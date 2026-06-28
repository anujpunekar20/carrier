package scraper_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/anujpunekar20/carrier/internal/ent"
	"github.com/anujpunekar20/carrier/internal/ent/job"
	"github.com/anujpunekar20/carrier/internal/scraper"
	"github.com/anujpunekar20/carrier/internal/services"
	"go.akshayshah.org/attest"
	_ "modernc.org/sqlite"
)

// setup creates an isolated in-memory SQLite DB and returns the ent client and JobService.
func setup(t *testing.T) (*ent.Client, *services.JobService) {
	t.Helper()

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

	return client, services.NewJobService(client)
}

// stubScraper implements Scraper with canned data for use in tests.
type stubScraper struct {
	name string
	jobs []scraper.Job
	err  error
}

func (s *stubScraper) Name() string { return s.name }
func (s *stubScraper) Scrape(_ context.Context) ([]scraper.Job, error) {
	return s.jobs, s.err
}

func makeJob(title, company, url string) scraper.Job {
	return scraper.Job{Title: title, Company: company, URL: url}
}

func TestRunner_SavesNewJobs(t *testing.T) {
	client, svc := setup(t)
	stub := &stubScraper{
		name: "test-source",
		jobs: []scraper.Job{
			makeJob("Engineer", "Acme", "https://example.com/1"),
			makeJob("Designer", "Globex", "https://example.com/2"),
		},
	}

	scraper.NewRunner(svc, stub).RunAll(t.Context())

	count, err := client.Job.Query().Count(t.Context())
	attest.Ok(t, err)
	attest.Equal(t, count, 2)

	jobs, _ := client.Job.Query().All(t.Context())
	for _, j := range jobs {
		attest.Equal(t, j.Source, "test-source")
		attest.False(t, j.ScrapedAt.IsZero())
		attest.Equal(t, j.Status, job.StatusSaved) // default status
	}
}

func TestRunner_SkipsDuplicateURL(t *testing.T) {
	client, svc := setup(t)
	stub := &stubScraper{
		name: "test-source",
		jobs: []scraper.Job{makeJob("Engineer", "Acme", "https://example.com/1")},
	}
	runner := scraper.NewRunner(svc, stub)

	runner.RunAll(t.Context())
	runner.RunAll(t.Context()) // second run — same URL

	count, err := client.Job.Query().Count(t.Context())
	attest.Ok(t, err)
	attest.Equal(t, count, 1) // still only 1 row
}

func TestRunner_PreservesUserEdits(t *testing.T) {
	client, svc := setup(t)
	stub := &stubScraper{
		name: "test-source",
		jobs: []scraper.Job{makeJob("Engineer", "Acme", "https://example.com/1")},
	}
	runner := scraper.NewRunner(svc, stub)

	runner.RunAll(t.Context())

	// user marks the job as applied
	j := client.Job.Query().OnlyX(t.Context())
	client.Job.UpdateOneID(j.ID).SetStatus(job.StatusApplied).SaveX(t.Context())

	runner.RunAll(t.Context()) // re-run — same URL, should not overwrite

	updated, _ := client.Job.Get(t.Context(), j.ID)
	attest.Equal(t, updated.Status, job.StatusApplied) // preserved
}

func TestRunner_PartialScrapeError(t *testing.T) {
	client, svc := setup(t)
	stub := &stubScraper{
		name: "test-source",
		jobs: []scraper.Job{makeJob("Engineer", "Acme", "https://example.com/1")},
		err:  errors.New("network timeout"),
	}

	scraper.NewRunner(svc, stub).RunAll(t.Context())

	// partial results before the error should still be persisted
	count, err := client.Job.Query().Count(t.Context())
	attest.Ok(t, err)
	attest.Equal(t, count, 1)
}

func TestRunner_MultipleScrapers(t *testing.T) {
	client, svc := setup(t)
	s1 := &stubScraper{
		name: "source-a",
		jobs: []scraper.Job{makeJob("Engineer", "Acme", "https://a.com/1")},
	}
	s2 := &stubScraper{
		name: "source-b",
		jobs: []scraper.Job{makeJob("Designer", "Globex", "https://b.com/1")},
	}

	scraper.NewRunner(svc, s1, s2).RunAll(t.Context())

	jobs, err := client.Job.Query().Order(ent.Asc(job.FieldSource)).All(t.Context())
	attest.Ok(t, err)
	attest.Equal(t, len(jobs), 2)
	attest.Equal(t, jobs[0].Source, "source-a")
	attest.Equal(t, jobs[1].Source, "source-b")
}
