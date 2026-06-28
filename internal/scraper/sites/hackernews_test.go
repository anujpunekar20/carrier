package sites_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anujpunekar20/carrier/internal/scraper/sites"
	"go.akshayshah.org/attest"
)

// hnFixture wires a test server that mimics the two Algolia API calls HackerNews makes.
func hnFixture(t *testing.T, storyID string, comments []map[string]any) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("query") != "" {
			// first call: story search
			json.NewEncoder(w).Encode(map[string]any{
				"hits": []map[string]any{
					{"objectID": storyID, "title": "Ask HN: Who is hiring? (June 2026)", "author": "whoishiring"},
				},
			})
		} else {
			// second call: comments
			json.NewEncoder(w).Encode(map[string]any{"hits": comments})
		}
	}))
	t.Cleanup(ts.Close)
	return ts
}

func TestHackerNews_Scrape(t *testing.T) {
	t.Run("parses well-formed comments", func(t *testing.T) {
		ts := hnFixture(t, "99999", []map[string]any{
			{
				"objectID":     "10001",
				"comment_text": "<p>Acme Corp | Software Engineer | Remote | Full-time</p><p>We are hiring engineers.</p>",
				"created_at":   "2026-06-01T12:00:00.000Z",
			},
			{
				"objectID":     "10002",
				"comment_text": "<p>Globex | Backend Developer | USA Only</p>",
				"created_at":   "2026-06-01T13:00:00.000Z",
			},
		})

		s := sites.NewHackerNews(ts.Client(), ts.URL)
		jobs, err := s.Scrape(context.Background())

		attest.Ok(t, err)
		attest.Equal(t, len(jobs), 2)

		attest.Equal(t, jobs[0].Company, "Acme Corp")
		attest.Equal(t, jobs[0].Title, "Software Engineer")
		attest.Equal(t, *jobs[0].Location, "Remote")
		attest.Equal(t, *jobs[0].EmploymentType, "Full-time")
		attest.Equal(t, jobs[0].URL, "https://news.ycombinator.com/item?id=10001")
		attest.True(t, jobs[0].PostedOn != nil)

		attest.Equal(t, jobs[1].Company, "Globex")
		attest.Equal(t, jobs[1].Title, "Backend Developer")
	})

	t.Run("skips comments without pipe separator", func(t *testing.T) {
		ts := hnFixture(t, "99999", []map[string]any{
			{"objectID": "10001", "comment_text": "<p>This is a meta discussion comment.</p>", "created_at": ""},
			{"objectID": "10002", "comment_text": "<p>Acme | Engineer | Remote</p>", "created_at": ""},
		})

		s := sites.NewHackerNews(ts.Client(), ts.URL)
		jobs, err := s.Scrape(context.Background())

		attest.Ok(t, err)
		attest.Equal(t, len(jobs), 1) // only the pipe-formatted one
	})

	t.Run("non-200 from Algolia returns error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer ts.Close()

		s := sites.NewHackerNews(ts.Client(), ts.URL)
		_, err := s.Scrape(context.Background())

		attest.Error(t, err)
	})

	t.Run("no hiring post found returns error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"hits": []any{}})
		}))
		defer ts.Close()

		s := sites.NewHackerNews(ts.Client(), ts.URL)
		_, err := s.Scrape(context.Background())

		attest.Error(t, err)
	})
}
