package types

import "time"

// Workflow represents a registered LangGraph workflow
type Workflow struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Owner     string    `json:"owner"`
	Project   string    `json:"project"`
	CreatedAt time.Time `json:"createdAt"`
}

// WorkflowVersion represents a version of a workflow with its image
type WorkflowVersion struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflowId"`
	Version     string                 `json:"version"`
	ImageDigest string                 `json:"imageDigest"` // Full digest: quay.io/org/repo@sha256:...
	Graphs      []WorkflowGraph        `json:"graphs"`      // Multiple graphs per image
	InputsSchema map[string]interface{} `json:"inputsSchema,omitempty"` // JSONSchema for UI
	CreatedAt   time.Time              `json:"createdAt"`
}

// WorkflowGraph represents a graph entry point in a workflow version
type WorkflowGraph struct {
	Name  string `json:"name"`  // Display name (e.g., "spec_kit")
	Entry string `json:"entry"` // Module:function (e.g., "app:build_app")
}

// CreateWorkflowRequest represents a request to register a new workflow
type CreateWorkflowRequest struct {
	Name        string                 `json:"name" binding:"required"`
	ImageDigest string                 `json:"imageDigest" binding:"required"` // Must be digest format
	Graphs      []WorkflowGraph        `json:"graphs" binding:"required"`
	InputsSchema map[string]interface{} `json:"inputsSchema,omitempty"`
}

// CreateWorkflowVersionRequest represents a request to add a new version to an existing workflow
type CreateWorkflowVersionRequest struct {
	Version     string                 `json:"version" binding:"required"`
	ImageDigest string                 `json:"imageDigest" binding:"required"`
	Graphs      []WorkflowGraph        `json:"graphs" binding:"required"`
	InputsSchema map[string]interface{} `json:"inputsSchema,omitempty"`
}

// WorkflowRef references a workflow for use in AgenticSession
type WorkflowRef struct {
	Name    string `json:"name" binding:"required"`
	Version string `json:"version,omitempty"` // Optional, defaults to latest
	Graph   string `json:"graph" binding:"required"` // Graph name from workflow version's graphs array
}

