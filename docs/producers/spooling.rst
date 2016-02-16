Spooling
========

The Spooling producer buffers messages and sends them again to the previous stream stored in the message.
This means the message must have been routed at least once before reaching the spooling producer.
If the previous and current stream is identical the message is dropped.
The Formatter configuration value is forced to "format.Serialize" and cannot be changed.
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

**Path**
  Path sets the output directory for spooling files.
  Spooling files will Files will be stored as "<path>/<stream>/<number>.spl".
  By default this is set to "/var/run/gollum/spooling".

**BatchMaxCount**
  BatchMaxCount defines the maximum number of messages stored in memory before a write to file is triggered.
  Set to 100 by default.

**BatchTimeoutSec**
  BatchTimeoutSec defines the maximum number of seconds to wait after the last message arrived before a batch is flushed automatically.
  By default this is set to 5.

**MaxFileSizeMB**
  MaxFileSizeMB sets the size in MB when a spooling file is rotated.
  Reading will start only after a file is rotated.
  Set to 512 MB by default.

**MaxFileAgeMin**
  MaxFileAgeMin defines the time in minutes after a spooling file is rotated.
  Reading will start only after a file is rotated.
  This setting divided by two will be used to define the wait time for reading, too.
  Set to 1 minute by default.

**BufferSizeByte**
  BufferSizeByte defines the initial size of the buffer that is used to parse messages from a spool file.
  If a message is larger than this size, the buffer will be resized.
  By default this is set to 8192.

**RespoolDelaySec**
  RespoolDelaySec sets the number of seconds to wait before trying to load existing spool files after a restart.
  This is useful for configurations that contain dynamic streams.
  By default this is set to 10.

**MaxMessagesSec**
  MaxMessagesSec sets the maximum number of messages that can be respooled per second.
  By default this is set to 100.
  Setting this value to 0 will cause respooling to work as fast as possible.

Example
-------

.. code-block:: yaml

- "producer.Spooling":
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
    Path: "/var/run/gollum/spooling"
    BatchMaxCount: 100
    BatchTimeoutSec: 5
    MaxFileSizeMB: 512
    MaxFileAgeMin: 1
    MessageSizeByte: 8192
    RespoolDelaySec: 10
    MaxMessagesSec: 100
