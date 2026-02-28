FROM golang:1.26-alpine AS builder

ARG APP_ENV=production
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
RUN echo "APP_ENV: ${APP_ENV:-production (default)}"
RUN addgroup -g 65532 nonroot && \
    adduser -D -u 65532 -G nonroot nonroot && \
    mkdir -p /data && \
    chown -R 65532:65532 /data

COPY . .

RUN if [ "$APP_ENV" = "development" ]; then \
        echo "Building with debug symbols (APP_ENV=development)"; \
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -gcflags="all=-N -l" -o main ./cmd/api; \
    else \
        echo "Building stripped binary (production)"; \
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o main ./cmd/api; \
    fi

FROM golang:1.26-alpine AS debug
RUN apk add --no-cache git
RUN go install github.com/go-delve/delve/cmd/dlv@latest
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /app/main /app/main
COPY --from=builder --chown=65532:65532 /data /data
USER 65532:65532
EXPOSE 2345 8080
ENTRYPOINT ["/go/bin/dlv"]
CMD ["--listen=:2345", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "/app/main", "--"]

# Production target
FROM scratch
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/main /main
COPY --from=builder --chown=65532:65532 /data /data

USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/main"]
