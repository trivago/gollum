FROM trivago/gollum:base-latest

COPY . /go/src/github.com/trivago/gollum

RUN make

RUN chmod +x /go/src/github.com/trivago/gollum/gollum

ENTRYPOINT ["/go/src/github.com/trivago/gollum/gollum"]
