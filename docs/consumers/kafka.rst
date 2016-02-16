Kafka
=====

Thes consumer reads data from a given kafka topic.
It is based on the sarama library so most settings are mapped to the settings from this library.
When attached to a fuse, this consumer will stop processing messages in case that fuse is burned.


Parameters
----------

**Enable**
  Enable switches the consumer on or off.
  By default this value is set to true.

**ID**
  ID allows this consumer to be found by other plugins by name.
  By default this is set to "" which does not register this consumer.

**Stream**
  Stream contains either a single string or a list of strings defining the message channels this consumer will produce.
  By default this is set to "*" which means only producers set to consume "all streams" will get these messages.

**Fuse**
  Fuse defines the name of a fuse to observe for this consumer.
  Producer may "burn" the fuse when they encounter errors.
  Consumers may react on this by e.g. closing connections to notify any writing services of the problem.
  Set to "" by default which disables the fuse feature for this consumer.
  It is up to the consumer implementation to react on a broken fuse in an appropriate manner.

**Topic**
  Topic defines the kafka topic to read from.
  By default this is set to "default".

**DefaultOffset**
  DefaultOffset defines where to start reading the topic.
  Valid values are "oldest" and "newest".
  If OffsetFile is defined the DefaultOffset setting will be ignored unless the file does not exist.
  By default this is set to "newest".

**OffsetFile**
  OffsetFile defines the path to a file that stores the current offset inside a given partition.
  If the consumer is restarted that offset is used to continue reading.
  By default this is set to "" which disables the offset file.

**MaxOpenRequests**
  MaxOpenRequests defines the number of simultanious connections are allowed.
  By default this is set to 5.

**ServerTimeoutSec**
  ServerTimeoutSec defines the time after which a connection is set to timed out.
  By default this is set to 30 seconds.

**MaxFetchSizeByte**
  MaxFetchSizeByte sets the maximum size of a message to fetch.
  Larger messages will be ignored.
  By default this is set to 0 (fetch all messages).

**MinFetchSizeByte**
  MinFetchSizeByte defines the minimum amout of data to fetch from Kafka per request.
  If less data is available the broker will wait.
  By default this is set to 1.

**FetchTimeoutMs**
  FetchTimeoutMs defines the time in milliseconds the broker will wait for MinFetchSizeByte to be reached before processing data anyway.
  By default this is set to 250ms.

**MessageBufferCount**
  MessageBufferCount sets the internal channel size for the kafka client.
  By default this is set to 256.

**PresistTimoutMs**
  PresistTimoutMs defines the time in milliseconds between writes to OffsetFile.
  By default this is set to 5000.
  Shorter durations reduce the amount of duplicate messages after a fail but increases I/O.

**ElectRetries**
  ElectRetries defines how many times to retry during a leader election.
  By default this is set to 3.

**ElectTimeoutMs**
  ElectTimeoutMs defines the number of milliseconds to wait for the cluster to elect a new leader.
  Defaults to 250.

**MetadataRefreshMs**
  MetadataRefreshMs set the interval in seconds for fetching cluster metadata.
  By default this is set to 10000.
  This corresponds to the JVM setting `topic.metadata.refresh.interval.ms`.

**Servers**
  Servers contains the list of all kafka servers to connect to.
  By default this is set to contain only "localhost:9092".

Example
-------

.. code-block:: yaml

	- "consumer.Kafka":
	    Enable: true
	    ID: ""
	    Fuse: ""
	    Stream:
	        - "foo"
	        - "bar"
	    Topic: "default"
	    DefaultOffset: "newest"
	    OffsetFile: ""
	    MaxOpenRequests: 5
	    ServerTimeoutSec: 30
	    MaxFetchSizeByte: 0
	    MinFetchSizeByte: 1
	    FetchTimeoutMs: 250
	    MessageBufferCount: 256
	    PresistTimoutMs: 5000
	    ElectRetries: 3
	    ElectTimeoutMs: 250
	    MetadataRefreshMs: 10000
	    Servers:
	        - "localhost:9092"
