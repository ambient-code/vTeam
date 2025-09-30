// Backend API Service for vTeam Platform
// Based on C4 Architecture: REST API for managing Kubernetes Custom Resources
// Technologies: Go, Gin Framework

package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jeremyeder/vteam/backend/internal/api"
	"github.com/jeremyeder/vteam/backend/internal/k8s"
	"github.com/jeremyeder/vteam/backend/internal/middleware"
)

func main() {
	// Initialize Kubernetes client
	k8sClient, err := k8s.NewClient()
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}

	// Initialize Gin router
	router := gin.Default()

	// Configure CORS for frontend communication
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", os.Getenv("FRONTEND_URL")},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Initialize RBAC middleware
	rbacMiddleware := middleware.NewRBACMiddleware(k8sClient.Clientset)

	// API v1 routes group
	v1 := router.Group("/api/v1")
	v1.Use(rbacMiddleware.Authorize())

	// Initialize API handlers with Kubernetes client
	projectsAPI := api.NewProjectsAPI(k8sClient.DynamicClient)
	sessionsAPI := api.NewSessionsAPI(k8sClient.DynamicClient)
	secretsAPI := api.NewSecretsAPI(k8sClient.Clientset)

	// Projects API endpoints
	v1.GET("/projects", projectsAPI.ListProjects)
	v1.POST("/projects", projectsAPI.CreateProject)
	v1.GET("/projects/:name", projectsAPI.GetProject)
	v1.PUT("/projects/:name", projectsAPI.UpdateProject)
	v1.DELETE("/projects/:name", projectsAPI.DeleteProject)

	// Sessions API endpoints
	v1.GET("/projects/:project/sessions", sessionsAPI.ListSessions)
	v1.POST("/projects/:project/sessions", sessionsAPI.CreateSession)
	v1.GET("/projects/:project/sessions/:name", sessionsAPI.GetSession)
	v1.PUT("/projects/:project/sessions/:name", sessionsAPI.UpdateSession)
	v1.DELETE("/projects/:project/sessions/:name", sessionsAPI.DeleteSession)
	v1.GET("/projects/:project/sessions/:name/status", sessionsAPI.GetSessionStatus)

	// Secrets API endpoints
	v1.GET("/projects/:project/secrets", secretsAPI.ListSecrets)
	v1.POST("/projects/:project/secrets", secretsAPI.CreateSecret)
	v1.PUT("/projects/:project/secrets/:name", secretsAPI.UpdateSecret)
	v1.DELETE("/projects/:project/secrets/:name", secretsAPI.DeleteSecret)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"service": "vteam-backend",
		})
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting vTeam Backend API on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}