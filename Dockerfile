# Multi-stage build for kv-stash
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
	-ldflags='-w -s -extldflags "-static"' \
	-o kvstash \
	./cmd/kvstash

# Final stage
FROM alpine:latest

# Install ca-certificates for TLS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S kvstash && \
	adduser -u 1001 -S kvstash -G kvstash

# Create data directory
RUN mkdir -p /data && chown kvstash:kvstash /data

# Copy binary from builder stage
COPY --from=builder /app/kvstash /usr/local/bin/kvstash

# Switch to non-root user
USER kvstash

# Expose ports
EXPOSE 6380 9100

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
	CMD nc -z localhost 6380 || exit 1

# Default command
CMD ["kvstash"]
