# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
# We copy to a subdirectory to match the module structure if needed, or just to keep it clean.
COPY agent/go.mod agent/go.sum ./agent/
WORKDIR /app/agent
RUN go mod download

# Copy source code (entire repo)
WORKDIR /app
COPY . .

# Build the binary
WORKDIR /app/agent
RUN CGO_ENABLED=0 GOOS=linux go build -v -o agent main.go

# Final stage
FROM google/cloud-sdk:alpine

WORKDIR /app

# Install dependencies
# google/cloud-sdk:alpine has gcloud. We need kubectl.
# Try installing via gcloud components or manual. Manual is often faster/reliable on alpine.
RUN apk add --no-cache ca-certificates bash curl \
    && curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" \
    && chmod +x kubectl \
    && mv kubectl /usr/local/bin/ \
    && gcloud components install gke-gcloud-auth-plugin --quiet

# Copy binary from builder
COPY --from=builder /app/agent/agent /app/agent

# Copy failure modes directory (needed for agent to scan)
COPY --from=builder /app/failure-modes /app/failure-modes
# Copy baseline directory (needed for revert)
COPY --from=builder /app/baseline /app/baseline
# Copy UI directory (needed for A2UI client)
COPY --from=builder /app/agent/ui /app/ui

# Environment variables
ENV PORT=8080
ENV APP_ROOT=/app

EXPOSE 8080

CMD ["/app/agent"]
