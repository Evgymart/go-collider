package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL string
	ServerPort  string
	AppEnv      string
}

func Load() (*Config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	return &Config{
		DatabaseURL: databaseURL,
		ServerPort:  serverPort,
		AppEnv:      os.Getenv("APP_ENV"),
	}, nil
}
