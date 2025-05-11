# Build stage
FROM golang:1.24.2 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go env -w GOPROXY=direct
RUN go mod download

COPY . .

RUN go install github.com/swaggo/swag/cmd/swag@latest

# Build the app using Makefile
RUN make release-static

# Final image with scratch (requires static binary)
FROM scratch

WORKDIR /app

COPY --from=builder /app/bin/mma_backend .
COPY --from=builder /app/migrations ./migrations

# Port default is 8080

EXPOSE 8080

# Default port 8080
ENV PORT=8080

# Default to production
ENV IS_PRODUCTION=true

# Entrypoint
CMD ["./mma_backend"]
