FROM golang:1.24.3 AS builder

# Set the Current Working Directory inside the container
WORKDIR /build

# Copy go mod and sum files separately to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o go-kanban ./cmd/server


# --------- Final minimal image ---------
FROM alpine:latest

# Create a non-root user (optional but good practice)
RUN adduser -D appuser

WORKDIR /app

# Copy built binary from builder
COPY --from=builder /build/go-kanban .

# Copy environment files if needed
COPY .env* ./

EXPOSE 8080

# Run the binary as non-root (optional)
USER appuser

CMD ["./go-kanban"]
