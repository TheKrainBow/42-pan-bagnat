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
RUN apt-get update && apt-get install -y \
	git \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
  && rm -rf /var/lib/apt/lists/*

# Add Docker’s official GPG key
RUN mkdir -p /etc/apt/keyrings \
  && curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
    | gpg --dearmor -o /etc/apt/keyrings/docker.gpg

# Set up the Docker stable repo
RUN echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
    https://download.docker.com/linux/ubuntu \
    $(lsb_release -cs) stable" \
  > /etc/apt/sources.list.d/docker.list

# Install Docker CLI + Compose plugin
RUN apt-get update && apt-get install -y \
    docker-ce-cli \
    docker-compose-plugin \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/main .
COPY ./templates /data/templates/

EXPOSE 8080

CMD ["./main"]