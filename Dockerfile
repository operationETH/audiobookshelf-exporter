FROM golang:1.22-alpine3.19 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /go/bin/audiobookshelf-exporter ./cmd/audiobookshelf-exporter
RUN update-ca-certificates

FROM scratch

# Copy binary and certificates
COPY --from=builder /go/bin/audiobookshelf-exporter /go/bin/audiobookshelf-exporter
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Metadata linking to GitHub repository
LABEL org.opencontainers.image.source="https://github.com/operationeth/audiobookshelf-exporter"

# Prometheus metrics port (default)
EXPOSE 9860

# Run exporter
ENTRYPOINT ["/go/bin/audiobookshelf-exporter"]
