package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
)

// WatchAgenticSessions watches for AgenticSession events across all namespaces
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

				if err := HandleAgenticSessionEvent(obj); err != nil {
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

// HandleAgenticSessionEvent handles AgenticSession events
func HandleAgenticSessionEvent(obj *unstructured.Unstructured) error {
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
		_ = UpdateAgenticSessionStatus(sessionNamespace, name, map[string]interface{}{"phase": "Pending"})
		phase = "Pending"
	}

	log.Printf("Processing AgenticSession %s with phase %s", name, phase)

	// Handle sessions marked for stopping
	if phase == "Stopped" {
		return HandleStoppedSession(currentObj)
	}

	// Only process if status is Pending
	if phase != "Pending" {
		return nil
	}

	// Ensure a per-project workspace PVC exists for runner artifacts
	if err := services.EnsureProjectWorkspacePVC(sessionNamespace); err != nil {
		log.Printf("Failed to ensure workspace PVC in %s: %v", sessionNamespace, err)
		// Continue; job may still run with ephemeral storage
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
	workspaceStorePath, workspaceStorePathFound, _ := unstructured.NestedString(spec, "paths", "workspace")
	messageStorePath, messageStorePathFound, _ := unstructured.NestedString(spec, "paths", "messages")
	// Extract git configuration
	gitConfig, _, _ := unstructured.NestedMap(spec, "gitConfig")
	gitUserName, _, _ := unstructured.NestedString(gitConfig, "user", "name")
	gitUserEmail, _, _ := unstructured.NestedString(gitConfig, "user", "email")
	sshKeySecret, _, _ := unstructured.NestedString(gitConfig, "authentication", "sshKeySecret")
	tokenSecret, _, _ := unstructured.NestedString(gitConfig, "authentication", "tokenSecret")
	repositories, _, _ := unstructured.NestedSlice(gitConfig, "repositories")

	// Marshal repositories to JSON string for runner env var
	reposJSON := "[]"
	if len(repositories) > 0 {
		if b, err := json.Marshal(repositories); err == nil {
			reposJSON = string(b)
		} else {
			log.Printf("Failed to marshal git repositories: %v", err)
		}
	}

	// Read runner secrets configuration from ProjectSettings in the session's namespace
	runnerSecretsName := ""
	{
		psGvr := types.GetProjectSettingsResource()
		if psObj, err := config.DynamicClient.Resource(psGvr).Namespace(sessionNamespace).Get(context.TODO(), "projectsettings", v1.GetOptions{}); err == nil {
			if psSpec, ok := psObj.Object["spec"].(map[string]interface{}); ok {
				if v, ok := psSpec["runnerSecretsName"].(string); ok {
					runnerSecretsName = strings.TrimSpace(v)
				}
			}
		}
	}

	appConfig := config.LoadConfig()

	// Create the Job
	job := createJobSpec(currentObj, jobName, sessionNamespace, name, appConfig, runnerSecretsName, prompt, timeout, interactive, model, temperature, maxTokens, workspaceStorePath, workspaceStorePathFound, messageStorePath, messageStorePathFound, gitUserName, gitUserEmail, sshKeySecret, tokenSecret, reposJSON)

	// If a runner secret is configured, mount it as a volume in addition to EnvFrom
	if runnerSecretsName != "" {
		job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: "runner-secrets",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{SecretName: runnerSecretsName},
			},
		})
		if len(job.Spec.Template.Spec.Containers) > 0 {
			job.Spec.Template.Spec.Containers[0].VolumeMounts = append(job.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
				Name:      "runner-secrets",
				MountPath: "/var/run/runner-secrets",
				ReadOnly:  true,
			})
		}
	}

	// Update status to Creating before attempting job creation
	if err := UpdateAgenticSessionStatus(sessionNamespace, name, map[string]interface{}{
		"phase":   "Creating",
		"message": "Creating Kubernetes job",
	}); err != nil {
		log.Printf("Failed to update AgenticSession status to Creating: %v", err)
		// Continue anyway - resource might have been deleted
	}

	// Create the job
	_, err = config.K8sClient.BatchV1().Jobs(sessionNamespace).Create(context.TODO(), job, v1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create job %s: %v", jobName, err)
		// Update status to Error if job creation fails and resource still exists
		UpdateAgenticSessionStatus(sessionNamespace, name, map[string]interface{}{
			"phase":   "Error",
			"message": fmt.Sprintf("Failed to create job: %v", err),
		})
		return fmt.Errorf("failed to create job: %v", err)
	}

	log.Printf("Created job %s for AgenticSession %s", jobName, name)

	// Update AgenticSession status to Running
	if err := UpdateAgenticSessionStatus(sessionNamespace, name, map[string]interface{}{
		"phase":     "Running",
		"message":   "Job created and running",
		"startTime": time.Now().Format(time.RFC3339),
		"jobName":   jobName,
	}); err != nil {
		log.Printf("Failed to update AgenticSession status to Running: %v", err)
		// Don't return error here - the job was created successfully
		// The status update failure might be due to the resource being deleted
	}

	// Start monitoring the job
	go MonitorJob(jobName, name, sessionNamespace)

	return nil
}

// HandleStoppedSession handles AgenticSessions marked as "Stopped" by terminating their associated jobs
func HandleStoppedSession(obj *unstructured.Unstructured) error {
	name := obj.GetName()
	sessionNamespace := obj.GetNamespace()

	log.Printf("Handling stopped session %s/%s", sessionNamespace, name)

	// Find the associated job
	jobName := fmt.Sprintf("%s-job", name)

	// Check if job exists
	job, err := config.K8sClient.BatchV1().Jobs(sessionNamespace).Get(context.TODO(), jobName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("Job %s for session %s not found, may have already been cleaned up", jobName, name)
			return nil
		}
		return fmt.Errorf("failed to get job %s: %v", jobName, err)
	}

	// Check if job is already completed or failed
	if job.Status.Succeeded > 0 || job.Status.Failed > 0 {
		log.Printf("Job %s is already completed (succeeded=%d, failed=%d)", jobName, job.Status.Succeeded, job.Status.Failed)
		return nil
	}

	// Delete the job to stop it
	// Use background deletion policy to immediately stop pods
	propagationPolicy := v1.DeletePropagationBackground
	err = config.K8sClient.BatchV1().Jobs(sessionNamespace).Delete(context.TODO(), jobName, v1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("Job %s already deleted", jobName)
			return nil
		}
		return fmt.Errorf("failed to delete job %s: %v", jobName, err)
	}

	log.Printf("Successfully deleted job %s for stopped session %s", jobName, name)
	return nil
}

// createJobSpec creates a Kubernetes Job specification for an AgenticSession
func createJobSpec(currentObj *unstructured.Unstructured, jobName, sessionNamespace, name string, appConfig *config.Config, runnerSecretsName, prompt string, timeout int64, interactive bool, model string, temperature float64, maxTokens int64, workspaceStorePath string, workspaceStorePathFound bool, messageStorePath string, messageStorePathFound bool, gitUserName, gitUserEmail, sshKeySecret, tokenSecret, reposJSON string) *batchv1.Job {
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
			ActiveDeadlineSeconds: int64Ptr(1800), // 30 minute timeout for safety
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
					// Hard anti-race: prefer runner to schedule on same node as ambient-content for RWO PVCs
					Affinity: &corev1.Affinity{
						PodAffinity: &corev1.PodAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
								{
									Weight: 100,
									PodAffinityTerm: corev1.PodAffinityTerm{
										LabelSelector: &v1.LabelSelector{MatchLabels: map[string]string{"app": "ambient-content"}},
										Namespaces:    []string{sessionNamespace},
										TopologyKey:   "kubernetes.io/hostname",
									},
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes: []corev1.Volume{
						{
							Name: "workspace",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "ambient-workspace",
								},
							},
						},
					},

					Containers: []corev1.Container{
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
								{Name: "workspace", MountPath: "/workspace", ReadOnly: true},
							},

							Env: buildEnvVars(currentObj, name, sessionNamespace, prompt, model, temperature, maxTokens, timeout, interactive, appConfig, workspaceStorePath, workspaceStorePathFound, messageStorePath, messageStorePathFound, gitUserName, gitUserEmail, sshKeySecret, tokenSecret, reposJSON),

							// If configured, import all keys from the runner Secret as environment variables
							EnvFrom: func() []corev1.EnvFromSource {
								if runnerSecretsName != "" {
									return []corev1.EnvFromSource{
										{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: runnerSecretsName}}},
									}
								}
								return []corev1.EnvFromSource{}
							}(),

							Resources: corev1.ResourceRequirements{},
						},
					},
				},
			},
		},
	}

	return job
}

// buildEnvVars builds the environment variables for the job container
func buildEnvVars(currentObj *unstructured.Unstructured, name, sessionNamespace, prompt, model string, temperature float64, maxTokens, timeout int64, interactive bool, appConfig *config.Config, workspaceStorePath string, workspaceStorePathFound bool, messageStorePath string, messageStorePathFound bool, gitUserName, gitUserEmail, sshKeySecret, tokenSecret, reposJSON string) []corev1.EnvVar {
	base := []corev1.EnvVar{
		{Name: "DEBUG", Value: "false"},
		{Name: "INTERACTIVE", Value: fmt.Sprintf("%t", interactive)},
		{Name: "AGENTIC_SESSION_NAME", Value: name},
		{Name: "AGENTIC_SESSION_NAMESPACE", Value: sessionNamespace},
		{Name: "PROMPT", Value: prompt},
		{Name: "LLM_MODEL", Value: model},
		{Name: "LLM_TEMPERATURE", Value: fmt.Sprintf("%.2f", temperature)},
		{Name: "LLM_MAX_TOKENS", Value: fmt.Sprintf("%d", maxTokens)},
		{Name: "TIMEOUT", Value: fmt.Sprintf("%d", timeout)},
		{Name: "BACKEND_API_URL", Value: fmt.Sprintf("http://backend-service.%s.svc.cluster.local:8080/api", appConfig.BackendNamespace)},
		{Name: "PVC_PROXY_API_URL", Value: fmt.Sprintf("http://ambient-content.%s.svc:8080", sessionNamespace)},
		{Name: "WORKSPACE_STORE_PATH", Value: func() string {
			if workspaceStorePathFound {
				return workspaceStorePath
			}
			return fmt.Sprintf("/sessions/%s/workspace", name)
		}()},
		{Name: "MESSAGE_STORE_PATH", Value: func() string {
			if messageStorePathFound {
				return messageStorePath
			}
			return fmt.Sprintf("/sessions/%s/messages.json", name)
		}()},
		{Name: "GIT_USER_NAME", Value: gitUserName},
		{Name: "GIT_USER_EMAIL", Value: gitUserEmail},
		{Name: "GIT_SSH_KEY_SECRET", Value: sshKeySecret},
		{Name: "GIT_TOKEN_SECRET", Value: tokenSecret},
		{Name: "GIT_REPOSITORIES", Value: reposJSON},
	}
	// If backend annotated the session with a runner token secret, inject bot token envs without refetching the CR
	if meta, ok := currentObj.Object["metadata"].(map[string]interface{}); ok {
		if anns, ok := meta["annotations"].(map[string]interface{}); ok {
			if v, ok := anns["ambient-code.io/runner-token-secret"].(string); ok && strings.TrimSpace(v) != "" {
				secretName := strings.TrimSpace(v)
				base = append(base, corev1.EnvVar{Name: "AUTH_MODE", Value: "bot_token"})
				base = append(base, corev1.EnvVar{
					Name: "BOT_TOKEN",
					ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
						Key:                  "token",
					}},
				})
			}
		}
	}
	// Add CR-provided envs last (override base when same key)
	if spec, ok := currentObj.Object["spec"].(map[string]interface{}); ok {
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
}

// MonitorJob monitors a job and updates the AgenticSession status
func MonitorJob(jobName, sessionName, sessionNamespace string) {
	log.Printf("Starting job monitoring for %s (session: %s/%s)", jobName, sessionNamespace, sessionName)

	for {
		time.Sleep(10 * time.Second)

		// First check if the AgenticSession still exists
		gvr := types.GetAgenticSessionResource()
		if _, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), sessionName, v1.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				log.Printf("AgenticSession %s no longer exists, stopping job monitoring for %s", sessionName, jobName)
				return
			}
			log.Printf("Error checking AgenticSession %s existence: %v", sessionName, err)
			// Continue monitoring even if we can't check the session
		}

		job, err := config.K8sClient.BatchV1().Jobs(sessionNamespace).Get(context.TODO(), jobName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				log.Printf("Job %s not found, stopping monitoring", jobName)
				return
			}
			log.Printf("Error getting job %s: %v", jobName, err)
			continue
		}

		if job.Status.Failed >= *job.Spec.BackoffLimit {
			log.Printf("Job %s failed after %d attempts", jobName, job.Status.Failed)

			// Get pod logs for error information
			errorMessage := "Job failed"
			if pods, err := config.K8sClient.CoreV1().Pods(sessionNamespace).List(context.TODO(), v1.ListOptions{
				LabelSelector: fmt.Sprintf("job-name=%s", jobName),
			}); err == nil && len(pods.Items) > 0 {
				// Try to get logs from the first pod
				pod := pods.Items[0]
				if logs, err := config.K8sClient.CoreV1().Pods(sessionNamespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).DoRaw(context.TODO()); err == nil {
					errorMessage = fmt.Sprintf("Job failed: %s", string(logs))
					if len(errorMessage) > 500 {
						errorMessage = errorMessage[:500] + "..."
					}
				}
			}

			// Update AgenticSession status to Failed
			UpdateAgenticSessionStatus(sessionNamespace, sessionName, map[string]interface{}{
				"phase":          "Failed",
				"message":        errorMessage,
				"completionTime": time.Now().Format(time.RFC3339),
			})
			// OwnerReferences handle cleanup after failure
			return
		}
	}
}

// UpdateAgenticSessionStatus updates the status of an AgenticSession
func UpdateAgenticSessionStatus(sessionNamespace, name string, statusUpdate map[string]interface{}) error {
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

var (
	boolPtr          = func(b bool) *bool { return &b }
	int32Ptr         = func(i int32) *int32 { return &i }
	int64Ptr         = func(i int64) *int64 { return &i }
	intstrFromString = func(s string) intstr.IntOrString { return intstr.Parse(s) }
)