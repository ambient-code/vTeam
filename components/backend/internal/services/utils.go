package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"ambient-code-backend/internal/types"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

// ParseSpec parses the spec from an unstructured object
func ParseSpec(spec map[string]interface{}) types.AgenticSessionSpec {
	result := types.AgenticSessionSpec{}

	if prompt, ok := spec["prompt"].(string); ok {
		result.Prompt = prompt
	}

	if interactive, ok := spec["interactive"].(bool); ok {
		result.Interactive = interactive
	}

	if displayName, ok := spec["displayName"].(string); ok {
		result.DisplayName = displayName
	}

	if timeout, ok := spec["timeout"].(float64); ok {
		result.Timeout = int(timeout)
	}

	if project, ok := spec["project"].(string); ok {
		result.Project = project
	}

	// Parse LLMSettings
	if llmSettings, ok := spec["llmSettings"].(map[string]interface{}); ok {
		if model, ok := llmSettings["model"].(string); ok {
			result.LLMSettings.Model = model
		}
		if temp, ok := llmSettings["temperature"].(float64); ok {
			result.LLMSettings.Temperature = temp
		}
		if maxTokens, ok := llmSettings["maxTokens"].(float64); ok {
			result.LLMSettings.MaxTokens = int(maxTokens)
		}
	}

	// Parse UserContext
	if userContext, ok := spec["userContext"].(map[string]interface{}); ok {
		uc := &types.UserContext{}
		if userID, ok := userContext["userId"].(string); ok {
			uc.UserID = userID
		}
		if displayName, ok := userContext["displayName"].(string); ok {
			uc.DisplayName = displayName
		}
		if groups, ok := userContext["groups"].([]interface{}); ok {
			for _, g := range groups {
				if groupStr, ok := g.(string); ok {
					uc.Groups = append(uc.Groups, groupStr)
				}
			}
		}
		result.UserContext = uc
	}

	// Parse BotAccount
	if botAccount, ok := spec["botAccount"].(map[string]interface{}); ok {
		ba := &types.BotAccountRef{}
		if name, ok := botAccount["name"].(string); ok {
			ba.Name = name
		}
		result.BotAccount = ba
	}

	// Parse ResourceOverrides
	if resourceOverrides, ok := spec["resourceOverrides"].(map[string]interface{}); ok {
		ro := &types.ResourceOverrides{}
		if cpu, ok := resourceOverrides["cpu"].(string); ok {
			ro.CPU = cpu
		}
		if memory, ok := resourceOverrides["memory"].(string); ok {
			ro.Memory = memory
		}
		if storageClass, ok := resourceOverrides["storageClass"].(string); ok {
			ro.StorageClass = storageClass
		}
		if priorityClass, ok := resourceOverrides["priorityClass"].(string); ok {
			ro.PriorityClass = priorityClass
		}
		result.ResourceOverrides = ro
	}

	// Parse GitConfig
	if gitConfig, ok := spec["gitConfig"].(map[string]interface{}); ok {
		gc := &types.GitConfig{}

		if user, ok := gitConfig["user"].(map[string]interface{}); ok {
			gu := &types.GitUser{}
			if name, ok := user["name"].(string); ok {
				gu.Name = name
			}
			if email, ok := user["email"].(string); ok {
				gu.Email = email
			}
			gc.User = gu
		}

		if auth, ok := gitConfig["authentication"].(map[string]interface{}); ok {
			ga := &types.GitAuthentication{}
			if sshKeySecret, ok := auth["sshKeySecret"].(string); ok {
				ga.SSHKeySecret = &sshKeySecret
			}
			if tokenSecret, ok := auth["tokenSecret"].(string); ok {
				ga.TokenSecret = &tokenSecret
			}
			gc.Authentication = ga
		}

		if repos, ok := gitConfig["repositories"].([]interface{}); ok {
			for _, r := range repos {
				if repo, ok := r.(map[string]interface{}); ok {
					gr := types.GitRepository{}
					if url, ok := repo["url"].(string); ok {
						gr.URL = url
					}
					if branch, ok := repo["branch"].(string); ok {
						gr.Branch = &branch
					}
					if clonePath, ok := repo["clonePath"].(string); ok {
						gr.ClonePath = &clonePath
					}
					gc.Repositories = append(gc.Repositories, gr)
				}
			}
		}
		result.GitConfig = gc
	}

	// Parse Paths
	if paths, ok := spec["paths"].(map[string]interface{}); ok {
		p := &types.Paths{}
		if workspace, ok := paths["workspace"].(string); ok {
			p.Workspace = workspace
		}
		if messages, ok := paths["messages"].(string); ok {
			p.Messages = messages
		}
		if inbox, ok := paths["inbox"].(string); ok {
			p.Inbox = inbox
		}
		result.Paths = p
	}

	return result
}

// ParseStatus parses the status from an unstructured object
func ParseStatus(status map[string]interface{}) *types.AgenticSessionStatus {
	result := &types.AgenticSessionStatus{}

	if phase, ok := status["phase"].(string); ok {
		result.Phase = phase
	}

	if message, ok := status["message"].(string); ok {
		result.Message = message
	}

	if startTime, ok := status["startTime"].(string); ok {
		result.StartTime = &startTime
	}

	if completionTime, ok := status["completionTime"].(string); ok {
		result.CompletionTime = &completionTime
	}

	if jobName, ok := status["jobName"].(string); ok {
		result.JobName = jobName
	}

	// New: result summary fields (top-level in status)
	if st, ok := status["subtype"].(string); ok {
		result.Subtype = st
	}

	if ie, ok := status["is_error"].(bool); ok {
		result.IsError = ie
	}
	if nt, ok := status["num_turns"].(float64); ok {
		result.NumTurns = int(nt)
	}
	if sid, ok := status["session_id"].(string); ok {
		result.SessionID = sid
	}
	if tcu, ok := status["total_cost_usd"].(float64); ok {
		result.TotalCostUSD = &tcu
	}
	if usage, ok := status["usage"].(map[string]interface{}); ok {
		result.Usage = usage
	}
	if res, ok := status["result"].(string); ok {
		result.Result = &res
	}

	if stateDir, ok := status["stateDir"].(string); ok {
		result.StateDir = stateDir
	}

	return result
}

// RFEWorkflowToCRObject converts RFEWorkflow to a Kubernetes custom resource object
func RFEWorkflowToCRObject(workflow *types.RFEWorkflow) map[string]interface{} {
	// Build spec
	spec := map[string]interface{}{
		"title":         workflow.Title,
		"description":   workflow.Description,
		"workspacePath": workflow.WorkspacePath,
	}
	if len(workflow.JiraLinks) > 0 {
		links := make([]map[string]interface{}, 0, len(workflow.JiraLinks))
		for _, l := range workflow.JiraLinks {
			links = append(links, map[string]interface{}{"path": l.Path, "jiraKey": l.JiraKey})
		}
		spec["jiraLinks"] = links
	}

	if len(workflow.Repositories) > 0 {
		repos := make([]map[string]interface{}, 0, len(workflow.Repositories))
		for _, r := range workflow.Repositories {
			rm := map[string]interface{}{
				"url": r.URL,
			}
			if r.Branch != nil {
				rm["branch"] = *r.Branch
			}
			if r.ClonePath != nil {
				rm["clonePath"] = *r.ClonePath
			}
			repos = append(repos, rm)
		}
		spec["repositories"] = repos
	}

	labels := map[string]string{
		"project":      workflow.Project,
		"rfe-workflow": workflow.ID,
	}

	return map[string]interface{}{
		"apiVersion": "vteam.ambient-code/v1alpha1",
		"kind":       "RFEWorkflow",
		"metadata": map[string]interface{}{
			"name":      workflow.ID,
			"namespace": workflow.Project,
			"labels":    labels,
		},
		"spec": spec,
	}
}

// UpsertProjectRFEWorkflowCR creates or updates an RFE workflow custom resource
func UpsertProjectRFEWorkflowCR(dyn dynamic.Interface, workflow *types.RFEWorkflow) error {
	if workflow.Project == "" {
		// Only manage CRD for project-scoped workflows
		return nil
	}
	if dyn == nil {
		return fmt.Errorf("no dynamic client provided")
	}
	gvr := types.GetRFEWorkflowResource()
	obj := &unstructured.Unstructured{Object: RFEWorkflowToCRObject(workflow)}
	// Try create, if exists then update
	_, err := dyn.Resource(gvr).Namespace(workflow.Project).Create(context.TODO(), obj, v1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			_, uerr := dyn.Resource(gvr).Namespace(workflow.Project).Update(context.TODO(), obj, v1.UpdateOptions{})
			if uerr != nil {
				return fmt.Errorf("failed to update RFEWorkflow CR: %v", uerr)
			}
			return nil
		}
		return fmt.Errorf("failed to create RFEWorkflow CR: %v", err)
	}
	return nil
}

// RFEFromUnstructured converts an unstructured object to RFEWorkflow
func RFEFromUnstructured(item *unstructured.Unstructured) *types.RFEWorkflow {
	meta := item.Object["metadata"].(map[string]interface{})
	spec := item.Object["spec"].(map[string]interface{})

	workflow := &types.RFEWorkflow{
		ID:            meta["name"].(string),
		Project:       meta["namespace"].(string),
		Title:         spec["title"].(string),
		Description:   spec["description"].(string),
		WorkspacePath: spec["workspacePath"].(string),
	}

	// Parse created/updated timestamps
	if creationTimestamp, ok := meta["creationTimestamp"].(string); ok {
		workflow.CreatedAt = creationTimestamp
	}

	// Parse repositories
	if repos, ok := spec["repositories"].([]interface{}); ok {
		for _, r := range repos {
			if repo, ok := r.(map[string]interface{}); ok {
				gr := types.GitRepository{
					URL: repo["url"].(string),
				}
				if branch, ok := repo["branch"].(string); ok {
					gr.Branch = &branch
				}
				if clonePath, ok := repo["clonePath"].(string); ok {
					gr.ClonePath = &clonePath
				}
				workflow.Repositories = append(workflow.Repositories, gr)
			}
		}
	}

	// Parse jira links
	if links, ok := spec["jiraLinks"].([]interface{}); ok {
		for _, l := range links {
			if link, ok := l.(map[string]interface{}); ok {
				jl := types.WorkflowJiraLink{
					Path:    link["path"].(string),
					JiraKey: link["jiraKey"].(string),
				}
				workflow.JiraLinks = append(workflow.JiraLinks, jl)
			}
		}
	}

	return workflow
}

// SanitizeName sanitizes a name for use as a Kubernetes resource name
func SanitizeName(input string) string {
	// Replace invalid characters with hyphens
	result := strings.ReplaceAll(input, " ", "-")
	result = strings.ReplaceAll(result, "_", "-")
	result = strings.ToLower(result)

	// Remove any characters that aren't alphanumeric or hyphens
	var sanitized strings.Builder
	for _, r := range result {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			sanitized.WriteRune(r)
		}
	}

	result = sanitized.String()

	// Ensure it doesn't start or end with a hyphen
	result = strings.Trim(result, "-")

	// Ensure it's not empty
	if result == "" {
		result = "unnamed"
	}

	return result
}

// ResolveWorkspaceAbsPath resolves a workspace path for a session
func ResolveWorkspaceAbsPath(stateBaseDir, sessionName, relOrAbs string) string {
	if filepath.IsAbs(relOrAbs) {
		return relOrAbs
	}
	if relOrAbs == "" {
		relOrAbs = "workspace"
	}
	return filepath.Join(stateBaseDir, sessionName, relOrAbs)
}

// ResolveWorkflowWorkspaceAbsPath resolves a workspace path for an RFE workflow
func ResolveWorkflowWorkspaceAbsPath(pvcBaseDir, workflowID, relOrAbs string) string {
	if filepath.IsAbs(relOrAbs) {
		return relOrAbs
	}
	if relOrAbs == "" {
		relOrAbs = "workspace"
	}
	return filepath.Join(pvcBaseDir, "rfe-workflows", workflowID, relOrAbs)
}