# modules-proxy/Dockerfile

FROM golang:1.25 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o modules-proxy .

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder /app/modules-proxy /modules-proxy
EXPOSE 9090
USER nonroot:nonroot
ENTRYPOINT ["/modules-proxy"]
