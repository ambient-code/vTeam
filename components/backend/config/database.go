package config

import (
	"fmt"
	"os"
	"strconv"
)

// DatabaseConfig contains the configuration for connecting to PostgreSQL
type DatabaseConfig struct {
	// Host is the database host
	Host string

	// Port is the database port
	Port int

	// Username is the database username
	Username string

	// Password is the database password
	Password string

	// Database is the name of the database to connect to
	Database string

	// SSLMode is the PostgreSQL SSL mode
	SSLMode string

	// MigrationsPath is the file path to database migrations
	MigrationsPath string

	// MaxConnections is the maximum number of connections in the pool
	MaxConnections int

	// MinConnections is the minimum number of connections in the pool
	MinConnections int

	// ConnectionTimeout is the maximum time (in seconds) to wait for a connection
	ConnectionTimeout int
}

// GetConnectionString returns a PostgreSQL connection string
func (c *DatabaseConfig) GetConnectionString() string {
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.SSLMode,
	)
}

// LoadDatabaseConfig loads database configuration from environment variables
func LoadDatabaseConfig() *DatabaseConfig {
	// Set defaults and override with environment variables if present
	config := &DatabaseConfig{
		Host:              getEnv("DB_HOST", "localhost"),
		Port:              getEnvAsInt("DB_PORT", 5432),
		Username:          getEnv("DB_USERNAME", "postgres"),
		Password:          getEnv("DB_PASSWORD", "postgres"),
		Database:          getEnv("DB_NAME", "ambient_code"),
		SSLMode:           getEnv("DB_SSLMODE", "disable"),
		MigrationsPath:    getEnv("DB_MIGRATIONS_PATH", "./db/migrations"),
		MaxConnections:    getEnvAsInt("DB_MAX_CONNECTIONS", 10),
		MinConnections:    getEnvAsInt("DB_MIN_CONNECTIONS", 2),
		ConnectionTimeout: getEnvAsInt("DB_CONNECTION_TIMEOUT", 5),
	}

	return config
}

// Helper function to get environment variable with fallback
func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// Helper function to get integer environment variable with fallback
func getEnvAsInt(key string, fallback int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return fallback
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return fallback
	}

	return value
}