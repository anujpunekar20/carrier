package handlers

import (
	"errors"
	"strconv"

	"github.com/anujpunekar20/carrier/internal/ent"
	"github.com/anujpunekar20/carrier/internal/ent/job"
	"github.com/anujpunekar20/carrier/internal/services"
	"github.com/gofiber/fiber/v3"
)

type JobHandler struct {
	service *services.JobService
}

func NewJobHandler(service *services.JobService) *JobHandler {
	return &JobHandler{service}
}

func (h *JobHandler) ListJobs(c fiber.Ctx) error {
	params := services.ListJobsParams{
		Query:          c.Query("q"),
		Company:        c.Query("company"),
		Location:       c.Query("location"),
		EmploymentType: c.Query("employment_type"),
		Source:         c.Query("source"),
		Page:           fiber.Query[int](c, "page", 1),
		Limit:          fiber.Query[int](c, "limit", 20),
	}

	result, err := h.service.ListJobs(c.Context(), params)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	return c.JSON(result)
}

func (h *JobHandler) GetJob(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid id",
		})
	}

	j, err := h.service.GetJob(c.Context(), id)
	if err != nil {
		if ent.IsNotFound(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "job not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	return c.JSON(j)
}

func (h *JobHandler) CreateJob(c fiber.Ctx) error {
	var input services.CreateJobInput
	if err := c.Bind().JSON(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if input.Title == "" || input.Company == "" || input.URL == "" || input.Source == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "title, company, url, and source are required",
		})
	}
	if input.ScrapedAt.IsZero() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "scraped_at is required",
		})
	}

	j, err := h.service.CreateJob(c.Context(), input)
	if err != nil {
		var constraintErr *ent.ConstraintError
		if errors.As(err, &constraintErr) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "a job with this url already exists",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	return c.Status(fiber.StatusCreated).JSON(j)
}

func (h *JobHandler) UpdateJob(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid id",
		})
	}

	var input services.UpdateJobInput
	if err := c.Bind().JSON(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if input.Status != nil {
		if err := job.StatusValidator(*input.Status); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "status must be one of: saved, applied, interview, offer, rejected",
			})
		}
	}

	j, err := h.service.UpdateJob(c.Context(), id, input)
	if err != nil {
		if ent.IsNotFound(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "job not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	return c.JSON(j)
}

func (h *JobHandler) DeleteJob(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid id",
		})
	}

	if err := h.service.DeleteJob(c.Context(), id); err != nil {
		if ent.IsNotFound(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "job not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
