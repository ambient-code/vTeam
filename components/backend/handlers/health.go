package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health returns a simple health check handler
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}
