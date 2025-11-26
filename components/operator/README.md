# Agentic Operator

Kubernetes operator watching Custom Resources and managing AgenticSession Job lifecycle.

## Features

- Watches AgenticSession CRs and spawns Jobs with runner pods
- Updates CR status based on Job completion
- Handles timeout and cleanup
- Reconnects watch on channel close
- Idempotent reconciliation

## Development

### Prerequisites

- Go 1.21+
- kubectl
- Kubernetes cluster access
- CRDs installed in cluster

### Quick Start

```bash
cd components/operator

# Build
go build -o operator .

# Run locally (requires k8s access and CRDs installed)
go run .
```

### Build

```bash
# Build binary
go build -o operator .

# Build container image
docker build -t operator .
# or
podman build -t operator .
```

### Testing

```bash
# Run tests
go test ./... -v

# Run tests with coverage
go test ./... -v -cover
```

### Linting

```bash
# Format code
gofmt -l .

# Run go vet
go vet ./...

# Run golangci-lint
golangci-lint run
```

**Pre-commit checklist**:
```bash
# Run all linting checks
gofmt -l .             # Should output nothing
go vet ./...
golangci-lint run

# Auto-format code
gofmt -w .
```

## Architecture

### Package Structure

```
operator/
├── internal/
│   ├── config/        # K8s client init, config loading
│   ├── types/         # GVR definitions, resource helpers
│   ├── handlers/      # Watch handlers (sessions, namespaces, projectsettings)
│   └── services/      # Reusable services (PVC provisioning, etc.)
└── main.go            # Watch coordination
```

### Key Patterns

See `AGENTS.md` in project root for:
- Watch loop with reconnection
- Reconciliation pattern
- Status updates (UpdateStatus subresource)
- Goroutine monitoring
- Error handling

## Reference Files

- `internal/handlers/sessions.go` - Watch loop, reconciliation, status updates
- `internal/config/config.go` - K8s client initialization
- `internal/types/resources.go` - GVR definitions
- `internal/services/infrastructure.go` - Reusable services
