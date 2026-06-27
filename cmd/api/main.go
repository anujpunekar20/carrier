package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/anujpunekar20/carrier/internal/config"
	"github.com/anujpunekar20/carrier/internal/database"
	"github.com/anujpunekar20/carrier/internal/handlers"
	"github.com/anujpunekar20/carrier/internal/routes"
	"github.com/anujpunekar20/carrier/internal/services"
	"github.com/anujpunekar20/carrier/internal/ent/migrate"
	"github.com/gofiber/fiber/v3"
)

func main() {
	cfg, err := config.NewConfig(".env")
	if err != nil {
		log.Fatal(err)
	}

	client, err := database.NewDB(*cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err := client.Schema.Create(
		context.Background(),
		migrate.WithDropIndex(true),
		migrate.WithDropColumn(true),
	); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	svc := services.NewJobService(client)
	handler := handlers.NewJobHandler(svc)
	app := fiber.New()
	routes.Register(app, handler)

	go func() {
		if err := app.Listen(":3000"); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server stopped gracefully")
	if err := client.Close(); err != nil {
		log.Fatalf("DB shutdown failed: %v", err)
	}
}
