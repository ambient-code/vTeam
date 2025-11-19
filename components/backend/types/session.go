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
	InitialPrompt        string             `json:"initialPrompt,omitempty"`
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
	ObservedGeneration int64               `json:"observedGeneration,omitempty"`
	Phase              string              `json:"phase,omitempty"`
	StartTime          *string             `json:"startTime,omitempty"`
	CompletionTime     *string             `json:"completionTime,omitempty"`
	ReconciledRepos    []ReconciledRepo    `json:"reconciledRepos,omitempty"`
	ReconciledWorkflow *ReconciledWorkflow `json:"reconciledWorkflow,omitempty"`
	SDKSessionID       string              `json:"sdkSessionId,omitempty"`
	SDKRestartCount    int                 `json:"sdkRestartCount,omitempty"`
	Conditions         []Condition         `json:"conditions,omitempty"`
}

type CreateAgenticSessionRequest struct {
	InitialPrompt   string       `json:"initialPrompt,omitempty"`
	DisplayName     string       `json:"displayName,omitempty"`
	LLMSettings     *LLMSettings `json:"llmSettings,omitempty"`
	Timeout         *int         `json:"timeout,omitempty"`
	Interactive     *bool        `json:"interactive,omitempty"`
	ParentSessionID string       `json:"parent_session_id,omitempty"`
	// Multi-repo support
	Repos                []SimpleRepo      `json:"repos,omitempty"`
	AutoPushOnComplete   *bool             `json:"autoPushOnComplete,omitempty"`
	UserContext          *UserContext      `json:"userContext,omitempty"`
	EnvironmentVariables map[string]string `json:"environmentVariables,omitempty"`
	Labels               map[string]string `json:"labels,omitempty"`
	Annotations          map[string]string `json:"annotations,omitempty"`
}

type CloneSessionRequest struct {
	TargetProject  string `json:"targetProject" binding:"required"`
	NewSessionName string `json:"newSessionName" binding:"required"`
}

type UpdateAgenticSessionRequest struct {
	InitialPrompt *string      `json:"initialPrompt,omitempty"`
	DisplayName   *string      `json:"displayName,omitempty"`
	Timeout       *int         `json:"timeout,omitempty"`
	LLMSettings   *LLMSettings `json:"llmSettings,omitempty"`
}

type CloneAgenticSessionRequest struct {
	TargetProject     string `json:"targetProject,omitempty"`
	TargetSessionName string `json:"targetSessionName,omitempty"`
	DisplayName       string `json:"displayName,omitempty"`
	InitialPrompt     string `json:"initialPrompt,omitempty"`
}

// WorkflowSelection represents a workflow to load into the session
type WorkflowSelection struct {
	GitURL string `json:"gitUrl" binding:"required"`
	Branch string `json:"branch,omitempty"`
	Path   string `json:"path,omitempty"`
}

// ReconciledRepo captures reconciliation state for a repository
type ReconciledRepo struct {
	URL      string  `json:"url"`
	Branch   string  `json:"branch"`
	Name     string  `json:"name,omitempty"`
	Status   string  `json:"status,omitempty"`
	ClonedAt *string `json:"clonedAt,omitempty"`
}

// ReconciledWorkflow captures reconciliation state for the active workflow
type ReconciledWorkflow struct {
	GitURL    string  `json:"gitUrl"`
	Branch    string  `json:"branch"`
	Path      string  `json:"path,omitempty"`
	Status    string  `json:"status,omitempty"`
	AppliedAt *string `json:"appliedAt,omitempty"`
}

// Condition mirrors metav1.Condition for API transport
type Condition struct {
	Type               string `json:"type"`
	Status             string `json:"status"`
	Reason             string `json:"reason,omitempty"`
	Message            string `json:"message,omitempty"`
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	ObservedGeneration int64  `json:"observedGeneration,omitempty"`
}
