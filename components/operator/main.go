package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

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
	if err := ensureSessionWorkspacePVC(sessionNamespace, pvcName, ownerRefs); err != nil {
		log.Printf("Failed to ensure session PVC %s in %s: %v", pvcName, sessionNamespace, err)
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
	interactive, _, _ := unstructured.NestedBool(spec, "interactive")

	llmSettings, _, _ := unstructured.NestedMap(spec, "llmSettings")
	model, _, _ := unstructured.NestedString(llmSettings, "model")
	temperature, _, _ := unstructured.NestedFloat64(llmSettings, "temperature")
	maxTokens, _, _ := unstructured.NestedInt64(llmSettings, "maxTokens")

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
									{Name: "BACKEND_API_URL", Value: fmt.Sprintf("http://backend-service.%s.svc.cluster.local:8080/api", backendNamespace)},
									// WebSocket URL used by runner-shell to connect back to backend
									{Name: "WEBSOCKET_URL", Value: fmt.Sprintf("ws://backend-service.%s.svc.cluster.local:8080/api/projects/%s/sessions/%s/ws", backendNamespace, sessionNamespace, name)},
									// S3 disabled; backend persists messages
								}
								// If backend annotated the session with a runner token secret, inject tokens from it
								// Secret contains: 'k8s-token' (for CR updates), 'token' (GitHub token), 'github-token-expires-at'
								if meta, ok := currentObj.Object["metadata"].(map[string]interface{}); ok {
									if anns, ok := meta["annotations"].(map[string]interface{}); ok {
										if v, ok := anns["ambient-code.io/runner-token-secret"].(string); ok && strings.TrimSpace(v) != "" {
											secretName := strings.TrimSpace(v)
											// K8s ServiceAccount token for CR status updates
											base = append(base, corev1.EnvVar{
												Name: "RUNNER_TOKEN",
												ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
													LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
													Key:                  "k8s-token",
												}},
											})
											// GitHub token for git operations (optional - will fall back to ProjectSettings secret)
											base = append(base, corev1.EnvVar{
												Name: "GITHUB_TOKEN",
												ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
													LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
													Key:                  "token",
													Optional:             boolPtr(true),
												}},
											})
											base = append(base, corev1.EnvVar{
												Name: "BOT_TOKEN",
												ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
													LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
													Key:                  "token",
													Optional:             boolPtr(true),
												}},
											})
										}
									}
								}
								// Add CR-provided envs last (override base when same key)
								if spec, ok := obj.Object["spec"].(map[string]interface{}); ok {
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

								// If a runner secret is configured via ProjectSettings, map its 'token' to GITHUB_TOKEN/BOT_TOKEN
								if runnerSecretsName != "" {
									// avoid duplicates if already set via annotation
									hasGitHubToken := false
									hasBotToken := false
									for i := range base {
										if base[i].Name == "GITHUB_TOKEN" {
											hasGitHubToken = true
										}
										if base[i].Name == "BOT_TOKEN" {
											hasBotToken = true
										}
									}
									if !hasGitHubToken {
										base = append(base, corev1.EnvVar{
											Name: "GITHUB_TOKEN",
											ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
												LocalObjectReference: corev1.LocalObjectReference{Name: runnerSecretsName},
												Key:                  "token",
											}},
										})
									}
									if !hasBotToken {
										base = append(base, corev1.EnvVar{
											Name: "BOT_TOKEN",
											ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
												LocalObjectReference: corev1.LocalObjectReference{Name: runnerSecretsName},
												Key:                  "token",
											}},
										})
									}
								}

								return base
							}(),

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
						{
							Name:            "ambient-content",
							Image:           contentServiceImage,
							ImagePullPolicy: imagePullPolicy,
							Env: []corev1.EnvVar{
								{Name: "CONTENT_SERVICE_MODE", Value: "true"},
								{Name: "STATE_BASE_DIR", Value: "/data/state"},
							},
							Ports:        []corev1.ContainerPort{{ContainerPort: 8080, Name: "http"}},
							VolumeMounts: []corev1.VolumeMount{{Name: "workspace", MountPath: "/data/state"}},
						},
					},
				},
			},
		},
	}

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
	if err := updateAgenticSessionStatus(sessionNamespace, name, map[string]interface{}{
		"phase":   "Creating",
		"message": "Creating Kubernetes job",
	}); err != nil {
		log.Printf("Failed to update AgenticSession status to Creating: %v", err)
		// Continue anyway - resource might have been deleted
	}

	// Create the job
	createdJob, err := k8sClient.BatchV1().Jobs(sessionNamespace).Create(context.TODO(), job, v1.CreateOptions{})
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
		"phase":     "Creating",
		"message":   "Job is being set up",
		"startTime": time.Now().Format(time.RFC3339),
		"jobName":   jobName,
	}); err != nil {
		log.Printf("Failed to update AgenticSession status to Creating: %v", err)
		// Don't return error here - the job was created successfully
		// The status update failure might be due to the resource being deleted
	}

	// Create a per-job Service pointing to the content sidecar
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
	if _, serr := k8sClient.CoreV1().Services(sessionNamespace).Create(context.TODO(), svc, v1.CreateOptions{}); serr != nil && !errors.IsAlreadyExists(serr) {
		log.Printf("Failed to create per-job content service for %s: %v", name, serr)
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

// ensureContentService deploys a per-namespace backend instance in CONTENT_SERVICE_MODE
func ensureContentService(namespace string) error {
	// removed: per-namespace content service no longer managed by operator
	return nil
}

// ensureSessionWorkspacePVC creates a per-session PVC owned by the AgenticSession to avoid multi-attach conflicts
func ensureSessionWorkspacePVC(namespace, pvcName string, ownerRefs []v1.OwnerReference) error {
	// Check if PVC exists
	if _, err := k8sClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), pvcName, v1.GetOptions{}); err == nil {
		return nil
	} else if !errors.IsNotFound(err) {
		return err
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: v1.ObjectMeta{
			Name:            pvcName,
			Namespace:       namespace,
			Labels:          map[string]string{"app": "ambient-workspace", "agentic-session": pvcName},
			OwnerReferences: ownerRefs,
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

func monitorJob(jobName, sessionName, sessionNamespace string) {
	log.Printf("Starting job monitoring for %s (session: %s/%s)", jobName, sessionNamespace, sessionName)

	mainContainerName := "ambient-code-runner"

	for {
		time.Sleep(5 * time.Second)

		// Ensure the AgenticSession still exists
		gvr := getAgenticSessionResource()
		if _, err := dynamicClient.Resource(gvr).Namespace(sessionNamespace).Get(context.TODO(), sessionName, v1.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				log.Printf("AgenticSession %s no longer exists, stopping job monitoring for %s", sessionName, jobName)
				return
			}
			log.Printf("Error checking AgenticSession %s existence: %v", sessionName, err)
		}

		// Get Job
		job, err := k8sClient.BatchV1().Jobs(sessionNamespace).Get(context.TODO(), jobName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				log.Printf("Job %s not found, stopping monitoring", jobName)
				return
			}
			log.Printf("Error getting job %s: %v", jobName, err)
			continue
		}

		// If K8s already marked the Job as succeeded, finalize
		if job.Status.Succeeded > 0 {
			log.Printf("Job %s marked succeeded by Kubernetes", jobName)
			_ = updateAgenticSessionStatus(sessionNamespace, sessionName, map[string]interface{}{
				"phase":          "Completed",
				"message":        "Job completed successfully",
				"completionTime": time.Now().Format(time.RFC3339),
			})
			_ = deleteJobAndPerJobService(sessionNamespace, jobName, sessionName)
			return
		}

		// If Job has failed according to backoff policy, mark failed
		if job.Spec.BackoffLimit != nil && job.Status.Failed >= *job.Spec.BackoffLimit {
			log.Printf("Job %s failed after %d attempts", jobName, job.Status.Failed)
			failureMsg := "Job failed"
			if pods, err := k8sClient.CoreV1().Pods(sessionNamespace).List(context.TODO(), v1.ListOptions{LabelSelector: fmt.Sprintf("job-name=%s", jobName)}); err == nil && len(pods.Items) > 0 {
				pod := pods.Items[0]
				if logs, err := k8sClient.CoreV1().Pods(sessionNamespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).DoRaw(context.TODO()); err == nil {
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
		pods, err := k8sClient.CoreV1().Pods(sessionNamespace).List(context.TODO(), v1.ListOptions{LabelSelector: fmt.Sprintf("job-name=%s", jobName)})
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
				_ = updateAgenticSessionStatus(sessionNamespace, sessionName, map[string]interface{}{
					"phase":   "Running",
					"message": "Agent is running",
				})
			}
			if cs.State.Terminated != nil {
				term := cs.State.Terminated
				if term.ExitCode == 0 {
					log.Printf("Main container finished successfully for job %s; finalizing session", jobName)
					_ = updateAgenticSessionStatus(sessionNamespace, sessionName, map[string]interface{}{
						"phase":          "Completed",
						"message":        "Job completed successfully",
						"completionTime": time.Now().Format(time.RFC3339),
					})
					_ = deleteJobAndPerJobService(sessionNamespace, jobName, sessionName)
					return
				}
				// Non-zero exit = failure
				msg := term.Message
				if msg == "" {
					msg = fmt.Sprintf("Main container exited with code %d", term.ExitCode)
				}
				_ = updateAgenticSessionStatus(sessionNamespace, sessionName, map[string]interface{}{
					"phase":          "Failed",
					"message":        msg,
					"completionTime": time.Now().Format(time.RFC3339),
				})
				_ = deleteJobAndPerJobService(sessionNamespace, jobName, sessionName)
				return
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
	if err := k8sClient.CoreV1().Services(namespace).Delete(context.TODO(), svcName, v1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to delete per-job service %s/%s: %v", namespace, svcName, err)
	}

	// Delete the Job with background propagation
	policy := v1.DeletePropagationBackground
	if err := k8sClient.BatchV1().Jobs(namespace).Delete(context.TODO(), jobName, v1.DeleteOptions{PropagationPolicy: &policy}); err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to delete job %s/%s: %v", namespace, jobName, err)
		return err
	}

	// Proactively delete Pods for this Job before removing PVC
	if pods, err := k8sClient.CoreV1().Pods(namespace).List(context.TODO(), v1.ListOptions{LabelSelector: fmt.Sprintf("job-name=%s", jobName)}); err == nil {
		for i := range pods.Items {
			p := pods.Items[i]
			if err := k8sClient.CoreV1().Pods(namespace).Delete(context.TODO(), p.Name, v1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
				log.Printf("Failed to delete pod %s/%s for job %s: %v", namespace, p.Name, jobName, err)
			}
		}
	} else if err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to list pods for job %s/%s: %v", namespace, jobName, err)
	}

	// Delete the per-session workspace PVC
	pvcName := fmt.Sprintf("ambient-workspace-%s", sessionName)
	if err := k8sClient.CoreV1().PersistentVolumeClaims(namespace).Delete(context.TODO(), pvcName, v1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		log.Printf("Failed to delete per-session PVC %s/%s: %v", namespace, pvcName, err)
	}
	return nil
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

				// Ensure shared workspace PVC exists
				if err := ensureProjectWorkspacePVC(namespace.Name); err != nil {
					log.Printf("Failed to ensure workspace PVC in %s: %v", namespace.Name, err)
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

var (
	boolPtr          = func(b bool) *bool { return &b }
	int32Ptr         = func(i int32) *int32 { return &i }
	int64Ptr         = func(i int64) *int64 { return &i }
	intstrFromString = func(s string) intstr.IntOrString { return intstr.Parse(s) }
)
