# WebSocket Protocol Specification

**Feature**: LangGraph Workflow Integration
**Version**: 1.0.0
**Date**: 2025-11-04

## Overview

This document defines the WebSocket protocol for real-time communication between workflow runners, the backend API, and frontend clients.

## Connection

### Endpoint

```
wss://{BACKEND_HOST}/ws?sessionName={sessionName}&token={bearerToken}
```

### Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `sessionName` | Yes | Unique identifier for the session (project-scoped or workflow session name) |
| `token` | Yes | Bearer token for authentication |

### Authentication

- Token is validated before WebSocket upgrade
- Invalid token returns HTTP 401 Unauthorized
- Token must have read access to the project containing the session

### Connection Lifecycle

1. **Client Initiates**: Frontend sends WebSocket connection request with query parameters
2. **Server Validates**: Backend authenticates token and validates session access
3. **Upgrade**: HTTP connection upgraded to WebSocket
4. **Active**: Bidirectional message exchange
5. **Close**: Either party can close the connection

### Reconnection Strategy

**Frontend Behavior:**
- On disconnect: Retry with exponential backoff
- Backoff intervals: 1s, 2s, 4s, 8s, 16s, 30s (max)
- Stop retrying when session status is terminal (`completed` or `failed`)

**Backend Behavior:**
- Maintains connection pool by session name
- Multiple clients can connect to the same session
- Messages broadcast to all connected clients for a session
- Automatic cleanup on disconnect

## Message Format

All messages are JSON objects with the following structure:

```json
{
  "type": "<message_type>",
  "timestamp": "<ISO 8601 timestamp>",
  "data": { ... }
}
```

### Message Direction

- **Agent → UI**: Workflow progress, system notifications, prompts
- **UI → Agent**: User responses to prompts

## Agent → UI Messages

### 1. System Message

Platform-generated notifications about session lifecycle.

```json
{
  "type": "system",
  "timestamp": "2025-11-04T10:00:15Z",
  "data": {
    "message": "Session started"
  }
}
```

**Use Cases:**
- Session start/stop notifications
- Job creation/completion
- Platform errors

### 2. Workflow Progress

Progress updates from running workflows.

```json
{
  "type": "workflow_progress",
  "timestamp": "2025-11-04T10:00:20Z",
  "data": {
    "step": "load_data",
    "progress": 0.25,
    "message": "Loading CSV file from storage",
    "metadata": {
      "rows_processed": 1000,
      "total_rows": 4000
    }
  }
}
```

**Fields:**
- `step` (string, required): Current workflow step identifier
- `progress` (number, optional): Progress percentage (0.0 to 1.0)
- `message` (string, required): Human-readable progress message
- `metadata` (object, optional): Additional context data

**Frequency**: Throttled to max 1 message per second per session

### 3. Workflow Waiting for Input

Workflow paused and waiting for human approval or input.

```json
{
  "type": "workflow_waiting_for_input",
  "timestamp": "2025-11-04T10:02:00Z",
  "data": {
    "prompt": "Found 3 statistical outliers. Remove them?",
    "options": ["approve", "reject"],
    "context": {
      "outliers": [
        {"date": "2024-01-15", "value": 250000},
        {"date": "2024-06-22", "value": 180000},
        {"date": "2024-12-03", "value": 320000}
      ]
    }
  }
}
```

**Fields:**
- `prompt` (string, required): Question or instruction for user
- `options` (array, optional): List of valid response options
- `context` (object, optional): Supporting data for user decision

**Backend Behavior:**
- Session status updated to `waiting_for_input`
- Checkpoint saved automatically by LangGraph

### 4. Agent Message

General messages from the workflow agent (similar to Claude Code sessions).

```json
{
  "type": "agent_message",
  "timestamp": "2025-11-04T10:04:00Z",
  "data": {
    "message": "Forecast generated successfully",
    "metadata": {
      "mae": 2500,
      "rmse": 3200
    }
  }
}
```

**Use Cases:**
- Informational messages
- Interim results
- Diagnostic information

### 5. Error Message

Error notifications from workflow execution.

```json
{
  "type": "error",
  "timestamp": "2025-11-04T10:03:30Z",
  "data": {
    "error": "Failed to load CSV file",
    "details": "HTTP 404: File not found at https://storage.example.com/sales_2024.csv",
    "recoverable": false
  }
}
```

**Fields:**
- `error` (string, required): Error message
- `details` (string, optional): Additional error context
- `recoverable` (boolean, required): Whether workflow can continue

**Backend Behavior:**
- If `recoverable=false`: Session status updated to `failed`

## UI → Agent Messages

### 1. User Input

User response to a workflow prompt.

```json
{
  "type": "user_input",
  "timestamp": "2025-11-04T10:03:30Z",
  "data": {
    "response": "approve",
    "metadata": {
      "comment": "Outliers look correct to remove"
    }
  }
}
```

**Fields:**
- `response` (any, required): User's response data (structure depends on prompt)
- `metadata` (object, optional): Additional context from user

**Backend Behavior:**
- Validates session is in `waiting_for_input` status
- Stores message in `session_messages` table
- Publishes to runner via polling or callback mechanism
- Runner resumes workflow execution

**Validation:**
- 400 Bad Request if session is not waiting for input
- 400 Bad Request if response data is invalid

### 2. User Message (General)

Generic user messages (for future interactive features).

```json
{
  "type": "user_message",
  "timestamp": "2025-11-04T10:05:00Z",
  "data": {
    "message": "Can you show more details?",
    "context": {}
  }
}
```

**Note**: Not used in MVP (workflows are not conversational like Claude Code sessions)

## Error Handling

### WebSocket Errors

**Connection Refused (HTTP 401):**
```json
{
  "error": "Unauthorized",
  "details": "Invalid or expired bearer token"
}
```

**Connection Closed by Server (1008 Policy Violation):**
- Invalid message format
- Unauthorized message type for user role
- Rate limiting exceeded

**Connection Closed by Server (1011 Internal Error):**
- Backend service error
- Database connection failure

### Message Validation Errors

Backend sends error message via WebSocket before closing:

```json
{
  "type": "error",
  "timestamp": "2025-11-04T10:05:00Z",
  "data": {
    "error": "Invalid message format",
    "details": "Missing required field 'response'"
  }
}
```

## Implementation Examples

### Frontend (TypeScript)

```typescript
function connectWorkflowSession(sessionName: string, token: string) {
  const ws = new WebSocket(
    `wss://${BACKEND_HOST}/ws?sessionName=${sessionName}&token=${token}`
  );

  ws.onopen = () => {
    console.log('WebSocket connected');
  };

  ws.onmessage = (event) => {
    const message = JSON.parse(event.data);

    switch (message.type) {
      case 'workflow_progress':
        updateProgressBar(message.data.progress);
        appendMessage(message.data.message);
        break;

      case 'workflow_waiting_for_input':
        showApprovalPrompt(message.data.prompt, message.data.options);
        break;

      case 'error':
        showError(message.data.error);
        break;
    }
  };

  ws.onerror = (error) => {
    console.error('WebSocket error:', error);
  };

  ws.onclose = (event) => {
    if (sessionStatus !== 'completed' && sessionStatus !== 'failed') {
      // Retry with exponential backoff
      setTimeout(() => connectWorkflowSession(sessionName, token), retryDelay);
    }
  };

  return ws;
}

function sendUserApproval(ws: WebSocket, response: string) {
  ws.send(JSON.stringify({
    type: 'user_input',
    timestamp: new Date().toISOString(),
    data: { response }
  }));
}
```

### Runner (Python)

```python
import asyncio
import websockets
import json
from datetime import datetime

async def send_progress(websocket, step: str, progress: float, message: str):
    """Send progress update to backend WebSocket"""
    await websocket.send(json.dumps({
        "type": "workflow_progress",
        "timestamp": datetime.utcnow().isoformat() + "Z",
        "data": {
            "step": step,
            "progress": progress,
            "message": message
        }
    }))

async def wait_for_user_input(websocket) -> dict:
    """Wait for user response to prompt"""
    # Runner sends prompt
    await websocket.send(json.dumps({
        "type": "workflow_waiting_for_input",
        "timestamp": datetime.utcnow().isoformat() + "Z",
        "data": {
            "prompt": "Approve outlier removal?",
            "options": ["approve", "reject"]
        }
    }))

    # Wait for user response
    while True:
        message_str = await websocket.recv()
        message = json.loads(message_str)

        if message["type"] == "user_input":
            return message["data"]["response"]
```

### Backend (Go)

```go
// WebSocket handler (simplified)
func handleWebSocket(c *gin.Context) {
    sessionName := c.Query("sessionName")
    token := c.Query("token")

    // Authenticate
    if !validateToken(token, sessionName) {
        c.JSON(401, gin.H{"error": "Unauthorized"})
        return
    }

    // Upgrade connection
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Printf("Upgrade error: %v", err)
        return
    }
    defer conn.Close()

    // Register connection in pool
    connectionPool.Add(sessionName, conn)
    defer connectionPool.Remove(sessionName, conn)

    // Message loop
    for {
        var msg WebSocketMessage
        err := conn.ReadJSON(&msg)
        if err != nil {
            break
        }

        // Handle user_input messages
        if msg.Type == "user_input" {
            handleUserInput(sessionName, msg.Data)
        }
    }
}

// Broadcast message to all clients of a session
func broadcastToSession(sessionName string, message WebSocketMessage) {
    connections := connectionPool.GetBySession(sessionName)
    for _, conn := range connections {
        conn.WriteJSON(message)
    }
}
```

## Security Considerations

1. **Authentication**: Bearer token required for all connections
2. **Authorization**: Token must have access to project containing session
3. **Message Validation**: All incoming messages validated before processing
4. **Rate Limiting**: Max 10 messages per second per client
5. **Payload Size**: Max 1MB per message (prevents DoS)
6. **Connection Limit**: Max 5 concurrent connections per session per user

## Performance Considerations

1. **Message Throttling**: Progress messages throttled to 1/second
2. **Connection Pooling**: Backend maintains connection pool per session
3. **Broadcast Optimization**: Single message published to all session clients
4. **Heartbeat**: Ping/Pong every 30 seconds to keep connection alive
5. **Buffering**: Limited message buffer (100 messages) for disconnected clients

## Future Enhancements (Out of MVP Scope)

- **Message History**: Fetch missed messages after reconnection
- **Compression**: WebSocket compression for large payloads
- **Binary Messages**: Support for binary data (e.g., plot images)
- **Message Acknowledgement**: Reliable delivery with ACK/NACK
- **Presence**: Online user indicators for collaborative sessions
