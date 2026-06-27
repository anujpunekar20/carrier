package database

import (
	"database/sql"
	"fmt"

	"github.com/anujpunekar20/carrier/internal/config"
	_ "github.com/lib/pq"
)

// Returns a new DB instance.
func NewDB(cfg config.Config) (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error initializing db: %w", err)
	}
	return db, nil
}
