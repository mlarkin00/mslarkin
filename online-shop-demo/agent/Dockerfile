# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -v -o agent main.go

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies and kubectl
RUN apk add --no-cache ca-certificates bash curl \
    && curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" \
    && chmod +x kubectl \
    && mv kubectl /usr/local/bin/

# Copy binary from builder
COPY --from=builder /app/agent /app/agent

# Copy failure modes directory (needed for agent to scan)
COPY --from=builder /app/failure-modes /app/failure-modes
# Copy baseline directory (needed for revert)
COPY --from=builder /app/baseline /app/baseline

# Environment variables
ENV PORT=8080

EXPOSE 8080

CMD ["/app/agent"]
