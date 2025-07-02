#Use the official Golang image to create a build artifact.
# This is based on Debian and sets the GOPATH to /go.
# https://hub.docker.com/_/golang
FROM golang:1.24 as go-builder
# Create and change to the app directory.
WORKDIR /app
# Retrieve application dependencies using go modules.
# Allows container builds to reuse downloaded dependencies.
COPY go.* ./
RUN go mod download
# Copy local code to the container image.
COPY . ./
# Build the binary.
# -mod=readonly ensures immutable go.mod and go.sum in container builds.
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -v -o workload

# Use the official lightweight Node.js image.
# https://hub.docker.com/_/node
FROM node:20-slim as node-builder

# Create and change to the app directory.
WORKDIR /src/public

# Copy application dependency manifests to the container image.
# A wildcard is used to ensure copying both package.json AND package-lock.json (when available).
# Copying this first prevents re-running npm install on every code change.
COPY ./public/package*.json ./
RUN npm install 
COPY ./public .
RUN npm run build

# Build the runtime container image from scratch, copying what is needed from the previous stage.  
# https://docs.docker.com/develop/develop-images/multistage-build/#use-multi-stage-builds
# FROM scratch
####
# Use alpine (instead of scratch) to add ca-certificates which is needed for gRPC in some cases
####
FROM alpine:3
RUN apk add --no-cache ca-certificates
# Copy the binary to the production image from the builder stage.
COPY --from=go-builder /app/workload /workload
# Copy the Vue application from the node-builder stage.
COPY --from=node-builder /src/public/dist ./public/dist
# Run the web service on container startup.
ENTRYPOINT ["/workload"]