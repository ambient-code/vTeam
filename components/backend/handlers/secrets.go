package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Runner secrets management
// Config is stored in ProjectSettings.spec.runnerSecretsName
// The Secret lives in the project namespace and stores key/value pairs for runners

// ListNamespaceSecrets handles GET /api/projects/:projectName/secrets -> { items: [{name, createdAt}] }
func ListNamespaceSecrets(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := GetK8sClientsForRequest(c)

	list, err := reqK8s.CoreV1().Secrets(projectName).List(c.Request.Context(), v1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list secrets in %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list secrets"})
		return
	}

	type Item struct {
		Name      string `json:"name"`
		CreatedAt string `json:"createdAt,omitempty"`
		Type      string `json:"type"`
	}
	items := []Item{}
	for _, s := range list.Items {
		// Only include runner/session secrets: Opaque + annotated
		if s.Type != corev1.SecretTypeOpaque {
			continue
		}
		if s.Annotations == nil || s.Annotations["ambient-code.io/runner-secret"] != "true" {
			continue
		}
		it := Item{Name: s.Name, Type: string(s.Type)}
		if !s.CreationTimestamp.IsZero() {
			it.CreatedAt = s.CreationTimestamp.Format(time.RFC3339)
		}
		items = append(items, it)
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// GetRunnerSecretsConfig handles GET /api/projects/:projectName/runner-secrets/config
func GetRunnerSecretsConfig(c *gin.Context) {
	projectName := c.Param("projectName")
	_, reqDyn := GetK8sClientsForRequest(c)

	gvr := GetProjectSettingsResource()
	// ProjectSettings is a singleton per namespace named 'projectsettings'
	obj, err := reqDyn.Resource(gvr).Namespace(projectName).Get(c.Request.Context(), "projectsettings", v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to read ProjectSettings for %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read runner secrets config"})
		return
	}

	secretName := ""
	if obj != nil {
		if spec, ok := GetSpecMap(obj); ok {
			if v, ok := spec["runnerSecretsName"].(string); ok {
				secretName = v
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"secretName": secretName})
}

// UpdateRunnerSecretsConfig handles PUT /api/projects/:projectName/runner-secrets/config { secretName }
func UpdateRunnerSecretsConfig(c *gin.Context) {
	projectName := c.Param("projectName")
	_, reqDyn := GetK8sClientsForRequest(c)

	var req struct {
		SecretName string `json:"secretName" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.SecretName) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "secretName is required"})
		return
	}

	// Operator owns ProjectSettings. If it exists, update; otherwise, return not found.
	gvr := GetProjectSettingsResource()
	obj, err := reqDyn.Resource(gvr).Namespace(projectName).Get(c.Request.Context(), "projectsettings", v1.GetOptions{})
	if errors.IsNotFound(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "ProjectSettings not found. Ensure the namespace is labeled ambient-code.io/managed=true and wait for operator."})
		return
	}
	if err != nil {
		log.Printf("Failed to read ProjectSettings for %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read runner secrets config"})
		return
	}

	// Update spec.runnerSecretsName
	spec, ok := GetSpecMap(obj)
	if !ok {
		spec = make(map[string]interface{})
	}
	if spec == nil {
		spec = map[string]interface{}{}
		obj.Object["spec"] = spec
	}
	spec["runnerSecretsName"] = req.SecretName

	if _, err := reqDyn.Resource(gvr).Namespace(projectName).Update(c.Request.Context(), obj, v1.UpdateOptions{}); err != nil {
		log.Printf("Failed to update ProjectSettings for %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update runner secrets config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"secretName": req.SecretName})
}

// ListRunnerSecrets handles GET /api/projects/:projectName/runner-secrets -> { data: { key: value } }
func ListRunnerSecrets(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, reqDyn := GetK8sClientsForRequest(c)

	// Read config
	gvr := GetProjectSettingsResource()
	obj, err := reqDyn.Resource(gvr).Namespace(projectName).Get(c.Request.Context(), "projectsettings", v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to read ProjectSettings for %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read runner secrets config"})
		return
	}
	secretName := ""
	if obj != nil {
		if spec, ok := GetSpecMap(obj); ok {
			if v, ok := spec["runnerSecretsName"].(string); ok {
				secretName = v
			}
		}
	}
	if secretName == "" {
		c.JSON(http.StatusOK, gin.H{"data": map[string]string{}})
		return
	}

	sec, err := reqK8s.CoreV1().Secrets(projectName).Get(c.Request.Context(), secretName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"data": map[string]string{}})
			return
		}
		log.Printf("Failed to get Secret %s/%s: %v", projectName, secretName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read runner secrets"})
		return
	}

	out := map[string]string{}
	for k, v := range sec.Data {
		out[k] = string(v)
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

// UpdateRunnerSecrets handles PUT /api/projects/:projectName/runner-secrets { data: { key: value } }
func UpdateRunnerSecrets(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, reqDyn := GetK8sClientsForRequest(c)

	var req struct {
		Data map[string]string `json:"data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Read config for secret name
	gvr := GetProjectSettingsResource()
	obj, err := reqDyn.Resource(gvr).Namespace(projectName).Get(c.Request.Context(), "projectsettings", v1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to read ProjectSettings for %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read runner secrets config"})
		return
	}
	secretName := ""
	if obj != nil {
		if spec, ok := obj.Object["spec"].(map[string]interface{}); ok {
			if v, ok := spec["runnerSecretsName"].(string); ok {
				secretName = strings.TrimSpace(v)
			}
		}
	}
	if secretName == "" {
		secretName = "ambient-runner-secrets"
	}

	// Do not create/update ProjectSettings here. The operator owns it.

	// Try to get existing Secret
	sec, err := reqK8s.CoreV1().Secrets(projectName).Get(c.Request.Context(), secretName, v1.GetOptions{})
	if errors.IsNotFound(err) {
		// Create new Secret
		newSec := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      secretName,
				Namespace: projectName,
				Labels:    map[string]string{"app": "ambient-runner-secrets"},
				Annotations: map[string]string{
					"ambient-code.io/runner-secret": "true",
				},
			},
			Type:       corev1.SecretTypeOpaque,
			StringData: req.Data,
		}
		if _, err := reqK8s.CoreV1().Secrets(projectName).Create(c.Request.Context(), newSec, v1.CreateOptions{}); err != nil {
			log.Printf("Failed to create Secret %s/%s: %v", projectName, secretName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create runner secrets"})
			return
		}
	} else if err != nil {
		log.Printf("Failed to get Secret %s/%s: %v", projectName, secretName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read runner secrets"})
		return
	} else {
		// Update existing - replace Data
		sec.Type = corev1.SecretTypeOpaque
		sec.Data = map[string][]byte{}
		for k, v := range req.Data {
			sec.Data[k] = []byte(v)
		}
		if _, err := reqK8s.CoreV1().Secrets(projectName).Update(c.Request.Context(), sec, v1.UpdateOptions{}); err != nil {
			log.Printf("Failed to update Secret %s/%s: %v", projectName, secretName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update runner secrets"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "runner secrets updated"})
}
