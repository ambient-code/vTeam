package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"ambient-code-backend/server"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// IngestRunEvent receives events from LangGraph runner pods
func IngestRunEvent(c *gin.Context) {
	project := c.Param("projectName")
	runID := c.Param("runId")

	var event struct {
		RunID       string                 `json:"run_id"`
		Seq         int                    `json:"seq"`
		Ts          string                 `json:"ts"`
		Type        string                 `json:"type"`
		Node        *string                `json:"node"`
		CheckpointID *string               `json:"checkpoint_id"`
		Payload     map[string]interface{} `json:"payload"`
	}

	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate run_id matches
	if event.RunID != runID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "run_id mismatch"})
		return
	}

	// Insert event
	payloadJSON, _ := json.Marshal(event.Payload)
	var checkpointID *string
	if event.CheckpointID != nil {
		checkpointID = event.CheckpointID
	}

	_, err := server.DB.Exec(
		"INSERT INTO run_events (run_id, seq, ts, kind, checkpoint_id, payload) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (run_id, seq) DO NOTHING",
		runID, event.Seq, event.Ts, event.Type, checkpointID, payloadJSON,
	)
	if err != nil {
		log.Printf("Failed to insert run event: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store event"})
		return
	}

	// Update AgenticSession status based on event type
	if event.Type == "node_start" || event.Type == "node_update" {
		// Update currentNode in status
		if event.Node != nil {
			updateSessionStatusFromEvent(project, runID, map[string]interface{}{
				"currentNode": *event.Node,
			})
		}
	} else if event.Type == "interrupt" {
		// Add condition for awaiting approval
		if event.CheckpointID != nil {
			updateSessionStatusFromEvent(project, runID, map[string]interface{}{
				"currentNode":  event.Node,
				"checkpointId": *event.CheckpointID,
				"conditions": []map[string]interface{}{
					{
						"type":               "AwaitingApproval",
						"status":             "True",
						"message":            fmt.Sprintf("Waiting for approval at node %s", *event.Node),
						"lastTransitionTime": time.Now().Format(time.RFC3339),
					},
				},
			})
		}
	} else if event.Type == "node_end" {
		// Clear conditions
		updateSessionStatusFromEvent(project, runID, map[string]interface{}{
			"conditions": []map[string]interface{}{},
		})
	} else if event.Type == "error" {
		updateSessionStatusFromEvent(project, runID, map[string]interface{}{
			"phase":   "Error",
			"message": fmt.Sprintf("Workflow error: %v", event.Payload),
		})
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetRunEvents retrieves events for a run
func GetRunEvents(c *gin.Context) {
	_ = c.Param("projectName") // project name from path, not used but kept for API consistency
	runID := c.Param("runId")

	rows, err := server.DB.Query(
		"SELECT seq, ts, kind, checkpoint_id, payload FROM run_events WHERE run_id = $1 ORDER BY seq ASC",
		runID,
	)
	if err != nil {
		log.Printf("Failed to query run events: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get run events"})
		return
	}
	defer rows.Close()

	events := []map[string]interface{}{}
	for rows.Next() {
		var seq int
		var ts time.Time
		var kind string
		var checkpointID sql.NullString
		var payloadJSON []byte

		if err := rows.Scan(&seq, &ts, &kind, &checkpointID, &payloadJSON); err != nil {
			log.Printf("Error scanning event: %v", err)
			continue
		}

		var payload map[string]interface{}
		if len(payloadJSON) > 0 {
			json.Unmarshal(payloadJSON, &payload)
		}

		event := map[string]interface{}{
			"seq":   seq,
			"ts":    ts.Format(time.RFC3339),
			"type":  kind,
			"payload": payload,
		}
		if checkpointID.Valid {
			event["checkpoint_id"] = checkpointID.String
		}

		events = append(events, event)
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

// ApproveRun approves an interrupted workflow run
func ApproveRun(c *gin.Context) {
	project := c.Param("projectName")
	runID := c.Param("runId")

	var req struct {
		Node     string                 `json:"node" binding:"required"`
		Decision map[string]interface{} `json:"decision" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current session to find checkpoint_id
	gvr := GetAgenticSessionV1Alpha1Resource()
	session, err := DynamicClient.Resource(gvr).Namespace(project).Get(c.Request.Context(), runID, v1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	status, _, _ := unstructured.NestedMap(session.Object, "status")
	checkpointID, _, _ := unstructured.NestedString(status, "checkpointId")

	if checkpointID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No checkpoint ID found"})
		return
	}

	// Get runner service URL
	runnerSvcName := fmt.Sprintf("langgraph-runner-%s", runID)
	runnerURL := fmt.Sprintf("http://%s.%s.svc.cluster.local:8000", runnerSvcName, project)

	// Call /resume endpoint
	resumeReq := map[string]interface{}{
		"checkpoint_id": checkpointID,
		"values":        req.Decision,
	}
	reqJSON, _ := json.Marshal(resumeReq)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(fmt.Sprintf("%s/resume", runnerURL), "application/json", strings.NewReader(string(reqJSON)))
	if err != nil {
		log.Printf("Failed to call /resume: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to resume workflow: %v", err)})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Resume failed with status %d", resp.StatusCode)})
		return
	}

	// Update session status
	updateSessionStatusFromEvent(project, runID, map[string]interface{}{
		"conditions": []map[string]interface{}{
			{
				"type":               "AwaitingApproval",
				"status":             "False",
				"lastTransitionTime": time.Now().Format(time.RFC3339),
			},
		},
	})

	c.JSON(http.StatusOK, gin.H{"status": "approved"})
}

// updateSessionStatusFromEvent is a helper to update AgenticSession status
func updateSessionStatusFromEvent(project, runID string, updates map[string]interface{}) {
	gvr := GetAgenticSessionV1Alpha1Resource()
	session, err := DynamicClient.Resource(gvr).Namespace(project).Get(context.TODO(), runID, v1.GetOptions{})
	if err != nil {
		log.Printf("Failed to get session for status update: %v", err)
		return
	}

	status, _, _ := unstructured.NestedMap(session.Object, "status")
	if status == nil {
		status = make(map[string]interface{})
	}

	for k, v := range updates {
		status[k] = v
	}

	session.Object["status"] = status
	_, err = DynamicClient.Resource(gvr).Namespace(project).UpdateStatus(context.TODO(), session, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update session status: %v", err)
	}
}

