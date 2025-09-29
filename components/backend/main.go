package main

import (
	"log"

	"ambient-code-backend/internal/config"
	"ambient-code-backend/internal/routes"
)

func main() {
	// Initialize Kubernetes clients
	if err := config.InitK8sClients(); err != nil {
		log.Fatalf("Failed to initialize Kubernetes clients: %v", err)
	}

	// Load application configuration
	appConfig := config.LoadConfig()

	// Setup routes
	r := routes.SetupRoutes(appConfig)

	log.Printf("Server starting on port %s", appConfig.Port)
	log.Printf("Using namespace: %s", appConfig.Namespace)

	if err := r.Run(":" + appConfig.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
