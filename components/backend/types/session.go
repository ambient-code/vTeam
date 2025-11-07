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
	// Multi-repo support (unified mapping)
	Repos         []SessionRepoMapping `json:"repos,omitempty"`
	MainRepoIndex *int                 `json:"mainRepoIndex,omitempty"`
}

// NamedGitRepo represents named repository types for multi-repo session support.
type NamedGitRepo struct {
	URL    string  `json:"url"`
	Branch *string `json:"branch,omitempty"`
}

type OutputNamedGitRepo struct {
	URL    string  `json:"url"`
	Branch *string `json:"branch,omitempty"`
}

// SessionRepoMapping is a unified session repo mapping.
type SessionRepoMapping struct {
	Input  NamedGitRepo        `json:"input"`
	Output *OutputNamedGitRepo `json:"output,omitempty"`
	Status *string             `json:"status,omitempty"`
}

type AgenticSessionStatus struct {
	Phase          string  `json:"phase,omitempty"`
	Message        string  `json:"message,omitempty"`
	StartTime      *string `json:"startTime,omitempty"`
	CompletionTime *string `json:"completionTime,omitempty"`
	JobName        string  `json:"jobName,omitempty"`
	StateDir       string  `json:"stateDir,omitempty"`
	// Result summary fields from runner
	Subtype      string                 `json:"subtype,omitempty"`
	IsError      bool                   `json:"is_error,omitempty"`
	NumTurns     int                    `json:"num_turns,omitempty"`
	SessionID    string                 `json:"session_id,omitempty"`
	TotalCostUSD *float64               `json:"total_cost_usd,omitempty"`
	Usage        map[string]interface{} `json:"usage,omitempty"`
	Result       *string                `json:"result,omitempty"`
}

type CreateAgenticSessionRequest struct {
	Prompt          string       `json:"prompt" binding:"required"`
	DisplayName     string       `json:"displayName,omitempty"`
	LLMSettings     *LLMSettings `json:"llmSettings,omitempty"`
	Timeout         *int         `json:"timeout,omitempty"`
	Interactive     *bool        `json:"interactive,omitempty"`
	WorkspacePath   string       `json:"workspacePath,omitempty"`
	ParentSessionID string       `json:"parent_session_id,omitempty"`
	// Multi-repo support (unified mapping)
	Repos                []SessionRepoMapping `json:"repos,omitempty"`
	MainRepoIndex        *int                 `json:"mainRepoIndex,omitempty"`
	AutoPushOnComplete   *bool                `json:"autoPushOnComplete,omitempty"`
	UserContext          *UserContext         `json:"userContext,omitempty"`
	BotAccount           *BotAccountRef       `json:"botAccount,omitempty"`
	ResourceOverrides    *ResourceOverrides   `json:"resourceOverrides,omitempty"`
	EnvironmentVariables map[string]string    `json:"environmentVariables,omitempty"`
	Labels               map[string]string    `json:"labels,omitempty"`
	Annotations          map[string]string    `json:"annotations,omitempty"`
}

type CloneSessionRequest struct {
	TargetProject  string `json:"targetProject" binding:"required"`
	NewSessionName string `json:"newSessionName" binding:"required"`
}
