FROM rockylinux/rockylinux:9-minimal

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=300s \
    CMD nc -z localhost 25565 || exit 1

RUN microdnf update -y && \
    microdnf install -y \
    wget \
    jq \
    libxml2 \
    nmap-ncat \
    tar && \
    microdnf clean all

WORKDIR /project

COPY . .

RUN chmod +x /project/entrypoint.sh && \
    mkdir /project/server

ENTRYPOINT ["/project/entrypoint.sh"]