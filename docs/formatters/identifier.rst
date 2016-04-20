Identifier
==========

Identifier is a formatter that will generate a (mostly) unique 64 bit identifier number from the message timestamp and sequence number.
The message payload will not be encoded.


Parameters
----------

**IdentifierType**
  IdentifierType defines the algorithm used to generate the message id.
  This my be one of the following: "hash", "time", "seq", "seqhex".
  By default this is set to "time".
   * When using "hash" the message payload will be hashed using fnv1a and returned as hex. 
   * When using "time" the id will be formatted YYMMDDHHmmSSxxxxxxx where x denotes the sequence number modulo 10000000. I.e. 10mil messages per second are possible before there is a collision. 
   * When using "seq" the id will be returned as the integer representation of the sequence number. 
   * When using "seqhex" the id will be returned as the hex representation of the sequence number. 

**IdentifierDataFormatter**
  IdentifierDataFormatter defines the formatter for the data that is used to build the identifier from.
  By default this is set to "format.Forward" .

Example
-------

.. code-block:: yaml

	- "stream.Broadcast":
	    Formatter: "format.Identifier"
	    IdentifierType: "hash"
