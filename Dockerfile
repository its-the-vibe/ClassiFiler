# syntax=docker/dockerfile:1

# ── Build stage ──────────────────────────────────────────────────────────────
FROM --platform=$BUILDPLATFORM golang:1.26.6-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Download dependencies first (cache-friendly layer).
COPY go.mod go.sum ./
RUN go mod download

# Build a fully static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /classifiler .

# ── Runtime stage (distroless) ────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian13:nonroot

# Copy the static binary.
COPY --from=builder /classifiler /classifiler

USER nonroot:nonroot

ENTRYPOINT ["/classifiler"]
