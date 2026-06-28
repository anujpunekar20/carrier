package sites

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/anujpunekar20/carrier/internal/scraper"
)

const hnItemBase = "https://news.ycombinator.com/item?id="

// HackerNews scrapes the monthly "Ask HN: Who is hiring?" thread
// via the Algolia HN search API (JSON — no HTML scraping of HN itself).
type HackerNews struct {
	client      *http.Client
	algoliaBase string
}

// NewHackerNews creates a HackerNews scraper.
// algoliaBase is the testability seam: production uses "https://hn.algolia.com",
// tests point it at a local httptest.Server.
func NewHackerNews(client *http.Client, algoliaBase string) *HackerNews {
	return &HackerNews{client: client, algoliaBase: algoliaBase}
}

// DefaultHackerNews returns a production-ready scraper.
func DefaultHackerNews() *HackerNews {
	return NewHackerNews(
		&http.Client{Timeout: 30 * time.Second},
		"https://hn.algolia.com",
	)
}

func (h *HackerNews) Name() string { return "hackernews" }

func (h *HackerNews) Scrape(ctx context.Context) ([]scraper.Job, error) {
	storyID, err := h.latestHiringPostID(ctx)
	if err != nil {
		return nil, fmt.Errorf("find hiring post: %w", err)
	}

	comments, err := h.fetchComments(ctx, storyID)
	if err != nil {
		return nil, fmt.Errorf("fetch comments for story %s: %w", storyID, err)
	}

	var jobs []scraper.Job
	for _, c := range comments {
		j, ok := parseHNComment(c)
		if !ok {
			continue
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// algoliaStoryHit is the shape of a hit from the Algolia stories search.
type algoliaStoryHit struct {
	ObjectID string `json:"objectID"`
	Title    string `json:"title"`
	Author   string `json:"author"`
}

// algoliaCommentHit is the shape of a hit from the Algolia comments search.
type algoliaCommentHit struct {
	ObjectID    string `json:"objectID"`
	CommentText string `json:"comment_text"`
	CreatedAt   string `json:"created_at"`
}

type algoliaSearchResult[T any] struct {
	Hits []T `json:"hits"`
}

func (h *HackerNews) latestHiringPostID(ctx context.Context) (string, error) {
	url := h.algoliaBase + "/api/v1/search?query=Ask+HN%3A+Who+is+hiring%3F&tags=story&restrictSearchableAttributes=title"

	var result algoliaSearchResult[algoliaStoryHit]
	if err := h.getJSON(ctx, url, &result); err != nil {
		return "", err
	}
	for _, hit := range result.Hits {
		if hit.Author == "whoishiring" && strings.Contains(hit.Title, "Who is hiring") {
			return hit.ObjectID, nil
		}
	}
	return "", fmt.Errorf("no hiring post found")
}

func (h *HackerNews) fetchComments(ctx context.Context, storyID string) ([]algoliaCommentHit, error) {
	url := fmt.Sprintf("%s/api/v1/search?tags=comment,story_%s&hitsPerPage=1000", h.algoliaBase, storyID)

	var result algoliaSearchResult[algoliaCommentHit]
	if err := h.getJSON(ctx, url, &result); err != nil {
		return nil, err
	}
	return result.Hits, nil
}

func (h *HackerNews) getJSON(ctx context.Context, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}

// parseHNComment attempts to extract a job from a raw HN comment.
// Returns (job, false) if the comment doesn't look like a job posting.
//
// Convention: first line of the comment is "Company | Role | Location | Type"
// (pipe-separated). Comments without at least one pipe are skipped — they are
// almost always meta-discussion, not job postings.
func parseHNComment(c algoliaCommentHit) (scraper.Job, bool) {
	if c.CommentText == "" {
		return scraper.Job{}, false
	}

	// comment_text arrives as HTML from Algolia; extract plain text paragraph by
	// paragraph so that <p> boundaries become newlines in the resulting string.
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(c.CommentText))
	if err != nil {
		return scraper.Job{}, false
	}

	var paragraphs []string
	doc.Find("p").Each(func(_ int, s *goquery.Selection) {
		if t := strings.TrimSpace(s.Text()); t != "" {
			paragraphs = append(paragraphs, t)
		}
	})
	if len(paragraphs) == 0 {
		// fallback: no <p> tags, use raw text
		if t := strings.TrimSpace(doc.Text()); t != "" {
			paragraphs = []string{t}
		}
	}
	if len(paragraphs) == 0 {
		return scraper.Job{}, false
	}

	fullText := strings.Join(paragraphs, "\n")
	firstLine := paragraphs[0]
	parts := strings.Split(firstLine, "|")
	if len(parts) < 2 {
		return scraper.Job{}, false // no pipe separator → not a job posting
	}

	company := strings.TrimSpace(parts[0])
	title := strings.TrimSpace(parts[1])
	if company == "" || title == "" {
		return scraper.Job{}, false
	}

	jobURL := hnItemBase + c.ObjectID
	j := scraper.Job{
		Title:       title,
		Company:     company,
		URL:         jobURL,
		Description: &fullText,
	}

	if len(parts) >= 3 {
		loc := strings.TrimSpace(parts[2])
		if loc != "" {
			j.Location = &loc
		}
	}

	if len(parts) >= 4 {
		et := strings.TrimSpace(parts[3])
		if et != "" {
			j.EmploymentType = &et
		}
	}

	if c.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, c.CreatedAt); err == nil {
			j.PostedOn = &t
		}
	}

	return j, true
}
