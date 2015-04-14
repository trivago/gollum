Socket
#############

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
  Defines the server address to connect to.
  This can either be any ip address and port like "localhost:5880" or a file
  like "unix:///var/gollum.socket". By default this is set to ":5880".
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
**Acknowledge**
  Set to true if a response "OK\n" is expected from the server after a batch has been sent.
  This correspons to the behavior of the :doc:`Socket consumer </consumers/socket>`.
  This setting is disabled by default.
  If Acknowledge is set to true and a IP-Address is given to Address, TCP is
  enforced to open the connection.

Example
-------

::

  - "producer.File":
    Enable: true
    Channel: 8192
    ChannelTimeoutMs: 100
    Address: "unix:///var/gollum.socket"
    ConnectionBufferSizeKB: 4096
    BatchSizeMaxKB: 16384
    BatchSizeByte: 4096
    BatchTimeoutSec: 5
    Acknowledge: true
    Stream:
        - "log"
        - "console"
