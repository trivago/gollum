Envelope
========

Envelope is a formatter that allows prefixing and/or postfixing a message with configurable strings.


Parameters
----------

**EnvelopePrefix**
  EnvelopePrefix defines the message prefix.
  By default this is set to "".
  Special characters like \n \r \t will be transformed into the actual control characters.

**EnvelopePostfix**
  EnvelopePostfix defines the message postfix.
  By default this is set to "\n".
  Special characters like \n \r \t will be transformed into the actual control characters.

**EnvelopeFormatter**
  EnvelopeFormatter defines the formatter for the data transferred as message.
  By default this is set to "format.Forward" .

Example
-------

.. code-block:: yaml

	- "stream.Broadcast":
	    Formatter: "format.Envelope"
	    EnvelopeFormatter: "format.Forward"
	    EnvelopePrefix: ""
	    EnvelopePostfix: "\n"
