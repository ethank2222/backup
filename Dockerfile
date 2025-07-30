# Build stage
FROM golang:1.21-alpine AS builder

# Install git (required for cloning repositories)
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY *.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o backup .

# Final stage
FROM alpine:latest

# Install git and coreutils (required for the backup system)
RUN apk add --no-cache git coreutils

# Create app user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/backup .

# Copy configuration files
COPY repositories.txt .
COPY README-backup.md .

# Change ownership to app user
RUN chown -R appuser:appgroup /app

# Switch to app user
USER appuser

# Expose port (if needed for health checks)
EXPOSE 8080

# Set the entrypoint
ENTRYPOINT ["./backup"] 