// Package handlers provides HTTP handlers for the backend API.
// This file contains shared types and helper functions for session handlers.
package handlers

import (
	"context"

	"ambient-code-backend/types"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Package-level variables for session handlers (set from main package)
var (
	GetAgenticSessionV1Alpha1Resource func() schema.GroupVersionResource
	DynamicClient                     dynamic.Interface
	GetGitHubToken                    func(context.Context, *kubernetes.Clientset, dynamic.Interface, string, string) (string, error)
	DeriveRepoFolderFromURL           func(string) string
)

// contentListItem represents a file/directory in the workspace
type contentListItem struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	IsDir      bool   `json:"isDir"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modifiedAt"`
}

// parseSpec parses AgenticSessionSpec with v1alpha1 fields
func parseSpec(spec map[string]interface{}) types.AgenticSessionSpec {
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

	if project, ok := spec["project"].(string); ok {
		result.Project = project
	}

	if timeout, ok := spec["timeout"].(float64); ok {
		result.Timeout = int(timeout)
	}

	if llmSettings, ok := spec["llmSettings"].(map[string]interface{}); ok {
		if model, ok := llmSettings["model"].(string); ok {
			result.LLMSettings.Model = model
		}
		if temperature, ok := llmSettings["temperature"].(float64); ok {
			result.LLMSettings.Temperature = temperature
		}
		if maxTokens, ok := llmSettings["maxTokens"].(float64); ok {
			result.LLMSettings.MaxTokens = int(maxTokens)
		}
	}

	// environmentVariables passthrough
	if env, ok := spec["environmentVariables"].(map[string]interface{}); ok {
		resultEnv := make(map[string]string, len(env))
		for k, v := range env {
			if s, ok := v.(string); ok {
				resultEnv[k] = s
			}
		}
		if len(resultEnv) > 0 {
			result.EnvironmentVariables = resultEnv
		}
	}

	if userContext, ok := spec["userContext"].(map[string]interface{}); ok {
		uc := &types.UserContext{}
		if userID, ok := userContext["userId"].(string); ok {
			uc.UserID = userID
		}
		if displayName, ok := userContext["displayName"].(string); ok {
			uc.DisplayName = displayName
		}
		if groups, ok := userContext["groups"].([]interface{}); ok {
			uc.Groups = make([]string, len(groups))
			for i, g := range groups {
				if groupStr, ok := g.(string); ok {
					uc.Groups[i] = groupStr
				}
			}
		}
		result.UserContext = uc
	}

	if botAccount, ok := spec["botAccount"].(map[string]interface{}); ok {
		ba := &types.BotAccountRef{}
		if name, ok := botAccount["name"].(string); ok {
			ba.Name = name
		}
		result.BotAccount = ba
	}

	// Parse repos (multi-repo support)
	if repos, ok := spec["repos"].([]interface{}); ok {
		for _, r := range repos {
			if repoMap, ok := r.(map[string]interface{}); ok {
				mapping := types.SessionRepoMapping{}

				// Parse input
				if input, ok := repoMap["input"].(map[string]interface{}); ok {
					if url, ok := input["url"].(string); ok {
						mapping.Input.URL = url
					}
					if branch, ok := input["branch"].(string); ok {
						mapping.Input.Branch = &branch
					}
				}

				// Parse output
				if output, ok := repoMap["output"].(map[string]interface{}); ok {
					outRepo := &types.OutputNamedGitRepo{}
					if url, ok := output["url"].(string); ok {
						outRepo.URL = url
					}
					if branch, ok := output["branch"].(string); ok {
						outRepo.Branch = &branch
					}
					mapping.Output = outRepo
				}

				// Parse status
				if status, ok := repoMap["status"].(string); ok {
					mapping.Status = &status
				}

				result.Repos = append(result.Repos, mapping)
			}
		}
	}

	// Parse mainRepoIndex
	if mainRepoIndex, ok := spec["mainRepoIndex"].(float64); ok {
		idx := int(mainRepoIndex)
		result.MainRepoIndex = &idx
	}

	// Parse resourceOverrides
	if ro, ok := spec["resourceOverrides"].(map[string]interface{}); ok {
		overrides := &types.ResourceOverrides{}

		if cpu, ok := ro["cpu"].(string); ok {
			overrides.CPU = cpu
		}

		if memory, ok := ro["memory"].(string); ok {
			overrides.Memory = memory
		}

		if storageClass, ok := ro["storageClass"].(string); ok {
			overrides.StorageClass = storageClass
		}

		if priorityClass, ok := ro["priorityClass"].(string); ok {
			overrides.PriorityClass = priorityClass
		}

		result.ResourceOverrides = overrides
	}

	return result
}

// parseStatus parses AgenticSessionStatus including v1alpha1 runner output fields
func parseStatus(status map[string]interface{}) *types.AgenticSessionStatus {
	if status == nil {
		return nil
	}

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
	if stateDir, ok := status["stateDir"].(string); ok {
		result.StateDir = stateDir
	}

	// Parse runner output fields from v1alpha1
	if subtype, ok := status["subtype"].(string); ok {
		result.Subtype = subtype
	}
	if isError, ok := status["is_error"].(bool); ok {
		result.IsError = isError
	}
	if numTurns, ok := status["num_turns"].(float64); ok {
		result.NumTurns = int(numTurns)
	}
	if sessionID, ok := status["session_id"].(string); ok {
		result.SessionID = sessionID
	}
	if totalCostUSD, ok := status["total_cost_usd"].(float64); ok {
		result.TotalCostUSD = &totalCostUSD
	}
	if usage, ok := status["usage"].(map[string]interface{}); ok {
		result.Usage = usage
	}

	// Parse result if present
	if r, ok := status["result"].(string); ok {
		result.Result = &r
	}

	return result
}
