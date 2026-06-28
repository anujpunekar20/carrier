package services

import (
	"context"
	"time"

	"github.com/anujpunekar20/carrier/internal/ent"
	"github.com/anujpunekar20/carrier/internal/ent/job"
)

type JobService struct {
	db *ent.Client
}

func NewJobService(db *ent.Client) *JobService {
	return &JobService{db}
}

type ListJobsParams struct {
	Query          string
	Company        string
	Location       string
	EmploymentType string
	Source         string
	Page           int
	Limit          int
}

type ListJobsResult struct {
	Jobs  []*ent.Job `json:"jobs"`
	Total int        `json:"total"`
	Page  int        `json:"page"`
	Limit int        `json:"limit"`
}

type CreateJobInput struct {
	Title          string     `json:"title"`
	Company        string     `json:"company"`
	URL            string     `json:"url"`
	Source         string     `json:"source"`
	ScrapedAt      time.Time  `json:"scraped_at"`
	Location       *string    `json:"location"`
	Salary         *string    `json:"salary"`
	EmploymentType *string    `json:"employment_type"`
	Description    *string    `json:"description"`
	PostedOn       *time.Time `json:"posted_on"`
}

func (s *JobService) ListJobs(ctx context.Context, p ListJobsParams) (*ListJobsResult, error) {
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 20
	}
	if p.Page <= 0 {
		p.Page = 1
	}

	q := s.db.Job.Query()

	if p.Query != "" {
		q = q.Where(job.Or(
			job.TitleContainsFold(p.Query),
			job.DescriptionContainsFold(p.Query),
		))
	}
	if p.Company != "" {
		q = q.Where(job.CompanyContainsFold(p.Company))
	}
	if p.Location != "" {
		q = q.Where(job.LocationContainsFold(p.Location))
	}
	if p.EmploymentType != "" {
		q = q.Where(job.EmploymentTypeEqualFold(p.EmploymentType))
	}
	if p.Source != "" {
		q = q.Where(job.SourceEqualFold(p.Source))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}

	jobs, err := q.
		Order(job.ByID()).
		Offset((p.Page - 1) * p.Limit).
		Limit(p.Limit).
		All(ctx)
	if err != nil {
		return nil, err
	}

	return &ListJobsResult{
		Jobs:  jobs,
		Total: total,
		Page:  p.Page,
		Limit: p.Limit,
	}, nil
}

func (s *JobService) GetJob(ctx context.Context, id int) (*ent.Job, error) {
	return s.db.Job.Get(ctx, id)
}

func (s *JobService) CreateJob(ctx context.Context, input CreateJobInput) (*ent.Job, error) {
	c := s.db.Job.Create().
		SetTitle(input.Title).
		SetCompany(input.Company).
		SetURL(input.URL).
		SetSource(input.Source).
		SetScrapedAt(input.ScrapedAt)

	if input.Location != nil {
		c = c.SetLocation(*input.Location)
	}
	if input.Salary != nil {
		c = c.SetSalary(*input.Salary)
	}
	if input.EmploymentType != nil {
		c = c.SetEmploymentType(*input.EmploymentType)
	}
	if input.Description != nil {
		c = c.SetDescription(*input.Description)
	}
	if input.PostedOn != nil {
		c = c.SetPostedOn(*input.PostedOn)
	}

	return c.Save(ctx)
}

func (s *JobService) DeleteJob(ctx context.Context, id int) error {
	return s.db.Job.DeleteOneID(id).Exec(ctx)
}

type UpdateJobInput struct {
	Status *job.Status `json:"status"`
	Notes  *string     `json:"notes"`
}

func (s *JobService) UpdateJob(ctx context.Context, id int, input UpdateJobInput) (*ent.Job, error) {
	u := s.db.Job.UpdateOneID(id)
	if input.Status != nil {
		u = u.SetStatus(*input.Status)
	}
	if input.Notes != nil {
		u = u.SetNotes(*input.Notes)
	}
	return u.Save(ctx)
}
