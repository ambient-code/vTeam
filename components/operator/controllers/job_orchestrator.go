// Job Orchestrator Component
// C4 Architecture: Creates and monitors Kubernetes Jobs for AI runners
// Manages the execution of Claude Code runner pods

package controllers

import (
	"bytes"
	"context"
	"fmt"
	"io"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	vteamv1alpha1 "github.com/jeremyeder/vteam/operator/api/v1alpha1"
)

// JobOrchestrator creates and monitors Jobs for AI execution
type JobOrchestrator struct {
	client    client.Client
	clientset kubernetes.Interface
}

// NewJobOrchestrator creates a new JobOrchestrator
func NewJobOrchestrator(client client.Client) *JobOrchestrator {
	// Create clientset for pod log access
	config, _ := rest.InClusterConfig()
	if config == nil {
		// Fallback for local development
		config, _ = rest.InClusterConfig()
	}

	clientset, _ := kubernetes.NewForConfig(config)

	return &JobOrchestrator{
		client:    client,
		clientset: clientset,
	}
}

// CreateJobForSession creates a Kubernetes Job for an AgenticSession
func (j *JobOrchestrator) CreateJobForSession(ctx context.Context, session *vteamv1alpha1.AgenticSession) (*batchv1.Job, error) {
	// Prepare environment variables
	env := []corev1.EnvVar{
		{
			Name:  "SESSION_NAME",
			Value: session.Name,
		},
		{
			Name:  "SESSION_NAMESPACE",
			Value: session.Namespace,
		},
		{
			Name:  "AGENT_TYPE",
			Value: session.Spec.Agent,
		},
		{
			Name:  "TASK_DESCRIPTION",
			Value: session.Spec.Task,
		},
	}

	// Add custom parameters as env vars
	for key, value := range session.Spec.Parameters {
		env = append(env, corev1.EnvVar{
			Name:  fmt.Sprintf("PARAM_%s", key),
			Value: value,
		})
	}

	// Add runner environment variables
	env = append(env, session.Spec.Runner.Env...)

	// Prepare volume mounts
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "workspace",
			MountPath: "/workspace",
		},
	}

	// Add secret volume mounts
	for i, secretName := range session.Spec.Secrets {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      fmt.Sprintf("secret-%d", i),
			MountPath: fmt.Sprintf("/secrets/%s", secretName),
			ReadOnly:  true,
		})
	}

	// Add custom volume mounts
	volumeMounts = append(volumeMounts, session.Spec.Runner.VolumeMounts...)

	// Prepare volumes
	volumes := []corev1.Volume{
		{
			Name: "workspace",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	// Add secret volumes
	for i, secretName := range session.Spec.Secrets {
		volumes = append(volumes, corev1.Volume{
			Name: fmt.Sprintf("secret-%d", i),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		})
	}

	// Create Job specification
	backoffLimit := int32(0)
	ttlSecondsAfterFinished := int32(3600) // Clean up after 1 hour

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-job-", session.Name),
			Namespace:    session.Namespace,
			Labels: map[string]string{
				"app":            "vteam",
				"session":        session.Name,
				"agent":          session.Spec.Agent,
				"component":      "runner",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       "vteam",
						"session":   session.Name,
						"agent":     session.Spec.Agent,
						"component": "runner",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "claude-runner",
							Image:           session.Spec.Runner.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command: []string{
								"/bin/sh",
								"-c",
								j.buildRunnerCommand(session),
							},
							Env:          env,
							Resources:    session.Spec.Runner.Resources,
							VolumeMounts: volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	// Set owner reference for garbage collection
	if err := controllerutil.SetControllerReference(session, job, j.client.Scheme()); err != nil {
		return nil, err
	}

	// Create the Job
	if err := j.client.Create(ctx, job); err != nil {
		return nil, err
	}

	return job, nil
}

// buildRunnerCommand builds the command for the runner container
func (j *JobOrchestrator) buildRunnerCommand(session *vteamv1alpha1.AgenticSession) string {
	// Base command for Claude Code CLI
	command := `
#!/bin/bash
set -e

echo "Starting vTeam AI Runner"
echo "Session: $SESSION_NAME"
echo "Agent: $AGENT_TYPE"
echo "Task: $TASK_DESCRIPTION"

# Initialize workspace
cd /workspace

# Load agent configuration based on type
case "$AGENT_TYPE" in
  "general-purpose")
    echo "Loading general-purpose agent..."
    export AGENT_CONFIG="/agents/general.yaml"
    ;;
  "sre-reliability-engineer")
    echo "Loading SRE agent..."
    export AGENT_CONFIG="/agents/sre.yaml"
    ;;
  "tdd-developer")
    echo "Loading TDD developer agent..."
    export AGENT_CONFIG="/agents/tdd.yaml"
    ;;
  *)
    echo "Loading default agent..."
    export AGENT_CONFIG="/agents/default.yaml"
    ;;
esac

# Execute Claude Code CLI with the task
echo "Executing AI task..."

# Create task file
cat > task.md << EOF
$TASK_DESCRIPTION
EOF

# Run Claude Code CLI (placeholder for actual implementation)
# In real implementation, this would:
# 1. Initialize Claude Code CLI
# 2. Load agent configuration
# 3. Execute the task
# 4. Store results back to CR

echo "=== AI EXECUTION START ==="

# Simulate AI execution (replace with actual Claude Code CLI)
python3 /app/runner.py \
  --task-file task.md \
  --agent "$AGENT_TYPE" \
  --output-format json

echo "=== AI EXECUTION END ==="

# Update session status via Kubernetes API
echo "Task completed successfully"
`
	return command
}

// GetJobOutput retrieves the output from a Job's pod logs
func (j *JobOrchestrator) GetJobOutput(ctx context.Context, job *batchv1.Job) (string, error) {
	// Find the pod created by this job
	podList := &corev1.PodList{}
	err := j.client.List(ctx, podList, client.InNamespace(job.Namespace), client.MatchingLabels{
		"job-name": job.Name,
	})

	if err != nil {
		return "", err
	}

	if len(podList.Items) == 0 {
		return "", fmt.Errorf("no pods found for job %s", job.Name)
	}

	// Get logs from the first pod
	pod := podList.Items[0]

	if j.clientset == nil {
		return "Log retrieval not available", nil
	}

	// Get pod logs
	req := j.clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
	logs, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer logs.Close()

	// Read logs
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, logs)
	if err != nil {
		return "", err
	}

	output := buf.String()

	// Extract output between markers if present
	startMarker := "=== AI EXECUTION START ==="
	endMarker := "=== AI EXECUTION END ==="

	startIdx := bytes.Index(buf.Bytes(), []byte(startMarker))
	endIdx := bytes.Index(buf.Bytes(), []byte(endMarker))

	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		startIdx += len(startMarker)
		output = string(buf.Bytes()[startIdx:endIdx])
	}

	return output, nil
}

// GetJobStatus returns the current status of a Job
func (j *JobOrchestrator) GetJobStatus(ctx context.Context, jobName, namespace string) (string, error) {
	job := &batchv1.Job{}
	err := j.client.Get(ctx, client.ObjectKey{
		Name:      jobName,
		Namespace: namespace,
	}, job)

	if err != nil {
		return "", err
	}

	if job.Status.Succeeded > 0 {
		return "Succeeded", nil
	}

	if job.Status.Failed > 0 {
		return "Failed", nil
	}

	if job.Status.Active > 0 {
		return "Running", nil
	}

	return "Pending", nil
}