# Multi-stage Dockerfile for CasDash
# Stage 1: Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application (with CGO for SQLite support)
# Set CFLAGS to handle musl libc compatibility
RUN CGO_ENABLED=1 GOOS=linux \
    CGO_CFLAGS="-D_LARGEFILE64_SOURCE" \
    go build -tags "sqlite_omit_load_extension" -o casdash main.go

# Stage 2: Asset builder (placeholder - assets embedded in Go)
FROM alpine:latest AS asset-builder

WORKDIR /app

# Create placeholder since assets are embedded in Go binary
RUN mkdir -p web/dist && echo "Assets embedded in Go binary" > web/dist/README.txt

# Stage 3: Final runtime image
FROM alpine:latest

# Install runtime dependencies (including SQLite libs for CGO)
RUN apk --no-cache add ca-certificates tzdata sqlite-libs

# Create non-root user
RUN addgroup -g 1001 -S casdash && \
    adduser -u 1001 -S casdash -G casdash

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/casdash .

# Copy migrations (required for database initialization)
COPY --from=builder /app/migrations ./migrations

# Assets are embedded in the binary, no need to copy separately

# Create data directory
RUN mkdir -p /data && chown -R casdash:casdash /data /app

# Switch to non-root user
USER casdash

# Expose port range for auto-selection
EXPOSE 64000-65535

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ./casdash --help || exit 1

# Set entrypoint
ENTRYPOINT ["./casdash"]