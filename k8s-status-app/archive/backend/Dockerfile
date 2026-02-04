# Start from official Go image for building
FROM golang:1.24 as builder
WORKDIR /workspace



# Copy backend
COPY . .

# set working directory to backend
WORKDIR /workspace

# Build
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -v -o server .

# Use distroless for a minimal and secure runtime image
FROM gcr.io/distroless/static-debian12
COPY --from=builder /workspace/server /server
ENV PORT=8080
ENTRYPOINT ["/server"]
