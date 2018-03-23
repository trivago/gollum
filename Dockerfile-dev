FROM golang:1.10-stretch

LABEL maintainer="marc.siebeneicher@trivago.com"
LABEL maintainer="arne.claus@trivago.com"

# install dependencies
RUN apt-get update && \
    apt-get install --no-install-recommends -qqy make git libc-dev libpcap-dev libsystemd-dev

ADD http://launchpadlibrarian.net/234454186/librdkafka1_0.8.6-1.1_amd64.deb /src/librdkafka1_0.8.6-1.1_amd64.deb
ADD http://launchpadlibrarian.net/234454185/librdkafka-dev_0.8.6-1.1_amd64.deb /src/librdkafka-dev_0.8.6-1.1_amd64.deb

RUN dpkg -i /src/librdkafka1_0.8.6-1.1_amd64.deb && \
    dpkg -i /src/librdkafka-dev_0.8.6-1.1_amd64.deb

# setup repository
ADD . /go/src/github.com/trivago/gollum/
WORKDIR /go/src/github.com/trivago/gollum

RUN cp contrib_loader.go.dist contrib_loader.go; \
    ln -s /go/src/github.com/trivago/gollum/gollum /usr/local/bin/gollum; \
    mkdir -p /etc/gollum && \
    chmod -R 755 /etc/gollum

# build
RUN make build

ENTRYPOINT ["/usr/local/bin/gollum"]