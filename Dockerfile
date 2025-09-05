FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND noninteractive
ENV LANG C.UTF-8

# Default versions
ENV INFLUXDB_VERSION=1.8.2
ENV GRAFANA_VERSION=7.2.0
ENV GO_VERSION=1.23.0

# Grafana database type
ENV GF_DATABASE_TYPE=sqlite3

# Fix bad proxy issue
COPY image/99fixbadproxy /etc/apt/apt.conf.d/99fixbadproxy

WORKDIR /root

# Clear previous sources
RUN ARCH= && dpkgArch="$(dpkg --print-architecture)" && \
    case "${dpkgArch##*-}" in \
      amd64) ARCH='amd64';; \
      arm64) ARCH='arm64';; \
      armhf) ARCH='armhf';; \
      armel) ARCH='armel';; \
      *)     echo "Unsupported architecture: ${dpkgArch}"; exit 1;; \
    esac && \
    rm /var/lib/apt/lists/* -vf \
    # Base dependencies
    && apt-get -y update \
    && apt-get -y dist-upgrade \
    && apt-get -y --force-yes install \
        apt-utils \
        ca-certificates \
        certbot \
        curl \
        git \
        htop \
        libfontconfig \
        nano \
        net-tools \
        python3-certbot-nginx \
        supervisor \
        wget \
        gnupg \
        nginx \
    && curl -fsSL https://deb.nodesource.com/setup_14.x | bash - \
    && apt-get install -y nodejs \
    && mkdir -p /var/log/supervisor \
    && curl -fsSLO https://dl.influxdata.com/influxdb/releases/influxdb_${INFLUXDB_VERSION}_${ARCH}.deb \
    && dpkg -i influxdb_${INFLUXDB_VERSION}_${ARCH}.deb \
    && rm influxdb_${INFLUXDB_VERSION}_${ARCH}.deb \
    && curl -fsSLO https://dl.grafana.com/oss/release/grafana_${GRAFANA_VERSION}_${ARCH}.deb \
    && dpkg -i grafana_${GRAFANA_VERSION}_${ARCH}.deb \
    && rm grafana_${GRAFANA_VERSION}_${ARCH}.deb \
    && curl -fsSLO https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz \
    && tar -C /opt -xzf go${GO_VERSION}.linux-amd64.tar.gz \
    && rm go${GO_VERSION}.linux-amd64.tar.gz \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

COPY image/profile /root/.profile
COPY image/influxdb.conf /etc/influxdb/influxdb.conf
COPY image/grafana.ini /etc/grafana/grafana.ini
COPY image/nginx.conf /etc/nginx/nginx.conf
COPY image/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY image/run.sh /run.sh
COPY image/dashboards.yaml /etc/grafana/provisioning/dashboards/dashboards.yaml
COPY image/go-build.sh /root/
COPY image/certbot-renew.sh /root/

RUN mkdir -p /root/go/src/github.com/loda-lang/loda-api
COPY cmd /root/go/src/github.com/loda-lang/loda-api/cmd
COPY shared /root/go/src/github.com/loda-lang/loda-api/shared
COPY util /root/go/src/github.com/loda-lang/loda-api/util
COPY go.mod go.sum /root/go/src/github.com/loda-lang/loda-api/
RUN chmod +x /root/go-build.sh
RUN /root/go-build.sh

RUN mkdir -p /var/log/loda
RUN rm /etc/grafana/provisioning/dashboards/sample.yaml
RUN chmod +x /run.sh

CMD /run.sh
