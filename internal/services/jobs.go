package services

import (
	"context"

	"github.com/anujpunekar20/carrier/internal/ent"
)

type JobService struct {
	db *ent.Client
}

func NewJobService(db *ent.Client) *JobService {
	return &JobService{db}
}

func (j *JobService) ListJobs(ctx context.Context) ([]*ent.Job, error) {
	return j.db.Job.Query().All(ctx)
}
