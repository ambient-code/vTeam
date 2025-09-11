package main

import (
	"context"
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
	k8sClient         *kubernetes.Clientset
	dynamicClient     dynamic.Interface
	namespace         string
	claudeRunnerImage string
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

	// Get claude-runner image from environment or use default
	claudeRunnerImage = os.Getenv("CLAUDE_RUNNER_IMAGE")
	if claudeRunnerImage == "" {
		claudeRunnerImage = "quay.io/gkrumbach07/claude-runner:latest"
	}

	log.Printf("Research Session Operator starting in namespace: %s", namespace)
	log.Printf("Using claude-runner image: %s", claudeRunnerImage)

	// Start watching ResearchSession resources
	go watchResearchSessions()

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

func getResearchSessionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "research.example.com",
		Version:  "v1",
		Resource: "researchsessions",
	}
}

func watchResearchSessions() {
	gvr := getResearchSessionResource()

	for {
		watcher, err := dynamicClient.Resource(gvr).Namespace(namespace).Watch(context.TODO(), v1.ListOptions{})
		if err != nil {
			log.Printf("Failed to create watcher: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("Watching for ResearchSession events...")

		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added, watch.Modified:
				obj := event.Object.(*unstructured.Unstructured)

				// Add small delay to avoid race conditions with rapid create/delete cycles
				time.Sleep(100 * time.Millisecond)

				if err := handleResearchSessionEvent(obj); err != nil {
					log.Printf("Error handling ResearchSession event: %v", err)
				}
			case watch.Deleted:
				obj := event.Object.(*unstructured.Unstructured)
				sessionName := obj.GetName()
				log.Printf("ResearchSession %s deleted", sessionName)

				// Cancel any ongoing job monitoring for this session
				// (We could implement this with a context cancellation if needed)
			case watch.Error:
				obj := event.Object.(*unstructured.Unstructured)
				log.Printf("Watch error for ResearchSession: %v", obj)
			}
		}

		log.Println("Watch channel closed, restarting...")
		watcher.Stop()
		time.Sleep(2 * time.Second)
	}
}

func handleResearchSessionEvent(obj *unstructured.Unstructured) error {
	name := obj.GetName()

	// Verify the resource still exists before processing
	gvr := getResearchSessionResource()
	currentObj, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("ResearchSession %s no longer exists, skipping processing", name)
			return nil
		}
		return fmt.Errorf("failed to verify ResearchSession %s exists: %v", name, err)
	}

	// Get the current status from the fresh object
	status, _, _ := unstructured.NestedMap(currentObj.Object, "status")
	phase, _, _ := unstructured.NestedString(status, "phase")

	log.Printf("Processing ResearchSession %s with phase %s", name, phase)

	// Only process if status is Pending
	if phase != "Pending" {
		return nil
	}

	// Create a Kubernetes Job for this ResearchSession
	jobName := fmt.Sprintf("%s-job", name)

	// Check if job already exists
	_, err = k8sClient.BatchV1().Jobs(namespace).Get(context.TODO(), jobName, v1.GetOptions{})
	if err == nil {
		log.Printf("Job %s already exists for ResearchSession %s", jobName, name)
		return nil
	}

	// Extract spec information from the fresh object
	spec, _, _ := unstructured.NestedMap(currentObj.Object, "spec")
	prompt, _, _ := unstructured.NestedString(spec, "prompt")
	websiteURL, _, _ := unstructured.NestedString(spec, "websiteURL")
	timeout, _, _ := unstructured.NestedInt64(spec, "timeout")

	llmSettings, _, _ := unstructured.NestedMap(spec, "llmSettings")
	model, _, _ := unstructured.NestedString(llmSettings, "model")
	temperature, _, _ := unstructured.NestedFloat64(llmSettings, "temperature")
	maxTokens, _, _ := unstructured.NestedInt64(llmSettings, "maxTokens")

	// Create the Job
	job := &batchv1.Job{
		ObjectMeta: v1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels: map[string]string{
				"research-session": name,
				"app":              "claude-runner",
			},
			OwnerReferences: []v1.OwnerReference{
				{
					APIVersion: "research.example.com/v1",
					Kind:       "ResearchSession",
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
						"research-session": name,
						"app":              "claude-runner",
					},
					// If you run a service mesh that injects sidecars and causes egress issues for Jobs:
					// Annotations: map[string]string{"sidecar.istio.io/inject": "false"},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,

					// ⚠️ Let OpenShift SCC choose UID/GID dynamically (restricted-v2 compatible)
					// SecurityContext omitted to allow SCC assignment

					// 🔧 Optional: Shared memory volume (may need to be removed if SCC restricts it)
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
					},

					Containers: []corev1.Container{
						{
							Name:  "claude-runner",
							Image: claudeRunnerImage,
							// 🔒 Container-level security (SCC-compatible, no privileged capabilities)
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: boolPtr(false),
								ReadOnlyRootFilesystem:   boolPtr(false), // Playwright needs to write temp files
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"}, // Drop all capabilities for security
								},
							},

							// 📦 Mount shared memory volume
							VolumeMounts: []corev1.VolumeMount{
								{Name: "dshm", MountPath: "/dev/shm"},
							},

							Env: []corev1.EnvVar{
								{Name: "RESEARCH_SESSION_NAME", Value: name},
								{Name: "RESEARCH_SESSION_NAMESPACE", Value: namespace},
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
											LocalObjectReference: corev1.LocalObjectReference{Name: "claude-research-secrets"},
											Key:                  "anthropic-api-key",
										},
									},
								},

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
							},

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
	if err := updateResearchSessionStatus(name, map[string]interface{}{
		"phase":   "Creating",
		"message": "Creating Kubernetes job",
	}); err != nil {
		log.Printf("Failed to update ResearchSession status to Creating: %v", err)
		// Continue anyway - resource might have been deleted
	}

	// Create the job
	_, err = k8sClient.BatchV1().Jobs(namespace).Create(context.TODO(), job, v1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create job %s: %v", jobName, err)
		// Update status to Error if job creation fails and resource still exists
		updateResearchSessionStatus(name, map[string]interface{}{
			"phase":   "Error",
			"message": fmt.Sprintf("Failed to create job: %v", err),
		})
		return fmt.Errorf("failed to create job: %v", err)
	}

	log.Printf("Created job %s for ResearchSession %s", jobName, name)

	// Update ResearchSession status to Running
	if err := updateResearchSessionStatus(name, map[string]interface{}{
		"phase":     "Running",
		"message":   "Job created and running",
		"startTime": time.Now().Format(time.RFC3339),
		"jobName":   jobName,
	}); err != nil {
		log.Printf("Failed to update ResearchSession status to Running: %v", err)
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

		// First check if the ResearchSession still exists
		gvr := getResearchSessionResource()
		if _, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), sessionName, v1.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				log.Printf("ResearchSession %s no longer exists, stopping job monitoring for %s", sessionName, jobName)
				return
			}
			log.Printf("Error checking ResearchSession %s existence: %v", sessionName, err)
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

			// Update ResearchSession status to Completed
			updateResearchSessionStatus(sessionName, map[string]interface{}{
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

			// Update ResearchSession status to Failed
			updateResearchSessionStatus(sessionName, map[string]interface{}{
				"phase":          "Failed",
				"message":        errorMessage,
				"completionTime": time.Now().Format(time.RFC3339),
			})
			return
		}
	}
}

func updateResearchSessionStatus(name string, statusUpdate map[string]interface{}) error {
	gvr := getResearchSessionResource()

	// Get current resource
	obj, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("ResearchSession %s no longer exists, skipping status update", name)
			return nil // Don't treat this as an error - resource was deleted
		}
		return fmt.Errorf("failed to get ResearchSession %s: %v", name, err)
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
			log.Printf("ResearchSession %s was deleted during status update, skipping", name)
			return nil // Don't treat this as an error - resource was deleted
		}
		return fmt.Errorf("failed to update ResearchSession status: %v", err)
	}

	return nil
}

var (
	boolPtr  = func(b bool) *bool { return &b }
	int32Ptr = func(i int32) *int32 { return &i }
	int64Ptr = func(i int64) *int64 { return &i }
)
