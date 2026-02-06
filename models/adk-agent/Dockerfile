FROM golang:1.24 AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOARCH=amd64 go build -v -o /server ./main.go
# Build the ADK UI launcher (adk-web) from custom wrapper
RUN CGO_ENABLED=0 GOARCH=amd64 go build -v -o /adk-web ./cmd/adk-web
RUN ls -lh /adk-web

# Use a minimal image with shell for debugging
FROM debian:bookworm-slim

WORKDIR /

COPY --from=builder /server /server
COPY --from=builder /adk-web /adk-web

# Ensure binaries are executable
RUN chmod +x /server /adk-web

# Install CA certificates for HTTPS
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Expose ports
EXPOSE 8080 8000

# Default command (can be overridden by k8s args)
CMD ["/server"]
