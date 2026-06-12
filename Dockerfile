FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/api

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 1001 appuser

COPY --from=builder /server /server
COPY --from=builder /app/migrations /migrations

USER appuser

EXPOSE 8080

ENTRYPOINT ["/server"]
