package database

import (
	"fmt"

	"github.com/anujpunekar20/carrier/internal/config"
	"github.com/anujpunekar20/carrier/internal/ent"
	_ "github.com/lib/pq"
)

func NewDB(cfg config.Config) (*ent.Client, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)
	client, err := ent.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("error initializing db: %w", err)
	}
	return client, nil
}
