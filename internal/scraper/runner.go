package scraper

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/anujpunekar20/carrier/internal/ent"
	"github.com/anujpunekar20/carrier/internal/services"
)

// Runner orchestrates a set of Scrapers and persists results via JobService.
type Runner struct {
	scrapers []Scraper
	jobs     *services.JobService
}

// NewRunner wires the runner. Follows the same constructor-injection pattern
// as the rest of the codebase: *ent.Client → JobService → Runner.
func NewRunner(jobs *services.JobService, scrapers ...Scraper) *Runner {
	return &Runner{jobs: jobs, scrapers: scrapers}
}

// RunAll runs every registered scraper sequentially.
// A scraper error is logged but does not abort subsequent scrapers.
func (r *Runner) RunAll(ctx context.Context) {
	for _, s := range r.scrapers {
		r.run(ctx, s)
	}
}

func (r *Runner) run(ctx context.Context, s Scraper) {
	jobs, scrapeErr := s.Scrape(ctx)
	if scrapeErr != nil {
		log.Printf("[scraper] %s: scrape error: %v", s.Name(), scrapeErr)
	}

	now := time.Now()
	saved, skipped, failed := 0, 0, 0

	for _, j := range jobs {
		loc := j.Location
		sal := j.Salary
		et := j.EmploymentType
		desc := j.Description
		po := j.PostedOn

		input := services.CreateJobInput{
			Title:          j.Title,
			Company:        j.Company,
			URL:            j.URL,
			Source:         s.Name(),
			ScrapedAt:      now,
			Location:       loc,
			Salary:         sal,
			EmploymentType: et,
			Description:    desc,
			PostedOn:       po,
		}

		_, err := r.jobs.CreateJob(ctx, input)
		switch {
		case err == nil:
			saved++
		case isConstraintError(err):
			skipped++ // duplicate URL — already in DB, user's edits preserved
		default:
			failed++
			log.Printf("[scraper] %s: save %q: %v", s.Name(), j.URL, err)
		}
	}

	log.Printf("[scraper] %s: saved=%d skipped=%d failed=%d", s.Name(), saved, skipped, failed)
}

func isConstraintError(err error) bool {
	var ce *ent.ConstraintError
	return errors.As(err, &ce)
}
