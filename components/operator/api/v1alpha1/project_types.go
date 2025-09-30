// Project Custom Resource Definition
// C4 Architecture: Represents a multi-tenant project in vTeam

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectSpec defines the desired state of Project
type ProjectSpec struct {
	// Description of the project
	Description string `json:"description"`

	// Owner is the username of the project owner
	Owner string `json:"owner"`

	// Members are the usernames of project members
	Members []string `json:"members,omitempty"`

	// Quotas define resource limits for the project
	Quotas ResourceQuotas `json:"quotas"`

	// Labels to apply to project resources
	Labels map[string]string `json:"labels,omitempty"`
}

// ResourceQuotas defines resource limits
type ResourceQuotas struct {
	// Maximum number of concurrent sessions
	MaxSessions int `json:"maxSessions"`

	// Maximum CPU allocation
	MaxCPU string `json:"maxCpu"`

	// Maximum memory allocation
	MaxMemory string `json:"maxMemory"`

	// Maximum storage allocation
	MaxStorage string `json:"maxStorage"`
}

// ProjectStatus defines the observed state of Project
type ProjectStatus struct {
	// Phase of the project lifecycle
	Phase string `json:"phase"`

	// Namespace created for this project
	Namespace string `json:"namespace,omitempty"`

	// ActiveSessions count
	ActiveSessions int `json:"activeSessions"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastUpdated timestamp
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Namespaced
//+kubebuilder:printcolumn:name="Owner",type=string,JSONPath=`.spec.owner`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Sessions",type=integer,JSONPath=`.status.activeSessions`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Project is the Schema for the projects API
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProjectList contains a list of Project
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}