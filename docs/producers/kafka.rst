Kafka
=====

This producers sends messages to a Kafka cluster using Shopify's `Sarama <https://github.com/Shopify/sarama>`_ library.
Any setting here reflects settings from this library.

Parameters
----------

**Enable**
  Can either be true or false to enable or disable this consumer.
**Stream**
  Defines either one or an aray of stream names this consumer sends messages to.
**Channel**
  Defines the number of messages that can be buffered by the internal channel.
  By default this is set to 8192.
**ChannelTimeoutMs**
  Defines a timeout in milliseconds for messages to wait if this producer's queue is full.

- A timeout of -1 or lower will discard the message without notice.
- A timeout of 0 will block until the queue is free. This is the default.
- A timeout of 1 or higher will wait n milliseconds for the queues to become available again.
  If this does not happen, the message will be send to the _DROPPED_ stream that can be processed by the :doc:`Loopback </consumers/loopback>` consumer.

**Format**
  Defines a message formatter to use. :doc:`Format.Forward </formatters/forward>` by default.
**ClientID**
  Set the id of this client. "gollum" by default.
**Partitioner**
  Defines the distribution algorithm to use.
  Valid values are: "random", "roundrobin" and "hash".
  By default "Hash" is used.
**RequiredAcks**
  Sets the acknowledgement level required by the broker. By default this is set to 1.

  - **0**: no responses required.
  - **1**: wait for the local commit.
  - **1**:  wait for all replicas to commit.
  - **>1**: wait for a specific number of commits.

**TimeoutMs**
  Defines the maximum time the broker will wait for acks.
  This setting becomes active when RequiredAcks is set to wait for multiple commits.
  By default this is set to 1500.
**SendRetries**
  Defines how many times to retry sending data before marking a server as not reachable.
  By default this is set to 3.
**Compression**
  Sets the method of compression to use.
  Valid values are: "None","Zip","Snappy".
  By default "None" is set.
**MaxOpenRequests**
  Defines the number of simultanious connections are allowed.
  By default this is set to 5.
**BatchMinCount**
  Sets the minimum number of messages required to trigger a flush.
  By default this is set to 1.
**BatchMaxCount**
  Defines the maximum number of messages processed per request.
  By default this is set to 0 for "unlimited".
**BatchSizeByte**
  Sets the mimimum number of bytes to collect before a new flush is triggered.
  By default this is set to 8192.
**BatchSizeMaxKB**
  Defines the maximum allowed message size.
  By default this is set to 1 MB.
**BatchTimeoutSec**
  Sets the minimum time in seconds to pass after wich a new flush will be triggered.
  By default this is set to 3.
**MessageBufferCount**
  Sets the internal channel size for the kafka client.
  By default this is set to 256.
**ServerTimeoutSec**
  Defines the time after which a connection is set to timed out.
  By default this is set to 30 seconds.
**SendTimeoutMs**
  Defines the number of milliseconds to wait for a server to resond before triggering a timeout.
  Defaults to 250.
**ElectRetries**
  Defines how many times to retry during a leader election.
  By default this is set to 3.
**ElectTimeoutMs**
  Defines the number of milliseconds to wait for the cluster to elect a new leader.
  Defaults to 250.
**MetadataRefreshMs**
  Set the interval in seconds for fetching cluster metadata.
  By default this is set to 10000.
  This corresponds to the JVM setting `topic.metadata.refresh.interval.ms`.
**Servers**
  Defines the list of all kafka servers to connect to.
  Expects the IP or DNS of the server to listen to, followed by a port.
**Topic**
  Maps a stream to a specific Kafka topic.
  If you define a mapping on "*" all streams that do not have a specific mapping will go to this topic (including internal streams).
  If no mapping to "*" is set the stream name is used as topic.

Example
-------

.. code-block:: yaml

  - "producer.Kafka":
    Enable: true
    ClientId: "weblog"
    Partitioner: "Roundrobin"
    RequiredAcks: 0
    TimeoutMs: 0
    SendRetries: 5
    Compression: "Snappy"
    MaxOpenRequests: 6
    BatchMinCount: 10
    BatchMaxCount: 0
    BatchSizeByte: 16384
    BatchSizeMaxKB: 524288
    BatchTimeoutSec: 5
    ServerTimeoutSec: 3
    SendTimeoutMs: 100
    ElectRetries: 3
    ElectTimeoutMs: 1000
    MetadataRefreshSec: 30
    Servers:
    	- "192.168.222.30:9092"
      - "192.168.222.31:9092"
    Topic:
      "*" : "server_log"
      "_GOLLUM_"  : "gollum_log"
    Stream:
      - "console"
      - "_GOLLUM_"
