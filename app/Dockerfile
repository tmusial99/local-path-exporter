# Stage 1: Builder
FROM --platform=$BUILDPLATFORM golang:1.26.4-alpine AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Target architecture arguments for cross-compilation
ARG TARGETOS
ARG TARGETARCH

# Build statically linked binary with size optimizations (-s -w)
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o exporter .

# Stage 2: Runner
FROM scratch
COPY --from=builder /app/exporter /exporter

# Run as root to stat files owned by other users on the host
USER 0:0
ENTRYPOINT ["/exporter"]
