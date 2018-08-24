FROM golang:alpine as builder

ADD . /go/src/github.com/trivago/gollum
RUN apk update && \
    apk add make git

WORKDIR /go/src/github.com/trivago/gollum
RUN make build

#############################################################################

FROM alpine:3.7

LABEL maintainer="marc.siebeneicher@trivago.com"
LABEL maintainer="arne.claus@trivago.com"

COPY --from=builder /go/src/github.com/trivago/gollum/gollum /usr/local/bin

RUN apk add ca-certificates

# /etc/gollum is meant to be mounted with config data, etc.
RUN mkdir -p /etc/gollum && \
    chmod -R 755 /etc/gollum

ENTRYPOINT ["/usr/local/bin/gollum"]