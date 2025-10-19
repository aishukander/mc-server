# Build Stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.* ./
RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o entrypoint .

# Runtime Stage
FROM rockylinux/rockylinux:9-minimal

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=300s \
    CMD nc -z localhost 25565 || exit 1

RUN microdnf update -y && \
    microdnf install -y \
    libxml2 \
    nmap-ncat && \
    microdnf clean all

WORKDIR /project

COPY --from=builder /build/entrypoint /project/entrypoint

RUN chmod +x /project/entrypoint && \
    mkdir /project/server

ENTRYPOINT ["/project/entrypoint"]