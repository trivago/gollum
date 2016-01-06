FROM golang:latest

COPY . /go/src/gollum
WORKDIR /go/src/gollum

RUN go get . && \
	go build -v

ENTRYPOINT ["/go/src/gollum/gollum"]
