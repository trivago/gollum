Proxy
=====

This producer is compatible to consumer.proxy.
Responses to messages sent to the given address are sent back to the original consumer of it is a compatible message source.
As with consumer.proxy the returned messages are partitioned by common message length algorithms.
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

**Address**
  Address stores the identifier to connect to.
  This can either be any ip address and port like "localhost:5880" or a file like "unix:///var/gollum.Proxy".
  By default this is set to ":5880".

**ConnectionBufferSizeKB**
  ConnectionBufferSizeKB sets the connection buffer size in KB.
  This also defines the size of the buffer used by the message parser.
  By default this is set to 1024, i.e. 1 MB buffer.

**TimeoutSec**
  TimeoutSec defines the maximum time in seconds a client is allowed to take for a response.
  By default this is set to 1.

**Partitioner**
  Partitioner defines the algorithm used to read messages from the stream.
  The messages will be sent as a whole, no cropping or removal will take place.
  By default this is set to "delimiter".
   - "delimiter" separates messages by looking for a delimiter string.
  The    delimiter is included into the left hand message.
   - "ascii" reads an ASCII encoded number at a given offset until a given    delimiter is found.
   - "binary" reads a binary number at a given offset and size  - "binary_le" is an alias for "binary"  - "binary_be" is the same as "binary" but uses big endian encoding  - "fixed" assumes fixed size messages.

**Delimiter**
  Delimiter defines the delimiter used by the text and delimiter partitioner.
  By default this is set to "\n".

**Offset**
  Offset defines the offset used by the binary and text partitioner.
  By default this is set to 0.
  This setting is ignored by the fixed partitioner.

**Size**
  Size defines the size in bytes used by the binary or fixed partitioner.
  For binary this can be set to 1,2,4 or 8.
  By default 4 is chosen.
  For fixed this defines the size of a message.
  By default 1 is chosen.

Example
-------

.. code-block:: yaml

	- "producer.Proxy":
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
	    Address: ":5880"
	    ConnectionBufferSizeKB: 1024
	    TimeoutSec: 1
	    Partitioner: "delimiter"
	    Delimiter: "\n"
	    Offset: 0
	    Size: 1
