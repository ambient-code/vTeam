// Package preflight provides environment validation and configuration checks for the operator.
package preflight

import (
	"context"
	"fmt"
	"log"
	"os"

	"ambient-code-operator/internal/config"
	"ambient-code-operator/internal/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ValidateVertexConfig validates Vertex AI configuration at operator startup
func ValidateVertexConfig(operatorNamespace string) error {
	log.Printf("Vertex AI mode enabled - validating configuration...")

	// Check required environment variables
	requiredEnvVars := map[string]string{
		"ANTHROPIC_VERTEX_PROJECT_ID":    os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID"),
		"CLOUD_ML_REGION":                os.Getenv("CLOUD_ML_REGION"),
		"GOOGLE_APPLICATION_CREDENTIALS": os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
	}

	for name, value := range requiredEnvVars {
		if value == "" {
			return fmt.Errorf("CLAUDE_CODE_USE_VERTEX=1 but %s is not set", name)
		}
		log.Printf("  %s: %s", name, value)
	}

	// Optional: Check if ambient-vertex secret exists in operator namespace
	// The secret will be copied to runner namespaces, but it's not required at operator startup
	// since runners handle the actual authentication
	_, err := config.K8sClient.CoreV1().Secrets(operatorNamespace).Get(
		context.TODO(),
		types.AmbientVertexSecretName,
		metav1.GetOptions{},
	)
	if err != nil {
		log.Printf("  Warning: secret '%s' not found in namespace '%s': %v", types.AmbientVertexSecretName, operatorNamespace, err)
		log.Printf("  Note: Create the secret with: kubectl create secret generic %s --from-file=ambient-code-key.json=/path/to/service-account.json -n %s",
			types.AmbientVertexSecretName, operatorNamespace)
		log.Printf("  The operator will continue, but sessions requiring Vertex AI will fail until the secret is created")
	} else {
		log.Printf("  Secret '%s' found in namespace '%s'", types.AmbientVertexSecretName, operatorNamespace)
	}

	log.Printf("Vertex AI configuration validated successfully")
	return nil
}
