FROM golang:1.14-alpine


WORKDIR /opt/pingpong
COPY . .
RUN go build -o /usr/local/bin/docker-entrypoint ./cmd/server/

ENTRYPOINT ["docker-entrypoint"]