# Gollum

Gollum is a log multiplexer that gathers messages from different sources and broadcasts them to a set of listeners.
There are a few basic terms used throughout Gollum:

* A "consumer" is a plugin that reads from an external source
* A "producer" is a plugin that writes to an external source
* A "stream" is a message channel between consumer(s) and producer(s)
* A "formatter" is a plugin that adds information to a message
* A "distributor" is a plugin that routes/filters messages on a given stream

## Consumers

* Console
* File
* Kafka
* Socket

## Producers

* Console
* ElasticSearch
* File
* Kafka
* Null
* Facebook Scribe
* Socket

## Installation

### From source

Install dependencies
```
$ go get -u github.com/artyom/scribe
$ go get -u github.com/artyom/thrift
$ go get -u github.com/Shopify/sarama
$ go get -u github.com/mattbaird/elastigo
$ go get -u gopkg.in/docker/docker.v1
$ go get -u gopkg.in/yaml.v1
```

Get source and compile.
Note that gollum has to be placed into $GOPATH/src/github.com/trivago if you did not fetch it directly from GitHub using go get.

```
$ go build
```

## Configuration

## Usage

### Options

#### `-c` or `--config` (file)

Load a YAML config file. Example files can be found in the gollum directory.

#### `-n` or `--numcpu` (number)

Number of cores to use.

#### `-p` or `--pidfile` (file)

Generate a pid file at a given path.

#### `-v` or `--version`

Show version and exit.

#### `-t` or `--throughput`

Write regular statistics about message / sec throughput.

#### `-cp` or `--cpuprofile`

Write go CPU profiler results to a given file.

#### `-mp` or `--memprofile`

Write go memory profiler results to a given file.

## License
