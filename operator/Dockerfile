# Dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /operator

# Final stage
FROM alpine:3.19

WORKDIR /

# Copy the binary from builder
COPY --from=builder /operator /operator

# Create non-root user
RUN adduser -D -u 1000 operator
USER operator

EXPOSE 8080
ENTRYPOINT ["/operator"]