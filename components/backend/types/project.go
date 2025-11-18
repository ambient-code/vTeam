package types

// AmbientProject represents project management types.
type AmbientProject struct {
	Name              string            `json:"name"`                  // Kubernetes namespace name
	DisplayName       string            `json:"displayName"`           // OpenShift display name (empty for k8s)
	Description       string            `json:"description,omitempty"` // OpenShift description (empty for k8s)
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	CreationTimestamp string            `json:"creationTimestamp"`
	Status            string            `json:"status"`
	IsOpenShift       bool              `json:"isOpenShift"` // true if running on OpenShift cluster
}

type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"displayName,omitempty"` // Optional: only used on OpenShift
	Description string `json:"description,omitempty"` // Optional: only used on OpenShift
}
