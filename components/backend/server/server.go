// Package server provides HTTP server setup, middleware, and routing configuration.
package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// RouterFunc is a function that can register routes on a Gin router
type RouterFunc func(r *gin.Engine)

// Run starts the server with the provided route registration function
func Run(registerRoutes RouterFunc) error {
	// Setup Gin router with custom logger that redacts tokens
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Redact token from query string
		path := param.Path
		if strings.Contains(param.Request.URL.RawQuery, "token=") {
			path = strings.Split(path, "?")[0] + "?token=[REDACTED]"
		}
		return fmt.Sprintf("[GIN] %s | %3d | %s | %s\n",
			param.Method,
			param.StatusCode,
			param.ClientIP,
			path,
		)
	}))

	// Middleware to populate user context from forwarded headers
	r.Use(forwardedIdentityMiddleware())

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	// Register routes
	registerRoutes(r)

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Printf("Using namespace: %s", Namespace)

	if err := r.Run(":" + port); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	return nil
}

// forwardedIdentityMiddleware populates Gin context from common OAuth proxy headers
func forwardedIdentityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if v := c.GetHeader("X-Forwarded-User"); v != "" {
			c.Set("userID", v)
		}
		// Prefer preferred username; fallback to user id
		name := c.GetHeader("X-Forwarded-Preferred-Username")
		if name == "" {
			name = c.GetHeader("X-Forwarded-User")
		}
		if name != "" {
			c.Set("userName", name)
		}
		if v := c.GetHeader("X-Forwarded-Email"); v != "" {
			c.Set("userEmail", v)
		}
		if v := c.GetHeader("X-Forwarded-Groups"); v != "" {
			c.Set("userGroups", strings.Split(v, ","))
		}
		// Also expose access token if present
		auth := c.GetHeader("Authorization")
		if auth != "" {
			c.Set("authorizationHeader", auth)
		}
		if v := c.GetHeader("X-Forwarded-Access-Token"); v != "" {
			c.Set("forwardedAccessToken", v)
		}
		c.Next()
	}
}

// RunContentService starts the server in content service mode
func RunContentService(registerContentRoutes RouterFunc) error {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		path := param.Path
		if strings.Contains(param.Request.URL.RawQuery, "token=") {
			path = strings.Split(path, "?")[0] + "?token=[REDACTED]"
		}
		return fmt.Sprintf("[GIN] %s | %3d | %s | %s\n",
			param.Method,
			param.StatusCode,
			param.ClientIP,
			path,
		)
	}))

	// Register content service routes
	registerContentRoutes(r)

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Content service starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		return fmt.Errorf("failed to start content service: %v", err)
	}
	return nil
}
