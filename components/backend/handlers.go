package main

import (
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Helper pointer functions (used by jira.go)
var (
	boolPtr   = func(b bool) *bool { return &b }
	stringPtr = func(s string) *string { return &s }
)

// rfeFromUnstructured converts an unstructured RFEWorkflow CR into our RFEWorkflow struct
// Kept in main package for jira.go dependency
func rfeFromUnstructured(item *unstructured.Unstructured) *RFEWorkflow {
	if item == nil {
		return nil
	}
	obj := item.Object
	spec, _ := obj["spec"].(map[string]interface{})

	created := ""
	if item.GetCreationTimestamp().Time != (time.Time{}) {
		created = item.GetCreationTimestamp().Time.UTC().Format(time.RFC3339)
	}
	wf := &RFEWorkflow{
		ID:            item.GetName(),
		Title:         fmt.Sprintf("%v", spec["title"]),
		Description:   fmt.Sprintf("%v", spec["description"]),
		Project:       item.GetNamespace(),
		WorkspacePath: fmt.Sprintf("%v", spec["workspacePath"]),
		CreatedAt:     created,
		UpdatedAt:     time.Now().UTC().Format(time.RFC3339),
	}

	// Parse umbrellaRepo/supportingRepos when present; fallback to repositories
	if um, ok := spec["umbrellaRepo"].(map[string]interface{}); ok {
		repo := GitRepository{}
		if u, ok := um["url"].(string); ok {
			repo.URL = u
		}
		if b, ok := um["branch"].(string); ok && strings.TrimSpace(b) != "" {
			repo.Branch = stringPtr(b)
		}
		wf.UmbrellaRepo = &repo
	}
	if srs, ok := spec["supportingRepos"].([]interface{}); ok {
		wf.SupportingRepos = make([]GitRepository, 0, len(srs))
		for _, r := range srs {
			if rm, ok := r.(map[string]interface{}); ok {
				repo := GitRepository{}
				if u, ok := rm["url"].(string); ok {
					repo.URL = u
				}
				if b, ok := rm["branch"].(string); ok && strings.TrimSpace(b) != "" {
					repo.Branch = stringPtr(b)
				}
				wf.SupportingRepos = append(wf.SupportingRepos, repo)
			}
		}
	} else if repos, ok := spec["repositories"].([]interface{}); ok {
		// Backward compatibility: map legacy repositories -> umbrellaRepo (first) + supportingRepos (rest)
		for i, r := range repos {
			if rm, ok := r.(map[string]interface{}); ok {
				repo := GitRepository{}
				if u, ok := rm["url"].(string); ok {
					repo.URL = u
				}
				if b, ok := rm["branch"].(string); ok && strings.TrimSpace(b) != "" {
					repo.Branch = stringPtr(b)
				}
				if i == 0 {
					rcopy := repo
					wf.UmbrellaRepo = &rcopy
				} else {
					wf.SupportingRepos = append(wf.SupportingRepos, repo)
				}
			}
		}
	}

	// Parse jiraLinks
	if links, ok := spec["jiraLinks"].([]interface{}); ok {
		for _, it := range links {
			if m, ok := it.(map[string]interface{}); ok {
				path := fmt.Sprintf("%v", m["path"])
				jiraKey := fmt.Sprintf("%v", m["jiraKey"])
				if strings.TrimSpace(path) != "" && strings.TrimSpace(jiraKey) != "" {
					wf.JiraLinks = append(wf.JiraLinks, WorkflowJiraLink{Path: path, JiraKey: jiraKey})
				}
			}
		}
	}

	// Parse parentOutcome
	if po, ok := spec["parentOutcome"].(string); ok && strings.TrimSpace(po) != "" {
		wf.ParentOutcome = stringPtr(strings.TrimSpace(po))
	}

	return wf
}

// extractTitleFromContent attempts to extract a title from markdown content
// by looking for the first # heading
// Kept in main package for jira.go dependency
func extractTitleFromContent(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}
