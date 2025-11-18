// Package websocket provides real-time WebSocket communication for session updates.
package websocket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

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

// Package-level variables
var (
	Hub          *SessionWebSocketHub
	StateBaseDir string
)

// Initialize WebSocket hub
func init() {
	Hub = &SessionWebSocketHub{
		sessions:   make(map[string]map[*SessionConnection]bool),
		register:   make(chan *SessionConnection),
		unregister: make(chan *SessionConnection),
		broadcast:  make(chan *SessionMessage),
	}
	go Hub.run()
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
						// Unregister in goroutine to avoid deadlock - hub select loop
						// can only process one case at a time, so blocking send would hang
						go func(conn *SessionConnection) {
							h.unregister <- conn
						}(sessionConn)
					}
				}
			}

			// Also persist to S3
			go persistMessageToS3(message)
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

	Hub.broadcast <- message
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

	Hub.broadcast <- message
}

// Helper functions

func persistMessageToS3(message *SessionMessage) {
	// Write messages to per-project content service path as JSONL append for now
	// Backend does not have project in this scope; persist to local state dir for durability
	path := fmt.Sprintf("%s/sessions/%s/messages.jsonl", StateBaseDir, message.SessionID)
	log.Printf("persistMessageToS3: path: %s", path)
	b, _ := json.Marshal(message)
	// Ensure dir
	_ = os.MkdirAll(fmt.Sprintf("%s/sessions/%s", StateBaseDir, message.SessionID), 0o755)
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

func retrieveMessagesFromS3(sessionID string) ([]SessionMessage, error) {
	// Read from local state JSONL path for now
	path := fmt.Sprintf("%s/sessions/%s/messages.jsonl", StateBaseDir, sessionID)
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("retrieveMessagesFromS3: read failed: %v", err)
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
