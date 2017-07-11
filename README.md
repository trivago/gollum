# Gollum v0.5.0

![Gollum](docs/src/gollum.png)

***This is a DEVELOPMENT branch.***
Please read the list of [breaking changes](https://github.com/trivago/gollum/wiki/Breaking050) from 0.4.x to 0.5.0.

[![GoDoc](https://godoc.org/github.com/trivago/gollum?status.svg)](https://godoc.org/github.com/trivago/gollum)
[![Documentation Status](https://readthedocs.org/projects/gollum/badge/?version=latest)](http://gollum.readthedocs.org/en/latest/)
[![Go Report Card](https://goreportcard.com/badge/github.com/trivago/gollum)](https://goreportcard.com/report/github.com/trivago/gollum)
[![Build Status](https://travis-ci.org/trivago/gollum.svg?branch=v0.4.3dev)](https://travis-ci.org/trivago/gollum)
[![Coverage Status](https://coveralls.io/repos/github/trivago/gollum/badge.svg?branch=master)](https://coveralls.io/github/trivago/gollum?branch=master)
[![Gitter](https://badges.gitter.im/trivago/gollum.svg)](https://gitter.im/trivago/gollum?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)
[![Apache V2 License](http://img.shields.io/badge/license-Apache%20V2-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0)

Gollum is an n:m multiplexer that gathers messages from different sources and broadcasts them to a set of destinations.

There are a few basic terms used throughout Gollum:

* "Consumers" read data from other services
* "Producers" write data to other services
* "Streams" route data between consumers and producers
* A "message" is a set of data passed between consumers and producers
* "Formatters" can transform the content of messages
* "Filters" can block/pass messages based on their content

Writing a custom plugin does not require you to change any additional code besides your new plugin file.

## Documentation

A how-to-use documentation can be found on [read the docs](http://gollum.readthedocs.org/en/latest/). Developers should use the [godoc pages](https://godoc.org/github.com/trivago/gollum) to get started. Plugin documentation is generated from the plugin source code. So if you feel that something is missing a look into the code may help.

If you can't find your answer in the documentation or have other questions you can reach us on [gitter](https://gitter.im/trivago/gollum?utm_source=share-link&utm_medium=link&utm_campaign=share-link), too.

## Consumers (reading data)

* `Console` read from stdin.
* `File` read from a file (like tail).
* `HTTP` read http requests.
* `Kafka` read from a [Kafka](http://kafka.apache.org/) topic.
* `Kinesis` read from a [Kinesis](https://aws.amazon.com/de/kinesis/) stream.
* `Profiler` Generate profiling messages.
* `Proxy` use in combination with a proxy producer to enable two-way communication.
* `PcapHTTP` to read http traffic from libpcap, e.g. for traffic forwarding.
* `Socket` read from a socket (gollum specific protocol).
* `Syslogd` read from a socket (syslogd protocol).
* `SystemD` read from the SystemD journal.

## Producers (writing data)

* `Console` write to stdin or stdout.
* `ElasticSearch` write to [elasticsearch](http://www.elasticsearch.org/) via http/bulk.
* `File` write to a file. Supports log rotation and compression.
* `Firehose` write data to a [Firehose](https://aws.amazon.com/de/firehose/) stream.
* `HTTPRequest` HTTP request forwarder.
* `InfluxDB` send data to an [InfluxDB](https://influxdb.com) server.
* `Kafka` write to a [Kafka](http://kafka.apache.org/) topic.
* `Kinesis` write data to a [Kinesis](https://aws.amazon.com/de/kinesis/) stream.
* `Null` like /dev/null.
* `Proxy` two-way communication proxy for simple protocols.
* `Redis` write data to [Redis](https://redis.io).
* `S3` write data to [Amazon S3](https://aws.amazon.com/de/s3/) stream.
* `Scribe` send messages to a [Facebook scribe](https://github.com/facebookarchive/scribe) server.
* `Socket` send messages to a socket (gollum specific protocol).
* `Spooling` write messages to disk and retry them later.
* `Websocket` send messages to a websocket.

## Streams (multiplexing)

* `Broadcast` send to all producers in a stream.
* `Random` send to a random producer in a stream.
* `RoundRobin` switch the producer after each send in a round robin fashion.
* `Route` convert streams to one or multiple others

## Formatters (modifying data)

* `Base64Encode` encode messages to base64.
* `Base64Decode` decode messages from base64.
* `Clear` clears a message
* `CollectdToInflux08` convert [CollectD](https://collectd.org) 0.8 data to [InfluxDB](https://influxdb.com) compatible values.
* `CollectdToInflux09` convert [CollectD](https://collectd.org) 0.9 data to [InfluxDB](https://influxdb.com) compatible values.
* `CollectdToInflux10` convert [CollectD](https://collectd.org) 0.10 data to [InfluxDB](https://influxdb.com) compatible values.
* `ExtractJSON` extracts a single field from a JSON object.
* `Envelope` add a prefix and/or postfix string to a message.
* `Forward` write the message without modifying it.
* `GrokToJSON` parse grok patterns into JSON fields.
* `Hostname` prepend the current machine's hostname to a message.
* `Identifier` hash the message to generate a (mostly) unique id.
* `JSON` write the message as a JSON object. Messages can be parsed to generate fields.
* `ProcessJSON` Modify fields of a JSON encoded message.
* `ProcessTSV` Modify fields of a TSV encoded message.
* `Runlength` prepend the length of the message.
* `Sequence` prepend the sequence number of the message.
* `SplitPick` split a message by token and pick a specific one by index.
* `SplitToJSON` tokenize a message and put the values into JSON fields.
* `StreamName` prepend the name of a stream to the payload.
* `StreamRevert` route a message to the previous stream (e.g. after it has been routed).
* `StreamRoute` route a message to another stream by reading a prefix.
* `TemplateJSON` run JSON data through golangs text/templat mechanism.
* `Timestamp` prepend a timestamp to the message.

## Filters (filtering data)

* `All` lets all message pass.
* `JSON` blocks or lets json messages pass based on their content.
* `None` blocks all messages.
* `Rate` blocks messages that go over a given messages per second rate.
* `RegExp` blocks or lets messages pass based on a regular expression.
* `Stream` blocks or lets messages pass based on their stream name.

## Installation

### From source

Installation from source requires the installation of the [Go toolchain](http://golang.org/).
Gollum supports the Go 1.5 vendor experiment that is automatically enabled when using the provided makefile.
With Go 1.6 and later you can also use `go build` directly without additional modifications.
Builds with Go 1.4 or earlier versions are not officially supported and might require additional steps and modifications.

```bash
make
./gollum --help
```

You can use the make file coming with gollum to trigger cross platform builds.
Make will produce ready to deploy .zip files with the corresponding platform builds inside the dist folder.

## Usage

To test gollum you can make a local profiler run with a predefined configuration:

```bash
gollum -c config/profile.conf -ps -ll 3
```

By default this test profiles the theoretic maximum throughput of 256 Byte messages.
You can enable different producers in that config to test the write performance of these producers, too.

### Configuration

Configuration files are written in the YAML format and have to be loaded via command line switch.
Each plugin has a different set of configuration options which are currently described in the plugin itself, i.e. you can find examples in the GoDocs.

### Commandline

#### `-c` or `--config` [file]

Use a given configuration file.

#### `-h` or `--help`

Print this help message.

#### `-ll` or `--loglevel` [0-3]

Set the loglevel [0-3]. Higher levels produce more messages as in 0=Errors, 1=Warnings, 2=Notes, 3=Debug.

#### `-m` or `--metrics` [port]

Port to use for metric queries. Set 0 to disable.

#### `-hc` or `--healthcheck` <host:port|:port|port>

Open a healthcheck HTTP endpoint at the specified listening address. Disabled by default.

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

#### `-r` or `--report`

Print detailed version report and quit.

#### `-tc` or `--testconfig` [file]

Test a given configuration file and exit.

#### `-tr` or `--trace` [file]

Write trace results to a given file.

#### `-v` or `--version`

Print version information and quit.

## Building

### Mac OS X

The easiest way to install go is by using homebrew:

```bash
brew install go
```

### Linux

Download Go from the [golang website](https://golang.org/dl/) and unzip it to e.g. /usr/local/go.
You have to set the GOROOT environment variable to the folder you chose:
`export GOROOT=/usr/local/go`

### Prerequisites

If you do not already have a GOPATH set up you need to create one.
The location is free of choice, we prefer to put it into each users home folder:

```bash
mkdir -p ~/go
export GOPATH=$(HOME)/go
```

You can download gollum via `go get github.com/trivago/gollum` or clone it directly into your GOPATH.
If you choose this way you need to download your dependencies directly from that folder

```bash
mkdir -p $(GOPATH)/src/github.com/trivago
cd $(GOPATH)/src/github.com/trivago
git clone https://github.com/trivago/gollum.git
cd gollum
```

### Build

Building gollum is as easy as `make` or `go build`.
When using Go 1.5 make sure to enable the go vendor experiment by setting `export GO15VENDOREXPERIMENT=1` or use `make`.
If you want to do cross platform builds use `make all` or specify one of the following platforms instead of "all":

* `current` build for current OS (default)
* `freebsd` build for FreeBSD
* `linux` build for Linux x64
* `mac` build for MacOS X
* `pi` build for Linux ARM
* `win` build for Windows

There are also supplementary targets for make:

* `clean` clean all artifacts created by the build process
* `test` run unittests
* `vendor` install [Glide](https://github.com/Masterminds/glide) and update all dependencies
* `aws` build for Linux x64 and generate an [Elastic Beanstalk](https://aws.amazon.com/de/elasticbeanstalk/) package

If you want to use native plugins (contrib/native) or self provided you will have to enable the corresponding imports in the file `contrib_loader.go`. You can copy the `contrib_loader.go.dist` file here and activate the plugins you want to use.

Doing so will disable the possibility to do cross-platform builds for most users.
Please check also the requirements for each plugin.

### Dockerfile

The repository contains a `Dockerfile` which enables you to build and run gollum inside a Docker container.

```bash
docker build -t trivago/gollum .
docker run -it --rm trivago/gollum -c config/profile.conf -ps -ll 3
```

To use your own configuration you could run:

```bash
docker run -it --rm -v /path/to/config.conf:/etc/gollum/gollum.conf:ro trivago/gollum -c /etc/gollum/gollum.conf
```

## Best practice

### Managing own plugins in a seperate git repository

You can add a own plugin module by simple using `git submodule`:

```bash
git submodule add -f https://github.com/YOUR_NAMESPACE/YOUR_REPO.git contrib/namespace
```

The by git created `.gitmodules` will be ignored by the gollum repository.

To activate your plugin you need to create a `contrib_loader.go` to be able to compile gollum with your own provided plugins.
You can copy the existing `contrib_loader.go.dist` to `contrib_loader.go` and update the import path to your package:

```bash
cp contrib_loader.go.dist contrib_loader.go
# open contrib_loader.go with an editor
# update package path
make current
```

## Debugging

If you want to use [Delve](https://github.com/derekparker/delve) for debugging you need to build gollum with some additional flags:

```bash
go build -ldflags='-s -linkmode=internal' -gcflags='-N -l'
```

With this debug build you are able to start a [Delve](https://github.com/derekparker/delve) remote debugger:

```bash
# for the gollum arguments pls use this format: ./gollum -- -c my/config.conf
dlv --listen=:2345 --headless=true --api-version=2 --log exec ./gollum -- -c testing/configs/test_router.conf -ll 3
```

## License

This project is released under the terms of the [Apache 2.0 license](http://www.apache.org/licenses/LICENSE-2.0).
