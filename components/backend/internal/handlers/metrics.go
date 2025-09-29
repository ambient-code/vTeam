package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetMetrics handles the /metrics endpoint
func GetMetrics(c *gin.Context) {
	// Simple metrics endpoint - could be extended with actual metrics
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"metrics": gin.H{
			"uptime": "unknown", // Could add actual uptime tracking
		},
	})
}

// HealthCheck handles the /health endpoint
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}