package types

import "k8s.io/apimachinery/pkg/runtime/schema"

// AgenticSession represents the structure of our custom resource
type AgenticSession struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Spec       AgenticSessionSpec     `json:"spec"`
	Status     *AgenticSessionStatus  `json:"status,omitempty"`
}

type AgenticSessionSpec struct {
	Prompt            string             `json:"prompt" binding:"required"`
	Interactive       bool               `json:"interactive,omitempty"`
	DisplayName       string             `json:"displayName"`
	LLMSettings       LLMSettings        `json:"llmSettings"`
	Timeout           int                `json:"timeout"`
	UserContext       *UserContext       `json:"userContext,omitempty"`
	BotAccount        *BotAccountRef     `json:"botAccount,omitempty"`
	ResourceOverrides *ResourceOverrides `json:"resourceOverrides,omitempty"`
	Project           string             `json:"project,omitempty"`
	GitConfig         *GitConfig         `json:"gitConfig,omitempty"`
	Paths             *Paths             `json:"paths,omitempty"`
}

type LLMSettings struct {
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"maxTokens"`
}

type Paths struct {
	Workspace string `json:"workspace,omitempty"`
	Messages  string `json:"messages,omitempty"`
	Inbox     string `json:"inbox,omitempty"`
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
	Prompt               string             `json:"prompt" binding:"required"`
	DisplayName          string             `json:"displayName,omitempty"`
	LLMSettings          *LLMSettings       `json:"llmSettings,omitempty"`
	Timeout              *int               `json:"timeout,omitempty"`
	Interactive          *bool              `json:"interactive,omitempty"`
	WorkspacePath        string             `json:"workspacePath,omitempty"`
	GitConfig            *GitConfig         `json:"gitConfig,omitempty"`
	UserContext          *UserContext       `json:"userContext,omitempty"`
	BotAccount           *BotAccountRef     `json:"botAccount,omitempty"`
	ResourceOverrides    *ResourceOverrides `json:"resourceOverrides,omitempty"`
	EnvironmentVariables map[string]string  `json:"environmentVariables,omitempty"`
	Labels               map[string]string  `json:"labels,omitempty"`
	Annotations          map[string]string  `json:"annotations,omitempty"`
}

type UpdateAgenticSessionRequest struct {
	DisplayName *string      `json:"displayName,omitempty"`
	LLMSettings *LLMSettings `json:"llmSettings,omitempty"`
	Timeout     *int         `json:"timeout,omitempty"`
}

type CloneAgenticSessionRequest struct {
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

type CloneSessionRequest struct {
	TargetProject  string `json:"targetProject" binding:"required"`
	NewSessionName string `json:"newSessionName" binding:"required"`
}

// getAgenticSessionV1Alpha1Resource returns the GroupVersionResource for AgenticSession v1alpha1
func GetAgenticSessionV1Alpha1Resource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1alpha1",
		Resource: "agenticsessions",
	}
}