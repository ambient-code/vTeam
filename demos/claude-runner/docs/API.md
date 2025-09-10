# Claude Research Runner - API Documentation

## Overview

The Claude Research Runner provides a REST API for managing research sessions. The API allows you to create, list, retrieve, and manage ResearchSession custom resources.

## Base URL

```
http://<backend-service-url>:8080/api
```

## Authentication

Currently, the API does not require authentication. In a production environment, you should implement proper authentication and authorization.

## Endpoints

### Health Check

#### GET /health

Check if the API service is healthy.

**Response:**
```json
{
  "status": "healthy"
}
```

### Research Sessions

#### GET /research-sessions

List all research sessions in the current namespace.

**Response:**
```json
[
  {
    "apiVersion": "research.example.com/v1",
    "kind": "ResearchSession",
    "metadata": {
      "name": "research-session-1234567890",
      "namespace": "default",
      "creationTimestamp": "2024-01-15T10:30:00Z",
      "uid": "abc123-def456-ghi789"
    },
    "spec": {
      "prompt": "Analyze the homepage and identify key features",
      "websiteURL": "https://example.com",
      "llmSettings": {
        "model": "claude-3-5-sonnet-20241022",
        "temperature": 0.7,
        "maxTokens": 4000
      },
      "timeout": 300
    },
    "status": {
      "phase": "Completed",
      "message": "Research completed successfully",
      "startTime": "2024-01-15T10:30:05Z",
      "completionTime": "2024-01-15T10:32:15Z",
      "jobName": "research-session-1234567890-job",
      "finalOutput": "Based on my analysis of the website..."
    }
  }
]
```

#### GET /research-sessions/{name}

Get a specific research session by name.

**Parameters:**
- `name` (path): The name of the research session

**Response:**
```json
{
  "apiVersion": "research.example.com/v1",
  "kind": "ResearchSession",
  "metadata": {
    "name": "research-session-1234567890",
    "namespace": "default",
    "creationTimestamp": "2024-01-15T10:30:00Z",
    "uid": "abc123-def456-ghi789"
  },
  "spec": {
    "prompt": "Analyze the homepage and identify key features",
    "websiteURL": "https://example.com",
    "llmSettings": {
      "model": "claude-3-5-sonnet-20241022",
      "temperature": 0.7,
      "maxTokens": 4000
    },
    "timeout": 300
  },
  "status": {
    "phase": "Completed",
    "message": "Research completed successfully",
    "startTime": "2024-01-15T10:30:05Z",
    "completionTime": "2024-01-15T10:32:15Z",
    "jobName": "research-session-1234567890-job",
    "finalOutput": "Based on my analysis of the website..."
  }
}
```

**Error Responses:**
- `404 Not Found`: Research session not found
- `500 Internal Server Error`: Server error

#### POST /research-sessions

Create a new research session.

**Request Body:**
```json
{
  "prompt": "Analyze the homepage and identify key features",
  "websiteURL": "https://example.com",
  "llmSettings": {
    "model": "claude-3-5-sonnet-20241022",
    "temperature": 0.7,
    "maxTokens": 4000
  },
  "timeout": 300
}
```

**Request Fields:**
- `prompt` (string, required): The research prompt for Claude
- `websiteURL` (string, required): The URL of the website to analyze
- `llmSettings` (object, optional): LLM configuration
  - `model` (string): Claude model to use (default: "claude-3-5-sonnet-20241022")
  - `temperature` (number): Model temperature (default: 0.7)
  - `maxTokens` (number): Maximum tokens (default: 4000)
- `timeout` (number, optional): Timeout in seconds (default: 300)

**Response:**
```json
{
  "message": "Research session created successfully",
  "name": "research-session-1234567890",
  "uid": "abc123-def456-ghi789"
}
```

**Error Responses:**
- `400 Bad Request`: Invalid request body
- `500 Internal Server Error`: Failed to create research session

#### DELETE /research-sessions/{name}

Delete a research session.

**Parameters:**
- `name` (path): The name of the research session

**Response:**
```json
{
  "message": "Research session deleted successfully"
}
```

**Error Responses:**
- `404 Not Found`: Research session not found
- `500 Internal Server Error`: Failed to delete research session

#### POST /research-sessions/{name}/stop

Stop a running research session.

**Parameters:**
- `name` (path): The name of the research session

**Response:**
```json
{
  "message": "Research session stopped successfully"
}
```

**Error Responses:**
- `404 Not Found`: Research session not found
- `400 Bad Request`: Session is not in a stoppable state
- `500 Internal Server Error`: Failed to stop research session

#### POST /research-sessions/{name}/restart

Restart a stopped or failed research session.

**Parameters:**
- `name` (path): The name of the research session

**Response:**
```json
{
  "message": "Research session restarted successfully"
}
```

**Error Responses:**
- `404 Not Found`: Research session not found
- `400 Bad Request`: Session is not in a restartable state
- `500 Internal Server Error`: Failed to restart research session

#### PUT /research-sessions/{name}/status

Update the status of a research session. This endpoint is primarily used by the Claude runner pods to update their progress.

**Parameters:**
- `name` (path): The name of the research session

**Request Body:**
```json
{
  "phase": "Running",
  "message": "Claude is analyzing the website",
  "startTime": "2024-01-15T10:30:05Z",
  "finalOutput": "Partial results..."
}
```

**Status Fields:**
- `phase` (string): Current phase ("Pending", "Running", "Completed", "Failed")
- `message` (string): Status message
- `startTime` (string): ISO 8601 timestamp when execution started
- `completionTime` (string): ISO 8601 timestamp when execution completed
- `jobName` (string): Name of the Kubernetes job
- `finalOutput` (string): Final research output from Claude

**Response:**
```json
{
  "message": "Research session status updated successfully"
}
```

## Data Models

### ResearchSession Spec

```json
{
  "prompt": "string (required)",
  "websiteURL": "string (required, must be valid URL)",
  "llmSettings": {
    "model": "string",
    "temperature": "number (0-2)",
    "maxTokens": "number (100-8000)"
  },
  "timeout": "number (60-1800)"
}
```

### ResearchSession Status

```json
{
  "phase": "string (Pending|Running|Completed|Failed|Stopped)",
  "message": "string",
  "startTime": "string (ISO 8601)",
  "completionTime": "string (ISO 8601)",
  "jobName": "string",
  "finalOutput": "string"
}
```

**Status Phases:**
- `Pending`: Research session created but not yet started
- `Running`: Claude Code is actively analyzing the website
- `Completed`: Research session finished successfully
- `Failed`: Research session encountered an error
- `Stopped`: Research session was manually stopped

## Error Handling

All endpoints return appropriate HTTP status codes:

- `200 OK`: Successful request
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request data
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

Error responses include a JSON object with an error message:

```json
{
  "error": "Description of the error"
}
```

## Rate Limiting

Currently, no rate limiting is implemented. In production, you should implement rate limiting to prevent abuse.

## Examples

### Create a Research Session

```bash
curl -X POST http://backend-service:8080/api/research-sessions \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Analyze this website and provide insights about its user experience",
    "websiteURL": "https://example.com",
    "llmSettings": {
      "model": "claude-3-5-sonnet-20241022",
      "temperature": 0.8,
      "maxTokens": 3000
    },
    "timeout": 600
  }'
```

### List All Research Sessions

```bash
curl http://backend-service:8080/api/research-sessions
```

### Get a Specific Research Session

```bash
curl http://backend-service:8080/api/research-sessions/research-session-1234567890
```

### Stop a Research Session

```bash
curl -X POST http://backend-service:8080/api/research-sessions/research-session-1234567890/stop
```

### Restart a Research Session

```bash
curl -X POST http://backend-service:8080/api/research-sessions/research-session-1234567890/restart
```

### Update Research Session Status (Internal Use)

```bash
curl -X PUT http://backend-service:8080/api/research-sessions/research-session-1234567890/status \
  -H "Content-Type: application/json" \
  -d '{
    "phase": "Completed",
    "message": "Analysis completed",
    "completionTime": "2024-01-15T10:32:15Z",
    "finalOutput": "The website analysis reveals..."
  }'
```

## Kubernetes Custom Resource

The API manages Kubernetes custom resources of type `ResearchSession`. You can also interact with these directly using `kubectl`:

```bash
# List research sessions
kubectl get researchsessions

# Get details of a specific session
kubectl describe researchsession research-session-1234567890

# Delete a research session
kubectl delete researchsession research-session-1234567890

# Create from YAML
kubectl apply -f - <<EOF
apiVersion: research.example.com/v1
kind: ResearchSession
metadata:
  name: my-research-session
spec:
  prompt: "Analyze the website structure"
  websiteURL: "https://example.com"
  llmSettings:
    model: "claude-3-5-sonnet-20241022"
    temperature: 0.7
    maxTokens: 4000
  timeout: 300
EOF
```

## WebSocket Support (Future)

Future versions may include WebSocket support for real-time updates on research session progress.

## Authentication (Future)

Future versions will include:
- JWT-based authentication
- Role-based access control
- API key management
- Rate limiting per user/API key
