# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files first to leverage Docker caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o jina-http-proxy .

# Runtime stage
FROM alpine:3.19

# Install certificates for HTTPS connections
RUN apk --no-cache add ca-certificates tzdata

# Create a non-root user to run the application
RUN adduser -D -H -h /app appuser
WORKDIR /app

# Copy the migrations directory
COPY --from=builder /app/migrations /app/migrations

# Copy the binary from the builder stage
COPY --from=builder /app/jina-http-proxy /app/

# Set ownership to the non-root user
RUN chown -R appuser:appuser /app
USER appuser

# Set environment variables
ENV GOOSE_MIGRATION_DIR=/app/migrations
# Note: GOOSE_DBSTRING should be provided at runtime

# Expose the API and proxy ports
EXPOSE 5555 5556

# Run the application
CMD ["/app/jina-http-proxy"]
