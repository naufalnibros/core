FROM golang:1.24.0-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.Version=docker" \
    -o /app/server .

FROM golang:1.24.0-alpine AS development

RUN go install github.com/air-verse/air@v1.61.7
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum* ./

RUN go mod download
COPY . .
EXPOSE 11082

CMD ["air", "-c", ".air.toml"]

FROM alpine:3.19 AS production

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/.env* ./
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 11082

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:11082/core/_health || exit 1

CMD ["./server"]
