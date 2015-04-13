Identifier
#############

Identifier generates a (mostly) unqiue ID from a message.

Parameters
----------

**IdentifierType**
  Defines the algorithm to use. "time" by default.

  - "hash" hashes the message with FNV1A-64 and returns the result as HEX
  - "time" returns YYMMDDHHmmSSxxxxxxx where x denotes the sequence number modulo 10.000.000. I.e. 10mil messages per second are possible before there is a collision.
  - "seq" returns the integer representation of the message's internal sequence number
  - "seqhex" returns the HEX representation of the sequence number

Example
-------

::

  - "stream.Broadcast":
    Formatter: "format.Identifier"
    IdentifierType: "hash"
