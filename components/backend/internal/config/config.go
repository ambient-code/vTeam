package config

import (
	"os"
)

// AppConfig holds application configuration
type AppConfig struct {
	Namespace    string
	StateBaseDir string
	PVCBaseDir   string
	Port         string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *AppConfig {
	config := &AppConfig{
		Namespace:    getEnvOrDefault("NAMESPACE", "default"),
		StateBaseDir: getEnvOrDefault("STATE_BASE_DIR", "/data/state"),
		PVCBaseDir:   getEnvOrDefault("PVC_BASE_DIR", "/workspace"),
		Port:         getEnvOrDefault("PORT", "8080"),
	}
	return config
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// IsContentServiceMode returns true if running in content service mode
func IsContentServiceMode() bool {
	return os.Getenv("CONTENT_SERVICE_MODE") == "true"
}
