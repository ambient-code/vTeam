package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// Pool is the global PostgreSQL connection pool
	Pool *pgxpool.Pool

	// ErrNoConnectionPool is returned when the database connection pool is not initialized
	ErrNoConnectionPool = errors.New("no database connection pool initialized")
)

// Initialize creates a new PostgreSQL connection pool and returns it
func Initialize(connString string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parse and validate connection config
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres connection string: %w", err)
	}

	// Set reasonable defaults for the connection pool
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	// Create the connection pool
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres connection pool: %w", err)
	}

	// Ping the database to verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}

	// Set the global connection pool
	Pool = pool

	log.Println("Successfully connected to PostgreSQL database")
	return pool, nil
}

// Close closes the connection pool
func Close() {
	if Pool != nil {
		Pool.Close()
		log.Println("PostgreSQL connection pool closed")
	}
}

// RunMigrations executes database migrations from the specified directory
func RunMigrations(migrationsPath, connString string) error {
	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		connString,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	if err == migrate.ErrNilVersion {
		log.Println("No migrations applied yet")
	} else {
		log.Printf("Migrations applied successfully. Current version: %d, Dirty: %t", version, dirty)
	}

	return nil
}