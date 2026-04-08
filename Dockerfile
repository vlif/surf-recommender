FROM golang:1.24.11-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

# Download dependencies
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o surf-recommender ./cmd/surf-recommender

# Use a small alpine image for the final container
FROM alpine:3.17

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/surf-recommender .

# Create a non-root user and switch to it
RUN adduser -D -h /app appuser
USER appuser

# Command to run when the container starts
ENTRYPOINT ["/app/surf-recommender"]
