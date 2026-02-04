# Start from official Go image for building
FROM golang:1.24 as builder
WORKDIR /workspace

# Copy shared utils
COPY go-mslarkin-utils ./go-mslarkin-utils

# Copy backend
COPY k8s-status-app/backend ./k8s-status-app/backend

# set working directory to backend
WORKDIR /workspace/k8s-status-app/backend

# Build
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -v -o server .

# Use distroless for a minimal and secure runtime image
FROM gcr.io/distroless/static-debian12
COPY --from=builder /workspace/k8s-status-app/backend/server /server
ENV PORT=8080
ENTRYPOINT ["/server"]
