FROM quay.io/podman/stable

WORKDIR /project

RUN dnf install -y 'dnf-command(copr)'
RUN dnf copr enable project-flotta/flotta -y

# Yum dependencies
RUN dnf update -y \
  && dnf install -y openssl procps-ng dmidecode nc \
  yggdrasil flotta-agent node_exporter

# Modify podman configuration
RUN sed -i s/netns=\"host\"/netns=\"private\"/g /etc/containers/containers.conf && \
    sed -i s/utsns=\"host\"/utsns=\"private\"/g /etc/containers/containers.conf

# Certificate reqs:
RUN mkdir /etc/pki/consumer && \
    openssl req -new -newkey rsa:4096 -x509 -sha256 -days 365 -nodes -out cert.pem -keyout key.pem -subj "/C=EU/ST=No/L=State/O=D/CN=www.example.com" && \
    mv cert.pem key.pem /etc/pki/consumer

# Default yggdrasil configuration should be replaced by volume with proper config:
RUN echo "" > /etc/yggdrasil/config.toml && \
    echo 'key-file = "/etc/pki/consumer/key.pem"' >> /etc/yggdrasil/config.toml && \
    echo 'cert-file = "/etc/pki/consumer/cert.pem"' >> /etc/yggdrasil/config.toml && \
    echo "client-id = \"$(cat /etc/machine-id)\"" >> /etc/yggdrasil/config.toml && \
    echo 'server = "172.17.0.1:8888"' >> /etc/yggdrasil/config.toml && \
    echo 'protocol = "http"' >> /etc/yggdrasil/config.toml && \
    echo 'path-prefix="api/flotta-management/v1"' >> /etc/yggdrasil/config.toml && \
    echo 'log-level="trace"' >> /etc/yggdrasil/config.toml

# enable yggdrasil service
RUN systemctl enable yggdrasild.service

ENTRYPOINT ["/sbin/init"]
