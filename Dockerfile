# Multi-stage Dockerfile for mithril

# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

# Bypass proxy when it returns 403 (e.g. in some Docker/network environments)
ENV GOPROXY=direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/binary .

# Final stage
FROM alpine:3.19

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

CMD ["./app"]
