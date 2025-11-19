package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"ambient-code-operator/internal/config"
	"ambient-code-operator/internal/types"

	authnv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	conditionReady                     = "Ready"
	conditionPVCReady                  = "PVCReady"
	conditionSecretsReady              = "SecretsReady"
	conditionJobCreated                = "JobCreated"
	conditionPodScheduled              = "PodScheduled"
	conditionRunnerStarted             = "RunnerStarted"
	conditionReposReconciled           = "ReposReconciled"
	conditionWorkflowReconciled        = "WorkflowReconciled"
	conditionTempContentPodReady       = "TempContentPodReady"
	conditionCompleted                 = "Completed"
	conditionFailed                    = "Failed"
	runnerTokenSecretAnnotation        = "ambient-code.io/runner-token-secret"
	runnerServiceAccountAnnotation     = "ambient-code.io/runner-sa"
	runnerTokenRefreshedAtAnnotation   = "ambient-code.io/token-refreshed-at"
	tempContentRequestedAnnotation     = "ambient-code.io/temp-content-requested"
	tempContentLastAccessedAnnotation  = "ambient-code.io/temp-content-last-accessed"
	runnerTokenRefreshTTL              = 45 * time.Minute
	tempContentInactivityTTL           = 10 * time.Minute
	defaultRunnerTokenSecretPrefix     = "ambient-runner-token-"
	defaultSessionServiceAccountPrefix = "ambient-session-"
)

type conditionUpdate struct {
	Type    string
	Status  string
	Reason  string
	Message string
}

// mutateAgenticSessionStatus loads the AgenticSession, applies the mutator to the status map, and persists the result.
func mutateAgenticSessionStatus(sessionNamespace, name string, mutator func(status map[string]interface{})) error {
	gvr := types.GetAgenticSessionResource()

	obj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("AgenticSession %s no longer exists, skipping status update", name)
			return nil
		}
		return fmt.Errorf("failed to get AgenticSession %s: %w", name, err)
	}

	if obj.Object["status"] == nil {
		obj.Object["status"] = make(map[string]interface{})
	}

	status, ok := obj.Object["status"].(map[string]interface{})
	if !ok {
		status = make(map[string]interface{})
		obj.Object["status"] = status
	}

	mutator(status)

	// Always derive phase from conditions if they exist
	if derived := derivePhaseFromConditions(status); derived != "" {
		status["phase"] = derived
	}

	_, err = config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).UpdateStatus(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("AgenticSession %s was deleted during status update, skipping", name)
			return nil
		}
		return fmt.Errorf("failed to update AgenticSession status: %w", err)
	}

	return nil
}

// updateAgenticSessionStatus merges the provided fields into status.
func updateAgenticSessionStatus(sessionNamespace, name string, statusUpdate map[string]interface{}) error {
	return mutateAgenticSessionStatus(sessionNamespace, name, func(status map[string]interface{}) {
		for key, value := range statusUpdate {
			status[key] = value
		}
	})
}

// ensureSessionIsInteractive forces spec.interactive=true so sessions can be restarted.
func ensureSessionIsInteractive(sessionNamespace, name string) error {
	gvr := types.GetAgenticSessionResource()

	obj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("AgenticSession %s no longer exists, skipping interactive update", name)
			return nil
		}
		return fmt.Errorf("failed to get AgenticSession %s: %w", name, err)
	}

	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil {
		return fmt.Errorf("failed to read spec for AgenticSession %s: %w", name, err)
	}
	if !found {
		log.Printf("AgenticSession %s has no spec; cannot update interactive flag", name)
		return nil
	}

	if interactive, _, _ := unstructured.NestedBool(spec, "interactive"); interactive {
		return nil
	}

	if err := unstructured.SetNestedField(obj.Object, true, "spec", "interactive"); err != nil {
		return fmt.Errorf("failed to set interactive flag for %s: %w", name, err)
	}

	_, err = config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Update(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to persist interactive flag for %s: %w", name, err)
	}

	return nil
}

// updateAnnotations updates annotations on the AgenticSession CR.
func updateAnnotations(sessionNamespace, name string, annotations map[string]string) error {
	gvr := types.GetAgenticSessionResource()

	obj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("AgenticSession %s no longer exists, skipping annotation update", name)
			return nil
		}
		return fmt.Errorf("failed to get AgenticSession %s: %w", name, err)
	}

	obj.SetAnnotations(annotations)

	_, err = config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Update(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to update annotations for %s: %w", name, err)
	}

	return nil
}

// clearAnnotation removes a specific annotation from the AgenticSession CR.
func clearAnnotation(sessionNamespace, name, annotationKey string) error {
	gvr := types.GetAgenticSessionResource()

	obj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get AgenticSession %s: %w", name, err)
	}

	annotations := obj.GetAnnotations()
	if annotations == nil {
		return nil
	}

	if _, exists := annotations[annotationKey]; !exists {
		return nil
	}

	delete(annotations, annotationKey)
	obj.SetAnnotations(annotations)

	_, err = config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Update(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to clear annotation %s for %s: %w", annotationKey, name, err)
	}

	return nil
}

// setCondition upserts a condition entry on the provided status map.
func setCondition(status map[string]interface{}, update conditionUpdate) {
	now := time.Now().UTC().Format(time.RFC3339)
	conditions, _ := status["conditions"].([]interface{})
	updated := false

	for i, c := range conditions {
		if existing, ok := c.(map[string]interface{}); ok {
			if strings.EqualFold(existing["type"].(string), update.Type) {
				if existing["status"] != update.Status {
					existing["lastTransitionTime"] = now
				}
				existing["status"] = update.Status
				if update.Reason != "" {
					existing["reason"] = update.Reason
				}
				if update.Message != "" {
					existing["message"] = update.Message
				}
				conditions[i] = existing
				updated = true
				break
			}
		}
	}

	if !updated {
		newCond := map[string]interface{}{
			"type":               update.Type,
			"status":             update.Status,
			"reason":             update.Reason,
			"message":            update.Message,
			"lastTransitionTime": now,
		}
		conditions = append(conditions, newCond)
	}

	status["conditions"] = conditions
}

// derivePhaseFromConditions determines the high-level phase from condition set.
func derivePhaseFromConditions(status map[string]interface{}) string {
	condStatus := func(condType string) string {
		conditions, _ := status["conditions"].([]interface{})
		for _, c := range conditions {
			if existing, ok := c.(map[string]interface{}); ok {
				if strings.EqualFold(existing["type"].(string), condType) {
					if val, ok := existing["status"].(string); ok {
						return val
					}
				}
			}
		}
		return ""
	}

	switch {
	case condStatus(conditionFailed) == "True":
		return "Failed"
	case condStatus(conditionCompleted) == "True":
		return "Completed"
	case condStatus(conditionRunnerStarted) == "True":
		return "Running"
	case condStatus(conditionJobCreated) == "True":
		return "Creating"
	case condStatus(conditionPVCReady) == "True":
		return "Pending"
	default:
		return ""
	}
}

// ensureFreshRunnerToken refreshes the runner SA token if it is older than the allowed TTL.
func ensureFreshRunnerToken(ctx context.Context, session *unstructured.Unstructured) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	namespace := session.GetNamespace()
	if namespace == "" {
		return fmt.Errorf("session namespace is empty")
	}

	annotations := session.GetAnnotations()
	secretName := strings.TrimSpace(annotations[runnerTokenSecretAnnotation])
	if secretName == "" {
		secretName = fmt.Sprintf("%s%s", defaultRunnerTokenSecretPrefix, session.GetName())
	}

	secret, err := config.K8sClient.CoreV1().Secrets(namespace).Get(ctx, secretName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to fetch runner token secret %s/%s: %w", namespace, secretName, err)
	}

	if secret.Annotations != nil {
		if refreshedAtStr := secret.Annotations[runnerTokenRefreshedAtAnnotation]; refreshedAtStr != "" {
			if refreshedAt, parseErr := time.Parse(time.RFC3339, refreshedAtStr); parseErr == nil {
				if time.Since(refreshedAt) < runnerTokenRefreshTTL {
					return nil
				}
			}
		}
	}

	saName := strings.TrimSpace(annotations[runnerServiceAccountAnnotation])
	if saName == "" {
		saName = fmt.Sprintf("%s%s", defaultSessionServiceAccountPrefix, session.GetName())
	}

	tokenReq := &authnv1.TokenRequest{Spec: authnv1.TokenRequestSpec{}}
	tokenResp, err := config.K8sClient.CoreV1().ServiceAccounts(namespace).CreateToken(ctx, saName, tokenReq, v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to mint token for %s/%s: %w", namespace, saName, err)
	}
	token := strings.TrimSpace(tokenResp.Status.Token)
	if token == "" {
		return fmt.Errorf("received empty token for %s/%s", namespace, saName)
	}

	secretCopy := secret.DeepCopy()
	if secretCopy.Data == nil {
		secretCopy.Data = map[string][]byte{}
	}
	secretCopy.Data["k8s-token"] = []byte(token)
	if secretCopy.Annotations == nil {
		secretCopy.Annotations = map[string]string{}
	}
	secretCopy.Annotations[runnerTokenRefreshedAtAnnotation] = time.Now().UTC().Format(time.RFC3339)

	if _, err := config.K8sClient.CoreV1().Secrets(namespace).Update(ctx, secretCopy, v1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update runner token secret %s/%s: %w", namespace, secretName, err)
	}

	log.Printf("Refreshed runner token for session %s/%s", namespace, session.GetName())
	return nil
}
