package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

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

// SessionWebSocketHub manages WebSocket connections for sessions
type SessionWebSocketHub struct {
	// Map of sessionID -> SessionConnection pointers
	sessions map[string]map[*SessionConnection]bool
	// Register new connections
	register chan *SessionConnection
	// Unregister connections
	unregister chan *SessionConnection
	// Broadcast messages to session
	broadcast chan *SessionMessage
	mu        sync.RWMutex
}

// SessionConnection represents a WebSocket connection to a session
type SessionConnection struct {
	SessionID string
	Conn      *websocket.Conn
	UserID    string
	writeMu   sync.Mutex // Protects concurrent writes to Conn
}

// SessionMessage represents a message in a session
type SessionMessage struct {
	SessionID string                 `json:"sessionId"`
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
	// Partial message support
	Partial *PartialMessageInfo `json:"partial,omitempty"`
}

// PartialMessageInfo for fragmented messages
type PartialMessageInfo struct {
	ID    string `json:"id"`
	Index int    `json:"index"`
	Total int    `json:"total"`
	Data  string `json:"data"`
}

var wsHub = &SessionWebSocketHub{
	sessions:   make(map[string]map[*SessionConnection]bool),
	register:   make(chan *SessionConnection),
	unregister: make(chan *SessionConnection),
	broadcast:  make(chan *SessionMessage),
}

// Initialize WebSocket hub
func init() {
	go wsHub.run()
}

// run starts the WebSocket hub
func (h *SessionWebSocketHub) run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			if h.sessions[conn.SessionID] == nil {
				h.sessions[conn.SessionID] = make(map[*SessionConnection]bool)
			}
			h.sessions[conn.SessionID][conn] = true
			h.mu.Unlock()
			log.Printf("WebSocket connection registered for session %s", conn.SessionID)

		case conn := <-h.unregister:
			h.mu.Lock()
			if connections, exists := h.sessions[conn.SessionID]; exists {
				if _, exists := connections[conn]; exists {
					delete(connections, conn)
					conn.Conn.Close()
					if len(connections) == 0 {
						delete(h.sessions, conn.SessionID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("WebSocket connection unregistered for session %s", conn.SessionID)

		case message := <-h.broadcast:
			h.mu.RLock()
			connections := h.sessions[message.SessionID]
			h.mu.RUnlock()

			if connections != nil {
				messageData, _ := json.Marshal(message)
				for sessionConn := range connections {
					// Lock write mutex before writing
					sessionConn.writeMu.Lock()
					err := sessionConn.Conn.WriteMessage(websocket.TextMessage, messageData)
					sessionConn.writeMu.Unlock()
					if err != nil {
						h.unregister <- sessionConn
					}
				}
			}

			// Also persist to S3
			go persistMessageToS3(message)
		}
	}
}

// handleSessionWebSocket handles WebSocket connections for sessions
// Route: /projects/:projectName/sessions/:sessionId/ws
func handleSessionWebSocket(c *gin.Context) {
	log.Printf("handleSessionWebSocket: %v", c)
	log.Printf("c.Param: %v", c.Param("sessionId"))
	sessionID := c.Param("sessionId")

	// Access enforced by RBAC on downstream resources

	// Best-effort user identity: prefer forwarded user, else extract ServiceAccount from bearer token
	var userIDStr string
	log.Printf("c: %v", c)
	if v, ok := c.Get("userID"); ok {
		if s, ok2 := v.(string); ok2 {
			userIDStr = s
		}
	}
	log.Printf("userIDStr: %s", userIDStr)
	if userIDStr == "" {
		if ns, sa, ok := extractServiceAccountFromAuth(c); ok {
			userIDStr = ns + ":" + sa
		}
	}
	log.Printf("userIDStr: %s", userIDStr)

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
	wsHub.register <- sessionConn

	// Handle messages from client
	go handleWebSocketMessages(sessionConn)

	// Keep connection alive
	go handleWebSocketPing(sessionConn)
}

// handleWebSocketMessages processes incoming WebSocket messages
func handleWebSocketMessages(conn *SessionConnection) {
	defer func() {
		wsHub.unregister <- conn
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
				// Broadcast all other messages to session listeners (UI and others)
				sessionMsg := &SessionMessage{
					SessionID: conn.SessionID,
					Type:      msgType,
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					Payload:   msg,
				}
				wsHub.broadcast <- sessionMsg
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

// SendMessageToSession sends a message to all connections for a session
func SendMessageToSession(sessionID string, messageType string, payload map[string]interface{}) {
	message := &SessionMessage{
		SessionID: sessionID,
		Type:      messageType,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   payload,
	}

	wsHub.broadcast <- message
}

// SendPartialMessage sends a fragmented message to a session
func SendPartialMessage(sessionID string, partialID string, index, total int, data string) {
	message := &SessionMessage{
		SessionID: sessionID,
		Type:      "message.partial",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   map[string]interface{}{},
		Partial: &PartialMessageInfo{
			ID:    partialID,
			Index: index,
			Total: total,
			Data:  data,
		},
	}

	wsHub.broadcast <- message
}

// getSessionMessagesWS handles GET /projects/:projectName/sessions/:sessionId/messages
// Retrieves messages from S3 storage
func getSessionMessagesWS(c *gin.Context) {
	sessionID := c.Param("sessionId")

	// Access enforced by RBAC on downstream resources

	messages, err := retrieveMessagesFromS3(sessionID)
	if err != nil {
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

// Helper functions

func persistMessageToS3(message *SessionMessage) {
	// Write messages to per-project content service path as JSONL append for now
	// Backend does not have project in this scope; persist to local state dir for durability
	path := fmt.Sprintf("%s/sessions/%s/messages.jsonl", stateBaseDir, message.SessionID)
	b, _ := json.Marshal(message)
	// Ensure dir
	_ = os.MkdirAll(fmt.Sprintf("%s/sessions/%s", stateBaseDir, message.SessionID), 0o755)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Printf("persistMessage: open failed: %v", err)
		return
	}
	defer f.Close()
	if _, err := f.Write(append(b, '\n')); err != nil {
		log.Printf("persistMessage: write failed: %v", err)
	}
}

// postSessionMessageWS handles POST /projects/:projectName/sessions/:sessionId/messages
// Accepts a generic JSON body. If a "type" string is provided, it will be used.
// Otherwise, defaults to "user_message" and wraps body under payload.
func postSessionMessageWS(c *gin.Context) {
	sessionID := c.Param("sessionId")

	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
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
	wsHub.broadcast <- message

	c.JSON(http.StatusAccepted, gin.H{"status": "queued"})
}

func retrieveMessagesFromS3(sessionID string) ([]SessionMessage, error) {
	// Read from local state JSONL path for now
	path := fmt.Sprintf("%s/sessions/%s/messages.jsonl", stateBaseDir, sessionID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []SessionMessage{}, nil
		}
		return nil, err
	}
	lines := bytes.Split(data, []byte("\n"))
	msgs := make([]SessionMessage, 0, len(lines))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var m SessionMessage
		if err := json.Unmarshal(line, &m); err == nil {
			msgs = append(msgs, m)
		}
	}
	return msgs, nil
}
