# Use the official Golang image to create a build artifact.
FROM golang:1.22 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app for the current architecture
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server .

# Use a minimal base image for the final image
FROM alpine:latest

# Install necessary dependencies
RUN apk --no-cache add ca-certificates

# Copy the pre-built binary file from the builder stage
COPY --from=builder /app/server /server

# Copy the .env file from the build context
COPY .env /app/.env

# Expose port 8080 to the outside world
EXPOSE 8080

# Set the working directory
WORKDIR /app

# Command to run the executable
CMD ["sh", "-c", "source /app/.env && /server"]
