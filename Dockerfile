# Build stage
FROM golang:1.24.2 AS builder

ARG TARGETARCH # Docker provides this build-time variable

WORKDIR /app

COPY go.mod go.sum ./
RUN go env -w GOPROXY=direct
RUN go mod download

COPY . .

RUN go install github.com/swaggo/swag/cmd/swag@latest

# Build the app using Makefile, selecting target based on TARGETARCH
RUN if [ "$TARGETARCH" = "arm64" ]; then \
    make release-static-arm64; \
    else \
    make release-static; \
    fi

# Rename the built binary to a consistent name for the next stage
RUN if [ "$TARGETARCH" = "arm64" ]; then \
    mv /app/bin/mma_backend-arm64 /app/bin/app_binary; \
    else \
    mv /app/bin/mma_backend /app/bin/app_binary; \
    fi

# Final image with scratch (requires static binary)
FROM scratch

ARG TARGETARCH # Make TARGETARCH available in this stage as well

WORKDIR /app

COPY --from=builder /app/bin/app_binary ./mma_backend
COPY --from=builder /app/migrations ./migrations

# Port default is 8080

EXPOSE 8080

# Default port 8080
ENV PORT=8080

# Default to production
ENV IS_PRODUCTION=true

# Entrypoint
CMD ["./mma_backend"]
