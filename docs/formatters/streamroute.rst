StreamRoute
===========

StreamRoute modifies the stream of a message by searching for a prefix in the message payload.
The prefix is extracted from all characters before the first colon ":".
If no prefix is found or the prefix is empty, the message stream is not changed.
The prefix will be removed from the message.
If the stream modulation routes messages to a stream configured by another plugin, this plugin will be used.

Parameters
----------

**StreamRouteFormatter**
  Defines an additional formatter applied after removing the prefix from the message. :doc:`Format.Forward </formatters/forward>` by default.
**StreamRouteDelimiter**
  Defines the control character that is searched for when extracting the prefix from the message.
  By default this is set to ":".

Example
-------

.. code-block:: yaml

  - "stream.Broadcast":
    Formatter: "format.StreamRoute"
    StreamRouteFormatter: "format.Forward"
    StreamRouteDelimiter: ":"
