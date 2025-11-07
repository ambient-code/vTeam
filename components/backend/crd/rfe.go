// Package crd provides Custom Resource Definition utilities and helpers.
package crd

import (
	"context"
	"fmt"

	"ambient-code-backend/types"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// GetRFEWorkflowResourceFunc is a function type that returns the RFEWorkflow GVR
type GetRFEWorkflowResourceFunc func() schema.GroupVersionResource

// GetRFEWorkflowResource is set by main package
var GetRFEWorkflowResource GetRFEWorkflowResourceFunc

// RFEWorkflowToCRObject converts an RFEWorkflow to a Kubernetes CR object
func RFEWorkflowToCRObject(workflow *types.RFEWorkflow) map[string]interface{} {
	// Build spec
	spec := map[string]interface{}{
		"title":         workflow.Title,
		"description":   workflow.Description,
		"branchName":    workflow.BranchName,
		"workspacePath": workflow.WorkspacePath,
	}
	if len(workflow.JiraLinks) > 0 {
		links := make([]map[string]interface{}, 0, len(workflow.JiraLinks))
		for _, l := range workflow.JiraLinks {
			links = append(links, map[string]interface{}{"path": l.Path, "jiraKey": l.JiraKey})
		}
		spec["jiraLinks"] = links
	}
	if workflow.ParentOutcome != nil && *workflow.ParentOutcome != "" {
		spec["parentOutcome"] = *workflow.ParentOutcome
	}

	// Prefer umbrellaRepo/supportingRepos; fallback to legacy repositories array
	if workflow.UmbrellaRepo != nil {
		u := map[string]interface{}{"url": workflow.UmbrellaRepo.URL}
		if workflow.UmbrellaRepo.Branch != nil {
			u["branch"] = *workflow.UmbrellaRepo.Branch
		}
		spec["umbrellaRepo"] = u
	}
	if len(workflow.SupportingRepos) > 0 {
		items := make([]map[string]interface{}, 0, len(workflow.SupportingRepos))
		for _, r := range workflow.SupportingRepos {
			rm := map[string]interface{}{"url": r.URL}
			if r.Branch != nil {
				rm["branch"] = *r.Branch
			}
			items = append(items, rm)
		}
		spec["supportingRepos"] = items
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

// UpsertProjectRFEWorkflowCR creates or updates an RFEWorkflow custom resource
func UpsertProjectRFEWorkflowCR(dyn dynamic.Interface, workflow *types.RFEWorkflow) error {
	if workflow.Project == "" {
		// Only manage CRD for project-scoped workflows
		return nil
	}
	if dyn == nil {
		return fmt.Errorf("no dynamic client provided")
	}
	gvr := GetRFEWorkflowResource()
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
