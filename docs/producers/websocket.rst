Websocket
#############

This producers writes messages to a websocket.

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

::

  - "producer.Console":
    Enable: true
    Channel: 8192
    ChannelTimeoutMs: 100
    Address: ":80"
    Path:    "/data"
    ReadTimeoutSec: 5
    Stream:
        - "log"
        - "console"
