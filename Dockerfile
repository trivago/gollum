FROM alpine:latest

COPY gollum /usr/local/bin
COPY config /var/gollum/

WORKDIR /var/gollum