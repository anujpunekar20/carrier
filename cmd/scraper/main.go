package main

import (
	"context"
	"log"
	"time"

	"github.com/anujpunekar20/carrier/internal/config"
	"github.com/anujpunekar20/carrier/internal/database"
	"github.com/anujpunekar20/carrier/internal/ent/migrate"
	"github.com/anujpunekar20/carrier/internal/scraper"
	"github.com/anujpunekar20/carrier/internal/scraper/sites"
	"github.com/anujpunekar20/carrier/internal/services"
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
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := client.Schema.Create(
		ctx,
		migrate.WithDropIndex(true),
		migrate.WithDropColumn(true),
	); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	svc := services.NewJobService(client)

	runner := scraper.NewRunner(
		svc,
		sites.DefaultWeWorkRemotely(),
		sites.DefaultHackerNews(),
	)

	runner.RunAll(ctx)

	// print a count per source so you can verify the run worked
	total, err := client.Job.Query().Count(ctx)
	if err != nil {
		log.Printf("count query failed: %v", err)
		return
	}
	log.Printf("total jobs in DB: %d", total)
}
