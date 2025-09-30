// Secrets API Component
// C4 Architecture: Secure storage of runner API keys
// Manages encrypted secrets in Kubernetes

package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SecretsAPI handles secure storage of API keys and credentials
type SecretsAPI struct {
	clientset kubernetes.Interface
}

// NewSecretsAPI creates a new Secrets API handler
func NewSecretsAPI(clientset kubernetes.Interface) *SecretsAPI {
	return &SecretsAPI{
		clientset: clientset,
	}
}

// SecretRequest represents a request to create/update a secret
type SecretRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        string            `json:"type"` // api-key, oauth-token, certificate
	Data        map[string]string `json:"data"`
}

// ListSecrets returns all secrets for a project (metadata only)
func (s *SecretsAPI) ListSecrets(c *gin.Context) {
	projectName := c.Param("project")
	namespace := fmt.Sprintf("vteam-%s", projectName)

	// List secrets in project namespace
	secrets, err := s.clientset.CoreV1().Secrets(namespace).List(
		context.TODO(),
		metav1.ListOptions{
			LabelSelector: "app=vteam,type=runner-secret",
		},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to list secrets: %v", err),
		})
		return
	}

	// Return metadata only, never expose secret data
	secretList := []map[string]interface{}{}
	for _, secret := range secrets.Items {
		secretList = append(secretList, map[string]interface{}{
			"name":        secret.Name,
			"description": secret.Annotations["description"],
			"type":        secret.Labels["secret-type"],
			"created":     secret.CreationTimestamp,
			"keys":        getSecretKeys(secret.Data),
		})
	}

	c.JSON(http.StatusOK, secretList)
}

// CreateSecret creates a new encrypted secret
func (s *SecretsAPI) CreateSecret(c *gin.Context) {
	projectName := c.Param("project")
	namespace := fmt.Sprintf("vteam-%s", projectName)

	var request SecretRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid secret data: %v", err),
		})
		return
	}

	// Validate secret type
	validTypes := map[string]bool{
		"api-key":     true,
		"oauth-token": true,
		"certificate": true,
		"credentials": true,
	}

	if !validTypes[request.Type] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid secret type. Must be: api-key, oauth-token, certificate, or credentials",
		})
		return
	}

	// Encode secret data
	encodedData := make(map[string][]byte)
	for key, value := range request.Data {
		encodedData[key] = []byte(value)
	}

	// Create Kubernetes Secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: namespace,
			Labels: map[string]string{
				"app":         "vteam",
				"type":        "runner-secret",
				"secret-type": request.Type,
				"project":     projectName,
				"owner":       c.GetString("user"),
			},
			Annotations: map[string]string{
				"description": request.Description,
				"created-by":  c.GetString("user"),
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: encodedData,
	}

	// Create the secret
	created, err := s.clientset.CoreV1().Secrets(namespace).Create(
		context.TODO(),
		secret,
		metav1.CreateOptions{},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create secret: %v", err),
		})
		return
	}

	// Return metadata only
	c.JSON(http.StatusCreated, gin.H{
		"name":        created.Name,
		"description": created.Annotations["description"],
		"type":        created.Labels["secret-type"],
		"created":     created.CreationTimestamp,
		"keys":        getSecretKeys(created.Data),
	})
}

// UpdateSecret updates an existing secret
func (s *SecretsAPI) UpdateSecret(c *gin.Context) {
	projectName := c.Param("project")
	secretName := c.Param("name")
	namespace := fmt.Sprintf("vteam-%s", projectName)

	var request SecretRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid secret data: %v", err),
		})
		return
	}

	// Get existing secret
	existing, err := s.clientset.CoreV1().Secrets(namespace).Get(
		context.TODO(),
		secretName,
		metav1.GetOptions{},
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Secret not found",
		})
		return
	}

	// Check ownership
	if existing.Labels["owner"] != c.GetString("user") && !c.GetBool("isAdmin") {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only secret owner can update the secret",
		})
		return
	}

	// Update secret data
	encodedData := make(map[string][]byte)
	for key, value := range request.Data {
		encodedData[key] = []byte(value)
	}

	existing.Data = encodedData
	if request.Description != "" {
		existing.Annotations["description"] = request.Description
	}

	// Update the secret
	updated, err := s.clientset.CoreV1().Secrets(namespace).Update(
		context.TODO(),
		existing,
		metav1.UpdateOptions{},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to update secret: %v", err),
		})
		return
	}

	// Return metadata only
	c.JSON(http.StatusOK, gin.H{
		"name":        updated.Name,
		"description": updated.Annotations["description"],
		"type":        updated.Labels["secret-type"],
		"updated":     updated.CreationTimestamp,
		"keys":        getSecretKeys(updated.Data),
	})
}

// DeleteSecret deletes a secret
func (s *SecretsAPI) DeleteSecret(c *gin.Context) {
	projectName := c.Param("project")
	secretName := c.Param("name")
	namespace := fmt.Sprintf("vteam-%s", projectName)

	// Get secret to check ownership
	secret, err := s.clientset.CoreV1().Secrets(namespace).Get(
		context.TODO(),
		secretName,
		metav1.GetOptions{},
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Secret not found",
		})
		return
	}

	// Check ownership
	if secret.Labels["owner"] != c.GetString("user") && !c.GetBool("isAdmin") {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only secret owner can delete the secret",
		})
		return
	}

	// Delete the secret
	err = s.clientset.CoreV1().Secrets(namespace).Delete(
		context.TODO(),
		secretName,
		metav1.DeleteOptions{},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to delete secret: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Secret %s deleted successfully", secretName),
	})
}

// getSecretKeys returns the keys in a secret without exposing values
func getSecretKeys(data map[string][]byte) []string {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	return keys
}