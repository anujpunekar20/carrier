package services

import (
	"database/sql"

	"github.com/anujpunekar20/carrier/internal/models"
)

type JobService struct {
	db *sql.DB
}

func NewJobService(db *sql.DB) *JobService {
	return &JobService{db}
}

func (j *JobService) ListJobs() ([]models.Job, error) {
	rows, err := j.db.Query("SELECT id, name FROM jobs")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []models.Job{}

	for rows.Next() {
		var job models.Job

		if err := rows.Scan(&job.ID, &job.Name); err != nil {
			return nil, err
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return jobs, nil
}
