# syntax=docker/dockerfile:1

# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.26.2 AS builder

WORKDIR /app

# Download dependencies first (cache-friendly layer).
COPY go.mod go.sum ./
RUN go mod download

# Build a fully static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o classifiler .

# ── Runtime stage (scratch) ───────────────────────────────────────────────────
FROM scratch

# Copy CA certificates for TLS connections (e.g., TLS-enabled Redis).
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy the static binary.
COPY --from=builder /app/classifiler /classifiler

ENTRYPOINT ["/classifiler"]
