package main

import (
	"log"

	"ambient-code-operator/internal/config"
	"ambient-code-operator/internal/handlers"
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
