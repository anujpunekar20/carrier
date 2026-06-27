package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/anujpunekar20/carrier/internal/config"
	"github.com/anujpunekar20/carrier/internal/database"
	"github.com/anujpunekar20/carrier/internal/handlers"
	"github.com/anujpunekar20/carrier/internal/routes"
	"github.com/anujpunekar20/carrier/internal/services"
	"github.com/gofiber/fiber/v3"
)

func main() {
	cfg, err := config.NewConfig(".env")
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.NewDB(*cfg)
	if err != nil {
		log.Fatal(err)
	}
	svc := services.NewJobService(db)
	handler := handlers.NewJobHandler(svc)
	app := fiber.New()
	routes.Register(app, handler)

	// Graceful shutdown: https://docs.gofiber.io/blog/fiber-v3-graceful-shutdown/#fibers-shutdown-method
	go func() {
		if err := app.Listen(":3000"); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server stopped gracefully")
	if err := db.Close(); err != nil {
		log.Fatalf("Db shutdown failed: %v", err)
	}
}
