Websocket
=========

This producers writes messages to a websocket.
This producer does not implement a fuse breaker.
See the `API documentation <http://gollum.readthedocs.org/en/latest/producers/websocket.html>`_ for additional details.

Parameters
----------

**Enable**
  Can either be true or false to enable or disable this producer.
**ID**
  Allows this producer to be found by other plugins by name.
  By default this is set to "" which does not register this producer.
**Stream**
  Defines either one or an aray of stream names this producer receives messages from.
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
**Address**
  Sets the address identifier to bind to.
  This is allowed be any IP address/dns and port like "localhost:5880".
  By default this is set to ":81".
**Path**
  Defines the URL to listen for.
  By default this is set to "/".
**ReadTimeoutSec**
  Specifies the maximum duration in seconds before timing out a request.
  By default this is set to 3 seconds.

Example
-------

.. code-block:: yaml

  - "producer.Websocket":
    Enable: true
    Channel: 8192
    ChannelTimeoutMs: 100
    Address: ":80"
    Path:    "/data"
    ReadTimeoutSec: 5
    Stream:
        - "log"
        - "console"
