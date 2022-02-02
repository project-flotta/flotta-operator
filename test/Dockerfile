FROM quay.io/podman/stable

WORKDIR /project

# Setup Yggdrasil & device-worker repo:
RUN curl -k https://copr.fedorainfracloud.org/coprs/ondramachacek/yggdrasil/repo/fedora-34/ondramachacek-yggdrasil-fedora-34.repo -o /etc/yum.repos.d/yggd.repo
RUN curl -k https://copr.fedorainfracloud.org/coprs/eloyocoto/k4e-test/repo/fedora-34/eloyocoto-k4e-test-fedora-34.repo -o /etc/yum.repos.d/flotta-agent.repo && \
    sed -i 's/\$releasever/34/g' /etc/yum.repos.d/flotta-agent.repo && \
    sed -i 's/\$basearch/x86_64/g' /etc/yum.repos.d/flotta-agent.repo

# Yum dependencies
RUN yum install -y golang openssl procps-ng dmidecode flotta-agent

# Modify podman configuration
RUN sed -i s/netns=\"host\"/netns=\"private\"/g /etc/containers/containers.conf && \
    sed -i s/utsns=\"host\"/utsns=\"private\"/g /etc/containers/containers.conf

# Certificate reqs:
RUN mkdir /etc/pki/consumer && \
    openssl req -new -newkey rsa:4096 -x509 -sha256 -days 365 -nodes -out cert.pem -keyout key.pem -subj "/C=EU/ST=No/L=State/O=D/CN=www.example.com" && \
    mv cert.pem key.pem /etc/pki/consumer

# Default yggdrasil configuration should be replaced by volume with proper config:
RUN echo 'key-file = "/etc/pki/consumer/key.pem"' >> /etc/yggdrasil/config.toml && \
    echo 'cert-file = "/etc/pki/consumer/cert.pem"' >> /etc/yggdrasil/config.toml && \
    echo 'client-id-source = "machine-id"' >> /etc/yggdrasil/config.toml && \
    echo 'http-server = "172.17.0.1:8888"' >> /etc/yggdrasil/config.toml && \
    echo 'transport = "http"' >> /etc/yggdrasil/config.toml && \
    sed -i 's/log-level = "error"/log-level = "trace"/g' /etc/yggdrasil/config.toml


# enable yggdrasil service
RUN systemctl enable yggdrasild.service

ENTRYPOINT ["/sbin/init"]
