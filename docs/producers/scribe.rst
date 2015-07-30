Facebook Scribe
===============

This producers sends messages to a scribe server (fb303).

Parameters
----------

**Enable**
  Can either be true or false to enable or disable this producer.
**ID**
  Allows this producer to be found by other plugins by name.
  By default this is set to "" which does not register this producer.
**Stream**
  Defines either one or an aray of stream names this producer recieves messages from.
**DropToStream**
  Defines the stream used for messages that are dropped after a timeout (see ChannelTimeoutMs).
  By default this is _DROPPED_.
**Channel**
  Defines the number of messages that can be buffered by the internal channel.
  By default this is set to 8192.
**ChannelTimeoutMs**
  Defines a timeout in milliseconds for messages to wait if this producer's queue is full.

  - A timeout of -1 or lower will discard the message without notice.
  - A timeout of 0 will block until the queue is free. This is the default.
  - A timeout of 1 or higher will wait n milliseconds for the queues to become available again.
    If this does not happen, the message will be send to the _DROPPED_ stream that can be processed by the :doc:`Loopback </consumers/loopback>` consumer.

**FlushTimeoutSec**
  Sets the maximum number of seconds to wait before a flush is aborted during shutdown.
  By default this is set to 0, which does not abort the flushing procedure.
**Format**
  Defines a message formatter to use. :doc:`Format.Forward </formatters/forward>` by default.
**Address**
  Defines the redis server address to connect to.
  This can be any ip address and port like "localhost:6379".
  By default this is set to ":6379".
**ConnectionBufferSizeKB**
  Sets the connection buffer size in KB.
  By default this is set to 1024, i.e. 1 MB buffer.
**BatchMaxCount**
  Defines the maximum number of messages that can be buffered before a flush is mandatory.
  If the buffer is full and a flush is still underway or cannot be triggered out of other reasons, the producer will block.
**BatchFlushCount**
  Defines the number of messages to be buffered before they are written to disk.
  This setting is clamped to BatchMaxCount.
  By default this is set to BatchMaxCount / 2.
**BatchTimeoutSec**
  Defines the number of seconds to wait after a message before a flush is triggered.
  The timer is reset after each new message.
  By default this is set to 5.
**Category**
  Maps a stream to a specific scribe category.
  If you define a mapping on "*" all streams that do not have a specific mapping will go to this category (including internal streams).
  If no mapping to "*" is set the stream name is used as category.

Example
-------

.. code-block:: yaml

  - "producer.Scribe":
    Enable: true
    Channel: 8192
    ChannelTimeoutMs: 100
    Address: "192.168.222.30:1463"
    ConnectionBufferSizeKB: 4096
    BatchSizeMaxKB: 16384
    BatchSizeByte: 4096
    BatchTimeoutSec: 2
    Category:
      "log" : "logs"
      "console"  : "user"
    Stream:
        - "log"
        - "console"
