Serialize
=========

Serialize is a formatter that serializes a message for later retrieval.


Parameters
----------

**SerializeFormatter**
  SerializeFormatter defines the formatter for the data transferred as message.
  By default this is set to "format.Forward".

**SerializeStringEncode**
  SerializeStringEncode causes the serialized data to be base64 encoded and newline separated.
  This is enabled by default.

Example
-------

.. code-block:: yaml

	    - "stream.Broadcast":
	        Formatter: "format.Serialize"
	        SerializeFormatter: "format.Envelope"
	        SerializeStringEncode: true
