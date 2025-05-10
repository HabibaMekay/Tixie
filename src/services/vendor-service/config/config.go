package config

import (
	"fmt"
	"os"
)

type Config struct {
	EventServiceURL string
}

var AppConfig Config

func LoadConfig() error {
	eventServiceURL := os.Getenv("EVENT_SERVICE_URL")
	if eventServiceURL == "" {
		return fmt.Errorf("EVENT_SERVICE_URL environment variable is required")
	}

	AppConfig = Config{
		EventServiceURL: eventServiceURL,
	}
	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
