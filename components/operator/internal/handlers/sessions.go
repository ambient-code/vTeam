// Package handlers implements Kubernetes watch handlers for AgenticSession, ProjectSettings, and Namespace resources.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"ambient-code-operator/internal/config"
	"ambient-code-operator/internal/services"
	"ambient-code-operator/internal/types"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/retry"
)

// WatchAgenticSessions watches for AgenticSession custom resources and creates jobs
func WatchAgenticSessions() {
	gvr := types.GetAgenticSessionResource()

	for {
		// Watch AgenticSessions across all namespaces
		watcher, err := config.DynamicClient.Resource(gvr).Watch(context.TODO(), v1.ListOptions{})
		if err != nil {
			log.Printf("Failed to create AgenticSession watcher: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("Watching for AgenticSession events across all namespaces...")

		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added, watch.Modified:
				obj := event.Object.(*unstructured.Unstructured)

				// Only process resources in managed namespaces
				ns := obj.GetNamespace()
				if ns == "" {
					continue
				}
				nsObj, err := config.K8sClient.CoreV1().Namespaces().Get(context.TODO(), ns, v1.GetOptions{})
				if err != nil {
					log.Printf("Failed to get namespace %s: %v", ns, err)
					continue
				}
				if nsObj.Labels["ambient-code.io/managed"] != "true" {
					// Skip unmanaged namespaces
					continue
				}

				// Add small delay to avoid race conditions with rapid create/delete cycles
				time.Sleep(100 * time.Millisecond)

				if err := handleAgenticSessionEvent(obj); err != nil {
					log.Printf("Error handling AgenticSession event: %v", err)
				}
			case watch.Deleted:
				obj := event.Object.(*unstructured.Unstructured)
				sessionName := obj.GetName()
				sessionNamespace := obj.GetNamespace()
				log.Printf("AgenticSession %s/%s deleted", sessionNamespace, sessionName)

				// Cancel any ongoing job monitoring for this session
				// (We could implement this with a context cancellation if needed)
				// OwnerReferences handle cleanup of per-session resources
			case watch.Error:
				obj := event.Object.(*unstructured.Unstructured)
				log.Printf("Watch error for AgenticSession: %v", obj)
			}
		}

		log.Println("AgenticSession watch channel closed, restarting...")
		watcher.Stop()
		time.Sleep(2 * time.Second)
	}
}

func handleAgenticSessionEvent(obj *unstructured.Unstructured) error {
	name := obj.GetName()
	sessionNamespace := obj.GetNamespace()

	// Verify the resource still exists before processing (in its own namespace)
	gvr := types.GetAgenticSessionResource()
	currentObj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("AgenticSession %s no longer exists, skipping processing", name)
			return nil
		}
		return fmt.Errorf("failed to verify AgenticSession %s exists: %v", name, err)
	}

	// Get the current status from the fresh object (status may be empty right after creation
	// because the API server drops .status on create when the status subresource is enabled)
	stMap, found, _ := unstructured.NestedMap(currentObj.Object, "status")
	phase := ""
	if found {
		if p, ok := stMap["phase"].(string); ok {
			phase = p
		}
	}
	// If status.phase is missing, treat as Pending and initialize it
	if phase == "" {
		_ = updateAgenticSessionStatus(sessionNamespace, name, map[string]interface{}{"phase": "Pending"})
		phase = "Pending"
	}

	log.Printf("Processing AgenticSession %s with phase %s", name, phase)

	// Handle Stopped phase - clean up running job if it exists
	if phase == "Stopped" {
		log.Printf("Session %s is stopped, checking for running job to clean up", name)
		jobName := fmt.Sprintf("%s-job", name)

		job, err := config.K8sClient.BatchV1().Jobs(sessionNamespace).Get(context.TODO(), jobName, v1.GetOptions{})
		if err == nil {
			// Job exists, check if it's still running or needs cleanup
			if job.Status.Active > 0 || (job.Status.Succeeded == 0 && job.Status.Failed == 0) {
				log.Printf("Job %s is still active, cleaning up job and pods", jobName)

				// First, delete the job itself with foreground propagation
				deletePolicy := v1.DeletePropagationForeground
				err = config.K8sClient.BatchV1().Jobs(sessionNamespace).Delete(context.TODO(), jobName, v1.DeleteOptions{
					PropagationPolicy: &deletePolicy,
				})
				if err != nil && !errors.IsNotFound(err) {
					log.Printf("Failed to delete job %s: %v", jobName, err)
				} else {
					log.Printf("Successfully deleted job %s for stopped session", jobName)
				}

				// Then, explicitly delete all pods for this job (by job-name label)
				podSelector := fmt.Sprintf("job-name=%s", jobName)
				log.Printf("Deleting pods with job-name selector: %s", podSelector)
				err = config.K8sClient.CoreV1().Pods(sessionNamespace).DeleteCollection(context.TODO(), v1.DeleteOptions{}, v1.ListOptions{
					LabelSelector: podSelector,
				})
				if err != nil && !errors.IsNotFound(err) {
					log.Printf("Failed to delete pods for job %s: %v (continuing anyway)", jobName, err)
				} else {
					log.Printf("Successfully deleted pods for job %s", jobName)
				}

				// Also delete any pods labeled with this session (in case owner refs are lost)
				sessionPodSelector := fmt.Sprintf("agentic-session=%s", name)
				log.Printf("Deleting pods with agentic-session selector: %s", sessionPodSelector)
				err = config.K8sClient.CoreV1().Pods(sessionNamespace).DeleteCollection(context.TODO(), v1.DeleteOptions{}, v1.ListOptions{
					LabelSelector: sessionPodSelector,
				})
				if err != nil && !errors.IsNotFound(err) {
					log.Printf("Failed to delete session-labeled pods: %v (continuing anyway)", err)
				} else {
					log.Printf("Successfully deleted session-labeled pods")
				}
			} else {
				log.Printf("Job %s already completed (Succeeded: %d, Failed: %d), no cleanup needed", jobName, job.Status.Succeeded, job.Status.Failed)
			}
		} else if !errors.IsNotFound(err) {
			log.Printf("Error checking job %s: %v", jobName, err)
		} else {
			log.Printf("Job %s not found, already cleaned up", jobName)
		}

		// Also cleanup ambient-vertex secret when session is stopped
		deleteCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := deleteAmbientVertexSecret(deleteCtx, sessionNamespace); err != nil {
			log.Printf("Warning: Failed to cleanup %s secret from %s: %v", types.AmbientVertexSecretName, sessionNamespace, err)
			// Continue - session cleanup is still successful
		}

		return nil
	}

	// Only process if status is Pending
	if phase != "Pending" {
		return nil
	}

	// Check for session continuation (parent session ID)
	parentSessionID := ""
	// Check annotations first
	annotations := currentObj.GetAnnotations()
	if val, ok := annotations["vteam.ambient-code/parent-session-id"]; ok {
		parentSessionID = strings.TrimSpace(val)
	}
	// Check environmentVariables as fallback
	if parentSessionID == "" {
		spec, _, _ := unstructured.NestedMap(currentObj.Object, "spec")
		if envVars, found, _ := unstructured.NestedStringMap(spec, "environmentVariables"); found {
			if val, ok := envVars["PARENT_SESSION_ID"]; ok {
				parentSessionID = strings.TrimSpace(val)
			}
		}
	}

	// Determine PVC name and owner references
	var pvcName string
	var ownerRefs []v1.OwnerReference
	reusingPVC := false

	if parentSessionID != "" {
		// Continuation: reuse parent's PVC
		pvcName = fmt.Sprintf("ambient-workspace-%s", parentSessionID)
		reusingPVC = true
		log.Printf("Session continuation: reusing PVC %s from parent session %s", pvcName, parentSessionID)
		// No owner refs - we don't own the parent's PVC
	} else {
		// New session: create fresh PVC with owner refs
		pvcName = fmt.Sprintf("ambient-workspace-%s", name)
		ownerRefs = []v1.OwnerReference{
			{
				APIVersion: "vteam.ambient-code/v1",
				Kind:       "AgenticSession",
				Name:       currentObj.GetName(),
				UID:        currentObj.GetUID(),
				Controller: boolPtr(true),
				// BlockOwnerDeletion intentionally omitted to avoid permission issues
			},
		}
	}

	// Ensure PVC exists (skip for continuation if parent's PVC should exist)
	if !reusingPVC {
		if err := services.EnsureSessionWorkspacePVC(sessionNamespace, pvcName, ownerRefs); err != nil {
			log.Printf("Failed to ensure session PVC %s in %s: %v", pvcName, sessionNamespace, err)
			// Continue; job may still run with ephemeral storage
		}
	} else {
		// Verify parent's PVC exists
		if _, err := config.K8sClient.CoreV1().PersistentVolumeClaims(sessionNamespace).Get(context.TODO(), pvcName, v1.GetOptions{}); err != nil {
			log.Printf("Warning: Parent PVC %s not found for continuation session %s: %v", pvcName, name, err)
			// Fall back to creating new PVC with current session's owner refs
			pvcName = fmt.Sprintf("ambient-workspace-%s", name)
			ownerRefs = []v1.OwnerReference{
				{
					APIVersion: "vteam.ambient-code/v1",
					Kind:       "AgenticSession",
					Name:       currentObj.GetName(),
					UID:        currentObj.GetUID(),
					Controller: boolPtr(true),
				},
			}
			if err := services.EnsureSessionWorkspacePVC(sessionNamespace, pvcName, ownerRefs); err != nil {
				log.Printf("Failed to create fallback PVC %s: %v", pvcName, err)
			}
		}
	}

	// Load config for this session
	appConfig := config.LoadConfig()

	// Check for ambient-vertex secret in the operator's namespace and copy it if Vertex is enabled
	// This will be used to conditionally mount the secret as a volume
	ambientVertexSecretCopied := false
	operatorNamespace := appConfig.BackendNamespace // Assuming operator runs in same namespace as backend
	vertexEnabled := os.Getenv("CLAUDE_CODE_USE_VERTEX") == "1"

	// Only attempt to copy the secret if Vertex AI is enabled
	if vertexEnabled {
		if ambientVertexSecret, err := config.K8sClient.CoreV1().Secrets(operatorNamespace).Get(context.TODO(), types.AmbientVertexSecretName, v1.GetOptions{}); err == nil {
			// Secret exists in operator namespace, copy it to the session namespace
			log.Printf("Found %s secret in %s, copying to %s", types.AmbientVertexSecretName, operatorNamespace, sessionNamespace)
			// Create context with timeout for secret copy operation
			copyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := copySecretToNamespace(copyCtx, ambientVertexSecret, sessionNamespace, currentObj); err != nil {
				return fmt.Errorf("failed to copy %s secret from %s to %s (CLAUDE_CODE_USE_VERTEX=1): %w", types.AmbientVertexSecretName, operatorNamespace, sessionNamespace, err)
			}
			ambientVertexSecretCopied = true
			log.Printf("Successfully copied %s secret to %s", types.AmbientVertexSecretName, sessionNamespace)
		} else if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to check for %s secret in %s (CLAUDE_CODE_USE_VERTEX=1): %w", types.AmbientVertexSecretName, operatorNamespace, err)
		} else {
			// Vertex enabled but secret not found - fail fast
			return fmt.Errorf("CLAUDE_CODE_USE_VERTEX=1 but %s secret not found in namespace %s", types.AmbientVertexSecretName, operatorNamespace)
		}
	} else {
		log.Printf("Vertex AI disabled (CLAUDE_CODE_USE_VERTEX=0), skipping %s secret copy", types.AmbientVertexSecretName)
	}

	// Create a Kubernetes Job for this AgenticSession
	jobName := fmt.Sprintf("%s-job", name)

	// Check if job already exists in the session's namespace
	_, err = config.K8sClient.BatchV1().Jobs(sessionNamespace).Get(context.TODO(), jobName, v1.GetOptions{})
	if err == nil {
		log.Printf("Job %s already exists for AgenticSession %s", jobName, name)
		return nil
	}

	// Extract spec information from the fresh object
	spec, _, _ := unstructured.NestedMap(currentObj.Object, "spec")
	prompt, _, _ := unstructured.NestedString(spec, "prompt")
	timeout, _, _ := unstructured.NestedInt64(spec, "timeout")
	interactive, _, _ := unstructured.NestedBool(spec, "interactive")

	llmSettings, _, _ := unstructured.NestedMap(spec, "llmSettings")
	model, _, _ := unstructured.NestedString(llmSettings, "model")
	temperature, _, _ := unstructured.NestedFloat64(llmSettings, "temperature")
	maxTokens, _, _ := unstructured.NestedInt64(llmSettings, "maxTokens")

	// Hardcoded secret names (convention over configuration)
	const runnerSecretsName = "ambient-runner-secrets"               // ANTHROPIC_API_KEY only (ignored when Vertex enabled)
	const integrationSecretsName = "ambient-non-vertex-integrations" // GIT_*, JIRA_*, custom keys (optional)

	// Check if integration secrets exist (optional)
	integrationSecretsExist := false
	if _, err := config.K8sClient.CoreV1().Secrets(sessionNamespace).Get(context.TODO(), integrationSecretsName, v1.GetOptions{}); err == nil {
		integrationSecretsExist = true
		log.Printf("Found %s secret in %s, will inject as env vars", integrationSecretsName, sessionNamespace)
	} else if !errors.IsNotFound(err) {
		log.Printf("Error checking for %s secret in %s: %v", integrationSecretsName, sessionNamespace, err)
	} else {
		log.Printf("No %s secret found in %s (optional, skipping)", integrationSecretsName, sessionNamespace)
	}

	// Extract input/output git configuration (support flat and nested forms)
	inputRepo, _, _ := unstructured.NestedString(spec, "inputRepo")
	inputBranch, _, _ := unstructured.NestedString(spec, "inputBranch")
	outputRepo, _, _ := unstructured.NestedString(spec, "outputRepo")
	outputBranch, _, _ := unstructured.NestedString(spec, "outputBranch")
	if v, found, _ := unstructured.NestedString(spec, "input", "repo"); found && strings.TrimSpace(v) != "" {
		inputRepo = v
	}
	if v, found, _ := unstructured.NestedString(spec, "input", "branch"); found && strings.TrimSpace(v) != "" {
		inputBranch = v
	}
	if v, found, _ := unstructured.NestedString(spec, "output", "repo"); found && strings.TrimSpace(v) != "" {
		outputRepo = v
	}
	if v, found, _ := unstructured.NestedString(spec, "output", "branch"); found && strings.TrimSpace(v) != "" {
		outputBranch = v
	}

	// Read autoPushOnComplete flag
	autoPushOnComplete, _, _ := unstructured.NestedBool(spec, "autoPushOnComplete")

	// Create the Job
	job := &batchv1.Job{
		ObjectMeta: v1.ObjectMeta{
			Name:      jobName,
			Namespace: sessionNamespace,
			Labels: map[string]string{
				"agentic-session": name,
				"app":             "ambient-code-runner",
			},
			OwnerReferences: []v1.OwnerReference{
				{
					APIVersion: "vteam.ambient-code/v1",
					Kind:       "AgenticSession",
					Name:       currentObj.GetName(),
					UID:        currentObj.GetUID(),
					Controller: boolPtr(true),
					// Remove BlockOwnerDeletion to avoid permission issues
					// BlockOwnerDeletion: boolPtr(true),
				},
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:          int32Ptr(3),
			ActiveDeadlineSeconds: int64Ptr(14400), // 4 hour timeout for safety
			// Auto-cleanup finished Jobs if TTL controller is enabled in the cluster
			TTLSecondsAfterFinished: int32Ptr(600),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"agentic-session": name,
						"app":             "ambient-code-runner",
					},
					// If you run a service mesh that injects sidecars and causes egress issues for Jobs:
					// Annotations: map[string]string{"sidecar.istio.io/inject": "false"},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					// Explicitly set service account for pod creation permissions
					AutomountServiceAccountToken: boolPtr(false),
					Volumes: []corev1.Volume{
						{
							Name: "workspace",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvcName,
								},
							},
						},
					},

					// InitContainer to ensure workspace directory structure exists
					InitContainers: []corev1.Container{
						{
							Name:  "init-workspace",
							Image: "registry.access.redhat.com/ubi8/ubi-minimal:latest",
							Command: []string{
								"sh", "-c",
								fmt.Sprintf("mkdir -p /workspace/sessions/%s/workspace && chmod 777 /workspace/sessions/%s/workspace && echo 'Workspace initialized'", name, name),
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "workspace", MountPath: "/workspace"},
							},
						},
					},

					// Flip roles so the content writer is the main container that keeps the pod alive
					Containers: []corev1.Container{
						{
							Name:            "ambient-content",
							Image:           appConfig.ContentServiceImage,
							ImagePullPolicy: appConfig.ImagePullPolicy,
							Env: []corev1.EnvVar{
								{Name: "CONTENT_SERVICE_MODE", Value: "true"},
								{Name: "STATE_BASE_DIR", Value: "/workspace"},
							},
							Ports: []corev1.ContainerPort{{ContainerPort: 8080, Name: "http"}},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromString("http"),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
							},
							VolumeMounts: []corev1.VolumeMount{{Name: "workspace", MountPath: "/workspace"}},
						},
						{
							Name:            "ambient-code-runner",
							Image:           appConfig.AmbientCodeRunnerImage,
							ImagePullPolicy: appConfig.ImagePullPolicy,
							// 🔒 Container-level security (SCC-compatible, no privileged capabilities)
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: boolPtr(false),
								ReadOnlyRootFilesystem:   boolPtr(false), // Playwright needs to write temp files
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"}, // Drop all capabilities for security
								},
							},

							VolumeMounts: []corev1.VolumeMount{
								{Name: "workspace", MountPath: "/workspace", ReadOnly: false},
								// Mount .claude directory for session state persistence
								// This enables SDK's built-in resume functionality
								{Name: "workspace", MountPath: "/app/.claude", SubPath: fmt.Sprintf("sessions/%s/.claude", name), ReadOnly: false},
							},

							Env: func() []corev1.EnvVar {
								base := []corev1.EnvVar{
									{Name: "DEBUG", Value: "true"},
									{Name: "INTERACTIVE", Value: fmt.Sprintf("%t", interactive)},
									{Name: "AGENTIC_SESSION_NAME", Value: name},
									{Name: "AGENTIC_SESSION_NAMESPACE", Value: sessionNamespace},
									// Provide session id and workspace path for the runner wrapper
									{Name: "SESSION_ID", Value: name},
									{Name: "WORKSPACE_PATH", Value: fmt.Sprintf("/workspace/sessions/%s/workspace", name)},
									{Name: "ARTIFACTS_DIR", Value: "_artifacts"},
									// Provide git input/output parameters to the runner
									{Name: "INPUT_REPO_URL", Value: inputRepo},
									{Name: "INPUT_BRANCH", Value: inputBranch},
									{Name: "OUTPUT_REPO_URL", Value: outputRepo},
									{Name: "OUTPUT_BRANCH", Value: outputBranch},
									{Name: "PROMPT", Value: prompt},
									{Name: "LLM_MODEL", Value: model},
									{Name: "LLM_TEMPERATURE", Value: fmt.Sprintf("%.2f", temperature)},
									{Name: "LLM_MAX_TOKENS", Value: fmt.Sprintf("%d", maxTokens)},
									{Name: "TIMEOUT", Value: fmt.Sprintf("%d", timeout)},
									{Name: "AUTO_PUSH_ON_COMPLETE", Value: fmt.Sprintf("%t", autoPushOnComplete)},
									{Name: "BACKEND_API_URL", Value: fmt.Sprintf("http://backend-service.%s.svc.cluster.local:8080/api", appConfig.BackendNamespace)},
									// WebSocket URL used by runner-shell to connect back to backend
									{Name: "WEBSOCKET_URL", Value: fmt.Sprintf("ws://backend-service.%s.svc.cluster.local:8080/api/projects/%s/sessions/%s/ws", appConfig.BackendNamespace, sessionNamespace, name)},
									// S3 disabled; backend persists messages
								}

								// Add Vertex AI configuration only if enabled
								if vertexEnabled {
									base = append(base,
										corev1.EnvVar{Name: "CLAUDE_CODE_USE_VERTEX", Value: "1"},
										corev1.EnvVar{Name: "CLOUD_ML_REGION", Value: os.Getenv("CLOUD_ML_REGION")},
										corev1.EnvVar{Name: "ANTHROPIC_VERTEX_PROJECT_ID", Value: os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID")},
										corev1.EnvVar{Name: "GOOGLE_APPLICATION_CREDENTIALS", Value: os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")},
									)
								} else {
									// Explicitly set to 0 when Vertex is disabled
									base = append(base, corev1.EnvVar{Name: "CLAUDE_CODE_USE_VERTEX", Value: "0"})
								}

								// Add PARENT_SESSION_ID if this is a continuation
								if parentSessionID != "" {
									base = append(base, corev1.EnvVar{Name: "PARENT_SESSION_ID", Value: parentSessionID})
									log.Printf("Session %s: passing PARENT_SESSION_ID=%s to runner", name, parentSessionID)
								}
								// If backend annotated the session with a runner token secret, inject only BOT_TOKEN
								// Secret contains: 'k8s-token' (for CR updates)
								// Prefer annotated secret name; fallback to deterministic name
								secretName := ""
								if meta, ok := currentObj.Object["metadata"].(map[string]interface{}); ok {
									if anns, ok := meta["annotations"].(map[string]interface{}); ok {
										if v, ok := anns["ambient-code.io/runner-token-secret"].(string); ok && strings.TrimSpace(v) != "" {
											secretName = strings.TrimSpace(v)
										}
									}
								}
								if secretName == "" {
									secretName = fmt.Sprintf("ambient-runner-token-%s", name)
								}
								base = append(base, corev1.EnvVar{
									Name: "BOT_TOKEN",
									ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
										Key:                  "k8s-token",
									}},
								})
								// Add CR-provided envs last (override base when same key)
								if spec, ok := currentObj.Object["spec"].(map[string]interface{}); ok {
									// Inject REPOS_JSON and MAIN_REPO_NAME from spec.repos and spec.mainRepoName if present
									if repos, ok := spec["repos"].([]interface{}); ok && len(repos) > 0 {
										// Use a minimal JSON serialization via fmt (we'll rely on client to pass REPOS_JSON too)
										// This ensures runner gets repos even if env vars weren't passed from frontend
										b, _ := json.Marshal(repos)
										base = append(base, corev1.EnvVar{Name: "REPOS_JSON", Value: string(b)})
									}
									if mrn, ok := spec["mainRepoName"].(string); ok && strings.TrimSpace(mrn) != "" {
										base = append(base, corev1.EnvVar{Name: "MAIN_REPO_NAME", Value: mrn})
									}
									// Inject MAIN_REPO_INDEX if provided
									if mriRaw, ok := spec["mainRepoIndex"]; ok {
										switch v := mriRaw.(type) {
										case int64:
											base = append(base, corev1.EnvVar{Name: "MAIN_REPO_INDEX", Value: fmt.Sprintf("%d", v)})
										case int32:
											base = append(base, corev1.EnvVar{Name: "MAIN_REPO_INDEX", Value: fmt.Sprintf("%d", v)})
										case int:
											base = append(base, corev1.EnvVar{Name: "MAIN_REPO_INDEX", Value: fmt.Sprintf("%d", v)})
										case float64:
											base = append(base, corev1.EnvVar{Name: "MAIN_REPO_INDEX", Value: fmt.Sprintf("%d", int64(v))})
										case string:
											if strings.TrimSpace(v) != "" {
												base = append(base, corev1.EnvVar{Name: "MAIN_REPO_INDEX", Value: v})
											}
										}
									}
									// Inject activeWorkflow environment variables if present
									if workflow, ok := spec["activeWorkflow"].(map[string]interface{}); ok {
										if gitURL, ok := workflow["gitUrl"].(string); ok && strings.TrimSpace(gitURL) != "" {
											base = append(base, corev1.EnvVar{Name: "ACTIVE_WORKFLOW_GIT_URL", Value: gitURL})
										}
										if branch, ok := workflow["branch"].(string); ok && strings.TrimSpace(branch) != "" {
											base = append(base, corev1.EnvVar{Name: "ACTIVE_WORKFLOW_BRANCH", Value: branch})
										}
										if path, ok := workflow["path"].(string); ok && strings.TrimSpace(path) != "" {
											base = append(base, corev1.EnvVar{Name: "ACTIVE_WORKFLOW_PATH", Value: path})
										}
									}
									if envMap, ok := spec["environmentVariables"].(map[string]interface{}); ok {
										for k, v := range envMap {
											if vs, ok := v.(string); ok {
												// replace if exists
												replaced := false
												for i := range base {
													if base[i].Name == k {
														base[i].Value = vs
														replaced = true
														break
													}
												}
												if !replaced {
													base = append(base, corev1.EnvVar{Name: k, Value: vs})
												}
											}
										}
									}
								}

								return base
							}(),

							// Import secrets as environment variables
							// - integrationSecretsName: Only if exists (GIT_TOKEN, JIRA_*, custom keys)
							// - runnerSecretsName: Only when Vertex disabled (ANTHROPIC_API_KEY)
							EnvFrom: func() []corev1.EnvFromSource {
								sources := []corev1.EnvFromSource{}

								// Only inject integration secrets if they exist (optional)
								if integrationSecretsExist {
									sources = append(sources, corev1.EnvFromSource{
										SecretRef: &corev1.SecretEnvSource{
											LocalObjectReference: corev1.LocalObjectReference{Name: integrationSecretsName},
										},
									})
									log.Printf("Injecting integration secrets from '%s' for session %s", integrationSecretsName, name)
								} else {
									log.Printf("Skipping integration secrets '%s' for session %s (not found or not configured)", integrationSecretsName, name)
								}

								// Only inject runner secrets (ANTHROPIC_API_KEY) when Vertex is disabled
								if !vertexEnabled && runnerSecretsName != "" {
									sources = append(sources, corev1.EnvFromSource{
										SecretRef: &corev1.SecretEnvSource{
											LocalObjectReference: corev1.LocalObjectReference{Name: runnerSecretsName},
										},
									})
									log.Printf("Injecting runner secrets from '%s' for session %s (Vertex disabled)", runnerSecretsName, name)
								} else if vertexEnabled && runnerSecretsName != "" {
									log.Printf("Skipping runner secrets '%s' for session %s (Vertex enabled)", runnerSecretsName, name)
								}

								return sources
							}(),

							Resources: corev1.ResourceRequirements{},
						},
					},
				},
			},
		},
	}

	// Note: No volume mounts needed for runner/integration secrets
	// All keys are injected as environment variables via EnvFrom above

	// If ambient-vertex secret was successfully copied, mount it as a volume
	if ambientVertexSecretCopied {
		job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, corev1.Volume{
			Name:         "vertex",
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: types.AmbientVertexSecretName}},
		})
		// Mount to the ambient-code-runner container by name
		for i := range job.Spec.Template.Spec.Containers {
			if job.Spec.Template.Spec.Containers[i].Name == "ambient-code-runner" {
				job.Spec.Template.Spec.Containers[i].VolumeMounts = append(job.Spec.Template.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
					Name:      "vertex",
					MountPath: "/app/vertex",
					ReadOnly:  true,
				})
				log.Printf("Mounted %s secret to /app/vertex in runner container for session %s", types.AmbientVertexSecretName, name)
				break
			}
		}
	}

	// Do not mount runner Secret volume; runner fetches tokens on demand

	// Update status to Creating before attempting job creation
	if err := updateAgenticSessionStatus(sessionNamespace, name, map[string]interface{}{
		"phase":   "Creating",
		"message": "Creating Kubernetes job",
	}); err != nil {
		log.Printf("Failed to update AgenticSession status to Creating: %v", err)
		// Continue anyway - resource might have been deleted
	}

	// Create the job
	createdJob, err := config.K8sClient.BatchV1().Jobs(sessionNamespace).Create(context.TODO(), job, v1.CreateOptions{})
	if err != nil {
		// If job already exists, this is likely a race condition from duplicate watch events - not an error
		if errors.IsAlreadyExists(err) {
			log.Printf("Job %s already exists (race condition), continuing", jobName)
			return nil
		}
		log.Printf("Failed to create job %s: %v", jobName, err)
		// Update status to Error if job creation fails and resource still exists
		updateAgenticSessionStatus(sessionNamespace, name, map[string]interface{}{
			"phase":   "Error",
			"message": fmt.Sprintf("Failed to create job: %v", err),
		})
		return fmt.Errorf("failed to create job: %v", err)
	}

	log.Printf("Created job %s for AgenticSession %s", jobName, name)

	// Update AgenticSession status to Running
	if err := updateAgenticSessionStatus(sessionNamespace, name, map[string]interface{}{
		"phase":     "Creating",
		"message":   "Job is being set up",
		"startTime": time.Now().Format(time.RFC3339),
		"jobName":   jobName,
	}); err != nil {
		log.Printf("Failed to update AgenticSession status to Creating: %v", err)
		// Don't return error here - the job was created successfully
		// The status update failure might be due to the resource being deleted
	}

	// Create a per-job Service pointing to the content container
	svc := &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      fmt.Sprintf("ambient-content-%s", name),
			Namespace: sessionNamespace,
			Labels:    map[string]string{"app": "ambient-code-runner", "agentic-session": name},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "batch/v1",
				Kind:       "Job",
				Name:       jobName,
				UID:        createdJob.UID,
				Controller: boolPtr(true),
			}},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"job-name": jobName},
			Ports:    []corev1.ServicePort{{Port: 8080, TargetPort: intstr.FromString("http"), Protocol: corev1.ProtocolTCP, Name: "http"}},
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
	if _, serr := config.K8sClient.CoreV1().Services(sessionNamespace).Create(context.TODO(), svc, v1.CreateOptions{}); serr != nil && !errors.IsAlreadyExists(serr) {
		log.Printf("Failed to create per-job content service for %s: %v", name, serr)
	}

	// Start monitoring the job
	go monitorJob(jobName, name, sessionNamespace)

	return nil
}

func monitorJob(jobName, sessionName, sessionNamespace string) {
	log.Printf("Starting job monitoring for %s (session: %s/%s)", jobName, sessionNamespace, sessionName)

	// Main is now the content container to keep service alive
	mainContainerName := "ambient-content"

	// Track if we've verified owner references
	ownerRefsChecked := false

	for {
		time.Sleep(5 * time.Second)

		// Ensure the AgenticSession still exists
		gvr := types.GetAgenticSessionResource()
		if _, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), sessionName, v1.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				log.Printf("AgenticSession %s no longer exists, stopping job monitoring for %s", sessionName, jobName)
				return
			}
			log.Printf("Error checking AgenticSession %s existence: %v", sessionName, err)
		}

		// Get Job
		job, err := config.K8sClient.BatchV1().Jobs(sessionNamespace).Get(context.TODO(), jobName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				log.Printf("Job %s not found, stopping monitoring", jobName)
				return
			}
			log.Printf("Error getting job %s: %v", jobName, err)
			continue
		}

		// Verify pod owner references once (diagnostic)
		if !ownerRefsChecked && job.Status.Active > 0 {
			pods, err := config.K8sClient.CoreV1().Pods(sessionNamespace).List(context.TODO(), v1.ListOptions{
				LabelSelector: fmt.Sprintf("job-name=%s", jobName),
			})
			if err == nil && len(pods.Items) > 0 {
				for _, pod := range pods.Items {
					hasJobOwner := false
					for _, ownerRef := range pod.OwnerReferences {
						if ownerRef.Kind == "Job" && ownerRef.Name == jobName {
							hasJobOwner = true
							break
						}
					}
					if !hasJobOwner {
						log.Printf("WARNING: Pod %s does NOT have Job %s as owner reference! This will prevent automatic cleanup.", pod.Name, jobName)
					} else {
						log.Printf("✓ Pod %s has correct Job owner reference", pod.Name)
					}
				}
				ownerRefsChecked = true
			}
		}

		// If K8s already marked the Job as succeeded, mark session Completed but defer cleanup
		// BUT: respect terminal statuses already set by wrapper (Failed, Completed)
		if job.Status.Succeeded > 0 {
			// Check current status before overriding
			gvr := types.GetAgenticSessionResource()
			currentObj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), sessionName, v1.GetOptions{})
			currentPhase := ""
			if err == nil && currentObj != nil {
				if status, found, _ := unstructured.NestedMap(currentObj.Object, "status"); found {
					if v, ok := status["phase"].(string); ok {
						currentPhase = v
					}
				}
			}
			// Only set to Completed if not already in a terminal state (Failed, Completed, Stopped)
			if currentPhase != "Failed" && currentPhase != "Completed" && currentPhase != "Stopped" {
				log.Printf("Job %s marked succeeded by Kubernetes, setting to Completed", jobName)
				_ = updateAgenticSessionStatus(sessionNamespace, sessionName, map[string]interface{}{
					"phase":          "Completed",
					"message":        "Job completed successfully",
					"completionTime": time.Now().Format(time.RFC3339),
				})
				// Ensure session is interactive so it can be restarted
				_ = ensureSessionIsInteractive(sessionNamespace, sessionName)
			} else {
				log.Printf("Job %s marked succeeded by Kubernetes, but status already %s (not overriding)", jobName, currentPhase)
			}
			// Do not delete here; defer cleanup until all repos are finalized
		}

		// If Job has failed according to backoff policy, mark failed
		if job.Spec.BackoffLimit != nil && job.Status.Failed >= *job.Spec.BackoffLimit {
			log.Printf("Job %s failed after %d attempts", jobName, job.Status.Failed)
			failureMsg := "Job failed"
			if pods, err := config.K8sClient.CoreV1().Pods(sessionNamespace).List(context.TODO(), v1.ListOptions{LabelSelector: fmt.Sprintf("job-name=%s", jobName)}); err == nil && len(pods.Items) > 0 {
				pod := pods.Items[0]
				if logs, err := config.K8sClient.CoreV1().Pods(sessionNamespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).DoRaw(context.TODO()); err == nil {
					failureMsg = fmt.Sprintf("Job failed: %s", string(logs))
					if len(failureMsg) > 500 {
						failureMsg = failureMsg[:500] + "..."
					}
				}
			}

			// Only update to Failed if not already in a terminal state
			gvr := types.GetAgenticSessionResource()
			if currentObj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), sessionName, v1.GetOptions{}); err == nil {
				currentPhase := ""
				if status, found, _ := unstructured.NestedMap(currentObj.Object, "status"); found {
					if v, ok := status["phase"].(string); ok {
						currentPhase = v
					}
				}
				if currentPhase != "Failed" && currentPhase != "Completed" && currentPhase != "Stopped" {
					_ = updateAgenticSessionStatus(sessionNamespace, sessionName, map[string]interface{}{
						"phase":          "Failed",
						"message":        failureMsg,
						"completionTime": time.Now().Format(time.RFC3339),
					})
					// Ensure session is interactive so it can be restarted
					_ = ensureSessionIsInteractive(sessionNamespace, sessionName)
				}
			}
			_ = deleteJobAndPerJobService(sessionNamespace, jobName, sessionName)
			return
		}

		// Inspect pods to determine main container state regardless of sidecar
		pods, err := config.K8sClient.CoreV1().Pods(sessionNamespace).List(context.TODO(), v1.ListOptions{LabelSelector: fmt.Sprintf("job-name=%s", jobName)})
		if err != nil {
			log.Printf("Error listing pods for job %s: %v", jobName, err)
			continue
		}

		// Check for job with no active pods (pod evicted/preempted/deleted)
		if len(pods.Items) == 0 && job.Status.Active == 0 && job.Status.Succeeded == 0 && job.Status.Failed == 0 {
			// Check current phase to see if this is unexpected
			gvr := types.GetAgenticSessionResource()
			if currentObj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), sessionName, v1.GetOptions{}); err == nil {
				currentPhase := ""
				if status, found, _ := unstructured.NestedMap(currentObj.Object, "status"); found {
					if v, ok := status["phase"].(string); ok {
						currentPhase = v
					}
				}
				// If session is Running but pod is gone, mark as Failed
				if currentPhase == "Running" || currentPhase == "Creating" {
					log.Printf("Job %s has no pods but session is %s, marking as Failed", jobName, currentPhase)
					_ = updateAgenticSessionStatus(sessionNamespace, sessionName, map[string]interface{}{
						"phase":          "Failed",
						"message":        "Job pod was deleted or evicted unexpectedly",
						"completionTime": time.Now().Format(time.RFC3339),
					})
					_ = deleteJobAndPerJobService(sessionNamespace, jobName, sessionName)
					return
				}
			}
			continue
		}

		if len(pods.Items) == 0 {
			continue
		}
		pod := pods.Items[0]

		// Check for pod-level failures (ImagePullBackOff, CrashLoopBackOff, etc.)
		if pod.Status.Phase == corev1.PodFailed {
			gvr := types.GetAgenticSessionResource()
			if currentObj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), sessionName, v1.GetOptions{}); err == nil {
				currentPhase := ""
				if status, found, _ := unstructured.NestedMap(currentObj.Object, "status"); found {
					if v, ok := status["phase"].(string); ok {
						currentPhase = v
					}
				}
				// Only update if not already in terminal state
				if currentPhase != "Failed" && currentPhase != "Completed" && currentPhase != "Stopped" {
					failureMsg := fmt.Sprintf("Pod failed: %s - %s", pod.Status.Reason, pod.Status.Message)
					log.Printf("Job %s pod in Failed phase, updating session to Failed: %s", jobName, failureMsg)
					_ = updateAgenticSessionStatus(sessionNamespace, sessionName, map[string]interface{}{
						"phase":          "Failed",
						"message":        failureMsg,
						"completionTime": time.Now().Format(time.RFC3339),
					})
					_ = deleteJobAndPerJobService(sessionNamespace, jobName, sessionName)
					return
				}
			}
		}

		// Check for containers in waiting state with errors (ImagePullBackOff, CrashLoopBackOff, etc.)
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil {
				waiting := cs.State.Waiting
				// Check for error states that indicate permanent failure
				errorStates := []string{"ImagePullBackOff", "ErrImagePull", "CrashLoopBackOff", "CreateContainerConfigError", "InvalidImageName"}
				for _, errState := range errorStates {
					if waiting.Reason == errState {
						gvr := types.GetAgenticSessionResource()
						if currentObj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), sessionName, v1.GetOptions{}); err == nil {
							currentPhase := ""
							if status, found, _ := unstructured.NestedMap(currentObj.Object, "status"); found {
								if v, ok := status["phase"].(string); ok {
									currentPhase = v
								}
							}
							// Only update if not already in terminal state and we've been in this state for a while
							if currentPhase == "Running" || currentPhase == "Creating" {
								failureMsg := fmt.Sprintf("Container %s failed: %s - %s", cs.Name, waiting.Reason, waiting.Message)
								log.Printf("Job %s container in error state, updating session to Failed: %s", jobName, failureMsg)
								_ = updateAgenticSessionStatus(sessionNamespace, sessionName, map[string]interface{}{
									"phase":          "Failed",
									"message":        failureMsg,
									"completionTime": time.Now().Format(time.RFC3339),
								})
								_ = deleteJobAndPerJobService(sessionNamespace, jobName, sessionName)
								return
							}
						}
					}
				}
			}
		}

		// If main container is running and phase hasn't been set to Running yet, update
		if cs := getContainerStatusByName(&pod, mainContainerName); cs != nil {
			if cs.State.Running != nil {
				// Avoid downgrading terminal phases; only set Running when not already terminal
				func() {
					gvr := types.GetAgenticSessionResource()
					obj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), sessionName, v1.GetOptions{})
					if err != nil || obj == nil {
						// Best-effort: still try to set Running
						_ = updateAgenticSessionStatus(sessionNamespace, sessionName, map[string]interface{}{
							"phase":   "Running",
							"message": "Agent is running",
						})
						return
					}
					status, _, _ := unstructured.NestedMap(obj.Object, "status")
					current := ""
					if v, ok := status["phase"].(string); ok {
						current = v
					}
					if current != "Completed" && current != "Stopped" && current != "Failed" && current != "Running" {
						_ = updateAgenticSessionStatus(sessionNamespace, sessionName, map[string]interface{}{
							"phase":   "Running",
							"message": "Agent is running",
						})
					}
				}()
			}
			if cs.State.Terminated != nil {
				log.Printf("Content container terminated for job %s; checking runner container status instead", jobName)
				// Don't use content container exit code - check runner instead below
			}
		}

		// Check runner container status (the actual work is done here, not in content container)
		runnerContainerName := "ambient-code-runner"
		runnerStatus := getContainerStatusByName(&pod, runnerContainerName)
		if runnerStatus != nil && runnerStatus.State.Terminated != nil {
			term := runnerStatus.State.Terminated

			// Get current CR status to check if wrapper already set it
			gvr := types.GetAgenticSessionResource()
			obj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), sessionName, v1.GetOptions{})
			currentPhase := ""
			if err == nil && obj != nil {
				status, _, _ := unstructured.NestedMap(obj.Object, "status")
				if v, ok := status["phase"].(string); ok {
					currentPhase = v
				}
			}

			// If wrapper already set status to Completed, clean up immediately
			if currentPhase == "Completed" || currentPhase == "Failed" {
				log.Printf("Runner exited for job %s with phase %s", jobName, currentPhase)

				// Ensure session is interactive so it can be restarted
				_ = ensureSessionIsInteractive(sessionNamespace, sessionName)

				// Clean up Job/Service immediately
				_ = deleteJobAndPerJobService(sessionNamespace, jobName, sessionName)

				// Keep PVC - it will be deleted via garbage collection when session CR is deleted
				// This allows users to restart completed sessions and reuse the workspace
				log.Printf("Session %s completed, keeping PVC for potential restart", sessionName)
				return
			}

			// Runner exit code 0 = success (fallback if wrapper didn't set status)
			if term.ExitCode == 0 {
				_ = updateAgenticSessionStatus(sessionNamespace, sessionName, map[string]interface{}{
					"phase":          "Completed",
					"message":        "Runner completed successfully",
					"completionTime": time.Now().Format(time.RFC3339),
				})
				// Ensure session is interactive so it can be restarted
				_ = ensureSessionIsInteractive(sessionNamespace, sessionName)
				log.Printf("Runner container exited successfully for job %s", jobName)
				// Will cleanup on next iteration
				continue
			}

			// Runner non-zero exit = failure
			msg := term.Message
			if msg == "" {
				msg = fmt.Sprintf("Runner container exited with code %d", term.ExitCode)
			}
			_ = updateAgenticSessionStatus(sessionNamespace, sessionName, map[string]interface{}{
				"phase":   "Failed",
				"message": msg,
			})
			// Ensure session is interactive so it can be restarted
			_ = ensureSessionIsInteractive(sessionNamespace, sessionName)
			log.Printf("Runner container failed for job %s: %s", jobName, msg)
			// Will cleanup on next iteration
			continue
		}

		// Note: Job/Pod cleanup now happens immediately when runner exits (see above)
		// This loop continues to monitor until cleanup happens
	}
}

// getContainerStatusByName returns the ContainerStatus for a given container name
func getContainerStatusByName(pod *corev1.Pod, name string) *corev1.ContainerStatus {
	for i := range pod.Status.ContainerStatuses {
		if pod.Status.ContainerStatuses[i].Name == name {
			return &pod.Status.ContainerStatuses[i]
		}
	}
	return nil
}

// deleteJobAndPerJobService deletes the Job and its associated per-job Service
func deleteJobAndPerJobService(namespace, jobName, sessionName string) error {
	// Delete Service first (it has ownerRef to Job, but delete explicitly just in case)
	svcName := fmt.Sprintf("ambient-content-%s", sessionName)
	if err := config.K8sClient.CoreV1().Services(namespace).Delete(context.TODO(), svcName, v1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to delete per-job service %s/%s: %v", namespace, svcName, err)
	}

	// Delete the Job with background propagation
	policy := v1.DeletePropagationBackground
	if err := config.K8sClient.BatchV1().Jobs(namespace).Delete(context.TODO(), jobName, v1.DeleteOptions{PropagationPolicy: &policy}); err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to delete job %s/%s: %v", namespace, jobName, err)
		return err
	}

	// Proactively delete Pods for this Job
	if pods, err := config.K8sClient.CoreV1().Pods(namespace).List(context.TODO(), v1.ListOptions{LabelSelector: fmt.Sprintf("job-name=%s", jobName)}); err == nil {
		for i := range pods.Items {
			p := pods.Items[i]
			if err := config.K8sClient.CoreV1().Pods(namespace).Delete(context.TODO(), p.Name, v1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
				log.Printf("Failed to delete pod %s/%s for job %s: %v", namespace, p.Name, jobName, err)
			}
		}
	} else if !errors.IsNotFound(err) {
		log.Printf("Failed to list pods for job %s/%s: %v", namespace, jobName, err)
	}

	// Delete the ambient-vertex secret if it was copied by the operator
	deleteCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := deleteAmbientVertexSecret(deleteCtx, namespace); err != nil {
		log.Printf("Failed to delete %s secret from %s: %v", types.AmbientVertexSecretName, namespace, err)
		// Don't return error - this is a non-critical cleanup step
	}

	// NOTE: PVC is kept for all sessions and only deleted via garbage collection
	// when the session CR is deleted. This allows sessions to be restarted.

	return nil
}

func updateAgenticSessionStatus(sessionNamespace, name string, statusUpdate map[string]interface{}) error {
	gvr := types.GetAgenticSessionResource()

	// Get current resource
	obj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("AgenticSession %s no longer exists, skipping status update", name)
			return nil // Don't treat this as an error - resource was deleted
		}
		return fmt.Errorf("failed to get AgenticSession %s: %v", name, err)
	}

	// Update status
	if obj.Object["status"] == nil {
		obj.Object["status"] = make(map[string]interface{})
	}

	status := obj.Object["status"].(map[string]interface{})
	for key, value := range statusUpdate {
		status[key] = value
	}

	// Update the resource with retry logic
	_, err = config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).UpdateStatus(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("AgenticSession %s was deleted during status update, skipping", name)
			return nil // Don't treat this as an error - resource was deleted
		}
		return fmt.Errorf("failed to update AgenticSession status: %v", err)
	}

	return nil
}

// ensureSessionIsInteractive updates a session's spec to set interactive: true
// This allows completed sessions to be restarted without requiring manual spec file removal
func ensureSessionIsInteractive(sessionNamespace, name string) error {
	gvr := types.GetAgenticSessionResource()

	// Get current resource
	obj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("AgenticSession %s no longer exists, skipping interactive update", name)
			return nil // Don't treat this as an error - resource was deleted
		}
		return fmt.Errorf("failed to get AgenticSession %s: %v", name, err)
	}

	// Check if spec exists and if interactive is already true
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil {
		return fmt.Errorf("failed to get spec from AgenticSession %s: %v", name, err)
	}
	if !found {
		log.Printf("AgenticSession %s has no spec, cannot update interactive", name)
		return nil
	}

	// Check current interactive value
	interactive, _, _ := unstructured.NestedBool(spec, "interactive")
	if interactive {
		log.Printf("AgenticSession %s is already interactive, no update needed", name)
		return nil
	}

	// Update spec to set interactive: true
	if err := unstructured.SetNestedField(obj.Object, true, "spec", "interactive"); err != nil {
		return fmt.Errorf("failed to set interactive field for AgenticSession %s: %v", name, err)
	}

	log.Printf("Setting interactive: true for AgenticSession %s to allow restart", name)

	// Update the resource (not UpdateStatus, since we're modifying spec)
	_, err = config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Update(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("AgenticSession %s was deleted during spec update, skipping", name)
			return nil // Don't treat this as an error - resource was deleted
		}
		return fmt.Errorf("failed to update AgenticSession spec: %v", err)
	}

	log.Printf("Successfully set interactive: true for AgenticSession %s", name)
	return nil
}

// CleanupExpiredTempContentPods removes temporary content pods that have exceeded their TTL
func CleanupExpiredTempContentPods() {
	log.Println("Starting temp content pod cleanup goroutine")
	for {
		time.Sleep(1 * time.Minute)

		// List all temp content pods across all namespaces
		pods, err := config.K8sClient.CoreV1().Pods("").List(context.TODO(), v1.ListOptions{
			LabelSelector: "app=temp-content-service",
		})
		if err != nil {
			log.Printf("Failed to list temp content pods: %v", err)
			continue
		}

		for _, pod := range pods.Items {
			// Check TTL annotation
			createdAtStr := pod.Annotations["vteam.ambient-code/created-at"]
			ttlStr := pod.Annotations["vteam.ambient-code/ttl"]

			if createdAtStr == "" || ttlStr == "" {
				continue
			}

			createdAt, err := time.Parse(time.RFC3339, createdAtStr)
			if err != nil {
				log.Printf("Failed to parse created-at for pod %s: %v", pod.Name, err)
				continue
			}

			ttlSeconds := int64(0)
			if _, err := fmt.Sscanf(ttlStr, "%d", &ttlSeconds); err != nil {
				log.Printf("Failed to parse TTL for pod %s: %v", pod.Name, err)
				continue
			}

			ttlDuration := time.Duration(ttlSeconds) * time.Second
			if time.Since(createdAt) > ttlDuration {
				log.Printf("Deleting expired temp content pod: %s/%s (age: %v, ttl: %v)",
					pod.Namespace, pod.Name, time.Since(createdAt), ttlDuration)
				if err := config.K8sClient.CoreV1().Pods(pod.Namespace).Delete(context.TODO(), pod.Name, v1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
					log.Printf("Failed to delete expired temp pod %s/%s: %v", pod.Namespace, pod.Name, err)
				}
			}
		}
	}
}

// copySecretToNamespace copies a secret to a target namespace with owner references
func copySecretToNamespace(ctx context.Context, sourceSecret *corev1.Secret, targetNamespace string, ownerObj *unstructured.Unstructured) error {
	// Check if secret already exists in target namespace
	existingSecret, err := config.K8sClient.CoreV1().Secrets(targetNamespace).Get(ctx, sourceSecret.Name, v1.GetOptions{})
	secretExists := err == nil
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error checking for existing secret: %w", err)
	}

	// Determine if we should set Controller: true
	// For shared secrets (like ambient-vertex), don't set Controller: true if secret already exists
	// to avoid conflicts when multiple sessions use the same secret
	shouldSetController := true
	if secretExists {
		// Check if existing secret already has a controller reference
		for _, ownerRef := range existingSecret.OwnerReferences {
			if ownerRef.Controller != nil && *ownerRef.Controller {
				shouldSetController = false
				log.Printf("Secret %s already has a controller reference, adding non-controller reference instead", sourceSecret.Name)
				break
			}
		}
	}

	// Create owner reference
	newOwnerRef := v1.OwnerReference{
		APIVersion: ownerObj.GetAPIVersion(),
		Kind:       ownerObj.GetKind(),
		Name:       ownerObj.GetName(),
		UID:        ownerObj.GetUID(),
	}
	if shouldSetController {
		newOwnerRef.Controller = boolPtr(true)
	}

	// Create a new secret in the target namespace
	newSecret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      sourceSecret.Name,
			Namespace: targetNamespace,
			Labels:    sourceSecret.Labels,
			Annotations: map[string]string{
				types.CopiedFromAnnotation: fmt.Sprintf("%s/%s", sourceSecret.Namespace, sourceSecret.Name),
			},
			OwnerReferences: []v1.OwnerReference{newOwnerRef},
		},
		Type: sourceSecret.Type,
		Data: sourceSecret.Data,
	}

	if secretExists {
		// Secret already exists, check if it needs to be updated
		log.Printf("Secret %s already exists in namespace %s, checking if update needed", sourceSecret.Name, targetNamespace)

		// Check if the existing secret has the correct owner reference
		hasOwnerRef := false
		for _, ownerRef := range existingSecret.OwnerReferences {
			if ownerRef.UID == ownerObj.GetUID() {
				hasOwnerRef = true
				break
			}
		}

		if hasOwnerRef {
			log.Printf("Secret %s already has correct owner reference, skipping", sourceSecret.Name)
			return nil
		}

		// Update the secret with owner reference using retry logic to handle race conditions
		return retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// Re-fetch the secret to get the latest version
			currentSecret, err := config.K8sClient.CoreV1().Secrets(targetNamespace).Get(ctx, sourceSecret.Name, v1.GetOptions{})
			if err != nil {
				return err
			}

			// Check again if there's already a controller reference (may have changed since last check)
			hasController := false
			for _, ownerRef := range currentSecret.OwnerReferences {
				if ownerRef.Controller != nil && *ownerRef.Controller {
					hasController = true
					break
				}
			}

			// Create a fresh owner reference based on current state
			// If there's already a controller, don't set Controller: true for the new reference
			ownerRefToAdd := newOwnerRef
			if hasController {
				ownerRefToAdd.Controller = nil
			}

			// Apply updates
			// Create a new slice to avoid mutating shared/cached data
			currentSecret.OwnerReferences = append([]v1.OwnerReference{}, currentSecret.OwnerReferences...)
			currentSecret.OwnerReferences = append(currentSecret.OwnerReferences, ownerRefToAdd)
			currentSecret.Data = sourceSecret.Data
			if currentSecret.Annotations == nil {
				currentSecret.Annotations = make(map[string]string)
			}
			currentSecret.Annotations[types.CopiedFromAnnotation] = fmt.Sprintf("%s/%s", sourceSecret.Namespace, sourceSecret.Name)

			// Attempt update
			_, err = config.K8sClient.CoreV1().Secrets(targetNamespace).Update(ctx, currentSecret, v1.UpdateOptions{})
			return err
		})
	}

	// Create the secret
	_, err = config.K8sClient.CoreV1().Secrets(targetNamespace).Create(ctx, newSecret, v1.CreateOptions{})
	return err
}

// deleteAmbientVertexSecret deletes the ambient-vertex secret from a namespace if it was copied
func deleteAmbientVertexSecret(ctx context.Context, namespace string) error {
	secret, err := config.K8sClient.CoreV1().Secrets(namespace).Get(ctx, types.AmbientVertexSecretName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Secret doesn't exist, nothing to do
			return nil
		}
		return fmt.Errorf("error checking for %s secret: %w", types.AmbientVertexSecretName, err)
	}

	// Check if this was a copied secret (has the annotation)
	if _, ok := secret.Annotations[types.CopiedFromAnnotation]; !ok {
		log.Printf("%s secret in namespace %s was not copied by operator, not deleting", types.AmbientVertexSecretName, namespace)
		return nil
	}

	log.Printf("Deleting copied %s secret from namespace %s", types.AmbientVertexSecretName, namespace)
	err = config.K8sClient.CoreV1().Secrets(namespace).Delete(ctx, types.AmbientVertexSecretName, v1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s secret: %w", types.AmbientVertexSecretName, err)
	}

	return nil
}

// Helper functions
var (
	boolPtr  = func(b bool) *bool { return &b }
	int32Ptr = func(i int32) *int32 { return &i }
	int64Ptr = func(i int64) *int64 { return &i }
)
