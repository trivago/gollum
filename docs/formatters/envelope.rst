Envelope
========

Envelope is a formatter that allows prefixing and/or postfixing a message with configurable strings.


Parameters
----------

**Prefix**
  Prefix defines the message prefix.
  By default this is set to "".
  Special characters like \n \r \t will be transformed into the actual control characters.

**Postfix**
  Postfix defines the message postfix.
  By default this is set to "\n".
  Special characters like \n \r \t will be transformed into the actual control characters.

**EnvelopeDataFormatter**
  EnvelopeDataFormatter defines the formatter for the data transferred as message.
  By default this is set to "format.Forward" .

Example
-------

.. code-block:: yaml

- "stream.Broadcast":
    Formatter: "format.Envelope"
    EnvelopeFormatter: "format.Forward"
    EnvelopePrefix: "<data>"
    EnvelopePostfix: "</data>\n"
