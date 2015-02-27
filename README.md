# Gollum

Gollum is a log multiplexer that gathers messages from different sources and broadcasts them to a set of listeners.
There are a few basic terms used throughout Gollum:

* A "consumer" is a plugin that reads from an external source
* A "producer" is a plugin that writes to an external source
* A "stream" is a message channel between consumer(s) and producer(s)
* A "formatter" is a plugin that adds information to a message
* A "distributor" is a plugin that routes/filters messages on a given stream

Writing a custom plugin does not require you to change any additional code besides your new plugin file.

## Consumers

* `Console` read from stdin.
* `File` read from a file (like tail).
* `Kafka` read from a Kafka topic using the Sarama Kafka library.
* `Socket` read from a socket (gollum specfic protocol).

## Producers

* `Console` write to stdin or stdout.
* `ElasticSearch` write to elastic search via http/bulk.
* `File` write to a file. Supports log rotation and compression.
* `Kafka` write to a Kafka topic using the Sarama Kafka library.
* `Null` like /dev/null.
* `Facebook Scribe` send messages to a scribe server.
* `Socket` send messages to a socket  (gollum specfic protocol).

## Distributors

* `Broadcast` send to all producers in a stream.
* `Random` send to a random roducers in a stream.
* `RoundRobin` switch the producer after each send in a round robin fashion.

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
