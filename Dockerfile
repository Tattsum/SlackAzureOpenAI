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

# Build the Go app for multiple architectures
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /server_amd64 .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /server_arm64 .

# Use a minimal base image for the final image
FROM alpine:latest

# Install necessary dependencies
RUN apk --no-cache add ca-certificates

# Copy the pre-built binary files from the builder stage
COPY --from=builder /server_amd64 /server

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["/server"]
