![Gollum](docs/gollum.png)

## Gollum

[![GoDoc](https://godoc.org/github.com/trivago/gollum?status.svg)](https://godoc.org/github.com/trivago/gollum)
[![Documentation Status](https://readthedocs.org/projects/gollum/badge/?version=latest)](https://readthedocs.org/projects/gollum/?badge=latest)

Gollum is a n:m multiplexer that gathers messages from different sources and broadcasts them to a set of destinations.

There are a few basic terms used throughout Gollum:

* "Consumers" read data from other services
* "Producers" write data to other services
* "Streams" route data between consumers and producers
* A "message" is a set of data passed between consumers and producers
* "Formatters" can transform the content of messages
* "Filters" can block/pass messages based on their content

Writing a custom plugin does not require you to change any additional code besides your new plugin file.

## Consumers (reading data)

* `Console` read from stdin.
* `File` read from a file (like tail).
* `Http` read http requests.
* `Kafka` read from a [Kafka](http://kafka.apache.org/) topic.
* `LoopBack` Process routed (e.g. dropped) messages.
* `Proxy` use in combination with a proxy producer to enable two-way communication.
* `Socket` read from a socket (gollum specfic protocol).
* `Syslogd` read from a socket (syslogd protocol).

## Producers (writing data)

* `Console` write to stdin or stdout.
* `ElasticSearch` write to [elasticsearch](http://www.elasticsearch.org/) via http/bulk.
* `File` write to a file. Supports log rotation and compression.
* `HttpReq` HTTP request forwarder.
* `Kafka` write to a [Kafka](http://kafka.apache.org/) topic.
* `Null` like /dev/null.
* `Proxy` two-way communication proxy for simple protocols.
* `Scribe` send messages to a [Facebook scribe](https://github.com/facebookarchive/scribe) server.
* `Socket` send messages to a socket (gollum specfic protocol).
* `Websocket` send messages to a websocket.

## Streams (multiplexing)

* `Broadcast` send to all producers in a stream.
* `Random` send to a random roducers in a stream.
* `RoundRobin` switch the producer after each send in a round robin fashion.

## Formatters (modifying data)

* `Base64Encode` encodes messages to base64.
* `Base64Decode` decodes messages from base64.
* `Envelope` add a prefix and/or postfix string to a message.
* `Forward` write the message without modifying it.
* `Hostname` prepends the current machine's hostname to a message.
* `Identifier` hashes the message to generate a (mostly) unique id.
* `JSON` write the message as a JSON object. Messages can be parsed to generate fields.
* `Runlength` prepends the length of the message.
* `Sequence` prepends the sequence number of the message.
* `StreamMod` route a message to another stream by reading a prefix.
* `Timestamp` prepends a timestamp to the message.

## Filters (filtering data)

* `All` lets all message pass.
* `Json` blocks or lets json messages pass based on their content.
* `None` blocks all messages.
* `RegExp` blocks or lets messages pass based on a regular expression.

## Installation

### From source

Installation from source requires the installation of the [Go toolchain](http://golang.org/).  
Gollum has [Godeps](https://github.com/tools/godep) support but this is considered optional.

```
$ go get .
$ go build
$ gollum --help
```

You can use the supplied make file to trigger cross platform builds.  
Make will produce ready to deploy .tar.gz files with the corresponding platform builds.  
This does require a cross platform golang build.  
Valid make targets (besides all and clean) are:
 * freebsd
 * linux
 * mac
 * pi
 * win

## Usage

To test gollum you can make a local profiler run with a predefined configuration:

```
$ gollum -c config/profile.conf -ps -ll 3
```

By default this test profiles the theoretic maximum throughput of 256 Byte messages.  
You can enable different producers to test the write performance of these producers, too.

## Configuration

Configuration files are written in the YAML format and have to be loaded via command line switch.
Each plugin has a different set of configuration options which are currently described in the plugin itself, i.e. you can find examples in the GoDocs.

### Commandline

#### `-c` or `--config` [file]

Use a given configuration file.

#### `-h` or `--help`

Print this help message.

#### `-ll` or `--loglevel` [0-3]

Set the loglevel [0-3]. Higher levels produce more messages.

#### `-m` or `--metrics` [port]

Port to use for metric queries. Set 0 to disable.

#### `-n` or `--numcpu` [number]

Number of CPUs to use. Set 0 for all CPUs.

#### `-p` or `--pidfile` [file]

Write the process id into a given file.

#### `-pc` or `--profilecpu` [file]

Write CPU profiler results to a given file.

#### `-pm` or `--profilemem` [file]

Write heap profile results to a given file.

#### `-ps` or `--profilespeed`

Write msg/sec measurements to log.

#### `-tc` or `--testconfig` [file]

Test a given configuration file and exit.

#### `-v` or `--version`

Print version information and quit.

## Building

### Mac OS X

The easiest way to install go is by using homebrew:  
`brew install go`

If you want to do cross platform builds you need to specify an additional option:  
`brew install go --with-cc-all`

### Linux

Download Go from the [golang website](https://golang.org/dl/) and unzip it to e.g. /usr/local/go.  
You have to set the GOROOT environment variable to the folder you chose:  
`export GOROOT=/usr/local/go`

### Prerequisites

If you do not already have a GOPATH set up you need to create one.  
The location is free of choice, we prefer to put it into each users home folder:
```
mkdir -p ~/go
export GOROOT=$(HOME)/go
```

You can download gollum via `go get github.com/trivago/gollum` or clone it directly into your GOPATH.  
If you choose this way you need to download your dependencies directly from that folder
```
mkdir -p $(GOPATH)/src/github.com/trivago
cd $(GOPATH)/src/github.com/trivago
git clone https://github.com/trivago/gollum.git
cd gollum
go get -u .
```

### Build

Building gollum is as easy as `go build`.  
If you want to do cross platform builds use `make all` or specifiy one of the following platforms instead of "all":
- freebsd
- linux
- mac
- pi
- win

Please not that building for windows will give you errors, which can be solved by removing the lines reported.  
If you want to use native plugins (contrib/native) you will have to enable the import in the file contrib/loader.go.
Doing so will disable the possibility to do cross-platform builds for most users.

### Solving dependency problems

If you got any errors during build regarding external dependencies (i.e. the error message points to another repository than github.com/trivago) you can restore the last dependency snapshot using [godep](https://github.com/tools/godep).
Install godep via `go get github.com/tools/godep` and restore the dependency via `godep restore` when inside the gollum base folder.

## License

This project is released under the terms of the [Apache 2.0 license](http://www.apache.org/licenses/LICENSE-2.0).
