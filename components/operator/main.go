package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
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
	contentServiceImage    string
	backendNamespace       string
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

	// Get backend namespace from environment or use operator namespace
	backendNamespace = os.Getenv("BACKEND_NAMESPACE")
	if backendNamespace == "" {
		backendNamespace = namespace // Default to same namespace as operator
	}

	// Get ambient-code runner image from environment or use default
	ambientCodeRunnerImage = os.Getenv("AMBIENT_CODE_RUNNER_IMAGE")
	if ambientCodeRunnerImage == "" {
		ambientCodeRunnerImage = "quay.io/ambient_code/vteam_claude_runner:latest"
	}

	// Image for per-namespace content service (defaults to backend image)
	contentServiceImage = os.Getenv("CONTENT_SERVICE_IMAGE")
	if contentServiceImage == "" {
		contentServiceImage = "quay.io/ambient_code/vteam_backend:latest"
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

	// Start watching for managed namespaces
	go watchNamespaces()

	// Start watching ProjectSettings resources
	go watchProjectSettings()

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
		Version:  "v1alpha1",
		Resource: "agenticsessions",
	}
}

func getProjectSettingsResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1alpha1",
		Resource: "projectsettings",
	}
}

func watchAgenticSessions() {
	gvr := getAgenticSessionResource()

	for {
		// Watch AgenticSessions across all namespaces
		watcher, err := dynamicClient.Resource(gvr).Watch(context.TODO(), v1.ListOptions{})
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
				nsObj, err := k8sClient.CoreV1().Namespaces().Get(context.TODO(), ns, v1.GetOptions{})
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
	gvr := getAgenticSessionResource()
	currentObj, err := dynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), name, v1.GetOptions{})
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

	// Ensure a per-project workspace PVC exists for runner artifacts
	if err := ensureProjectWorkspacePVC(sessionNamespace); err != nil {
		log.Printf("Failed to ensure workspace PVC in %s: %v", sessionNamespace, err)
		// Continue; job may still run with ephemeral storage
	}

	// Create a Kubernetes Job for this AgenticSession
	jobName := fmt.Sprintf("%s-job", name)

	// Check if job already exists in the session's namespace
	_, err = k8sClient.BatchV1().Jobs(sessionNamespace).Get(context.TODO(), jobName, v1.GetOptions{})
	if err == nil {
		log.Printf("Job %s already exists for AgenticSession %s", jobName, name)
		return nil
	}

	// Extract spec information from the fresh object
	spec, _, _ := unstructured.NestedMap(currentObj.Object, "spec")
	prompt, _, _ := unstructured.NestedString(spec, "prompt")
	timeout, _, _ := unstructured.NestedInt64(spec, "timeout")

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

	// Read runner secrets configuration from ProjectSettings in the session's namespace
	runnerSecretsName := ""
	{
		psGvr := getProjectSettingsResource()
		if psObj, err := dynamicClient.Resource(psGvr).Namespace(sessionNamespace).Get(context.TODO(), "projectsettings", v1.GetOptions{}); err == nil {
			if psSpec, ok := psObj.Object["spec"].(map[string]interface{}); ok {
				if v, ok := psSpec["runnerSecretsName"].(string); ok {
					runnerSecretsName = strings.TrimSpace(v)
				}
			}
		}
	}

	// Ensure ServiceAccount exists in the target namespace
	if err := ensureServiceAccount(sessionNamespace); err != nil {
		log.Printf("Warning: Failed to ensure ServiceAccount in namespace %s: %v", sessionNamespace, err)
	}

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
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: "claude-runner",

					// ⚠️ Let OpenShift SCC choose UID/GID dynamically (restricted-v2 compatible)
					// SecurityContext omitted to allow SCC assignment

					// 🔧 Shared memory volume for browser
					Volumes: []corev1.Volume{
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

							// 📦 Mount shared memory volume only
							VolumeMounts: []corev1.VolumeMount{
								{Name: "dshm", MountPath: "/dev/shm"},
								{Name: "workspace", MountPath: "/workspace"},
							},

							Env: func() []corev1.EnvVar {
								base := []corev1.EnvVar{
									{Name: "AGENTIC_SESSION_NAME", Value: name},
									{Name: "AGENTIC_SESSION_NAMESPACE", Value: sessionNamespace},
									{Name: "PROMPT", Value: prompt},
									{Name: "LLM_MODEL", Value: model},
									{Name: "LLM_TEMPERATURE", Value: fmt.Sprintf("%.2f", temperature)},
									{Name: "LLM_MAX_TOKENS", Value: fmt.Sprintf("%d", maxTokens)},
									{Name: "TIMEOUT", Value: fmt.Sprintf("%d", timeout)},
									{Name: "BACKEND_API_URL", Value: fmt.Sprintf("http://backend-service.%s.svc.cluster.local:8080/api", backendNamespace)},

									// 🔑 Git configuration environment variables
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
								// Always attempt to map ANTHROPIC_API_KEY from the runner secret in this namespace.
								// Fallback to default name if ProjectSettings.spec.runnerSecretsName is not set.
								secretName := runnerSecretsName
								if secretName == "" {
									secretName = "ambient-code-secrets"
								}
								base = append(base, corev1.EnvVar{
									Name: "ANTHROPIC_API_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
											Key:                  "anthropic-api-key",
										},
									},
								})
								// If backend annotated the session with a runner token secret, inject bot token envs
								gvr := getAgenticSessionResource()
								if obj, err := dynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), name, v1.GetOptions{}); err == nil {
									if meta, ok := obj.Object["metadata"].(map[string]interface{}); ok {
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
								}
								return base
							}(),

							// If configured, import all keys from the runner Secret as environment variables
							EnvFrom: func() []corev1.EnvFromSource {
								if runnerSecretsName == "" {
									return nil
								}
								return []corev1.EnvFromSource{
									{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: runnerSecretsName}}},
								}
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
	if err := updateAgenticSessionStatus(sessionNamespace, name, map[string]interface{}{
		"phase":   "Creating",
		"message": "Creating Kubernetes job",
	}); err != nil {
		log.Printf("Failed to update AgenticSession status to Creating: %v", err)
		// Continue anyway - resource might have been deleted
	}

	// Create the job
	_, err = k8sClient.BatchV1().Jobs(sessionNamespace).Create(context.TODO(), job, v1.CreateOptions{})
	if err != nil {
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
	go monitorJob(jobName, name, sessionNamespace)

	return nil
}

// ensureProjectWorkspacePVC creates a per-namespace PVC for runner workspace if missing
func ensureProjectWorkspacePVC(namespace string) error {
	// Check if PVC exists
	if _, err := k8sClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), "ambient-workspace", v1.GetOptions{}); err == nil {
		return nil
	} else if !errors.IsNotFound(err) {
		return err
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: v1.ObjectMeta{
			Name:      "ambient-workspace",
			Namespace: namespace,
			Labels:    map[string]string{"app": "ambient-workspace"},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("5Gi"),
				},
			},
		},
	}
	if _, err := k8sClient.CoreV1().PersistentVolumeClaims(namespace).Create(context.TODO(), pvc, v1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}
	return nil
}

// ensureContentService deploys a per-namespace content service that mounts the project PVC RW
func ensureContentService(namespace string) error {
	// Check Service
	if _, err := k8sClient.CoreV1().Services(namespace).Get(context.TODO(), "ambient-content", v1.GetOptions{}); err == nil {
		return nil
	} else if !errors.IsNotFound(err) {
		return err
	}

	// Deployment
	replicas := int32(1)
	deploy := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      "ambient-content",
			Namespace: namespace,
			Labels:    map[string]string{"app": "ambient-content"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &v1.LabelSelector{MatchLabels: map[string]string{"app": "ambient-content"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{Labels: map[string]string{"app": "ambient-content"}},
				Spec: corev1.PodSpec{
					// Keep content service singleton for RWO PVC; rely on runner job podAffinity (set below) to co-locate with content if needed
					Containers: []corev1.Container{
						{
							Name:  "content",
							Image: contentServiceImage,
							Env: []corev1.EnvVar{
								{Name: "NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
								{Name: "CONTENT_SERVICE_MODE", Value: "true"},
								{Name: "STATE_BASE_DIR", Value: "/data"},
							},
							Ports:        []corev1.ContainerPort{{ContainerPort: 8080, Name: "http"}},
							VolumeMounts: []corev1.VolumeMount{{Name: "workspace", MountPath: "/data"}},
						},
					},
					Volumes: []corev1.Volume{
						{Name: "workspace", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "ambient-workspace"}}},
					},
				},
			},
		},
	}
	if _, err := k8sClient.AppsV1().Deployments(namespace).Create(context.TODO(), deploy, v1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	// Service
	svc := &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      "ambient-content",
			Namespace: namespace,
			Labels:    map[string]string{"app": "ambient-content"},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "ambient-content"},
			Ports:    []corev1.ServicePort{{Name: "http", Port: 8080, TargetPort: intstrFromString("http")}},
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
	if _, err := k8sClient.CoreV1().Services(namespace).Create(context.TODO(), svc, v1.CreateOptions{}); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// cleanupSessionResources removes per-session resources (SA, Role, RoleBinding, Secret)
// created for a given AgenticSession. Best-effort; ignores not found errors.
// cleanup handled via Kubernetes OwnerReferences on session-scoped resources

func monitorJob(jobName, sessionName, sessionNamespace string) {
	log.Printf("Starting job monitoring for %s (session: %s/%s)", jobName, sessionNamespace, sessionName)

	for {
		time.Sleep(10 * time.Second)

		// First check if the AgenticSession still exists
		gvr := getAgenticSessionResource()
		if _, err := dynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), sessionName, v1.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				log.Printf("AgenticSession %s no longer exists, stopping job monitoring for %s", sessionName, jobName)
				return
			}
			log.Printf("Error checking AgenticSession %s existence: %v", sessionName, err)
			// Continue monitoring even if we can't check the session
		}

		job, err := k8sClient.BatchV1().Jobs(sessionNamespace).Get(context.TODO(), jobName, v1.GetOptions{})
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
			updateAgenticSessionStatus(sessionNamespace, sessionName, map[string]interface{}{
				"phase":          "Completed",
				"message":        "Job completed successfully",
				"completionTime": time.Now().Format(time.RFC3339),
			})
			// OwnerReferences handle cleanup after successful completion
			return
		}

		if job.Status.Failed >= *job.Spec.BackoffLimit {
			log.Printf("Job %s failed after %d attempts", jobName, job.Status.Failed)

			// Get pod logs for error information
			errorMessage := "Job failed"
			if pods, err := k8sClient.CoreV1().Pods(sessionNamespace).List(context.TODO(), v1.ListOptions{
				LabelSelector: fmt.Sprintf("job-name=%s", jobName),
			}); err == nil && len(pods.Items) > 0 {
				// Try to get logs from the first pod
				pod := pods.Items[0]
				if logs, err := k8sClient.CoreV1().Pods(sessionNamespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).DoRaw(context.TODO()); err == nil {
					errorMessage = fmt.Sprintf("Job failed: %s", string(logs))
					if len(errorMessage) > 500 {
						errorMessage = errorMessage[:500] + "..."
					}
				}
			}

			// Update AgenticSession status to Failed
			updateAgenticSessionStatus(sessionNamespace, sessionName, map[string]interface{}{
				"phase":          "Failed",
				"message":        errorMessage,
				"completionTime": time.Now().Format(time.RFC3339),
			})
			// OwnerReferences handle cleanup after failure
			return
		}
	}
}

func updateAgenticSessionStatus(sessionNamespace, name string, statusUpdate map[string]interface{}) error {
	gvr := getAgenticSessionResource()

	// Get current resource
	obj, err := dynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), name, v1.GetOptions{})
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
	_, err = dynamicClient.Resource(gvr).Namespace(sessionNamespace).UpdateStatus(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("AgenticSession %s was deleted during status update, skipping", name)
			return nil // Don't treat this as an error - resource was deleted
		}
		return fmt.Errorf("failed to update AgenticSession status: %v", err)
	}

	return nil
}

func watchNamespaces() {
	for {
		watcher, err := k8sClient.CoreV1().Namespaces().Watch(context.TODO(), v1.ListOptions{
			LabelSelector: "ambient-code.io/managed=true",
		})
		if err != nil {
			log.Printf("Failed to create namespace watcher: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("Watching for managed namespaces...")

		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added:
				namespace := event.Object.(*corev1.Namespace)
				log.Printf("Detected new managed namespace: %s", namespace.Name)

				// Auto-create ProjectSettings for this namespace
				if err := createDefaultProjectSettings(namespace.Name); err != nil {
					log.Printf("Error creating default ProjectSettings for namespace %s: %v", namespace.Name, err)
				}

				// Ensure shared workspace PVC and content service exist
				if err := ensureProjectWorkspacePVC(namespace.Name); err != nil {
					log.Printf("Failed to ensure workspace PVC in %s: %v", namespace.Name, err)
				}
				if err := ensureContentService(namespace.Name); err != nil {
					log.Printf("Failed to ensure content service in %s: %v", namespace.Name, err)
				}
			case watch.Error:
				obj := event.Object.(*unstructured.Unstructured)
				log.Printf("Watch error for namespaces: %v", obj)
			}
		}

		log.Println("Namespace watch channel closed, restarting...")
		watcher.Stop()
		time.Sleep(2 * time.Second)
	}
}

func watchProjectSettings() {
	gvr := getProjectSettingsResource()

	for {
		// Watch across all namespaces for ProjectSettings
		watcher, err := dynamicClient.Resource(gvr).Watch(context.TODO(), v1.ListOptions{})
		if err != nil {
			log.Printf("Failed to create ProjectSettings watcher: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("Watching for ProjectSettings events...")

		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added, watch.Modified:
				obj := event.Object.(*unstructured.Unstructured)

				// Add small delay to avoid race conditions
				time.Sleep(100 * time.Millisecond)

				if err := handleProjectSettingsEvent(obj); err != nil {
					log.Printf("Error handling ProjectSettings event: %v", err)
				}
			case watch.Deleted:
				obj := event.Object.(*unstructured.Unstructured)
				settingsName := obj.GetName()
				settingsNamespace := obj.GetNamespace()
				log.Printf("ProjectSettings %s/%s deleted", settingsNamespace, settingsName)
			case watch.Error:
				obj := event.Object.(*unstructured.Unstructured)
				log.Printf("Watch error for ProjectSettings: %v", obj)
			}
		}

		log.Println("ProjectSettings watch channel closed, restarting...")
		watcher.Stop()
		time.Sleep(2 * time.Second)
	}
}

func createDefaultProjectSettings(namespaceName string) error {
	gvr := getProjectSettingsResource()

	// Check if ProjectSettings already exists in this namespace (singleton named 'projectsettings')
	_, err := dynamicClient.Resource(gvr).Namespace(namespaceName).Get(context.TODO(), "projectsettings", v1.GetOptions{})
	if err == nil {
		log.Printf("ProjectSettings already exists in namespace %s", namespaceName)
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("error checking existing ProjectSettings: %v", err)
	}

	// Create default ProjectSettings (minimal: only groupAccess)
	defaultSettings := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "vteam.ambient-code/v1alpha1",
			"kind":       "ProjectSettings",
			"metadata": map[string]interface{}{
				// Enforce singleton: fixed name 'projectsettings'
				"name":      "projectsettings",
				"namespace": namespaceName,
			},
			"spec": map[string]interface{}{
				"groupAccess": []interface{}{},
			},
		},
	}

	_, err = dynamicClient.Resource(gvr).Namespace(namespaceName).Create(context.TODO(), defaultSettings, v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create default ProjectSettings: %v", err)
	}

	log.Printf("Created default ProjectSettings for namespace %s", namespaceName)
	return nil
}

func handleProjectSettingsEvent(obj *unstructured.Unstructured) error {
	name := obj.GetName()
	namespace := obj.GetNamespace()

	// Verify the resource still exists before processing
	gvr := getProjectSettingsResource()
	currentObj, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("ProjectSettings %s/%s no longer exists, skipping processing", namespace, name)
			return nil
		}
		return fmt.Errorf("failed to verify ProjectSettings %s/%s exists: %v", namespace, name, err)
	}

	log.Printf("Reconciling ProjectSettings %s/%s", namespace, name)
	return reconcileProjectSettings(currentObj)
}

func reconcileProjectSettings(obj *unstructured.Unstructured) error {
	namespace := obj.GetNamespace()
	name := obj.GetName()

	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")

	// Reconcile group access (RoleBindings)
	groupBindingsCreated := 0
	if groupAccess, found, _ := unstructured.NestedSlice(spec, "groupAccess"); found {
		for _, accessInterface := range groupAccess {
			access := accessInterface.(map[string]interface{})
			groupName, _, _ := unstructured.NestedString(access, "groupName")
			role, _, _ := unstructured.NestedString(access, "role")
			if groupName != "" && role != "" {
				if err := ensureRoleBinding(namespace, groupName, role); err != nil {
					log.Printf("Error creating RoleBinding for group %s in namespace %s: %v", groupName, namespace, err)
					continue
				}
				groupBindingsCreated++
			}
		}
	}

	// Update status with reconciliation results (only fields defined in CRD)
	statusUpdate := map[string]interface{}{
		"groupBindingsCreated": groupBindingsCreated,
	}

	return updateProjectSettingsStatus(namespace, name, statusUpdate)
}

// Bot ServiceAccounts are no longer managed here; access keys handle authentication.

func ensureRoleBinding(namespace, groupName, role string) error {
	// Map role to ClusterRole used for ambient project access
	roleName := mapRoleToKubernetesRole(role)
	rbName := fmt.Sprintf("%s-%s", groupName, role)

	// Check if RoleBinding already exists
	_, err := k8sClient.RbacV1().RoleBindings(namespace).Get(context.TODO(), rbName, v1.GetOptions{})
	if err == nil {
		log.Printf("RoleBinding %s already exists in namespace %s", rbName, namespace)
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("error checking existing RoleBinding: %v", err)
	}

	// Create RoleBinding
	rb := &rbacv1.RoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:      rbName,
			Namespace: namespace,
			Labels: map[string]string{
				"ambient-code.io/managed": "true",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     roleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     "Group",
				Name:     groupName,
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
	}

	_, err = k8sClient.RbacV1().RoleBindings(namespace).Create(context.TODO(), rb, v1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create RoleBinding: %v", err)
	}

	log.Printf("Created RoleBinding %s for group %s in namespace %s", rbName, groupName, namespace)
	return nil
}

func mapRoleToKubernetesRole(role string) string {
	switch strings.ToLower(role) {
	case "admin":
		return "ambient-project-admin"
	case "edit":
		return "ambient-project-edit"
	case "view":
		return "ambient-project-view"
	default:
		return "ambient-project-view" // Default to view role
	}
}

func updateProjectSettingsStatus(namespace, name string, statusUpdate map[string]interface{}) error {
	gvr := getProjectSettingsResource()

	// Get current resource
	obj, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("ProjectSettings %s/%s no longer exists, skipping status update", namespace, name)
			return nil
		}
		return fmt.Errorf("failed to get ProjectSettings %s/%s: %v", namespace, name, err)
	}

	// Update status
	if obj.Object["status"] == nil {
		obj.Object["status"] = make(map[string]interface{})
	}

	status := obj.Object["status"].(map[string]interface{})
	for key, value := range statusUpdate {
		status[key] = value
	}

	// Update the resource
	_, err = dynamicClient.Resource(gvr).Namespace(namespace).UpdateStatus(context.TODO(), obj, v1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("ProjectSettings %s/%s was deleted during status update, skipping", namespace, name)
			return nil
		}
		return fmt.Errorf("failed to update ProjectSettings status: %v", err)
	}

	return nil
}


// ensureServiceAccount creates a ServiceAccount and RoleBinding for the runner in the target namespace
func ensureServiceAccount(targetNamespace string) error {
	// Create ServiceAccount if it doesn't exist
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Name:      "claude-runner",
			Namespace: targetNamespace,
		},
	}

	_, err := k8sClient.CoreV1().ServiceAccounts(targetNamespace).Get(context.TODO(), "claude-runner", v1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = k8sClient.CoreV1().ServiceAccounts(targetNamespace).Create(context.TODO(), serviceAccount, v1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create ServiceAccount: %v", err)
		}
		log.Printf("Created ServiceAccount claude-runner in namespace %s", targetNamespace)
	} else if err != nil {
		return fmt.Errorf("failed to get ServiceAccount: %v", err)
	}

	// Create RoleBinding to give the ServiceAccount access to the project
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name:      "claude-runner-project-access",
			Namespace: targetNamespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "claude-runner",
				Namespace: targetNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "edit", // Standard Kubernetes edit role for project access
		},
	}

	_, err = k8sClient.RbacV1().RoleBindings(targetNamespace).Get(context.TODO(), "claude-runner-project-access", v1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = k8sClient.RbacV1().RoleBindings(targetNamespace).Create(context.TODO(), roleBinding, v1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create RoleBinding: %v", err)
		}
		log.Printf("Created RoleBinding claude-runner-project-access in namespace %s", targetNamespace)
	} else if err != nil {
		return fmt.Errorf("failed to get RoleBinding: %v", err)
	}

	return nil
}

var (
	boolPtr          = func(b bool) *bool { return &b }
	int32Ptr         = func(i int32) *int32 { return &i }
	int64Ptr         = func(i int64) *int64 { return &i }
	intstrFromString = func(s string) intstr.IntOrString { return intstr.Parse(s) }
)
