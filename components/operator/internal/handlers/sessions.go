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

	// Only process if status is Pending
	if phase != "Pending" {
		return nil
	}

	// Ensure a per-session workspace PVC exists for this job to avoid multi-attach
	pvcName := fmt.Sprintf("ambient-workspace-%s", name)
	ownerRefs := []v1.OwnerReference{
		{
			APIVersion: "vteam.ambient-code/v1",
			Kind:       "AgenticSession",
			Name:       currentObj.GetName(),
			UID:        currentObj.GetUID(),
			Controller: boolPtr(true),
			// BlockOwnerDeletion intentionally omitted to avoid permission issues
		},
	}
	if err := services.EnsureSessionWorkspacePVC(sessionNamespace, pvcName, ownerRefs); err != nil {
		log.Printf("Failed to ensure session PVC %s in %s: %v", pvcName, sessionNamespace, err)
		// Continue; job may still run with ephemeral storage
	}

	// Load config for this session
	appConfig := config.LoadConfig()

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

							// If configured, import all keys from the runner Secret as environment variables
							EnvFrom: func() []corev1.EnvFromSource {
								if runnerSecretsName != "" {
									return []corev1.EnvFromSource{{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: runnerSecretsName}}}}
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

	// If a runner secret is configured, mount it as a volume in addition to EnvFrom
	if strings.TrimSpace(runnerSecretsName) != "" {
		job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, corev1.Volume{
			Name:         "runner-secrets",
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: runnerSecretsName}},
		})
		if len(job.Spec.Template.Spec.Containers) > 0 {
			job.Spec.Template.Spec.Containers[0].VolumeMounts = append(job.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
				Name:      "runner-secrets",
				MountPath: "/var/run/runner-secrets",
				ReadOnly:  true,
			})
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
			_ = updateAgenticSessionStatus(sessionNamespace, sessionName, map[string]interface{}{
				"phase":          "Failed",
				"message":        failureMsg,
				"completionTime": time.Now().Format(time.RFC3339),
			})
			_ = deleteJobAndPerJobService(sessionNamespace, jobName, sessionName)
			return
		}

		// Inspect pods to determine main container state regardless of sidecar
		pods, err := config.K8sClient.CoreV1().Pods(sessionNamespace).List(context.TODO(), v1.ListOptions{LabelSelector: fmt.Sprintf("job-name=%s", jobName)})
		if err != nil {
			log.Printf("Error listing pods for job %s: %v", jobName, err)
			continue
		}
		if len(pods.Items) == 0 {
			continue
		}
		pod := pods.Items[0]

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

			// If wrapper already set status to Completed, respect that (don't override)
			if currentPhase == "Completed" {
				log.Printf("Runner exited for job %s; status already set to Completed by wrapper", jobName)
				continue
			}

			// Runner exit code 0 = success
			if term.ExitCode == 0 {
				_ = updateAgenticSessionStatus(sessionNamespace, sessionName, map[string]interface{}{
					"phase":          "Completed",
					"message":        "Runner completed successfully",
					"completionTime": time.Now().Format(time.RFC3339),
				})
				log.Printf("Runner container exited successfully for job %s", jobName)
				continue
			}

			// Runner non-zero exit = failure (only if not already Completed)
			msg := term.Message
			if msg == "" {
				msg = fmt.Sprintf("Runner container exited with code %d", term.ExitCode)
			}
			_ = updateAgenticSessionStatus(sessionNamespace, sessionName, map[string]interface{}{
				"phase":   "Failed",
				"message": msg,
			})
			log.Printf("Runner container failed for job %s: %s", jobName, msg)
			continue
		}

		// Check CR repo statuses; if session marked Completed/Stopped and all repos are finalized, cleanup
		{
			gvr := types.GetAgenticSessionResource()
			obj, err := config.DynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), sessionName, v1.GetOptions{})
			if err == nil && obj != nil {
				status, _, _ := unstructured.NestedMap(obj.Object, "status")
				phase := ""
				if v, ok := status["phase"].(string); ok {
					phase = v
				}
				spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
				repos, _, _ := unstructured.NestedSlice(spec, "repos")
				// Only finalize when there is at least one repo and ALL are in a final state
				allFinal := false
				if len(repos) > 0 {
					allFinal = true
					for _, r := range repos {
						m, ok := r.(map[string]interface{})
						if !ok {
							continue
						}
						st, _ := m["status"].(string)
						st = strings.ToLower(strings.TrimSpace(st))
						if st != "pushed" && st != "abandoned" {
							allFinal = false
							break
						}
					}
				}
				if (phase == "Completed" || phase == "Stopped") && allFinal {
					log.Printf("All repos finalized for %s; cleaning up job and service", sessionName)
					_ = deleteJobAndPerJobService(sessionNamespace, jobName, sessionName)
					return
				}
			}
		}
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

	// Proactively delete Pods for this Job before removing PVC
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

	// Delete the per-session workspace PVC
	pvcName := fmt.Sprintf("ambient-workspace-%s", sessionName)
	if err := config.K8sClient.CoreV1().PersistentVolumeClaims(namespace).Delete(context.TODO(), pvcName, v1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to delete per-session PVC %s/%s: %v", namespace, pvcName, err)
	}
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

// Helper functions
var (
	boolPtr  = func(b bool) *bool { return &b }
	int32Ptr = func(i int32) *int32 { return &i }
	int64Ptr = func(i int64) *int64 { return &i }
)
