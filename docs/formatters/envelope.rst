Envelope
========

Envelope allows to pre- or postfix messages with a given string.
This formatter allows a nested formatter to further modify the message between pre- and postfix.
Prefix and Postfix may contain standard escape characters, i.e. "\r", "\n" and "\t".

Parameters
----------

**EnvelopeFormatter**
  Defines an additional formatter applied before adding pre- and postfix. :doc:`Format.Forward </formatters/forward>` by default.

**Prefix**
  Defines a string to be prepended to the message. Empty by default.

**Postfix**
  Defines a string to be appended to the message. "\n" by default.

Example
-------

.. code-block:: yaml

  - "stream.Broadcast":
    Formatter: "format.Envelope"
    EnvelopeFormatter: "format.Forward"
    Prefix: "<data>"
    Postfix: "</data>\n"
