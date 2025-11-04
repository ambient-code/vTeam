# Build stage
FROM registry.access.redhat.com/ubi9/go-toolset:1.24 AS builder

WORKDIR /app

USER 0

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies (with retry and DNS workaround for cross-platform builds)
RUN GOPROXY=direct GOSUMDB=off go mod download || \
    (sleep 2 && GOPROXY=direct GOSUMDB=off go mod download) || \
    (sleep 5 && GOPROXY=direct GOSUMDB=off go mod download)

# Copy the source code
COPY . .

# Build the application for linux/amd64 (required for OpenShift/K8s)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o main .

# Final stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

RUN microdnf install -y git && microdnf clean all
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Default agents directory
ENV AGENTS_DIR=/app/agents

# Set executable permissions and make accessible to any user
RUN chmod +x ./main && chmod 775 /app

USER 1001

# Expose port
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
