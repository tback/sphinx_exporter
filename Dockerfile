FROM        quay.io/prometheus/busybox:latest
MAINTAINER  Till Backhaus <till@backha.us>

EXPOSE      9104
COPY sphinx_exporter /bin/sphinx_exporter

ENTRYPOINT ["/bin/sphinx_exporter"]
