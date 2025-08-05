# Strigoi Security Platform - Multi-stage Docker build
# Build stage
FROM golang:1.20-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary with security flags
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -extldflags '-static'" \
    -a -installsuffix cgo \
    -o strigoi \
    ./cmd/strigoi

# Build audit tool
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o audit \
    ./security_audit/cmd/audit

# Final stage - minimal runtime
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    libpcap \
    tzdata \
    && update-ca-certificates

# Create non-root user
RUN addgroup -g 1000 strigoi && \
    adduser -u 1000 -G strigoi -s /bin/sh -D strigoi

# Copy binaries from builder
COPY --from=builder /build/strigoi /usr/local/bin/strigoi
COPY --from=builder /build/audit /usr/local/bin/audit

# Create necessary directories
RUN mkdir -p /var/log/strigoi /var/lib/strigoi /etc/strigoi && \
    chown -R strigoi:strigoi /var/log/strigoi /var/lib/strigoi /etc/strigoi

# Copy default configuration
COPY --chown=strigoi:strigoi configs/default.yaml /etc/strigoi/config.yaml

# Set security capabilities
RUN setcap cap_net_raw,cap_net_admin=+ep /usr/local/bin/strigoi

# Switch to non-root user
USER strigoi

# Set environment variables
ENV STRIGOI_CONFIG=/etc/strigoi/config.yaml \
    STRIGOI_LOG_LEVEL=info \
    STRIGOI_LOG_DIR=/var/log/strigoi

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD strigoi health || exit 1

# Expose ports
EXPOSE 8080 9090

# Volume mounts
VOLUME ["/var/log/strigoi", "/var/lib/strigoi", "/etc/strigoi"]

# Default command
ENTRYPOINT ["strigoi"]
CMD ["serve"]

# Labels
LABEL org.opencontainers.image.title="Strigoi Security Platform" \
      org.opencontainers.image.description="Advanced security monitoring and threat detection platform" \
      org.opencontainers.image.vendor="Strigoi" \
      org.opencontainers.image.version="1.0.0" \
      org.opencontainers.image.url="https://github.com/macawi-ai/strigoi" \
      org.opencontainers.image.source="https://github.com/macawi-ai/strigoi" \
      org.opencontainers.image.licenses="MIT"