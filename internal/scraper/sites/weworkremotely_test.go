package sites_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anujpunekar20/carrier/internal/scraper/sites"
	"go.akshayshah.org/attest"
)

func TestWeWorkRemotely_Scrape(t *testing.T) {
	t.Run("parses job cards correctly", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "testdata/weworkremotely_jobs.html")
		}))
		defer ts.Close()

		s := sites.NewWeWorkRemotely(ts.Client(), ts.URL)
		jobs, err := s.Scrape(context.Background())

		attest.Ok(t, err)
		// fixture has 4 cards; 2 are malformed (missing title, missing <a>) — expect 2
		attest.Equal(t, len(jobs), 2)

		attest.Equal(t, jobs[0].Title, "Software Engineer")
		attest.Equal(t, jobs[0].Company, "Acme Corp")
		attest.Equal(t, *jobs[0].Location, "Worldwide")
		attest.Equal(t, jobs[0].URL, ts.URL+"/remote-jobs/programming/111-software-engineer-acme")

		attest.Equal(t, jobs[1].Title, "Backend Developer")
		attest.Equal(t, jobs[1].Company, "Globex")
		attest.Equal(t, *jobs[1].Location, "USA Only")
	})

	t.Run("empty page returns empty slice", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<html><body><section class="jobs"><ul></ul></section></body></html>`))
		}))
		defer ts.Close()

		s := sites.NewWeWorkRemotely(ts.Client(), ts.URL)
		jobs, err := s.Scrape(context.Background())

		attest.Ok(t, err)
		attest.Equal(t, len(jobs), 0)
	})

	t.Run("non-200 response returns error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer ts.Close()

		s := sites.NewWeWorkRemotely(ts.Client(), ts.URL)
		_, err := s.Scrape(context.Background())

		attest.Error(t, err)
	})

	t.Run("context cancellation returns error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "testdata/weworkremotely_jobs.html")
		}))
		defer ts.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel before the request

		s := sites.NewWeWorkRemotely(ts.Client(), ts.URL)
		_, err := s.Scrape(ctx)

		attest.Error(t, err)
	})
}
