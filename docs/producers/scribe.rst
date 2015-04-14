Facebook Scribe
################

This producers sends messages to a scribe server (fb303).

Parameters
----------

**Enable**
  Can either be true or false to enable or disable this producer.
**Stream**
  Defines either one or an aray of stream names this producer recieves messages from.
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
**Address**
  Defines the redis server address to connect to.
  This can be any ip address and port like "localhost:6379".
  By default this is set to ":6379".
**ConnectionBufferSizeKB**
  Sets the connection buffer size in KB.
  By default this is set to 1024, i.e. 1 MB buffer.
**BatchSizeMaxKB**
  Defines the internal file buffer size in KB.
  This producers allocates a front- and a backbuffer of this size.
  If the frontbuffer is filled up completely a flush is triggered and the frontbuffer becomes available for writing again.
  Messages larger than BatchSizeMaxKB are rejected.
  By default this is set to 8192 (8MB)
**BatchSizeByte**
  Defines the number of bytes to be buffered before a flush is triggered.
  By default this is set to 8192 (8KB).
**BatchTimeoutSec**
  Defines the number of seconds to wait after a message before a flush is triggered.
  The timer is reset after each new message.
  By default this is set to 5.
**Category**
  Maps a stream to a specific scribe category.
  If you define a mapping on "*" all streams that do not have a specific mapping will go to this category (including internal streams).
  By default a mapping "*" to "default" is used.
  If no explicit mapping to "*" is set this mapping is preserved, i.e. streams without a mapping will use the "default" category.

Example
-------

::

  - "producer.Console":
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
