FROM golang:1.14-alpine

ARG CMD_PATH
ENV DD_PROPAGATION_STYLE_INJECT=Datadog,B3
ENV DD_PROPAGATION_STYLE_EXTRACT=Datadog,B3

WORKDIR /opt/pingpong
COPY . .
RUN go build -o /usr/local/bin/docker-entrypoint $CMD_PATH

ENTRYPOINT ["docker-entrypoint"]