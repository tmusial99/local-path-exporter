# Stage 1: Builder
FROM golang:1.25.4-alpine AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build statically linked binary with size optimizations (-s -w)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o exporter main.go

# Stage 2: Runner
FROM scratch
COPY --from=builder /app/exporter /exporter

# Run as root to access host filesystem (required for local-path)
USER 0:0
ENTRYPOINT ["/exporter"]
