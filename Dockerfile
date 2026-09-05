# Build Stage
FROM golang:1.27.1-alpine AS builder
ARG APP_VERSION=development

WORKDIR /build

COPY go.* ./
RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o entrypoint .

# Runtime Stage
FROM alpine
ARG APP_VERSION
LABEL org.opencontainers.image.source="https://github.com/aishukander/mc-server"

ENV \
    APP_VERSION=${APP_VERSION}

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=300s \
    CMD nc -z localhost 25565 || exit 1

RUN apk add --no-cache \
    libxml2 \
    netcat-openbsd

WORKDIR /project

COPY --from=builder /build/entrypoint /project/entrypoint
COPY LICENSE /project/LICENSE

RUN chmod +x /project/entrypoint && \
    mkdir /project/server

ENTRYPOINT ["/project/entrypoint"]