KafkaProducer
=============

NOTICE: This producer is not included in standard builds.
To enable it you need to trigger a custom build with native plugins enabled.
The kafka producer writes messages to a kafka cluster.
This producer is backed by the native librdkafka (0.8.6) library so most settings relate to that library.
This producer does not implement a fuse breaker.

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

**SendRetries**
  SendRetries is mapped to message.send.max.retries.
  This defines the number of times librdkafka will try to re-send a message if it did not succeed.
  Set to 0 by default (don't retry).

**Compression**
  Compression is mapped to compression.codec.
  Please note that "zip" has to be used instead of "gzip".
  Possible values are "none", "zip" and "snappy".
  By default this is set to "none".

**TimeoutMs**
  TimeoutMs is mapped to request.timeout.ms.
  This defines the number of milliseconds to wait until a request is marked as failed.
  By default this is set to 1.5sec.

**BatchSizeMaxKB**
  BatchSizeMaxKB is mapped to message.max.bytes (x1024).
  This defines the maximum message size in KB.
  By default this is set to 1 MB.
  Messages above this size are rejected.

**BatchMaxMessages**
  BatchMaxMessages is mapped to queue.buffering.max.messages.
  This defines the maximum number of messages that can be pending at any given moment in time.
  If this limit is hit additional messages will be rejected.
  This value is set to 100.000 by default and should be adjusted according to your average message throughput.

**BatchMinMessages**
  BatchMinMessages is mapped to batch.num.messages.
  This defines the minimum number of messages required for a batch to be sent.
  This is set to 1000 by default and should be significantly lower than BatchMaxMessages to avoid messages to be rejected.

**BatchTimeoutMs**
  BatchTimeoutMs is mapped to queue.buffering.max.ms.
  This defines the number of milliseconds to wait until a batch is flushed to kafka.
  Set to 1sec by default.

**ServerTimeoutSec**
  ServerTimeoutSec is mapped to socket.timeout.ms.
  Defines the time in seconds after a server is defined as "not reachable".
  Set to 1 minute by default.

**ServerMaxFails**
  ServerMaxFails is mapped to socket.max.fails.
  Number of retries after a server is marked as "failing".

**MetadataTimeoutMs**
  MetadataTimeoutMs is mapped to metadata.request.timeout.ms.
  Number of milliseconds a metadata request may take until considered as failed.
  Set to 1.5 seconds by default.

**MetadataRefreshMs**
  MetadataRefreshMs is mapped to topic.metadata.refresh.interval.ms.
  Interval in milliseconds for querying metadata.
  Set to 5 minutes by default.

**SecurityProtocol**
  SecurityProtocol is mapped to security.protocol.
  Protocol used to communicate with brokers.
  Set to plaintext by default.

**SslCipherSuites**
  SslCipherSuites is mapped to ssl.cipher.suites.
  Cipher Suites to use when connection via TLS/SSL.
  Not set by default.

**SslKeyLocation**
  SslKeyLocation is mapped to ssl.key.location.
  Path to client's private key (PEM) for used for authentication.
  Not set by default.

**SslKeyPassword**
  SslKeyPassword is mapped to ssl.key.password.
  Private key passphrase.
  Not set by default.

**SslCertificateLocation**
  SslCertificateLocation is mapped to ssl.certificate.location.
  Path to client's public key (PEM) used for authentication.
  Not set by default.

**SslCaLocation**
  SslCaLocation is mapped to ssl.ca.location.
  File or directory path to CA certificate(s) for verifying the broker's key.
  Not set by default.

**SslCrlLocation**
  SslCrlLocation is mapped to ssl.crl.location.
  Path to CRL for verifying broker's certificate validity.
  Not set by default.

**SaslMechanism**
  SaslMechanism is mapped to sasl.mechanisms.
  SASL mechanism to use for authentication.
  Not set by default.

**SaslUsername**
  SaslUsername is mapped to sasl.username.
  SASL username for use with the PLAIN mechanism.
  Not set by default.

**SaslPassword**
  SaslPassword is mapped to sasl.password.
  SASL password for use with the PLAIN mechanism.
  Not set by default.

**Servers**
  Servers defines the list of brokers to produce messages to.

**Topic**
  Topic defines a stream to topic mapping.
  If a stream is not mapped a topic named like the stream is assumed.

**KeyFormatter**
  KeyFormatter can define a formatter that extracts the key for a kafka message from the message payload.
  By default this is an empty string, which disables this feature.
  A good formatter for this can be format.Identifier.

**KeyFormatterFirst**
  KeyFormatterFirst can be set to true to apply the key formatter to the unformatted message.
  By default this is set to false, so that key formatter uses the message after Formatter has been applied.
  KeyFormatter does never affect the payload of the message sent to kafka.

**FilterAfterFormat**
  FilterAfterFormat behaves like Filter but allows filters to be executed after the formatter has run.
  By default no such filter is set.

Example
-------

.. code-block:: yaml

	- "native.KafkaProducer":
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
	    RequiredAcks: 1
	    TimeoutMs: 1500
	    SendRetries: 0
	    Compression: "none"
	    BatchSizeMaxKB: 1024
	    BatchMaxMessages: 100000
	    BatchMinMessages: 1000
	    BatchTimeoutMs: 1000
	    ServerTimeoutSec: 60
	    ServerMaxFails: 3
	    MetadataTimeoutMs: 1500
	    MetadataRefreshMs: 300000
	    SecurityProtocol: "plaintext"
	    SslCipherSuites: ""
	    SslKeyLocation: ""
	    SslKeyPassword: ""
	    SslCertificateLocation: ""
	    SslCaLocation: ""
	    SslCrlLocation: ""
	    SaslMechanism: ""
	    SaslUsername: ""
	    SaslPassword: ""
	    KeyFormatter: ""
	    Servers:
	        - "localhost:9092"
	    Topic:
	        "console" : "console"
