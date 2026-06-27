package middleware

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

// ErrorHandler is a Fiber ErrorHandler that always returns JSON.
func ErrorHandler(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	msg := "internal server error"

	var fe *fiber.Error
	if errors.As(err, &fe) {
		code = fe.Code
		msg = fe.Message
	}

	return c.Status(code).JSON(fiber.Map{"error": msg})
}

// Register attaches global middleware to app in order: recover → logger → cors.
func Register(app *fiber.App) {
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())
}
