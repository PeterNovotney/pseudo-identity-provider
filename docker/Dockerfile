# Stage 1: Build Angular Frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy package files
COPY frontend/package*.json ./

# Install dependencies
RUN npm install

# Copy frontend source
COPY frontend/ ./

# Build Angular application
RUN npm run build

# Stage 2: Build Go Backend
FROM golang:1.21-alpine AS backend-builder

WORKDIR /app

# Copy go mod files
COPY src/go.mod src/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY src/ ./

# Build the standalone server
RUN CGO_ENABLED=0 GOOS=linux go build -o /pseudo-idp ./standalone/standalone_main.go

# Stage 3: Final Image
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
ARG USER_ID=1000
ARG GROUP_ID=1000
RUN addgroup -g ${GROUP_ID} appuser && \
    adduser -D -u ${USER_ID} -G appuser appuser

# Create directory structure that matches the relative paths in standalone_main.go
# The binary is run from /app/src/ and expects ../src/static/browser to exist
RUN mkdir -p /app/src /app/config && \
    chown -R appuser:appuser /app

# Copy the compiled binary from backend-builder
COPY --from=backend-builder /pseudo-idp /app/src/pseudo-idp

# Copy the built frontend from frontend-builder (Angular builds to src/static)
# Path from binary perspective: ../src/static/browser (from /app/src/ -> /app/src/static/browser)
COPY --from=frontend-builder /app/src/static /app/src/static

# Copy example configuration (not in the volume mount path to avoid being overwritten)
COPY docker/config.example.json /app/config.example.json

# Copy entrypoint script
COPY docker/docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

# Set default environment variables
ENV LOG_USERNAME=admin
ENV LOG_PASSWORD=$$2a$$10$$FySeNH6jNxiGhHh/jgNP1.9PBgy3kOGgCiawrUCq1F.GYS4cosv8y
ENV PORT=8080

# Load config file by default
ENV CONFIG_FILE=/app/config/config.json

# Expose port 8080
EXPOSE 8080

# Define volume for configuration
VOLUME ["/app/config"]

# Switch to non-root user
USER appuser

# Set working directory to match where the binary expects to run from (src directory)
WORKDIR /app/src

# Use entrypoint to setup config before starting the app
ENTRYPOINT ["/docker-entrypoint.sh"]

# Run the standalone server
CMD ["/app/src/pseudo-idp"]
