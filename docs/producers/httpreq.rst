HTTPRequest
===========

This producer sends messages that already are valid http request to a given webserver.
This producer uses a fuse breaker when a request fails with an error code > 400 or the connection is down.
See the `API documentation <http://gollum.readthedocs.org/en/latest/producers/httpreq.html>`_ for additional details.

Parameters
----------

**Enable**
  Can either be true or false to enable or disable this producer.
**ID**
  Allows this producer to be found by other plugins by name.
  By default this is set to "" which does not register this producer.
**Fuse**
  Defines the name of the fuse this producer is attached to.
  When left empty no fuse is attached. This is the default value.
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
**RawData**
  Switches between creating POST data from the incoming message (false) and passing the message as HTTP request without changes (true).
  This setting is enabled by default.
**Encoding**
  Defines the payload encoding when RawData is set to false.
  Set to "text/plain; charset=utf-8" by default.

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
  Defines the server address to connect to.
  This can be any ip address and port like "localhost:5880". By default this is set to ":80".

Example
-------

.. code-block:: yaml

  - "producer.HTTPRequest":
    Enable: true
    Address: "testing:80"
    RawData: true
    Encoding: "text/plain; charset=utf-8"
    Stream: "http"
