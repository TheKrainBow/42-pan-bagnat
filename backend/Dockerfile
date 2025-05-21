# backend/Dockerfile

# 1. Use the latest Go image as a builder
FROM golang:latest AS builder

# 2. Set the working directory
WORKDIR /app

# 3. Copy go.mod and go.sum, then download dependencies
COPY srcs/go.mod srcs/go.sum ./
RUN go mod download

# 4. Copy the rest of the source code
COPY ./srcs .

# 5. Build the Go app
RUN go build -o main .

FROM ubuntu:22.04

# Install required dependencies
RUN apt-get update && apt-get install -y build-essential libc6

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]