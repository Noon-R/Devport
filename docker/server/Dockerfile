# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY server/go.mod server/go.sum* ./
RUN go mod download

# Copy source code
COPY server/ ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o devport .

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates and git
RUN apk --no-cache add ca-certificates git

# Copy binary from builder
COPY --from=builder /app/devport .

# Copy static files if present
COPY --from=builder /app/static ./static 2>/dev/null || true

# Create data directory
RUN mkdir -p /data/.devport

# Expose port
EXPOSE 9870

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9870/health || exit 1

# Run
ENV WORK_DIR=/data
ENV DATA_DIR=/data/.devport
CMD ["./devport"]
