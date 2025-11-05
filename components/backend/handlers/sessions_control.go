// Package handlers provides HTTP handlers for the backend API.
// This file contains session control operations (start, stop, status update).
package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"ambient-code-backend/types"

	"github.com/gin-gonic/gin"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// StartSession starts or restarts an agentic session.
// For continuations, it sets parent-session-id annotation and regenerates tokens.
func StartSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	gvr := GetAgenticSessionV1Alpha1Resource()

	// Get current resource
	item, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	// Ensure runner role has required permissions (update if needed for existing sessions)
	if err := ensureRunnerRolePermissions(c, reqK8s, project, sessionName); err != nil {
		log.Printf("Warning: failed to ensure runner role permissions for %s: %v", sessionName, err)
		// Non-fatal - continue with restart
	}

	// Clean up temp-content pod if it exists to free the PVC
	// This prevents Multi-Attach errors when the session job tries to mount the workspace
	if reqK8s != nil {
		tempPodName := fmt.Sprintf("temp-content-%s", sessionName)
		if err := reqK8s.CoreV1().Pods(project).Delete(c.Request.Context(), tempPodName, v1.DeleteOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				log.Printf("StartSession: failed to delete temp-content pod %s (non-fatal): %v", tempPodName, err)
			}
		} else {
			log.Printf("StartSession: deleted temp-content pod %s to free PVC", tempPodName)
		}
	}

	// Check if this is a continuation (session is in a terminal phase)
	// Terminal phases from CRD: Completed, Failed, Stopped, Error
	isActualContinuation := false
	currentPhase := ""
	if currentStatus, ok := GetStatusMap(item); ok {
		if phase, ok := currentStatus["phase"].(string); ok {
			currentPhase = phase
			terminalPhases := []string{"Completed", "Failed", "Stopped", "Error"}
			for _, terminalPhase := range terminalPhases {
				if phase == terminalPhase {
					isActualContinuation = true
					log.Printf("StartSession: Detected continuation - session is in terminal phase: %s", phase)
					break
				}
			}
		}
	}

	if !isActualContinuation {
		log.Printf("StartSession: Not a continuation - current phase is: %s (not in terminal phases)", currentPhase)
	}

	// Only set parent session annotation if this is an actual continuation
	// Don't set it on first start, even though StartSession can be called for initial creation
	if isActualContinuation {
		annotations := item.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations["vteam.ambient-code/parent-session-id"] = sessionName
		item.SetAnnotations(annotations)
		log.Printf("StartSession: Set parent-session-id annotation to %s for continuation (has completion time)", sessionName)

		// For headless sessions being continued, force interactive mode
		if spec, ok := GetSpecMap(item); ok {
			if interactive, ok := spec["interactive"].(bool); !ok || !interactive {
				// Session was headless, convert to interactive
				spec["interactive"] = true
				log.Printf("StartSession: Converting headless session to interactive for continuation")
			}
		}

		// Update the metadata and spec to persist the annotation and interactive flag
		item, err = reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
		if err != nil {
			log.Printf("Failed to update agentic session metadata %s in project %s: %v", sessionName, project, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session metadata"})
			return
		}

		// Regenerate runner token for continuation (old token may have expired)
		log.Printf("StartSession: Regenerating runner token for session continuation")
		if err := provisionRunnerTokenForSession(c, reqK8s, reqDyn, project, sessionName); err != nil {
			log.Printf("Warning: failed to regenerate runner token for session %s/%s: %v", project, sessionName, err)
			// Non-fatal: continue anyway, operator may retry
		} else {
			log.Printf("StartSession: Successfully regenerated runner token for continuation")

			// Delete the old job so operator creates a new one
			// This ensures fresh token and clean state
			jobName := fmt.Sprintf("ambient-runner-%s", sessionName)
			log.Printf("StartSession: Deleting old job %s to allow operator to create fresh one", jobName)
			propagationPolicy := v1.DeletePropagationBackground
			if err := reqK8s.BatchV1().Jobs(project).Delete(c.Request.Context(), jobName, v1.DeleteOptions{
				PropagationPolicy: &propagationPolicy,
			}); err != nil {
				if !errors.IsNotFound(err) {
					log.Printf("Warning: failed to delete old job %s: %v", jobName, err)
				} else {
					log.Printf("StartSession: Job %s already gone", jobName)
				}
			} else {
				log.Printf("StartSession: Successfully deleted old job %s", jobName)
			}
		}
	} else {
		log.Printf("StartSession: Not setting parent-session-id (first run, no completion time)")
	}

	// Now update status to trigger start (using the fresh object from Update)
	if item.Object["status"] == nil {
		item.Object["status"] = make(map[string]interface{})
	}

	status, ok := GetStatusMap(item)
	if !ok {
		status = make(map[string]interface{})
		item.Object["status"] = status
	}

	// Set to Pending so operator will process it (operator only acts on Pending phase)
	status["phase"] = "Pending"
	status["message"] = "Session restart requested"
	// Clear completion time from previous run
	delete(status, "completionTime")
	// Update start time for this run
	status["startTime"] = time.Now().Format(time.RFC3339)

	// Update the status subresource (must use UpdateStatus, not Update)
	updated, err := reqDyn.Resource(gvr).Namespace(project).UpdateStatus(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to start agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start agentic session"})
		return
	}

	// Parse and return updated session
	metadata, ok := GetMetadataMap(updated)
	if !ok {
		log.Printf("Updated session %s missing metadata", sessionName)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid session data"})
		return
	}

	session := types.AgenticSession{
		APIVersion: updated.GetAPIVersion(),
		Kind:       updated.GetKind(),
		Metadata:   metadata,
	}

	if spec, ok := GetSpecMap(updated); ok {
		session.Spec = parseSpec(spec)
	}

	if status, ok := GetStatusMap(updated); ok {
		session.Status = parseStatus(status)
	}

	c.JSON(http.StatusAccepted, session)
}

// StopSession stops a running agentic session by deleting its job and pods.
func StopSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	gvr := GetAgenticSessionV1Alpha1Resource()

	// Get current resource
	item, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	// Check current status
	status, ok := GetStatusMap(item)
	if !ok {
		status = make(map[string]interface{})
		item.Object["status"] = status
	}

	currentPhase, _ := status["phase"].(string)
	if currentPhase == "Completed" || currentPhase == "Failed" || currentPhase == "Stopped" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cannot stop session in %s state", currentPhase)})
		return
	}

	log.Printf("Attempting to stop agentic session %s in project %s (current phase: %s)", sessionName, project, currentPhase)

	// Get job name from status
	jobName, jobExists := status["jobName"].(string)
	if !jobExists || jobName == "" {
		// Try to derive job name if not in status
		jobName = fmt.Sprintf("%s-job", sessionName)
		log.Printf("Job name not in status, trying derived name: %s", jobName)
	}

	// Delete the job and its pods
	log.Printf("Attempting to delete job %s for session %s", jobName, sessionName)

	// First, delete the job itself with foreground propagation
	deletePolicy := v1.DeletePropagationForeground
	err = reqK8s.BatchV1().Jobs(project).Delete(context.TODO(), jobName, v1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("Job %s not found (may have already completed or been deleted)", jobName)
		} else {
			log.Printf("Failed to delete job %s: %v", jobName, err)
			// Don't fail the request if job deletion fails - continue with status update
			log.Printf("Continuing with status update despite job deletion failure")
		}
	} else {
		log.Printf("Successfully deleted job %s for agentic session %s", jobName, sessionName)
	}

	// Then, explicitly delete all pods for this job (by job-name label)
	podSelector := fmt.Sprintf("job-name=%s", jobName)
	log.Printf("Deleting pods with job-name selector: %s", podSelector)
	err = reqK8s.CoreV1().Pods(project).DeleteCollection(context.TODO(), v1.DeleteOptions{}, v1.ListOptions{
		LabelSelector: podSelector,
	})
	if err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to delete pods for job %s: %v (continuing anyway)", jobName, err)
	} else {
		log.Printf("Successfully deleted pods for job %s", jobName)
	}

	// Also delete any pods labeled with this session (in case owner refs are lost)
	sessionPodSelector := fmt.Sprintf("agentic-session=%s", sessionName)
	log.Printf("Deleting pods with agentic-session selector: %s", sessionPodSelector)
	err = reqK8s.CoreV1().Pods(project).DeleteCollection(context.TODO(), v1.DeleteOptions{}, v1.ListOptions{
		LabelSelector: sessionPodSelector,
	})
	if err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to delete session pods: %v (continuing anyway)", err)
	} else {
		log.Printf("Successfully deleted session-labeled pods")
	}

	// Update status to Stopped
	status["phase"] = "Stopped"
	status["message"] = "Session stopped by user"
	status["completionTime"] = time.Now().Format(time.RFC3339)

	// Also set interactive: true in spec so session can be restarted
	if spec, ok := GetSpecMap(item); ok {
		if interactive, ok := spec["interactive"].(bool); !ok || !interactive {
			log.Printf("Setting interactive: true for stopped session %s to allow restart", sessionName)
			spec["interactive"] = true
			// Update spec first (must use Update, not UpdateStatus)
			item, err = reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
			if err != nil {
				log.Printf("Failed to update session spec for %s: %v (continuing with status update)", sessionName, err)
				// Continue anyway - status update is more important
			}
		}
	}

	// Update the resource using UpdateStatus for status subresource
	updated, err := reqDyn.Resource(gvr).Namespace(project).UpdateStatus(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Session was deleted while we were trying to update it
			log.Printf("Agentic session %s was deleted during stop operation", sessionName)
			c.JSON(http.StatusOK, gin.H{"message": "Session no longer exists (already deleted)"})
			return
		}
		log.Printf("Failed to update agentic session status %s: %v", sessionName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agentic session status"})
		return
	}

	// Parse and return updated session
	metadata, ok := GetMetadataMap(updated)
	if !ok {
		log.Printf("Updated session %s missing metadata", sessionName)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid session data"})
		return
	}

	session := types.AgenticSession{
		APIVersion: updated.GetAPIVersion(),
		Kind:       updated.GetKind(),
		Metadata:   metadata,
	}

	if spec, ok := GetSpecMap(updated); ok {
		session.Spec = parseSpec(spec)
	}

	if status, ok := GetStatusMap(updated); ok {
		session.Status = parseStatus(status)
	}

	log.Printf("Successfully stopped agentic session %s", sessionName)
	c.JSON(http.StatusAccepted, session)
}

// UpdateSessionStatus writes selected fields to PVC-backed files and updates CR status.
// PUT /api/projects/:projectName/agentic-sessions/:sessionName/status
func UpdateSessionStatus(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, reqDyn := GetK8sClientsForRequest(c)

	var statusUpdate map[string]interface{}
	if err := c.ShouldBindJSON(&statusUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := GetAgenticSessionV1Alpha1Resource()

	// Get current resource
	item, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	// Ensure status map
	if item.Object["status"] == nil {
		item.Object["status"] = make(map[string]interface{})
	}
	status, ok := GetStatusMap(item)
	if !ok {
		status = make(map[string]interface{})
		item.Object["status"] = status
	}

	// Accept standard fields and result summary fields from runner
	allowed := map[string]struct{}{
		"phase": {}, "completionTime": {}, "cost": {}, "message": {},
		"subtype": {}, "duration_ms": {}, "duration_api_ms": {}, "is_error": {},
		"num_turns": {}, "session_id": {}, "total_cost_usd": {}, "usage": {}, "result": {},
	}
	for k := range statusUpdate {
		if _, ok := allowed[k]; !ok {
			delete(statusUpdate, k)
		}
	}

	// Merge remaining fields into status
	for k, v := range statusUpdate {
		status[k] = v
	}

	// Update only the status subresource (requires agenticsessions/status perms)
	if _, err := reqDyn.Resource(gvr).Namespace(project).UpdateStatus(context.TODO(), item, v1.UpdateOptions{}); err != nil {
		log.Printf("Failed to update agentic session status %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agentic session status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "agentic session status updated"})
}

// ensureRunnerRolePermissions checks and updates runner role permissions if needed.
// This is used to add missing permissions to existing sessions during restart.
func ensureRunnerRolePermissions(c *gin.Context, reqK8s *kubernetes.Clientset, project string, sessionName string) error {
	roleName := fmt.Sprintf("ambient-session-%s-role", sessionName)

	// Get existing role
	existingRole, err := reqK8s.RbacV1().Roles(project).Get(c.Request.Context(), roleName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("Role %s not found for session %s - will be created by operator", roleName, sessionName)
			return nil
		}
		return fmt.Errorf("get role: %w", err)
	}

	// Check if role has selfsubjectaccessreviews permission
	hasSelfSubjectAccessReview := false
	for _, rule := range existingRole.Rules {
		for _, apiGroup := range rule.APIGroups {
			if apiGroup == "authorization.k8s.io" {
				for _, resource := range rule.Resources {
					if resource == "selfsubjectaccessreviews" {
						hasSelfSubjectAccessReview = true
						break
					}
				}
			}
		}
	}

	if hasSelfSubjectAccessReview {
		log.Printf("Role %s already has selfsubjectaccessreviews permission", roleName)
		return nil
	}

	// Add missing permission
	log.Printf("Updating role %s to add selfsubjectaccessreviews permission", roleName)
	existingRole.Rules = append(existingRole.Rules, rbacv1.PolicyRule{
		APIGroups: []string{"authorization.k8s.io"},
		Resources: []string{"selfsubjectaccessreviews"},
		Verbs:     []string{"create"},
	})

	_, err = reqK8s.RbacV1().Roles(project).Update(c.Request.Context(), existingRole, v1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}

	log.Printf("Successfully updated role %s with selfsubjectaccessreviews permission", roleName)
	return nil
}
