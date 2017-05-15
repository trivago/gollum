Proxy
=====

The proxy consumer reads messages directly as-is from a given socket.
Messages are extracted by standard message size algorithms (see Partitioner).
This consumer can be used with any compatible proxy producer to establish a two-way communication.


Parameters
----------

**Enable**
  Enable switches the consumer on or off.
  By default this value is set to true.

**ID**
  ID allows this consumer to be found by other plugins by name.
  By default this is set to "" which does not register this consumer.

**Stream**
  Stream contains either a single string or a list of strings defining the message channels this consumer will produce.
  By default this is set to "*" which means only producers set to consume "all streams" will get these messages.

**Address**
  Address defines the protocol, host and port or socket to bind to.
  This can either be any ip address and port like "localhost:5880" or a file like "unix:///var/gollum.socket".
  By default this is set to ":5880".
  UDP is not supported.

**Partitioner**
  Partitioner defines the algorithm used to read messages from the stream.
  The messages will be sent as a whole, no cropping or removal will take place.
  By default this is set to "delimiter".
   * "delimiter" separates messages by looking for a delimiter string. The delimiter is included into the left hand message. 
   * "ascii" reads an ASCII number at a given offset until a given delimiter is found. Everything to the right of and including the delimiter is removed from the message. 
   * "binary" reads a binary number at a given offset and size. 
   * "binary_le" is an alias for "binary". 
   * "binary_be" is the same as "binary" but uses big endian encoding. 
   * "fixed" assumes fixed size messages. 

**Delimiter**
  Delimiter defines the delimiter used by the text and delimiter partitioner.
  By default this is set to "\n".

**Offset**
  Offset defines the offset used by the binary and text partitioner.
  By default this is set to 0.
  This setting is ignored by the fixed partitioner.

**Size**
  Size defines the size in bytes used by the binary or fixed partitioner.
  For binary this can be set to 1,2,4 or 8.
  By default 4 is chosen.
  For fixed this defines the size of a message.
  By default 1 is chosen.

Example
-------

.. code-block:: yaml

	- "consumer.Proxy":
	    Enable: true
	    ID: ""
	    Stream:
	        - "foo"
	        - "bar"
	    Address: ":5880"
	    Partitioner: "delimiter"
	    Delimiter: "\n"
	    Offset: 0
	    Size: 1
