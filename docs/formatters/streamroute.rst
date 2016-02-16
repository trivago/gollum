StreamRoute
===========

StreamRoute is a formatter that modifies a message's stream by reading a prefix from the message's data (and discarding it).
The prefix is defined by everything before a given delimiter in the message.
If no delimiter is found or the prefix is empty the message stream is not changed.


Parameters
----------

**StreamRouteFormatter**
  StreamRouteFormatter defines the formatter applied after reading the stream.
  This formatter is applied to the data after StreamRouteDelimiter.
  By default this is set to "format.Forward".

**StreamRouteStreamFormatter**
  StreamRouteStreamFormatter is used when StreamRouteFormatStream is set to true.
  By default this is the same value as StreamRouteFormatter.

**StreamRouteDelimiter**
  StreamRouteDelimiter defines the delimiter to search when extracting the stream name.
  By default this is set to ":".

**StreamRouteFormatStream**
  StreamRouteFormatStream can be set to true to apply StreamRouteFormatter to both parts of the message (stream and data).
  Set to false by default.

Example
-------

.. code-block:: yaml

- "stream.Broadcast":
    Formatter: "format.StreamRoute"
    StreamRouteFormatter: "format.Forward"
    StreamRouteStreamFormatter: "format.Forward"
    StreamRouteDelimiter: "$"
    StreamRouteFormatBoth: false
