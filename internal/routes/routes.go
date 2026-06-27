package routes

import (
	"github.com/anujpunekar20/carrier/internal/handlers"
	"github.com/gofiber/fiber/v3"
)

func Register(app *fiber.App, jobHandler *handlers.JobHandler) {
	api := app.Group("/api/v1")
	jobs := api.Group("/jobs")

	jobs.Get("/list", jobHandler.ListJobs)
	jobs.Get("/:id", jobHandler.GetJob)
	jobs.Post("/", jobHandler.CreateJob)
	jobs.Delete("/:id", jobHandler.DeleteJob)
}
