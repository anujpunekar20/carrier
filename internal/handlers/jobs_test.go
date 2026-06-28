package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/anujpunekar20/carrier/internal/ent"
	"github.com/anujpunekar20/carrier/internal/handlers"
	"github.com/anujpunekar20/carrier/internal/routes"
	"github.com/anujpunekar20/carrier/internal/services"
	"github.com/gofiber/fiber/v3"
	"go.akshayshah.org/attest"
	_ "modernc.org/sqlite"
)

// setup creates an isolated in-memory SQLite DB, wires the full stack, and
// returns the Fiber app and the ent client for seeding data.
func setup(t *testing.T) (*fiber.App, *ent.Client) {
	t.Helper()

	// Open raw DB with modernc's "sqlite" driver, then tell ent to treat it
	// as the "sqlite3" dialect via entsql.OpenDB.
	// - Unique name per test gives each test an isolated in-memory DB.
	// - MaxOpenConns(1) keeps a single connection so PRAGMA sticks.
	// - PRAGMA foreign_keys = ON is required by ent's schema migrator.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable foreign_keys: %v", err)
	}

	drv := entsql.OpenDB("sqlite3", db)
	client := ent.NewClient(ent.Driver(drv))

	if err := client.Schema.Create(t.Context()); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}

	t.Cleanup(func() {
		client.Close()
		db.Close()
	})

	svc := services.NewJobService(client)
	handler := handlers.NewJobHandler(svc)

	app := fiber.New()
	routes.Register(app, handler)

	return app, client
}

// seed inserts a job and returns it.
func seed(t *testing.T, client *ent.Client, title, company, location, source string) *ent.Job {
	t.Helper()
	j, err := client.Job.Create().
		SetTitle(title).
		SetCompany(company).
		SetLocation(location).
		SetURL(fmt.Sprintf("https://example.com/%s/%s", company, title)).
		SetSource(source).
		SetScrapedAt(time.Now()).
		Save(t.Context())
	if err != nil {
		t.Fatalf("seed job: %v", err)
	}
	return j
}

// do sends a test request and returns the response.
func do(t *testing.T, app *fiber.App, method, path string, body any) *http.Response {
	t.Helper()
	var b *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		b = bytes.NewReader(raw)
	} else {
		b = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, b)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	return resp
}

// decode decodes the response body into v.
func decode(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func TestJobHandlers(t *testing.T) {
	t.Run("ListJobs", func(t *testing.T) {
		t.Run("empty", func(t *testing.T) {
			app, _ := setup(t)
			resp := do(t, app, http.MethodGet, "/api/v1/jobs/list", nil)

			attest.Equal(t, resp.StatusCode, http.StatusOK)

			var result services.ListJobsResult
			decode(t, resp, &result)
			attest.Equal(t, result.Total, 0)
			attest.Equal(t, len(result.Jobs), 0)
			attest.Equal(t, result.Page, 1)
			attest.Equal(t, result.Limit, 20)
		})

		t.Run("returns all", func(t *testing.T) {
			app, client := setup(t)
			seed(t, client, "Engineer", "Google", "NYC", "linkedin")
			seed(t, client, "Designer", "Apple", "SF", "indeed")

			resp := do(t, app, http.MethodGet, "/api/v1/jobs/list", nil)
			var result services.ListJobsResult
			decode(t, resp, &result)
			attest.Equal(t, result.Total, 2)
			attest.Equal(t, len(result.Jobs), 2)
		})

		t.Run("search by query", func(t *testing.T) {
			app, client := setup(t)
			seed(t, client, "Software Engineer", "Google", "NYC", "linkedin")
			seed(t, client, "Product Designer", "Apple", "SF", "indeed")

			resp := do(t, app, http.MethodGet, "/api/v1/jobs/list?q=engineer", nil)
			var result services.ListJobsResult
			decode(t, resp, &result)
			attest.Equal(t, result.Total, 1)
			attest.Equal(t, result.Jobs[0].Title, "Software Engineer")
		})

		t.Run("filter by company", func(t *testing.T) {
			app, client := setup(t)
			seed(t, client, "Engineer", "Google", "NYC", "linkedin")
			seed(t, client, "Designer", "Apple", "SF", "indeed")

			resp := do(t, app, http.MethodGet, "/api/v1/jobs/list?company=google", nil)
			var result services.ListJobsResult
			decode(t, resp, &result)
			attest.Equal(t, result.Total, 1)
			attest.Equal(t, result.Jobs[0].Company, "Google")
		})

		t.Run("pagination", func(t *testing.T) {
			app, client := setup(t)
			for i := range 5 {
				seed(t, client, fmt.Sprintf("Job %d", i), "Acme", "NY", "web")
			}

			resp := do(t, app, http.MethodGet, "/api/v1/jobs/list?page=2&limit=2", nil)
			var result services.ListJobsResult
			decode(t, resp, &result)
			attest.Equal(t, result.Total, 5)
			attest.Equal(t, len(result.Jobs), 2)
			attest.Equal(t, result.Page, 2)
			attest.Equal(t, result.Limit, 2)
		})
	})

	t.Run("GetJob", func(t *testing.T) {
		t.Run("found", func(t *testing.T) {
			app, client := setup(t)
			j := seed(t, client, "Engineer", "Google", "NYC", "linkedin")

			resp := do(t, app, http.MethodGet, fmt.Sprintf("/api/v1/jobs/%d", j.ID), nil)
			attest.Equal(t, resp.StatusCode, http.StatusOK)

			var got ent.Job
			decode(t, resp, &got)
			attest.Equal(t, got.ID, j.ID)
		})

		t.Run("not found", func(t *testing.T) {
			app, _ := setup(t)
			resp := do(t, app, http.MethodGet, "/api/v1/jobs/9999", nil)
			attest.Equal(t, resp.StatusCode, http.StatusNotFound)
		})

		t.Run("invalid id", func(t *testing.T) {
			app, _ := setup(t)
			resp := do(t, app, http.MethodGet, "/api/v1/jobs/abc", nil)
			attest.Equal(t, resp.StatusCode, http.StatusBadRequest)
		})
	})

	t.Run("CreateJob", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			app, _ := setup(t)
			body := map[string]any{
				"title":      "Engineer",
				"company":    "Google",
				"url":        "https://google.com/jobs/1",
				"source":     "linkedin",
				"scraped_at": time.Now(),
			}
			resp := do(t, app, http.MethodPost, "/api/v1/jobs/", body)
			attest.Equal(t, resp.StatusCode, http.StatusCreated)

			var got ent.Job
			decode(t, resp, &got)
			attest.Equal(t, got.Title, "Engineer")
			attest.Equal(t, got.Company, "Google")
		})

		t.Run("missing required fields", func(t *testing.T) {
			app, _ := setup(t)
			body := map[string]any{"title": "Engineer"}
			resp := do(t, app, http.MethodPost, "/api/v1/jobs/", body)
			attest.Equal(t, resp.StatusCode, http.StatusBadRequest)
		})

		t.Run("duplicate url", func(t *testing.T) {
			app, client := setup(t)
			seed(t, client, "Engineer", "Google", "NYC", "linkedin")

			body := map[string]any{
				"title":      "Another Engineer",
				"company":    "Google",
				"url":        "https://example.com/Google/Engineer", // same URL as seeded job
				"source":     "indeed",
				"scraped_at": time.Now(),
			}
			resp := do(t, app, http.MethodPost, "/api/v1/jobs/", body)
			attest.Equal(t, resp.StatusCode, http.StatusConflict)
		})
	})

	t.Run("UpdateJob", func(t *testing.T) {
		t.Run("update status", func(t *testing.T) {
			app, client := setup(t)
			j := seed(t, client, "Engineer", "Google", "NYC", "linkedin")

			resp := do(t, app, http.MethodPatch, fmt.Sprintf("/api/v1/jobs/%d", j.ID), map[string]any{
				"status": "applied",
			})
			attest.Equal(t, resp.StatusCode, http.StatusOK)

			var got ent.Job
			decode(t, resp, &got)
			attest.Equal(t, got.Status.String(), "applied")
		})

		t.Run("update notes", func(t *testing.T) {
			app, client := setup(t)
			j := seed(t, client, "Engineer", "Google", "NYC", "linkedin")

			resp := do(t, app, http.MethodPatch, fmt.Sprintf("/api/v1/jobs/%d", j.ID), map[string]any{
				"notes": "reached out to recruiter",
			})
			attest.Equal(t, resp.StatusCode, http.StatusOK)

			var got ent.Job
			decode(t, resp, &got)
			attest.Equal(t, got.Notes, "reached out to recruiter")
		})

		t.Run("update status and notes together", func(t *testing.T) {
			app, client := setup(t)
			j := seed(t, client, "Engineer", "Google", "NYC", "linkedin")

			resp := do(t, app, http.MethodPatch, fmt.Sprintf("/api/v1/jobs/%d", j.ID), map[string]any{
				"status": "interview",
				"notes":  "phone screen scheduled",
			})
			attest.Equal(t, resp.StatusCode, http.StatusOK)

			var got ent.Job
			decode(t, resp, &got)
			attest.Equal(t, got.Status.String(), "interview")
			attest.Equal(t, got.Notes, "phone screen scheduled")
		})

		t.Run("invalid status", func(t *testing.T) {
			app, client := setup(t)
			j := seed(t, client, "Engineer", "Google", "NYC", "linkedin")

			resp := do(t, app, http.MethodPatch, fmt.Sprintf("/api/v1/jobs/%d", j.ID), map[string]any{
				"status": "ghosted",
			})
			attest.Equal(t, resp.StatusCode, http.StatusBadRequest)
		})

		t.Run("not found", func(t *testing.T) {
			app, _ := setup(t)
			resp := do(t, app, http.MethodPatch, "/api/v1/jobs/9999", map[string]any{
				"status": "applied",
			})
			attest.Equal(t, resp.StatusCode, http.StatusNotFound)
		})
	})

	t.Run("DeleteJob", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			app, client := setup(t)
			j := seed(t, client, "Engineer", "Google", "NYC", "linkedin")

			resp := do(t, app, http.MethodDelete, fmt.Sprintf("/api/v1/jobs/%d", j.ID), nil)
			attest.Equal(t, resp.StatusCode, http.StatusNoContent)
		})

		t.Run("not found", func(t *testing.T) {
			app, _ := setup(t)
			resp := do(t, app, http.MethodDelete, "/api/v1/jobs/9999", nil)
			attest.Equal(t, resp.StatusCode, http.StatusNotFound)
		})
	})
}
