package types

// AgenticSession represents the structure of our custom resource
type AgenticSession struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Spec       AgenticSessionSpec     `json:"spec"`
	Status     *AgenticSessionStatus  `json:"status,omitempty"`
}

type AgenticSessionSpec struct {
	Prompt               string             `json:"prompt" binding:"required"`
	Interactive          bool               `json:"interactive,omitempty"`
	DisplayName          string             `json:"displayName"`
	LLMSettings          LLMSettings        `json:"llmSettings"`
	Timeout              int                `json:"timeout"`
	UserContext          *UserContext       `json:"userContext,omitempty"`
	BotAccount           *BotAccountRef     `json:"botAccount,omitempty"`
	ResourceOverrides    *ResourceOverrides `json:"resourceOverrides,omitempty"`
	EnvironmentVariables map[string]string  `json:"environmentVariables,omitempty"`
	Project              string             `json:"project,omitempty"`
	// Multi-repo support
	Repos []SimpleRepo `json:"repos,omitempty"`
	// Active workflow for dynamic workflow switching
	ActiveWorkflow *WorkflowSelection `json:"activeWorkflow,omitempty"`
}

// SimpleRepo represents a simplified repository configuration
type SimpleRepo struct {
	URL    string  `json:"url"`
	Branch *string `json:"branch,omitempty"`
}

type AgenticSessionStatus struct {
	Phase    string `json:"phase,omitempty"`
	Message  string `json:"message,omitempty"`
	IsError  bool   `json:"is_error,omitempty"`
}

type CreateAgenticSessionRequest struct {
	Prompt          string       `json:"prompt" binding:"required"`
	DisplayName     string       `json:"displayName,omitempty"`
	LLMSettings     *LLMSettings `json:"llmSettings,omitempty"`
	Timeout         *int         `json:"timeout,omitempty"`
	Interactive     *bool        `json:"interactive,omitempty"`
	ParentSessionID string       `json:"parent_session_id,omitempty"`
	// Multi-repo support
	Repos              []SimpleRepo `json:"repos,omitempty"`
	AutoPushOnComplete *bool        `json:"autoPushOnComplete,omitempty"`
	UserContext          *UserContext         `json:"userContext,omitempty"`
	EnvironmentVariables map[string]string    `json:"environmentVariables,omitempty"`
	Labels               map[string]string    `json:"labels,omitempty"`
	Annotations          map[string]string    `json:"annotations,omitempty"`
}

type CloneSessionRequest struct {
	TargetProject  string `json:"targetProject" binding:"required"`
	NewSessionName string `json:"newSessionName" binding:"required"`
}

type UpdateAgenticSessionRequest struct {
	Prompt      *string      `json:"prompt,omitempty"`
	DisplayName *string      `json:"displayName,omitempty"`
	Timeout     *int         `json:"timeout,omitempty"`
	LLMSettings *LLMSettings `json:"llmSettings,omitempty"`
}

type CloneAgenticSessionRequest struct {
	TargetProject     string `json:"targetProject,omitempty"`
	TargetSessionName string `json:"targetSessionName,omitempty"`
	DisplayName       string `json:"displayName,omitempty"`
	Prompt            string `json:"prompt,omitempty"`
}

// WorkflowSelection represents a workflow to load into the session
type WorkflowSelection struct {
	GitURL string `json:"gitUrl" binding:"required"`
	Branch string `json:"branch,omitempty"`
	Path   string `json:"path,omitempty"`
}
