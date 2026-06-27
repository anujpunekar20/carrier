package handlers

import (
	"github.com/anujpunekar20/carrier/internal/services"
	"github.com/gofiber/fiber/v3"
)

type JobHandler struct {
	service *services.JobService
}

func NewJobHandler(service *services.JobService) *JobHandler {
	return &JobHandler{service}
}

func (j *JobHandler) ListJobs(c fiber.Ctx) error {
	jobs, err := j.service.ListJobs()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	return c.JSON(jobs)
}
