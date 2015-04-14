Kafka
#############

This consumer reads data from a kafka cluster using Shopify's `Sarama <https://github.com/Shopify/sarama>`_ library.
Any setting here reflects settings from this library.


Parameters
----------

**Enable**
    Can either be true or false to enable or disable this consumer.
**Stream**
    Defines either one or an aray of stream names this consumer sends messages to.
**DefaultOffset**
  Defines the offset inside the topic to start reading from if no OffsetFile is defined or found.
  Valid values are "newest", "oldest" and any number based index. By default this is set to "newest".
**OffsetFile**
  Defines a file to store the current offset to.
  The offset is updated in intervalls and can be used to continue reading after a restart.
**ClientID**
  Set the id of this client. "gollum" by default.
**MaxOpenRequests**
  Defines the number of simultanious connections are allowed.
  By default this is set to 5.
**ServerTimeoutSec**
  Defines the time after which a connection is set to timed
  out. By default this is set to 30 seconds.
**MaxFetchSizeByte**
  Defines the maximum size of a message to fetch. Larger messages
  will be ignored. By default this is set to 0 (fetch all messages).
**MinFetchSizeByte**
  Defines the minimum amout of data to fetch from Kafka per request.
  If less data is available the broker will wait
  By default this is set to 1.
**FetchTimeoutMs**
  Defines the time in milliseconds the broker will wait for MinFetchSizeByte to be reached before processing data anyway.
  By default this is set to 250ms.
**MessageBufferCount**
  Defines the internal channel size for the Kafka client.
  By default this is set to 256.
**PresistTimoutMs**
  Defines the time in milliseconds between writes to OffsetFile.
  By default this is set to 5000.
  Shorter durations reduce the amount of duplicate messages after a fail but increases I/O.
**ElectRetries**
  Defines how many times to retry during a leader election.
  By default this is set to 3.
**ElectTimeoutMs**
  Defines the number of milliseconds to wait for the cluster to elect a new leader.
  By defaults set to 250.
**MetadataRefreshMs**
  Defines the interval in seconds for fetching cluster metadata.
  By default this is set to 10000.
  This corresponds to the JVM setting `topic.metadata.refresh.interval.ms`.
**Servers**
  Contains the list of all kafka servers to connect to.

Offsets
-------

**Oldest**
  Start reading at the end of the file if no offset file is set and/or found.
**Neweset**
  Start reading at the beginning of the file if no offset file is set and/or found.
**Any number**
  Start at the specified message index.

Example
-------

::

  - "consumer.Kafka":
    Enable: true
    DefaultOffset: "Newest"
    OffsetFile: "/tmp/gollum_kafka.idx"
    ClientID: "logger"
    MaxOpenRequests: 6
    ServerTimeoutSec: 10
    MaxFetchSizeByte: 8192
    MinFetchSizeByte: 0
    FetchTimeoutMs: 500
    MessageBufferCount: 1024
    PresistTimoutMs: 1000
    ElectRetries: 5
    ElectTimeoutMs: 300
    MetadataRefreshMs: 3000
    Servers:
      - "192.168.222.30:9092"
      - "192.168.222.31:9092"
    Stream: "kafka"
