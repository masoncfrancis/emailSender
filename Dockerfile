# Stage 1: Builder
# Use a Go base image to build the application.
# golang:1.22-alpine is a good choice for smaller image size.
FROM golang:alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to download dependencies first.
# This helps leverage Docker's layer caching.
COPY go.mod .
COPY go.sum .

# Download Go modules
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
# -o app: specifies the output file name as 'app'
# -ldflags "-s -w": reduces the binary size by omitting symbol and debug info
# -tags netgo: ensures the binary does not rely on CGO for networking,
#              making it truly static and runnable on scratch.
# -installsuffix netgo: another flag for static linking.
RUN CGO_ENABLED=0 go build -o app -ldflags "-s -w" -tags netgo ./main.go

# Stage 2: Final Image (based on scratch)
# Use a minimal scratch image for the final production image.
# This image will only contain the compiled binary and necessary runtime files.
FROM scratch

# Set the working directory
WORKDIR /app

# Copy the compiled application binary from the builder stage
COPY --from=builder /app/app .

# --- IMPORTANT NOTE ON .ENV FILE ---
# For production environments, it's generally NOT recommended to bake sensitive
# .env files directly into your Docker image.
# Instead, consider using:
# 1. Kubernetes Secrets
# 2. Docker Secrets
# 3. Environment variables passed at runtime (e.g., `docker run -e SMTP_HOST=...`)
#
# For demonstration purposes, if you absolutely need the .env file inside the image,
# you would copy it from your host machine here.
# Example (uncomment if you choose this method, but be aware of security implications):
# COPY .env .

# Expose the port that your Fiber application listens on
# Ensure this matches the port defined in your Go application (default 3000)
EXPOSE 3000

# Define the entry point for the container
# This command will be executed when the container starts
CMD ["./app"]
