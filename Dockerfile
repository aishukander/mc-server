FROM debian:bookworm-slim

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=300s \
    CMD nc -z localhost 25565 || exit 1

RUN apt-get update && \
    apt-get install -y \
    wget \
    jq \
    libxml2-utils \
    netcat-traditional && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /project

COPY . .

RUN chmod +x /project/entrypoint.sh && \
    mkdir /project/server

ENTRYPOINT ["/project/entrypoint.sh"]