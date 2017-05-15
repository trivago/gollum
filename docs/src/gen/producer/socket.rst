Socket
======

The socket producer connects to a service over a TCP, UDP or unix domain socket based connection.


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

**Address**
  Address stores the identifier to connect to.
  This can either be any ip address and port like "localhost:5880" or a file like "unix:///var/gollum.socket".
  By default this is set to ":5880".

**ConnectionBufferSizeKB**
  ConnectionBufferSizeKB sets the connection buffer size in KB.
  By default this is set to 1024, i.e. 1 MB buffer.

**BatchMaxCount**
  BatchMaxCount defines the maximum number of messages that can be buffered before a flush is mandatory.
  If the buffer is full and a flush is still underway or cannot be triggered out of other reasons, the producer will block.
  By default this is set to 8192.

**BatchFlushCount**
  BatchFlushCount defines the number of messages to be buffered before they are written to disk.
  This setting is clamped to BatchMaxCount.
  By default this is set to BatchMaxCount / 2.

**BatchTimeoutSec**
  BatchTimeoutSec defines the maximum number of seconds to wait after the last message arrived before a batch is flushed automatically.
  By default this is set to 5.

**Acknowledge**
  Acknowledge can be set to a non-empty value to expect the given string as a response from the server after a batch has been sent.
  This setting is disabled by default, i.e. set to "".
  If Acknowledge is enabled and a IP-Address is given to Address, TCP is used to open the connection, otherwise UDP is used.

**AckTimeoutMs**
  AckTimeoutMs defines the time in milliseconds to wait for a response from the server.
  After this timeout the send is marked as failed.
  Defaults to 2000.

Example
-------

.. code-block:: yaml

	- "producer.Socket":
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
	    Enable: true
	    Address: ":5880"
	    ConnectionBufferSizeKB: 1024
	    BatchMaxCount: 8192
	    BatchFlushCount: 4096
	    BatchTimeoutSec: 5
	    Acknowledge: ""
