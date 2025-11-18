// Package jira provides JIRA integration (currently disabled - was RFE-specific).
// Kept for potential future use.
package jira

/*
// This package was RFE-specific and has been commented out.
// Uncomment and refactor when adding Jira support for sessions or other features.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ambient-code-backend/git"
	"ambient-code-backend/handlers"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Handler dependencies
type Handler struct {
	GetK8sClientsForRequest    func(*gin.Context) (*kubernetes.Clientset, dynamic.Interface)
	GetProjectSettingsResource func() schema.GroupVersionResource
	GetRFEWorkflowResource     func() schema.GroupVersionResource
}

// Commented out RFE-specific functions
// Add Jira integration functions here when ready for session-based Jira support
*/
