package sites

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/anujpunekar20/carrier/internal/scraper"
)

// WeWorkRemotely scrapes job listings from weworkremotely.com.
type WeWorkRemotely struct {
	client  *http.Client
	baseURL string
}

// NewWeWorkRemotely creates a WeWorkRemotely scraper.
// baseURL is the testability seam: production uses "https://weworkremotely.com",
// tests point it at a local httptest.Server.
func NewWeWorkRemotely(client *http.Client, baseURL string) *WeWorkRemotely {
	return &WeWorkRemotely{client: client, baseURL: baseURL}
}

// DefaultWeWorkRemotely returns a production-ready scraper.
func DefaultWeWorkRemotely() *WeWorkRemotely {
	return NewWeWorkRemotely(
		&http.Client{Timeout: 30 * time.Second},
		"https://weworkremotely.com",
	)
}

func (w *WeWorkRemotely) Name() string { return "weworkremotely" }

func (w *WeWorkRemotely) Scrape(ctx context.Context) ([]scraper.Job, error) {
	url := w.baseURL + "/categories/remote-programming-jobs"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; carrier-scraper/1.0)")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var jobs []scraper.Job

	doc.Find("section.jobs li.feature").Each(func(_ int, sel *goquery.Selection) {
		title := strings.TrimSpace(sel.Find("span.title").Text())
		company := strings.TrimSpace(sel.Find("span.company").Text())
		region := strings.TrimSpace(sel.Find("span.region").Text())
		href, exists := sel.Find("a").Attr("href")

		if title == "" || company == "" || !exists || href == "" {
			return // skip malformed or incomplete cards
		}

		jobURL := w.baseURL + href
		j := scraper.Job{
			Title:   title,
			Company: company,
			URL:     jobURL,
		}
		if region != "" {
			j.Location = &region
		}
		jobs = append(jobs, j)
	})

	return jobs, nil
}
