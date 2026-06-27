package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anujpunekar20/carrier/internal/middleware"
	"github.com/gofiber/fiber/v3"
	"go.akshayshah.org/attest"
)

func newApp() *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	middleware.Register(app)
	return app
}

func TestErrorHandler(t *testing.T) {
	t.Run("fiber error maps to correct status and JSON body", func(t *testing.T) {
		app := newApp()
		app.Get("/teapot", func(c fiber.Ctx) error {
			return fiber.NewError(fiber.StatusTeapot, "i am a teapot")
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/teapot", nil))
		attest.Ok(t, err)
		attest.Equal(t, resp.StatusCode, http.StatusTeapot)
		attest.Equal(t, resp.Header.Get("Content-Type"), "application/json; charset=utf-8")
	})

	t.Run("generic error maps to 500", func(t *testing.T) {
		app := newApp()
		app.Get("/boom", func(c fiber.Ctx) error {
			return errors.New("something went wrong")
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/boom", nil))
		attest.Ok(t, err)
		attest.Equal(t, resp.StatusCode, http.StatusInternalServerError)
	})
}

func TestRecoverMiddleware(t *testing.T) {
	app := newApp()
	app.Get("/panic", func(c fiber.Ctx) error {
		panic("unexpected panic")
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/panic", nil))
	attest.Ok(t, err)
	attest.Equal(t, resp.StatusCode, http.StatusInternalServerError)
}

func TestCORSMiddleware(t *testing.T) {
	app := newApp()
	app.Get("/ping", func(c fiber.Ctx) error {
		return c.SendString("pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	resp, err := app.Test(req)
	attest.Ok(t, err)
	attest.Equal(t, resp.StatusCode, http.StatusOK)
	attest.Equal(t, resp.Header.Get("Access-Control-Allow-Origin"), "*")
}
