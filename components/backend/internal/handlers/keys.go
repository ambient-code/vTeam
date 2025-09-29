package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"time"

	"ambient-code-backend/internal/middleware"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ProjectKeyItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"createdAt"`
	LastUsed    string `json:"lastUsed,omitempty"`
}

// ListProjectKeys lists all access keys for a project
func ListProjectKeys(c *gin.Context) {
	project := c.GetString("project")
	reqK8s, _ := middleware.GetK8sClientsForRequest(c)

	// List secrets with the access key label
	secrets, err := reqK8s.CoreV1().Secrets(project).List(context.TODO(), v1.ListOptions{
		LabelSelector: "ambient-code.io/access-key=true",
	})
	if err != nil {
		log.Printf("Failed to list access key secrets in %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list access keys"})
		return
	}

	keys := []ProjectKeyItem{}
	for _, secret := range secrets.Items {
		key := ProjectKeyItem{
			ID:   secret.Name,
			Name: secret.Annotations["ambient-code.io/key-name"],
		}
		if desc := secret.Annotations["ambient-code.io/description"]; desc != "" {
			key.Description = desc
		}
		if !secret.CreationTimestamp.IsZero() {
			key.CreatedAt = secret.CreationTimestamp.Time.Format(time.RFC3339)
		}
		if lastUsed := secret.Annotations["ambient-code.io/last-used"]; lastUsed != "" {
			key.LastUsed = lastUsed
		}
		keys = append(keys, key)
	}

	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

// CreateProjectKey creates a new access key for a project
func CreateProjectKey(c *gin.Context) {
	project := c.GetString("project")
	reqK8s, _ := middleware.GetK8sClientsForRequest(c)

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description,omitempty"`
		ExpiresAt   string `json:"expiresAt,omitempty"` // RFC3339 format
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Generate a secure random key
	keyBytes := make([]byte, 32) // 256-bit key
	if _, err := rand.Read(keyBytes); err != nil {
		log.Printf("Failed to generate random key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access key"})
		return
	}
	keyValue := base64.StdEncoding.EncodeToString(keyBytes)

	// Create a unique key ID
	keyId := fmt.Sprintf("key-%d", time.Now().Unix())

	// Create secret to store the key
	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      keyId,
			Namespace: project,
			Labels: map[string]string{
				"app":                          "ambient-access-keys",
				"ambient-code.io/access-key":   "true",
				"ambient-code.io/managed":      "true",
			},
			Annotations: map[string]string{
				"ambient-code.io/key-name": req.Name,
				"ambient-code.io/created":  time.Now().UTC().Format(time.RFC3339),
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte(keyValue),
		},
	}

	if req.Description != "" {
		secret.Annotations["ambient-code.io/description"] = req.Description
	}
	if req.ExpiresAt != "" {
		// Validate the expiration date format
		if _, err := time.Parse(time.RFC3339, req.ExpiresAt); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expiresAt format, use RFC3339"})
			return
		}
		secret.Annotations["ambient-code.io/expires-at"] = req.ExpiresAt
	}

	_, err := reqK8s.CoreV1().Secrets(project).Create(context.TODO(), secret, v1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create access key secret in %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create access key"})
		return
	}

	// Return the key details (including the key value for this one time)
	response := gin.H{
		"id":        keyId,
		"name":      req.Name,
		"key":       keyValue, // Only returned once during creation
		"createdAt": secret.Annotations["ambient-code.io/created"],
	}
	if req.Description != "" {
		response["description"] = req.Description
	}
	if req.ExpiresAt != "" {
		response["expiresAt"] = req.ExpiresAt
	}

	c.JSON(http.StatusCreated, response)
}

// DeleteProjectKey deletes an access key from a project
func DeleteProjectKey(c *gin.Context) {
	project := c.GetString("project")
	keyId := c.Param("keyId")
	reqK8s, _ := middleware.GetK8sClientsForRequest(c)

	// Verify the secret exists and is an access key
	secret, err := reqK8s.CoreV1().Secrets(project).Get(context.TODO(), keyId, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Access key not found"})
			return
		}
		log.Printf("Failed to get access key secret %s in %s: %v", keyId, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get access key"})
		return
	}

	// Verify it's actually an access key
	if secret.Labels["ambient-code.io/access-key"] != "true" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Access key not found"})
		return
	}

	// Delete the secret
	err = reqK8s.CoreV1().Secrets(project).Delete(context.TODO(), keyId, v1.DeleteOptions{})
	if err != nil {
		log.Printf("Failed to delete access key secret %s in %s: %v", keyId, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete access key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Access key deleted successfully"})
}