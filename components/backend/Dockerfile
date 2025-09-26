# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git and build dependencies
RUN apk add --no-cache git build-base

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application (with flags to avoid segfault)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates git
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Default agents directory
ENV AGENTS_DIR=/app/agents

# Set executable permissions and make accessible to any user
RUN chmod +x ./main && chmod 755 /app

# Expose port
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
