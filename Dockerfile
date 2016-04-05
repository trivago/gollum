FROM trivago/gollum:base-latest

COPY . /go/src/github.com/trivago/gollum

RUN make

ENTRYPOINT ["/go/src/gollum/gollum"]
