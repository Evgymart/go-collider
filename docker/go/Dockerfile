FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download

COPY cmd/ ./cmd/

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main ./cmd/main.go
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/main .

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

USER 1000:1000
EXPOSE 8080

CMD ["./main"]