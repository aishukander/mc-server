FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y \
    curl \
    jq \
    libxml2-utils && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /project

COPY . .

RUN chmod +x /project/entrypoint.sh && \
    mkdir /project/server

ENTRYPOINT ["/project/entrypoint.sh"]