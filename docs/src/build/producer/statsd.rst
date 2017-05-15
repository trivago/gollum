Statsd
======

This producer sends increment events to a statsd server.


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

**BatchMaxMessages**
  BatchMaxMessages defines the maximum number of messages to send per batch.
  By default this is set to 500.

**BatchTimeoutSec**
  BatchTimeoutSec defines the number of seconds after which a batch is flushed automatically.
  By default this is set to 10.

**Prefix**
  Prefix defines the prefix for stats metric names.
  By default this is set to "gollum.".

**Server**
  Server defines the server and port to send statsd metrics to.
  By default this is set to "localhost:8125".

**UseMessage**
  UseMessage defines whether to cast the message to string and increment the metric by that value.
  If this is set to true and the message fails to cast to an integer, then the message with be ignored.
  If this is set to false then each message will increment by 1.
  By default this is set to false.

**StreamMapping**
  StreamMapping defines a translation from gollum stream to statsd metric name.
  If no mapping is given the gollum stream name is used as the metric name.

Example
-------

.. code-block:: yaml

	- "producer.Statsd":
	    Enable: true
	    ID: ""
	    Channel: 8192
	    ChannelTimeoutMs: 0
	    ShutdownTimeoutMs: 3000
	    Formatter: "format.Forward"
	    Filter: "filter.All"
	    DropToStream: "_DROPPED_"
	    Stream:
	        - "foo"
	        - "bar"
	    BatchMaxMessages: 500
	    BatchTimeoutSec: 10
	    Prefix: "gollum."
	    Server: "localhost:8125"
	    UseMessage: false
	    StreamMapping:
	        "*" : "default"
