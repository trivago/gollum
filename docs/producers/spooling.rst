Spooling
========

The Spooling producer buffers messages and sends them again to the previous stream stored in the message.
This means the message must have been routed at least once before reaching the spooling producer.
If the previous and current stream is identical the message is dropped.

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
**Filter**
  Defines a message filter to apply before formatting. :doc:`Filter.All </filters/all>` by default.
**RetryDelayMs**
  Denotes the number of milliseconds before a message is send again.
  The message is removed from the list after sending.
  If the time is set to 0, messages are not buffered but resent directly.
  By default this is set to 2000 (2 seconds).
**MaxMessageCount**
  Denotes the maximum number of messages to store before dropping new incoming messages.
  By default this is set to 0 which means all messages are stored.

Example
-------

.. code-block:: yaml

  - "producer.Spooling":
    Enable: true
    RetryDelayMs: 2000
    MaxMessageCount: 10000
    Stream:
        - "spooling"
