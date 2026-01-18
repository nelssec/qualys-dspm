# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /dspm ./cmd/server

# Frontend build stage
FROM node:20-alpine AS frontend

WORKDIR /app

COPY web/package*.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

# Final stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /dspm /app/dspm
COPY --from=frontend /app/dist /app/static
COPY migrations /app/migrations
COPY config.prod.yaml /app/config.yaml

ENV DSPM_STATIC_DIR=/app/static
ENV CONFIG_PATH=/app/config.yaml

EXPOSE 8080

ENTRYPOINT ["/app/dspm"]
