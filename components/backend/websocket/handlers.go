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

	for {
		select {
		case <-ticker.C:
			// Lock write mutex before writing ping
			conn.writeMu.Lock()
			err := conn.Conn.WriteMessage(websocket.PingMessage, nil)
			conn.writeMu.Unlock()
			if err != nil {
				return
			}
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
