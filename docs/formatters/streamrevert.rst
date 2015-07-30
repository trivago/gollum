StreamRoute
===========

StreamRevert is a formatter that recovers the last used stream from a message and sets it as a new target stream.
Streams change whenever the Stream.Route or Message.Route function is used.
This e.g. happens after a Drop call.

Parameters
----------

**StreamRevertFormatter**
  Defines the formatter applied after reading the stream.
  This formatter is applied to the data after StreamRevertDelimiter.
  By default this is set to "format.Forward"

Example
-------

.. code-block:: yaml

  - "stream.Broadcast":
    Formatter: "format.StreamRoute"
    StreamRouteFormatter: "format.Forward"
    StreamRouteDelimiter: ":"
