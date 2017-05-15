Timestamp
=========

Timestamp is a formatter that allows prefixing a message with a timestamp (time of arrival at gollum) as well as postfixing it with a delimiter string.


Parameters
----------

**Timestamp**
  Timestamp defines a Go time format string that is used to format the actual timestamp that prefixes the message.
  By default this is set to "2006-01-02 15:04:05 MST | ".

**TimestampFormatter**
  TimestampFormatter defines the formatter for the data transferred as message.
  By default this is set to "format.Forward" .

Example
-------

.. code-block:: yaml

	- "stream.Broadcast":
	    Formatter: "format.Timestamp"
	    TimestampFormatter: "format.Envelope"
	    Timestamp: "2006-01-02T15:04:05.000 MST | "
