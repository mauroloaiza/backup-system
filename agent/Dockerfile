# ── Build stage ─────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o backupsmc-agent cmd/agent/main.go

# ── Runtime stage ────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache \
    ca-certificates \
    rclone \
    mysql-client \
    postgresql-client \
    tzdata

WORKDIR /app

COPY --from=builder /app/backupsmc-agent .

ENV TZ=America/Bogota

ENTRYPOINT ["./backupsmc-agent"]
