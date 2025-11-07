package main

import (
	"log"
	"os"

	"ambient-code-operator/internal/config"
	"ambient-code-operator/internal/handlers"
	"ambient-code-operator/internal/preflight"
)

func main() {
	// Initialize Kubernetes clients
	if err := config.InitK8sClients(); err != nil {
		log.Fatalf("Failed to initialize Kubernetes clients: %v", err)
	}

	// Load application configuration
	appConfig := config.LoadConfig()

	log.Printf("Agentic Session Operator starting in namespace: %s", appConfig.Namespace)
	log.Printf("Using ambient-code runner image: %s", appConfig.AmbientCodeRunnerImage)

	// Validate Vertex AI configuration at startup if enabled
	if os.Getenv("CLAUDE_CODE_USE_VERTEX") == "1" {
		if err := preflight.ValidateVertexConfig(appConfig.Namespace); err != nil {
			log.Fatalf("Vertex AI validation failed: %v", err)
		}
	}

	// Start watching AgenticSession resources
	go handlers.WatchAgenticSessions()

	// Start watching for managed namespaces
	go handlers.WatchNamespaces()

	// Start watching ProjectSettings resources
	go handlers.WatchProjectSettings()

	// Start cleanup of expired temporary content pods
	go handlers.CleanupExpiredTempContentPods()

	// Keep the operator running
	select {}
}
