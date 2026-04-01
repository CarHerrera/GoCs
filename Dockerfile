# STAGE 1: Build the Go binary
FROM golang:alpine AS builder

# Install build essentials if needed
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# STAGE 2: Final Runtime Image
FROM alpine:latest

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# COPY THE VITE FILES
COPY --from=builder /app/client ./client


RUN mkdir uploads

# Expose the port your Fiber app runs on
EXPOSE 4000

# Run the binary
CMD ["./main"]