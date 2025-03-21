package config

import (
	"fmt"
	"os"
)

func LoadConfig() (*Config, error) {
	databaseURL := os.Getenv("GOOSE_DBSTRING")
	if databaseURL == "" {
		return nil, fmt.Errorf("%w: DATABASE_URL", ErrMissingEnv)
	}

	migrationDir := os.Getenv("GOOSE_MIGRATION_DIR")
	if migrationDir == "" {
		return nil, fmt.Errorf("%w: GOOSE_MIGRATION_DIR", ErrMissingEnv)
	}

	return &Config{
		DatabaseURL:  databaseURL,
		MigrationDir: migrationDir,
	}, nil
}
