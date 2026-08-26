package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv         string
	Port           string
	DatabaseURL    string
	ClerkSecretKey string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	config := &Config{
		AppEnv:         os.Getenv("APP_ENV"),
		Port:           os.Getenv("PORT"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		ClerkSecretKey: os.Getenv("CLERK_SECRET_KEY"),
	}

	if config.AppEnv == "" {
		config.AppEnv = "development"
	}

	if config.Port == "" {
		config.Port = "8080"
	}

	if config.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if config.ClerkSecretKey == "" {
		return nil, fmt.Errorf("CLERK_SECRET_KEY is required")
	}

	return config, nil
}
