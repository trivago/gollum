# Gollum

Gollum is a n:m log multiplexer that gathers messages from different sources and broadcasts them to a set of listeners.

There are a few basic terms used throughout Gollum:

* A "consumer" is a plugin that reads from an external source
* A "producer" is a plugin that writes to an external source
* A "stream" is a message channel between consumer(s) and producer(s)
* A "formatter" is a plugin that adds information to a message
* A "distributor" is a plugin that routes/filters messages on a given stream

Writing a custom plugin does not require you to change any additional code besides your new plugin file.

## Consumers (reading data)

* `Console` read from stdin.
* `File` read from a file (like tail).
* `Kafka` read from a [Kafka](http://kafka.apache.org/) topic.
* `Socket` read from a socket (gollum specfic protocol).

## Producers (writing data)

* `Console` write to stdin or stdout.
* `ElasticSearch` write to [elasticsearch](http://www.elasticsearch.org/) via http/bulk.
* `File` write to a file. Supports log rotation and compression.
* `Kafka` write to a [Kafka](http://kafka.apache.org/) topic.
* `Null` like /dev/null.
* `Facebook Scribe` send messages to a [scribe](https://github.com/facebookarchive/scribe) server.
* `Socket` send messages to a socket (gollum specfic protocol).

## Formatters (modifying data)

TODO

## Distributors (multiplexing)

* `Broadcast` send to all producers in a stream.
* `Random` send to a random roducers in a stream.
* `RoundRobin` switch the producer after each send in a round robin fashion.

## Installation

### From source

Installation from source requires the installation of the [Go toolchain](http://golang.org/).

```
$ go get .
$ go build .
$ gollum --help
```

## Usage

The simplest usage is to make a local profiler run with a predefined consiguration:

```
$ gollum -c gollum_profile.conf
```

This configuration generates random messages (via *consumer.Profiler*) and write this to a file called *log_profile.log* with file rotation after 512MB per file (via *producer.File*).

## Configuration

Configuration files are written in the YAML format and have to be loaded via command line switch.
Each plugin has a different set of configuration options which are currently described in the plugin itself, i.e. you can find examples in the GoDocs.

### Commandline

#### `-c` or `--config` (file)

Load a YAML config file. Example files can be found in the gollum directory.

#### `-n` or `--numcpu` (number)

Number of cpu cores to use.

#### `-p` or `--pidfile` (file)

Generate a pid file at a given path.

#### `-v` or `--version`

Show version and exit.

#### `-t` or `--throughput`

Write regular statistics about message / sec throughput to stdout.

#### `-cp` or `--cpuprofile` (file)

Write go CPU profiler results to a given file.

#### `-mp` or `--memprofile` (file)

Write go memory profiler results to a given file.

### Configuration file

TODO

## Use cases

TODO

### Nginx logs to Kafka

To aggregate logs by a [nginx](http://nginx.org/) web server *Gollum* can be used.
Configure a *Gollum* syslogd consumer like **

```
...
- "consumer.Syslogd":
    Enable: true
    Channel: 1024
    Stream: "profile"
    Format: 3164
    Address: 0.0.0.0:5880
...

# TODO Add Kafka producer
```
This consumer will listen to 0.0.0.0:5880 and follow the [RFC 3164](http://tools.ietf.org/html/rfc3164) for the *profile* stream and a buffer of 1024 messages.

An example *nginx.conf* can look like
```
http {
  ...
  log_format syslogd "$remote_addr - $remote_user [$time_local] '$request' $status $body_bytes_sent '$http_referer' '$http_user_agent'\n";
  access_log syslog:server=192.168.7.52:5880 syslogd;
  ...
}
```
Important note: A syslog message will be delimited by a newline. The *\n* at the end of *log_format* is important here.

References:
* [Logging to syslog @ Nginx docs](http://nginx.org/en/docs/syslog.html)
* [Module ngx_http_log_module @ Nginx docs](http://nginx.org/en/docs/http/ngx_http_log_module.html)

### Business logging (by PHP) to Kafka

TODO

### Accesslog parsing for Elasticsearch

TODO

### Log aggregation by many servers to files

TODO

## License

This project is released under the terms of the [Apache 2.0 license](http://www.apache.org/licenses/LICENSE-2.0).
