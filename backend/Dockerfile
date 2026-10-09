# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM golang:1.27-alpine AS builder

# CA bundle for the scratch final stage (outbound HTTPS: Anthropic API, TLS SMTP). ISS-103
RUN apk add --no-cache ca-certificates && test -s /etc/ssl/certs/ca-certificates.crt

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api

# ── Stage 2: final ────────────────────────────────────────────────────────────
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

WORKDIR /app
COPY --from=builder /api /app/api
COPY --from=builder /app/migrations /app/migrations

EXPOSE 8080

ENTRYPOINT ["/app/api"]
