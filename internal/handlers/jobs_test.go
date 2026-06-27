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

	"github.com/anujpunekar20/carrier/internal/ent"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/anujpunekar20/carrier/internal/handlers"
	"github.com/anujpunekar20/carrier/internal/routes"
	"github.com/anujpunekar20/carrier/internal/services"
	"github.com/gofiber/fiber/v3"
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

// --- ListJobs ---

func TestListJobs_Empty(t *testing.T) {
	app, _ := setup(t)
	resp := do(t, app, http.MethodGet, "/api/v1/jobs/list", nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	var result services.ListJobsResult
	decode(t, resp, &result)
	if result.Total != 0 || len(result.Jobs) != 0 {
		t.Fatalf("want empty list, got total=%d", result.Total)
	}
	if result.Page != 1 || result.Limit != 20 {
		t.Fatalf("want page=1 limit=20, got page=%d limit=%d", result.Page, result.Limit)
	}
}

func TestListJobs_ReturnsAll(t *testing.T) {
	app, client := setup(t)
	seed(t, client, "Engineer", "Google", "NYC", "linkedin")
	seed(t, client, "Designer", "Apple", "SF", "indeed")

	resp := do(t, app, http.MethodGet, "/api/v1/jobs/list", nil)
	var result services.ListJobsResult
	decode(t, resp, &result)
	if result.Total != 2 || len(result.Jobs) != 2 {
		t.Fatalf("want 2 jobs, got total=%d", result.Total)
	}
}

func TestListJobs_SearchByQuery(t *testing.T) {
	app, client := setup(t)
	seed(t, client, "Software Engineer", "Google", "NYC", "linkedin")
	seed(t, client, "Product Designer", "Apple", "SF", "indeed")

	resp := do(t, app, http.MethodGet, "/api/v1/jobs/list?q=engineer", nil)
	var result services.ListJobsResult
	decode(t, resp, &result)
	if result.Total != 1 || result.Jobs[0].Title != "Software Engineer" {
		t.Fatalf("want 1 engineer job, got total=%d", result.Total)
	}
}

func TestListJobs_FilterByCompany(t *testing.T) {
	app, client := setup(t)
	seed(t, client, "Engineer", "Google", "NYC", "linkedin")
	seed(t, client, "Designer", "Apple", "SF", "indeed")

	resp := do(t, app, http.MethodGet, "/api/v1/jobs/list?company=google", nil)
	var result services.ListJobsResult
	decode(t, resp, &result)
	if result.Total != 1 || result.Jobs[0].Company != "Google" {
		t.Fatalf("want 1 Google job, got total=%d", result.Total)
	}
}

func TestListJobs_Pagination(t *testing.T) {
	app, client := setup(t)
	for i := range 5 {
		seed(t, client, fmt.Sprintf("Job %d", i), "Acme", "NY", "web")
	}

	resp := do(t, app, http.MethodGet, "/api/v1/jobs/list?page=2&limit=2", nil)
	var result services.ListJobsResult
	decode(t, resp, &result)
	if result.Total != 5 {
		t.Fatalf("want total=5, got %d", result.Total)
	}
	if len(result.Jobs) != 2 {
		t.Fatalf("want 2 jobs on page 2, got %d", len(result.Jobs))
	}
	if result.Page != 2 || result.Limit != 2 {
		t.Fatalf("want page=2 limit=2, got page=%d limit=%d", result.Page, result.Limit)
	}
}

// --- GetJob ---

func TestGetJob_Found(t *testing.T) {
	app, client := setup(t)
	j := seed(t, client, "Engineer", "Google", "NYC", "linkedin")

	resp := do(t, app, http.MethodGet, fmt.Sprintf("/api/v1/jobs/%d", j.ID), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	var got ent.Job
	decode(t, resp, &got)
	if got.ID != j.ID {
		t.Fatalf("want job id=%d, got %d", j.ID, got.ID)
	}
}

func TestGetJob_NotFound(t *testing.T) {
	app, _ := setup(t)
	resp := do(t, app, http.MethodGet, "/api/v1/jobs/9999", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404, got %d", resp.StatusCode)
	}
}

func TestGetJob_InvalidID(t *testing.T) {
	app, _ := setup(t)
	resp := do(t, app, http.MethodGet, "/api/v1/jobs/abc", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}
}

// --- CreateJob ---

func TestCreateJob_Success(t *testing.T) {
	app, _ := setup(t)
	body := map[string]any{
		"title":      "Engineer",
		"company":    "Google",
		"url":        "https://google.com/jobs/1",
		"source":     "linkedin",
		"scraped_at": time.Now(),
	}
	resp := do(t, app, http.MethodPost, "/api/v1/jobs/", body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("want 201, got %d", resp.StatusCode)
	}
	var got ent.Job
	decode(t, resp, &got)
	if got.Title != "Engineer" || got.Company != "Google" {
		t.Fatalf("unexpected job: %+v", got)
	}
}

func TestCreateJob_MissingRequiredFields(t *testing.T) {
	app, _ := setup(t)
	body := map[string]any{"title": "Engineer"}
	resp := do(t, app, http.MethodPost, "/api/v1/jobs/", body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}
}

func TestCreateJob_DuplicateURL(t *testing.T) {
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
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("want 409, got %d", resp.StatusCode)
	}
}

// --- DeleteJob ---

func TestDeleteJob_Success(t *testing.T) {
	app, client := setup(t)
	j := seed(t, client, "Engineer", "Google", "NYC", "linkedin")

	resp := do(t, app, http.MethodDelete, fmt.Sprintf("/api/v1/jobs/%d", j.ID), nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("want 204, got %d", resp.StatusCode)
	}
}

func TestDeleteJob_NotFound(t *testing.T) {
	app, _ := setup(t)
	resp := do(t, app, http.MethodDelete, "/api/v1/jobs/9999", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404, got %d", resp.StatusCode)
	}
}
