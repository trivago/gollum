# Gollum

Gollum is a log multiplexer that gathers messages from different sources and broadcasts them to a set of listeners.
There are three basic terms used throughout Gollum:

* A "consumer" is a plugin that reads from an external source
* A "producer" is a plugin that writes to an external source
* A "stream" is a message channel between consumer(s) and producer(s)

## Features

* Foo
* Bar
* Baz

## Installation

### From source

Install dependencies
```
$ go get -u github.com/artyom/scribe
$ go get -u github.com/artyom/thrift
$ go get -u launchpad.net/goyaml
```

Get source and compile.
Note that gollum has to be placed into $GOPATH/src/github.com/trivago if you did
not fetch it directly from GitHub using go get.

```
$ go build
```

## Configuration

## Usage

### Options

#### `-c`

Load a YAML config file. Example files can be found in the gollum directory.

#### `-v`

Show version and exit

#### `-cpuprofile`

Write profiler results to a given file

## License
