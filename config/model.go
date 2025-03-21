package config

import "errors"

type Config struct {
	DatabaseURL  string
	MigrationDir string
}

var ErrMissingEnv = errors.New("missing environment variable")
