# Build stage for downloading dependencies
FROM --platform=$BUILDPLATFORM golang:1.24.2 AS deps
WORKDIR /app

# Copy only the dependency files first for better caching
COPY go.mod go.sum ./

# Set Go environment variables
ENV CGO_ENABLED=0 \
    GO111MODULE=on \
    GOPROXY=direct

# Download dependencies
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Build stage
FROM deps AS builder

# Install swag
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Copy the rest of the application
COPY . .

# Build arguments
ARG TARGETOS
ARG TARGETARCH

# Install make and other build essentials
RUN apt-get update && apt-get install -y make

# Set the default shell to bash
SHELL ["/bin/bash", "-c"]

# Build the application
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    make swagger && \
    GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -a -installsuffix cgo -o /app/bin/app_binary ./cmd/mma_backend

# Final image with distroless (smaller than alpine, more secure than scratch)
FROM gcr.io/distroless/static-debian12:latest-${TARGETARCH:-amd64}

# Set working directory
WORKDIR /app

# Copy the binary and migrations from the builder stage
COPY --from=builder --chown=nonroot:nonroot /app/bin/app_binary /app/mma_backend
COPY --from=builder --chown=nonroot:nonroot /app/migrations /app/migrations

# Use non-root user for security
USER nonroot:nonroot

# Port default is 8080

EXPOSE 8080

# Default port 8080
ENV PORT=8080

# Default to production
ENV IS_PRODUCTION=true

# Entrypoint
CMD ["./mma_backend"]
