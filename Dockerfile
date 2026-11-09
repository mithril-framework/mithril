# Multi-stage Dockerfile for mithril

# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Views, static assets and migrations are embedded (embed.FS): the binary is the whole app.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/binary .

# Final stage
FROM alpine:3.22

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /app/binary ./app

# IANA zone; set at build: docker build --build-arg TZ=America/New_York .
# Override at run: docker run -e TZ=... (or mount .env; app loads it if TZ unset).
ARG TZ=UTC
ENV TZ=$TZ

RUN chown -R appuser:appgroup /app
USER appuser

EXPOSE 4000

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O- http://localhost:4000/health || exit 1

ENV APP_ENV=production
# Migrations: docker run --env-file .env IMAGE ./app migrate up
CMD ["./app"]
