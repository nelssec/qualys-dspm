# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
ARG VERSION=dev
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -o /dspm ./cmd/dspm

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /dspm /app/dspm

# Copy migrations (for init containers)
COPY --from=builder /app/migrations /app/migrations

# Create non-root user
RUN adduser -D -g '' dspm
USER dspm

EXPOSE 8080

ENTRYPOINT ["/app/dspm"]
CMD ["-config", "/app/config.yaml"]
