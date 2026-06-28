package scraper

import (
	"context"
	"time"
)

// Job is the result type produced by site implementations.
// Source and ScrapedAt are intentionally absent — the Runner fills them in
// from Scraper.Name() and time.Now(), ensuring consistency across all sites.
type Job struct {
	Title          string
	Company        string
	URL            string
	Location       *string
	Salary         *string
	EmploymentType *string
	Description    *string
	PostedOn       *time.Time
}

// Scraper is implemented by every site-specific scraper.
type Scraper interface {
	// Name returns the canonical source label stored in the DB (e.g. "weworkremotely").
	// Must be lowercase and stable across runs.
	Name() string

	// Scrape fetches job listings from the site.
	// Returning a partial list alongside a non-nil error is valid;
	// the Runner persists whichever jobs were collected before the failure.
	Scrape(ctx context.Context) ([]Job, error)
}
