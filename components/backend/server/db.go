package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq" // PostgreSQL driver
)

var (
	DB *sql.DB
)

// InitDB initializes the database connection
func InitDB() error {
	pgHost := os.Getenv("POSTGRES_HOST")
	if pgHost == "" {
		pgHost = "postgres-service"
	}

	pgPort := os.Getenv("POSTGRES_PORT")
	if pgPort == "" {
		pgPort = "5432"
	}

	pgUser := os.Getenv("POSTGRES_USER")
	if pgUser == "" {
		pgUser = "langgraph"
	}

	pgPassword := os.Getenv("POSTGRES_PASSWORD")
	if pgPassword == "" {
		pgPassword = "langgraph-change-me"
	}

	pgDB := os.Getenv("POSTGRES_DB")
	if pgDB == "" {
		pgDB = "langgraph"
	}

	pgSSLMode := os.Getenv("POSTGRES_SSLMODE")
	if pgSSLMode == "" {
		pgSSLMode = "disable" // Default for internal cluster communication
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		pgHost, pgPort, pgUser, pgPassword, pgDB, pgSSLMode)

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %v", err)
	}

	// Test connection
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	// Create workflow tables if they don't exist
	if err := createWorkflowTables(); err != nil {
		return fmt.Errorf("failed to create workflow tables: %v", err)
	}

	log.Printf("Database connection initialized: %s@%s:%s/%s", pgUser, pgHost, pgPort, pgDB)
	return nil
}

// createWorkflowTables creates the workflow registry tables
func createWorkflowTables() error {
	workflowsTable := `
	CREATE TABLE IF NOT EXISTS workflows (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		owner TEXT NOT NULL,
		project TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		UNIQUE(project, name)
	)`

	workflowVersionsTable := `
	CREATE TABLE IF NOT EXISTS workflow_versions (
		id TEXT PRIMARY KEY,
		workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
		version TEXT NOT NULL,
		image_digest TEXT NOT NULL,
		graphs JSONB NOT NULL,
		inputs_schema JSONB,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		UNIQUE(workflow_id, version)
	)`

	runEventsTable := `
	CREATE TABLE IF NOT EXISTS run_events (
		id SERIAL PRIMARY KEY,
		run_id TEXT NOT NULL,
		seq INTEGER NOT NULL,
		ts TIMESTAMP NOT NULL DEFAULT NOW(),
		kind TEXT NOT NULL,
		checkpoint_id TEXT,
		payload JSONB,
		UNIQUE(run_id, seq)
	)`

	runEventsIdx := `
	CREATE INDEX IF NOT EXISTS idx_run_events_run_id ON run_events(run_id, ts DESC)`

	if _, err := DB.Exec(workflowsTable); err != nil {
		return fmt.Errorf("failed to create workflows table: %v", err)
	}

	if _, err := DB.Exec(workflowVersionsTable); err != nil {
		return fmt.Errorf("failed to create workflow_versions table: %v", err)
	}

	if _, err := DB.Exec(runEventsTable); err != nil {
		return fmt.Errorf("failed to create run_events table: %v", err)
	}

	if _, err := DB.Exec(runEventsIdx); err != nil {
		return fmt.Errorf("failed to create run_events index: %v", err)
	}

	log.Println("Workflow tables created/verified")
	return nil
}

// Helper functions for JSONB handling
func jsonbMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func jsonbUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

