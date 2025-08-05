FROM golang:1.24-alpine AS builder

# Install dependencies
RUN apk add --no-cache git build-base

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o whatsapp-bridge .

# Create final lightweight image
FROM alpine:latest

# Install dependencies required for runtime
RUN apk add --no-cache ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/whatsapp-bridge /app/whatsapp-bridge

# Create directory for database
RUN mkdir -p /app/store

# Expose port 8080 for Cloud Run
EXPOSE 8080

# Set environment variable for port
ENV PORT=8080

# Set Supabase environment variables (to be overridden at runtime)
ENV SUPABASE_URL=""
ENV SUPABASE_JWT_SECRET=""

# Run the application
CMD ["/app/whatsapp-bridge"]
