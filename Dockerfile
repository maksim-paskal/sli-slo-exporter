FROM alpine:latest

WORKDIR /app/

COPY ./sre-metrics-exporter /app/sre-metrics-exporter

RUN apk upgrade \
&& addgroup -g 30523 -S app \
&& adduser -u 30523 -D -S -G app app

USER 30523

ENTRYPOINT [ "/app/sre-metrics-exporter" ]