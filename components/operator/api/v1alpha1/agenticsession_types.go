// AgenticSession Custom Resource Definition
// C4 Architecture: Represents an AI-powered automation session

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AgenticSessionSpec defines the desired state of AgenticSession
type AgenticSessionSpec struct {
	// Description of the session's purpose
	Description string `json:"description"`

	// Task to be executed by the AI agent
	Task string `json:"task"`

	// Agent type to use for this session
	Agent string `json:"agent"`

	// Parameters for the agent execution
	Parameters map[string]string `json:"parameters,omitempty"`

	// Secrets to mount in the runner pod
	Secrets []string `json:"secrets,omitempty"`

	// Timeout in seconds for the session
	Timeout int32 `json:"timeout,omitempty"`

	// Runner configuration
	Runner RunnerConfig `json:"runner"`
}

// RunnerConfig defines the runner pod configuration
type RunnerConfig struct {
	// Container image for the runner
	Image string `json:"image"`

	// Resource requirements
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Environment variables
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Volume mounts for the runner
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
}

// AgenticSessionStatus defines the observed state of AgenticSession
type AgenticSessionStatus struct {
	// Phase of the session (Pending, Running, Succeeded, Failed)
	Phase string `json:"phase"`

	// Message providing details about the current phase
	Message string `json:"message,omitempty"`

	// JobName is the name of the Kubernetes Job created
	JobName string `json:"jobName,omitempty"`

	// StartTime of the session
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// EndTime of the session
	EndTime *metav1.Time `json:"endTime,omitempty"`

	// Output from the AI execution
	Output string `json:"output,omitempty"`

	// Error message if the session failed
	Error string `json:"error,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Namespaced,shortName=session;sessions
//+kubebuilder:printcolumn:name="Agent",type=string,JSONPath=`.spec.agent`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Job",type=string,JSONPath=`.status.jobName`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// AgenticSession is the Schema for the agenticsessions API
type AgenticSession struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgenticSessionSpec   `json:"spec,omitempty"`
	Status AgenticSessionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AgenticSessionList contains a list of AgenticSession
type AgenticSessionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgenticSession `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AgenticSession{}, &AgenticSessionList{})
}