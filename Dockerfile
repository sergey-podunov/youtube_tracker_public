# Stage 1: Build the application
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy modules and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Generate code (based on your Makefile's 'generate' target)
RUN go generate ./...

# Build the application for Linux
RUN CGO_ENABLED=0 GOOS=linux go build -a -o /ytracker ./cmd/app/main.go

# Stage 2: Create the final, lightweight image
FROM alpine:latest

# Set default environment variables for the running container.
# These can be overridden at runtime.
ENV APP_PORT="8081"

# It is recommended to set DB_URL at runtime for security and flexibility.
# e.g., using -e DB_URL="your_db_connection_string" in `docker run`.

# Create a non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy the compiled application from the builder stage
COPY --from=builder /ytracker /usr/local/bin/ytracker

USER appuser

# Expose the port defined by APP_PORT. The default is 8080.
EXPOSE ${APP_PORT}

# Set the entrypoint for the container
ENTRYPOINT ["/usr/local/bin/ytracker"]