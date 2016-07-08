Kafka
=====

The kafka producer writes messages to a kafka cluster.
This producer is backed by the sarama library so most settings relate to that library.
This producer uses a fuse breaker if any connection reports an error.


Parameters
----------

**Enable**
  Enable switches the consumer on or off.
  By default this value is set to true.

**ID**
  ID allows this producer to be found by other plugins by name.
  By default this is set to "" which does not register this producer.

**Channel**
  Channel sets the size of the channel used to communicate messages.
  By default this value is set to 8192.

**ChannelTimeoutMs**
  ChannelTimeoutMs sets a timeout in milliseconds for messages to wait if this producer's queue is full.
  A timeout of -1 or lower will drop the message without notice.
  A timeout of 0 will block until the queue is free.
  This is the default.
  A timeout of 1 or higher will wait x milliseconds for the queues to become available again.
  If this does not happen, the message will be send to the retry channel.

**ShutdownTimeoutMs**
  ShutdownTimeoutMs sets a timeout in milliseconds that will be used to detect a blocking producer during shutdown.
  By default this is set to 3 seconds.
  If processing a message takes longer to process than this duration, messages will be dropped during shutdown.

**Stream**
  Stream contains either a single string or a list of strings defining the message channels this producer will consume.
  By default this is set to "*" which means "listen to all streams but the internal".

**DropToStream**
  DropToStream defines the stream used for messages that are dropped after a timeout (see ChannelTimeoutMs).
  By default this is _DROPPED_.

**Formatter**
  Formatter sets a formatter to use.
  Each formatter has its own set of options which can be set here, too.
  By default this is set to format.Forward.
  Each producer decides if and when to use a Formatter.

**Filter**
  Filter sets a filter that is applied before formatting, i.e. before a message is send to the message queue.
  If a producer requires filtering after formatting it has to define a separate filter as the producer decides if and where to format.

**Fuse**
  Fuse defines the name of a fuse to burn if e.g. the producer encounteres a lost connection.
  Each producer defines its own fuse breaking logic if necessary / applyable.
  Disable fuse behavior for a producer by setting an empty  name or a FuseTimeoutSec <= 0.
  By default this is set to "".

**FuseTimeoutSec**
  FuseTimeoutSec defines the interval in seconds used to check if the fuse can be recovered.
  Note that automatic fuse recovery logic depends on each producer's implementation.
  By default this setting is set to 10.

**ClientId**
  ClientId sets the client id of this producer.
  By default this is "gollum".

**Partitioner**
  Partitioner sets the distribution algorithm to use.
  Valid values are: "Random","Roundrobin" and "Hash".
  By default "Roundrobin" is set.

**KeyFormatter**
  KeyFormatter can define a formatter that extracts the key for a kafka message from the message payload.
  By default this is an empty string, which disables this feature.
  A good formatter for this can be format.Identifier.

**RequiredAcks**
  RequiredAcks defines the acknowledgment level required by the broker.
  0 = No responses required.
  1 = wait for the local commit.
  -1 = wait for all replicas to commit.
  >1 = wait for a specific number of commits.
  By default this is set to 1.

**TimeoutMs**
  TimeoutMs denotes the maximum time the broker will wait for acks.
  This setting becomes active when RequiredAcks is set to wait for multiple commits.
  By default this is set to 10 seconds.

**SendRetries**
  SendRetries defines how many times to retry sending data before marking a server as not reachable.
  By default this is set to 0.

**Compression**
  Compression sets the method of compression to use.
  Valid values are: "None","Zip" and "Snappy".
  By default "None" is set.

**MaxOpenRequests**
  MaxOpenRequests defines the number of simultanious connections are allowed.
  By default this is set to 5.

**BatchMinCount**
  BatchMinCount sets the minimum number of messages required to trigger a flush.
  By default this is set to 1.

**BatchMaxCount**
  BatchMaxCount defines the maximum number of messages processed per request.
  By default this is set to 0 for "unlimited".

**BatchSizeByte**
  BatchSizeByte sets the minimum number of bytes to collect before a new flush is triggered.
  By default this is set to 8192.

**BatchSizeMaxKB**
  BatchSizeMaxKB defines the maximum allowed message size.
  By default this is set to 1024.

**BatchTimeoutMs**
  BatchTimeoutMs sets the minimum time in milliseconds to pass after wich a new flush will be triggered.
  By default this is set to 3.

**MessageBufferCount**
  MessageBufferCount sets the internal channel size for the kafka client.
  By default this is set to 8192.

**ServerTimeoutSec**
  ServerTimeoutSec defines the time after which a connection is set to timed out.
  By default this is set to 30 seconds.

**SendTimeoutMs**
  SendTimeoutMs defines the number of milliseconds to wait for a server to resond before triggering a timeout.
  Defaults to 250.

**ElectRetries**
  ElectRetries defines how many times to retry during a leader election.
  By default this is set to 3.

**ElectTimeoutMs**
  ElectTimeoutMs defines the number of milliseconds to wait for the cluster to elect a new leader.
  Defaults to 250.

**GracePeriodMs**
  GracePeriodMs defines the number of milliseconds to wait for Sarama to accept a single message.
  After this period a message is dropped.
  By default this is set to 100ms.

**MetadataRefreshMs**
  MetadataRefreshMs set the interval in seconds for fetching cluster metadata.
  By default this is set to 600000 (10 minutes).
  This corresponds to the JVM setting `topic.metadata.refresh.interval.ms`.

**Servers**
  Servers contains the list of all kafka servers to connect to.
   By default this is set to contain only "localhost:9092".

**Topic**
  Topic maps a stream to a specific kafka topic.
  You can define the wildcard stream (*) here, too.
  If defined, all streams that do not have a specific mapping will go to this topic (including _GOLLUM_).
  If no topic mappings are set the stream names will be used as topic.

Example
-------

.. code-block:: yaml

	- "producer.Kafka":
	    Enable: true
	    ID: ""
	    Channel: 8192
	    ChannelTimeoutMs: 0
	    ShutdownTimeoutMs: 3000
	    Formatter: "format.Forward"
	    Filter: "filter.All"
	    DropToStream: "_DROPPED_"
	    Fuse: ""
	    FuseTimeoutSec: 5
	    Stream:
	        - "foo"
	        - "bar"
	    ClientId: "weblog"
	    Partitioner: "Roundrobin"
	    RequiredAcks: 1
	    TimeoutMs: 1500
	    GracePeriodMs: 10
	    SendRetries: 0
	    Compression: "None"
	    MaxOpenRequests: 5
	    MessageBufferCount: 256
	    BatchMinCount: 1
	    BatchMaxCount: 0
	    BatchSizeByte: 8192
	    BatchSizeMaxKB: 1024
	    BatchTimeoutMs: 3000
	    ServerTimeoutSec: 30
	    SendTimeoutMs: 250
	    ElectRetries: 3
	    ElectTimeoutMs: 250
	    MetadataRefreshMs: 10000
	    Servers:
	        - "localhost:9092"
	    Topic:
	        "console" : "console"
