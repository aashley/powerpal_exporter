ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest
LABEL maintainer="Adam Ashley <aashley@adamashley.name>"

ARG ARCH="amd64"
ARG OS="linux"
COPY .build/${OS}-${ARCH}/powerpal_exporter  /bin/powerpal_exporter

EXPOSE      9915
ENTRYPOINT  [ "/bin/powerpal_exporter" ]

