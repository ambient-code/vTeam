package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	k8sClient              *kubernetes.Clientset
	dynamicClient          dynamic.Interface
	namespace              string
	ambientCodeRunnerImage string
	imagePullPolicy        corev1.PullPolicy
)

func main() {
	// Initialize Kubernetes clients
	if err := initK8sClients(); err != nil {
		log.Fatalf("Failed to initialize Kubernetes clients: %v", err)
	}

	// Get namespace from environment or use default
	namespace = os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	// Get ambient-code runner image from environment or use default
	ambientCodeRunnerImage = os.Getenv("AMBIENT_CODE_RUNNER_IMAGE")
	if ambientCodeRunnerImage == "" {
		ambientCodeRunnerImage = "quay.io/ambient_code/vteam_claude_runner:latest"
	}

	// Get image pull policy from environment or use default
	imagePullPolicyStr := os.Getenv("IMAGE_PULL_POLICY")
	if imagePullPolicyStr == "" {
		imagePullPolicyStr = "Always"
	}
	imagePullPolicy = corev1.PullPolicy(imagePullPolicyStr)

	log.Printf("Agentic Session Operator starting in namespace: %s", namespace)
	log.Printf("Using ambient-code runner image: %s", ambientCodeRunnerImage)

	// Start watching AgenticSession resources
	go watchAgenticSessions()

	// Keep the operator running
	select {}
}

func initK8sClients() error {
	var config *rest.Config
	var err error

	// Try in-cluster config first
	if config, err = rest.InClusterConfig(); err != nil {
		// If in-cluster config fails, try kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = fmt.Sprintf("%s/.kube/config", os.Getenv("HOME"))
		}

		if config, err = clientcmd.BuildConfigFromFlags("", kubeconfig); err != nil {
			return fmt.Errorf("failed to create Kubernetes config: %v", err)
		}
	}

	// Create standard Kubernetes client
	k8sClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	// Create dynamic client for custom resources
	dynamicClient, err = dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %v", err)
	}

	return nil
}

func getAgenticSessionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1",
		Resource: "agenticsessions",
	}
}

func watchAgenticSessions() {
	gvr := getAgenticSessionResource()

	for {
		watcher, err := dynamicClient.Resource(gvr).Namespace(namespace).Watch(context.TODO(), v1.ListOptions{})
		if err != nil {
			log.Printf("Failed to create watcher: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("Watching for AgenticSession events...")

		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added, watch.Modified:
				obj := event.Object.(*unstructured.Unstructured)

				// Add small delay to avoid race conditions with rapid create/delete cycles
				time.Sleep(100 * time.Millisecond)

				if err := handleAgenticSessionEvent(obj); err != nil {
					log.Printf("Error handling AgenticSession event: %v", err)
				}
			case watch.Deleted:
				obj := event.Object.(*unstructured.Unstructured)
				sessionName := obj.GetName()
				log.Printf("AgenticSession %s deleted", sessionName)

				// Cancel any ongoing job monitoring for this session
				// (We could implement this with a context cancellation if needed)
			case watch.Error:
				obj := event.Object.(*unstructured.Unstructured)
				log.Printf("Watch error for AgenticSession: %v", obj)
			}
		}

		log.Println("Watch channel closed, restarting...")
		watcher.Stop()
		time.Sleep(2 * time.Second)
	}
}

func handleAgenticSessionEvent(obj *unstructured.Unstructured) error {
	name := obj.GetName()

	// Verify the resource still exists before processing
	gvr := getAgenticSessionResource()
	currentObj, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("AgenticSession %s no longer exists, skipping processing", name)
			return nil
		}
		return fmt.Errorf("failed to verify AgenticSession %s exists: %v", name, err)
	}

	// Get the current status from the fresh object
	status, _, _ := unstructured.NestedMap(currentObj.Object, "status")
	phase, _, _ := unstructured.NestedString(status, "phase")

	log.Printf("Processing AgenticSession %s with phase %s", name, phase)

	// Only process if status is Pending or not set (newly created resources)
	if phase != "Pending" && phase != "" {
		return nil
	}

	// If phase is empty (new resource), initialize it to Pending
	if phase == "" {
		if err := updateAgenticSessionStatus(name, map[string]interface{}{
			"phase": "Pending",
		}); err != nil {
			log.Printf("Failed to initialize AgenticSession status to Pending: %v", err)
			return err
		}
		phase = "Pending"
		log.Printf("Initialized AgenticSession %s status to Pending", name)
	}

	// Create a Kubernetes Job for this AgenticSession
	jobName := fmt.Sprintf("%s-job", name)

	// Check if job already exists
	_, err = k8sClient.BatchV1().Jobs(namespace).Get(context.TODO(), jobName, v1.GetOptions{})
	if err == nil {
		log.Printf("Job %s already exists for AgenticSession %s", jobName, name)
		return nil
	}

	// Extract spec information from the fresh object
	spec, _, _ := unstructured.NestedMap(currentObj.Object, "spec")
	prompt, _, _ := unstructured.NestedString(spec, "prompt")
	websiteURL, _, _ := unstructured.NestedString(spec, "websiteURL")
	timeout, _, _ := unstructured.NestedInt64(spec, "timeout")

	// For RFE workflows, websiteURL may be empty - this is expected and valid

	llmSettings, _, _ := unstructured.NestedMap(spec, "llmSettings")
	model, _, _ := unstructured.NestedString(llmSettings, "model")
	temperature, _, _ := unstructured.NestedFloat64(llmSettings, "temperature")
	maxTokens, _, _ := unstructured.NestedInt64(llmSettings, "maxTokens")

	// Extract Git configuration
	gitConfig, gitConfigExists, _ := unstructured.NestedMap(spec, "gitConfig")
	var gitUserName, gitUserEmail string
	var gitRepositoriesJSON string

	if gitConfigExists {
		// Get Git user configuration
		gitUser, _, _ := unstructured.NestedMap(gitConfig, "user")
		gitUserName, _, _ = unstructured.NestedString(gitUser, "name")
		gitUserEmail, _, _ = unstructured.NestedString(gitUser, "email")

		// Get Git repositories and serialize to JSON for environment variable
		gitRepositories, _, _ := unstructured.NestedSlice(gitConfig, "repositories")
		if len(gitRepositories) > 0 {
			if reposBytes, err := json.Marshal(gitRepositories); err == nil {
				gitRepositoriesJSON = string(reposBytes)
			}
		}
	}

	// Extract additional environment variables (for RFE workflows and other custom configurations)
	additionalEnvVars := make(map[string]string)
	if envVars, envVarsExist, _ := unstructured.NestedMap(spec, "environmentVariables"); envVarsExist {
		for key, value := range envVars {
			if strValue, ok := value.(string); ok {
				additionalEnvVars[key] = strValue
			} else {
				// Convert other types to string
				additionalEnvVars[key] = fmt.Sprintf("%v", value)
			}
		}
	}

	// Create the Job
	job := &batchv1.Job{
		ObjectMeta: v1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
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
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: "claude-runner",

					// ⚠️ Let OpenShift SCC choose UID/GID dynamically (restricted-v2 compatible)
					// SecurityContext omitted to allow SCC assignment

					// 🔧 Shared memory volume for browser, workspace storage for RFE workflows, and MCP servers config
					Volumes: func() []corev1.Volume {
						volumes := []corev1.Volume{
							{
								Name: "dshm",
								VolumeSource: corev1.VolumeSource{
									EmptyDir: &corev1.EmptyDirVolumeSource{
										Medium:    corev1.StorageMediumMemory,
										SizeLimit: resource.NewQuantity(256*1024*1024, resource.BinarySI),
									},
								},
							},
							{
								Name: "mcp-servers-config",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "mcp-servers-config",
										},
										Items: []corev1.KeyToPath{
											{
												Key:  ".mcp.json",
												Path: ".mcp.json",
											},
										},
									},
								},
							},
						}

						// Add workspace PVC for RFE workflows if SHARED_WORKSPACE env var is set
						if _, hasSharedWorkspace := additionalEnvVars["SHARED_WORKSPACE"]; hasSharedWorkspace {
							volumes = append(volumes, corev1.Volume{
								Name: "workspace",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "vteam-workspace-pvc",
									},
								},
							})
						}

						return volumes
					}(),

					Containers: []corev1.Container{
						{
							Name:            "ambient-code-runner",
							Image:           ambientCodeRunnerImage,
							ImagePullPolicy: imagePullPolicy,
							// 🔒 Container-level security (SCC-compatible, no privileged capabilities)
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: boolPtr(false),
								ReadOnlyRootFilesystem:   boolPtr(false), // Playwright needs to write temp files
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"}, // Drop all capabilities for security
								},
							},

							// 📦 Mount shared memory volume, workspace for RFE workflows, and MCP servers config
							VolumeMounts: func() []corev1.VolumeMount {
								mounts := []corev1.VolumeMount{
									{Name: "dshm", MountPath: "/dev/shm"},
									{Name: "mcp-servers-config", MountPath: "/app/.mcp.json", SubPath: ".mcp.json"},
								}

								// Add workspace mount for RFE workflows if SHARED_WORKSPACE env var is set
								if _, hasSharedWorkspace := additionalEnvVars["SHARED_WORKSPACE"]; hasSharedWorkspace {
									mounts = append(mounts, corev1.VolumeMount{
										Name:      "workspace",
										MountPath: "/workspace",
									})
								}

								return mounts
							}(),

							Env: func() []corev1.EnvVar {
								// Base environment variables
								baseEnvVars := []corev1.EnvVar{
									{Name: "AGENTIC_SESSION_NAME", Value: name},
									{Name: "AGENTIC_SESSION_NAMESPACE", Value: namespace},
									{Name: "PROMPT", Value: prompt},
									{Name: "WEBSITE_URL", Value: websiteURL},
									{Name: "LLM_MODEL", Value: model},
									{Name: "LLM_TEMPERATURE", Value: fmt.Sprintf("%.2f", temperature)},
									{Name: "LLM_MAX_TOKENS", Value: fmt.Sprintf("%d", maxTokens)},
									{Name: "TIMEOUT", Value: fmt.Sprintf("%d", timeout)},
									{Name: "BACKEND_API_URL", Value: os.Getenv("BACKEND_API_URL")},

									// 🔑 Anthropic key from Secret
									{
										Name: "ANTHROPIC_API_KEY",
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: &corev1.SecretKeySelector{
												LocalObjectReference: corev1.LocalObjectReference{Name: "ambient-code-secrets"},
												Key:                  "anthropic-api-key",
											},
										},
									},

									// 🔧 Git configuration environment variables
									{Name: "GIT_USER_NAME", Value: gitUserName},
									{Name: "GIT_USER_EMAIL", Value: gitUserEmail},
									{Name: "GIT_REPOSITORIES", Value: gitRepositoriesJSON},

									// ✅ Use /tmp for SCC-assigned random UID (OpenShift compatible)
									{Name: "HOME", Value: "/tmp"},
									{Name: "XDG_CONFIG_HOME", Value: "/tmp/.config"},
									{Name: "XDG_CACHE_HOME", Value: "/tmp/.cache"},
									{Name: "XDG_DATA_HOME", Value: "/tmp/.local/share"},

									// 🧊 Playwright/Chromium optimized for containers with shared memory
									{Name: "PW_CHROMIUM_ARGS", Value: "--no-sandbox --disable-gpu"},

									// 📁 Playwright browser cache in writable location
									{Name: "PLAYWRIGHT_BROWSERS_PATH", Value: "/tmp/.cache/ms-playwright"},

									// (Optional) proxy envs if your cluster requires them:
									// { Name: "HTTPS_PROXY", Value: "http://proxy.corp:3128" },
									// { Name: "NO_PROXY",    Value: ".svc,.cluster.local,10.0.0.0/8" },
								}

								// Add additional environment variables from spec (for RFE workflows and custom configurations)
								for key, value := range additionalEnvVars {
									baseEnvVars = append(baseEnvVars, corev1.EnvVar{
										Name:  key,
										Value: value,
									})
								}

								return baseEnvVars
							}(),

							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1000m"),
									corev1.ResourceMemory: resource.MustParse("2Gi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("2000m"),
									corev1.ResourceMemory: resource.MustParse("4Gi"),
								},
							},
						},
					},
				},
			},
		},
	}

	// Update status to Creating before attempting job creation
	if err := updateAgenticSessionStatus(name, map[string]interface{}{
		"phase":   "Creating",
		"message": "Creating Kubernetes job",
	}); err != nil {
		log.Printf("Failed to update AgenticSession status to Creating: %v", err)
		// Continue anyway - resource might have been deleted
	}

	// Create the job
	_, err = k8sClient.BatchV1().Jobs(namespace).Create(context.TODO(), job, v1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create job %s: %v", jobName, err)
		// Update status to Error if job creation fails and resource still exists
		updateAgenticSessionStatus(name, map[string]interface{}{
			"phase":   "Error",
			"message": fmt.Sprintf("Failed to create job: %v", err),
		})
		return fmt.Errorf("failed to create job: %v", err)
	}

	log.Printf("Created job %s for AgenticSession %s", jobName, name)

	// Update AgenticSession status to Running
	if err := updateAgenticSessionStatus(name, map[string]interface{}{
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
	go monitorJob(jobName, name)

	return nil
}

func monitorJob(jobName, sessionName string) {
	log.Printf("Starting job monitoring for %s (session: %s)", jobName, sessionName)

	for {
		time.Sleep(10 * time.Second)

		// First check if the AgenticSession still exists
		gvr := getAgenticSessionResource()
		if _, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), sessionName, v1.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				log.Printf("AgenticSession %s no longer exists, stopping job monitoring for %s", sessionName, jobName)
				return
			}
			log.Printf("Error checking AgenticSession %s existence: %v", sessionName, err)
			// Continue monitoring even if we can't check the session
		}

		job, err := k8sClient.BatchV1().Jobs(namespace).Get(context.TODO(), jobName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				log.Printf("Job %s not found, stopping monitoring", jobName)
				return
			}
			log.Printf("Error getting job %s: %v", jobName, err)
			continue
		}

		// Check job status
		if job.Status.Succeeded > 0 {
			log.Printf("Job %s completed successfully", jobName)

			// Update AgenticSession status to Completed
			updateAgenticSessionStatus(sessionName, map[string]interface{}{
				"phase":          "Completed",
				"message":        "Job completed successfully",
				"completionTime": time.Now().Format(time.RFC3339),
			})
			return
		}

		if job.Status.Failed >= *job.Spec.BackoffLimit {
			log.Printf("Job %s failed after %d attempts", jobName, job.Status.Failed)

			// Get pod logs for error information
			errorMessage := "Job failed"
			if pods, err := k8sClient.CoreV1().Pods(namespace).List(context.TODO(), v1.ListOptions{
				LabelSelector: fmt.Sprintf("job-name=%s", jobName),
			}); err == nil && len(pods.Items) > 0 {
				// Try to get logs from the first pod
				pod := pods.Items[0]
				if logs, err := k8sClient.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).DoRaw(context.TODO()); err == nil {
					errorMessage = fmt.Sprintf("Job failed: %s", string(logs))
					if len(errorMessage) > 500 {
						errorMessage = errorMessage[:500] + "..."
					}
				}
			}

			// Update AgenticSession status to Failed
			updateAgenticSessionStatus(sessionName, map[string]interface{}{
				"phase":          "Failed",
				"message":        errorMessage,
				"completionTime": time.Now().Format(time.RFC3339),
			})
			return
		}
	}
}

func updateAgenticSessionStatus(name string, statusUpdate map[string]interface{}) error {
	gvr := getAgenticSessionResource()

	// Get current resource
	obj, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
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
	_, err = dynamicClient.Resource(gvr).Namespace(namespace).UpdateStatus(context.TODO(), obj, v1.UpdateOptions{})
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
	boolPtr  = func(b bool) *bool { return &b }
	int32Ptr = func(i int32) *int32 { return &i }
	int64Ptr = func(i int64) *int64 { return &i }
)
