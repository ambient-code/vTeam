package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"ambient-code-backend/handlers"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development - should be restricted in production
		return true
	},
}

// HandleSessionWebSocket handles WebSocket connections for sessions
// Route: /projects/:projectName/sessions/:sessionId/ws
func HandleSessionWebSocket(c *gin.Context) {
	sessionID := c.Param("sessionId")
	log.Printf("handleSessionWebSocket for session: %s", sessionID)

	// Access enforced by RBAC on downstream resources

	// Best-effort user identity: prefer forwarded user, else extract ServiceAccount from bearer token
	var userIDStr string
	if v, ok := c.Get("userID"); ok {
		if s, ok2 := v.(string); ok2 {
			userIDStr = s
		}
	}
	if userIDStr == "" {
		if ns, sa, ok := handlers.ExtractServiceAccountFromAuth(c); ok {
			userIDStr = ns + ":" + sa
		}
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	sessionConn := &SessionConnection{
		SessionID: sessionID,
		Conn:      conn,
		UserID:    userIDStr,
	}

	// Register connection
	Hub.register <- sessionConn

	// Handle messages from client
	go handleWebSocketMessages(sessionConn)

	// Keep connection alive
	go handleWebSocketPing(sessionConn)
}

// handleWebSocketMessages processes incoming WebSocket messages
func handleWebSocketMessages(conn *SessionConnection) {
	defer func() {
		Hub.unregister <- conn
	}()

	for {
		messageType, messageData, err := conn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		if messageType == websocket.TextMessage {
			var msg map[string]interface{}
			if err := json.Unmarshal(messageData, &msg); err != nil {
				log.Printf("Failed to parse WebSocket message: %v", err)
				continue
			}

			// Handle control messages
			if msgType, ok := msg["type"].(string); ok {
				if msgType == "ping" {
					// Respond with pong
					pong := map[string]interface{}{
						"type":      "pong",
						"timestamp": time.Now().UTC().Format(time.RFC3339),
					}
					pongData, _ := json.Marshal(pong)
					// Lock write mutex before writing pong
					conn.writeMu.Lock()
					_ = conn.Conn.WriteMessage(websocket.TextMessage, pongData)
					conn.writeMu.Unlock()
					continue
				}
				// Extract payload from runner message to avoid double-nesting
				// Runner sends: {type, seq, timestamp, payload}
				// We only want to store the payload field
				payload, ok := msg["payload"].(map[string]interface{})
				if !ok {
					payload = msg // Fallback for legacy format
				}
				// Broadcast all other messages to session listeners (UI and others)
				sessionMsg := &SessionMessage{
					SessionID: conn.SessionID,
					Type:      msgType,
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					Payload:   payload,
				}
				Hub.broadcast <- sessionMsg
			}
		}
	}
}

// handleWebSocketPing sends periodic ping messages
func handleWebSocketPing(conn *SessionConnection) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Lock write mutex before writing ping
		conn.writeMu.Lock()
		err := conn.Conn.WriteMessage(websocket.PingMessage, nil)
		conn.writeMu.Unlock()
		if err != nil {
			return
		}
	}
}

// GetSessionMessagesWS handles GET /projects/:projectName/sessions/:sessionId/messages
// Retrieves messages from S3 storage
func GetSessionMessagesWS(c *gin.Context) {
	sessionID := c.Param("sessionId")

	// Access enforced by RBAC on downstream resources

	messages, err := retrieveMessagesFromS3(sessionID)
	if err != nil {
		log.Printf("getSessionMessagesWS: retrieve failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to retrieve messages: %v", err),
		})
		return
	}

	// Optional consolidation of partial messages
	includeParam := strings.ToLower(strings.TrimSpace(c.Query("include_partial_messages")))
	includePartials := includeParam == "1" || includeParam == "true" || includeParam == "yes"

	collapsed := make([]SessionMessage, 0, len(messages))
	activePartialIndex := -1
	for _, m := range messages {
		if m.Type == "message.partial" {
			if includePartials {
				if activePartialIndex >= 0 {
					collapsed[activePartialIndex] = m
				} else {
					collapsed = append(collapsed, m)
					activePartialIndex = len(collapsed) - 1
				}
			}
			// If not including partials, simply skip adding them
			continue
		}
		// On any non-partial, clear active partial placeholder
		activePartialIndex = -1
		collapsed = append(collapsed, m)
	}

	c.JSON(http.StatusOK, gin.H{
		"sessionId": sessionID,
		"messages":  collapsed,
	})
}

// PostSessionMessageWS handles POST /projects/:projectName/sessions/:sessionId/messages
// Accepts a generic JSON body. If a "type" string is provided, it will be used.
// Otherwise, defaults to "user_message" and wraps body under payload.
func PostSessionMessageWS(c *gin.Context) {
	sessionID := c.Param("sessionId")

	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		log.Printf("postSessionMessageWS: bind failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}

	msgType := "user_message"
	if v, ok := body["type"].(string); ok && v != "" {
		msgType = v
		// Remove type from payload to avoid duplication
		delete(body, "type")
	}

	message := &SessionMessage{
		SessionID: sessionID,
		Type:      msgType,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   body,
	}

	// Broadcast to session listeners (runner) and persist
	Hub.broadcast <- message

	c.JSON(http.StatusAccepted, gin.H{"status": "queued"})
}

// GetSessionMessagesClaudeFormat handles GET /projects/:projectName/sessions/:sessionId/messages/claude-format
// Transforms stored messages into Claude SDK format for session continuation
func GetSessionMessagesClaudeFormat(c *gin.Context) {
	sessionID := c.Param("sessionId")

	messages, err := retrieveMessagesFromS3(sessionID)
	if err != nil {
		log.Printf("GetSessionMessagesClaudeFormat: retrieve failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to retrieve messages: %v", err),
		})
		return
	}

	log.Printf("GetSessionMessagesClaudeFormat: retrieved %d messages for session %s", len(messages), sessionID)

	// Filter to only conversational messages (user and agent)
	// Exclude: system.message, agent.waiting, agent.running, etc.
	conversationalMessages := []SessionMessage{}
	for _, msg := range messages {
		msgType := strings.ToLower(strings.TrimSpace(msg.Type))
		// Normalize dots to underscores for comparison (stored as "agent.message" but we check "agent_message")
		normalizedType := strings.ReplaceAll(msgType, ".", "_")

		// Only include actual conversation messages
		if normalizedType == "user_message" || normalizedType == "agent_message" {
			// Additional validation - ensure payload is not empty
			if len(msg.Payload) == 0 {
				log.Printf("GetSessionMessagesClaudeFormat: filtering out %s with empty payload", msg.Type)
				continue
			}
			conversationalMessages = append(conversationalMessages, msg)
			log.Printf("GetSessionMessagesClaudeFormat: keeping message type=%s", msg.Type)
		} else {
			log.Printf("GetSessionMessagesClaudeFormat: filtering out non-conversational message type=%s", msg.Type)
		}
	}

	log.Printf("GetSessionMessagesClaudeFormat: filtered to %d conversational messages", len(conversationalMessages))

	claudeMessages := transformToClaudeFormat(conversationalMessages)

	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"messages":   claudeMessages,
	})
}

// transformToClaudeFormat converts SessionMessage to Claude SDK control protocol format
// For session continuation via connect(), the CLI validates BOTH:
//  1. Envelope "type" must be "user" or "control"
//  2. Message "role" must be "user" (NOT "assistant")
//
// Attempting to send assistant messages fails with:
//
//	"Expected message role 'user', got 'assistant'"
//
// Required format (from client.py lines 184-190):
//
//	{
//	  "type": "user",
//	  "message": {"role": "user", "content": "..."},
//	  "parent_tool_use_id": "..."  // optional
//	}
//
// The CLI will replay the conversation from user inputs and regenerate all assistant responses
func transformToClaudeFormat(messages []SessionMessage) []map[string]interface{} {
	result := []map[string]interface{}{}

	for _, msg := range messages {
		log.Printf("transformToClaudeFormat: processing message type=%s", msg.Type)

		// Normalize message type (stored as "agent.message" but we check "agent_message")
		msgType := strings.ToLower(strings.TrimSpace(msg.Type))
		normalizedType := strings.ReplaceAll(msgType, ".", "_")

		switch normalizedType {
		case "user_message":
			// Extract user content - can be text or tool_result blocks
			content := extractUserMessageContent(msg.Payload)
			if content != nil {
				// Extract parent_tool_use_id if present (for tool result chaining)
				parentToolUseID := extractParentToolUseID(msg.Payload)

				// SDK control protocol format: wraps API messages in envelope
				message := map[string]interface{}{
					"type": "user",
					"message": map[string]interface{}{
						"role":    "user",
						"content": content,
					},
				}
				// Only include parent_tool_use_id if it exists
				if parentToolUseID != "" {
					message["parent_tool_use_id"] = parentToolUseID
				}

				result = append(result, message)
				log.Printf("transformToClaudeFormat: added user message (parent_tool_use_id=%s)", parentToolUseID)
			} else {
				log.Printf("transformToClaudeFormat: skipping user_message with empty content")
			}

		case "agent_message":
			// Extract assistant content - can be text blocks, tool_use blocks, thinking blocks, etc.
			content := extractAssistantMessageContent(msg.Payload)
			if content != nil {
				// Extract model and parent_tool_use_id
				parentToolUseID := extractParentToolUseID(msg.Payload)

				// SDK control protocol: BOTH envelope type AND message role must be "user"
				// CLI validates both levels and rejects role="assistant"
				message := map[string]interface{}{
					"type": "user", // Control protocol envelope
					"message": map[string]interface{}{
						"role":    "user", // Must be "user" even for assistant messages!
						"content": content,
					},
				}
				// Only include parent_tool_use_id if it exists
				if parentToolUseID != "" {
					message["parent_tool_use_id"] = parentToolUseID
				}

				result = append(result, message)
				log.Printf("transformToClaudeFormat: added assistant message as envelope type=user (parent_tool_use_id=%s)", parentToolUseID)
			} else {
				log.Printf("transformToClaudeFormat: skipping agent_message with no extractable content")
			}

		default:
			log.Printf("transformToClaudeFormat: skipping message with unknown type=%s (normalized=%s)", msg.Type, normalizedType)
		}
	}

	log.Printf("transformToClaudeFormat: result: %+v", result)

	// Validate all messages have proper structure
	// Envelope "type" must be "user" or "control" (control protocol level)
	// Inner "message.role" can be "user" or "assistant" (conversation level)
	validated := []map[string]interface{}{}
	for i, msg := range result {
		envType, hasType := msg["type"].(string)
		if !hasType || (envType != "user" && envType != "control") {
			log.Printf("transformToClaudeFormat: INVALID message at index %d - envelope type must be 'user' or 'control', got: %v", i, envType)
			continue
		}

		// Validate message envelope structure
		if msg["message"] == nil {
			log.Printf("transformToClaudeFormat: INVALID message at index %d - missing 'message' envelope", i)
			continue
		}

		validated = append(validated, msg)
	}

	log.Printf("transformToClaudeFormat: returning %d messages (envelope type='user', conversation includes user+assistant)", len(validated))
	return validated
}

// extractUserMessageContent extracts content from user message payload
// Returns content as string (simple text) or []interface{} (content blocks with tool_result)
func extractUserMessageContent(payload map[string]interface{}) interface{} {
	// Check if payload already has properly formatted content
	if content, ok := payload["content"]; ok {
		// Content is already in correct format (string or array of blocks)
		switch v := content.(type) {
		case string:
			if v != "" {
				return v
			}
		case []interface{}:
			if len(v) > 0 {
				return v
			}
		}
	}

	// Check for tool_result block - must be in array format
	if toolResult := extractToolResult(payload); toolResult != nil {
		return []interface{}{toolResult}
	}

	// Try to extract simple text content
	if text, ok := payload["text"].(string); ok && text != "" {
		return text
	}

	// Check for text_block format from runner
	if msgType, ok := payload["type"].(string); ok && msgType == "text_block" {
		if text, ok := payload["text"].(string); ok && text != "" {
			return text
		}
	}

	return nil
}

// extractAssistantMessageContent extracts content from assistant message payload
// Returns content as array of content blocks (text, thinking, tool_use, etc.)
// Assistant messages must have content as an array, not a simple string
// Supports SDK types: TextBlock, ThinkingBlock, ToolUseBlock
func extractAssistantMessageContent(payload map[string]interface{}) interface{} {
	// Check if payload already has properly formatted content array
	if content, ok := payload["content"].([]interface{}); ok && len(content) > 0 {
		return content
	}

	// Build content blocks array from various payload formats
	var contentBlocks []map[string]interface{}

	// Check for thinking block (extended thinking)
	if thinking, signature := extractThinkingBlock(payload); thinking != "" {
		block := map[string]interface{}{
			"type":     "thinking",
			"thinking": thinking,
		}
		if signature != "" {
			block["signature"] = signature
		}
		contentBlocks = append(contentBlocks, block)
	}

	// Check for text block
	if text := extractTextBlock(payload); text != "" {
		contentBlocks = append(contentBlocks, map[string]interface{}{
			"type": "text",
			"text": text,
		})
	}

	// Check for tool_use block
	if tool, input, id := extractToolUse(payload); tool != "" {
		block := map[string]interface{}{
			"type":  "tool_use",
			"name":  tool,
			"input": input,
		}
		if id != "" {
			block["id"] = id
		}
		contentBlocks = append(contentBlocks, block)
	}

	if len(contentBlocks) > 0 {
		// Convert to []interface{} for JSON marshaling
		result := make([]interface{}, len(contentBlocks))
		for i, block := range contentBlocks {
			result[i] = block
		}
		return result
	}

	return nil
}

func extractModel(payload map[string]interface{}) string {
	// Check for model at top level
	if model, ok := payload["model"].(string); ok && model != "" {
		return model
	}

	// Default model if not specified
	return "claude-3-7-sonnet-latest"
}

func extractParentToolUseID(payload map[string]interface{}) string {
	// Check for parent_tool_use_id at top level
	if parentID, ok := payload["parent_tool_use_id"].(string); ok && parentID != "" {
		return parentID
	}

	// Check if this is a tool_result and extract the tool_use_id as parent
	if toolResult, ok := payload["tool_result"].(map[string]interface{}); ok {
		if toolUseID, ok := toolResult["tool_use_id"].(string); ok {
			return toolUseID
		}
	}

	return ""
}

func extractThinkingBlock(payload map[string]interface{}) (thinking string, signature string) {
	// Check if this is a thinking block
	if msgType, ok := payload["type"].(string); ok && msgType == "thinking" {
		thinking, _ = payload["thinking"].(string)
		signature, _ = payload["signature"].(string)
		return thinking, signature
	}

	// Check nested content for thinking block
	if content, ok := payload["content"].(map[string]interface{}); ok {
		if thinking, ok := content["thinking"].(string); ok {
			signature, _ := content["signature"].(string)
			return thinking, signature
		}
	}

	return "", ""
}

func extractTextBlock(payload map[string]interface{}) string {
	if content, ok := payload["content"].(map[string]interface{}); ok {
		if text, ok := content["text"].(string); ok {
			return text
		}
	}
	if text, ok := payload["text"].(string); ok {
		return text
	}
	if msgType, ok := payload["type"].(string); ok && msgType == "text_block" {
		if text, ok := payload["text"].(string); ok {
			return text
		}
	}
	return ""
}

func extractToolUse(payload map[string]interface{}) (tool string, input map[string]interface{}, id string) {
	toolName, hasTool := payload["tool"].(string)
	toolInput, hasInput := payload["input"].(map[string]interface{})
	toolID, _ := payload["id"].(string)

	if hasTool && hasInput {
		return toolName, toolInput, toolID
	}
	return "", nil, ""
}

func extractToolResult(payload map[string]interface{}) map[string]interface{} {
	if toolResult, ok := payload["tool_result"].(map[string]interface{}); ok {
		result := map[string]interface{}{
			"type": "tool_result",
		}
		if toolUseID, ok := toolResult["tool_use_id"].(string); ok {
			result["tool_use_id"] = toolUseID
		}
		if content := toolResult["content"]; content != nil {
			result["content"] = content
		}
		if isError, ok := toolResult["is_error"].(bool); ok {
			result["is_error"] = isError
		}
		return result
	}
	return nil
}
