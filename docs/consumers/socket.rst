Socket
======

The socket consumer listens to an arbitrary port.
Messages are separated from the stream by using a specific paritioner method.
In combination with the :doc:`Socket Producer </producers/socket>` this can be used to built Gollum based message networks.

Parameters
----------

**Enable**
    Can either be true or false to enable or disable this consumer.
**Stream**
    Defines either one or an aray of stream names this consumer sends messages to.
**Address**
  Defines the protocol, address/DNS and port to listen to.
  The protocol can either be "socket://" for unix domain, "tcp://" for TCP or "udp://" for UDP sockets.
  In addtion to that, any protocol supported by `net.Dial <http://golang.org/pkg/net/#Dial>`_ is possible here.
**Acknowledge**
  When set to a non-empty value, the socket consumer will send the given string after recieving a message or batch of messages.
  Acknowledge is disabled by default, i.e. set to "".
  If Acknowledge is enabled and a IP-Address is given to Address, TCP is enforced to open the connection.
**Partitioner**
  The partitioner defines the algorithm used to separate messages from the stream.
  By default this is set to "delimiter".
   - "delimiter" separates messages by looking for a delimiter string. Thedelimiter is removed from the message.
   - "ascii" reads an ASCII encoded number at a given offset until a given delimiter is found. Everything left from and including the delimiter is removed from the message.
   - "binary" reads a binary number at a given offset and size
   - "binary_le" is an alias for "binary"
   - "binary_be" is the same as "binary" but uses big endian encoding
   - "fixed" assumes fixed size messages
**Delimiter**
  Defines the delimiter used by the "text" and "delimiter" partitioner.
  By default this is set to "\n".
**Offset**
  Defines the offset in bytes used by the binary and text paritioner.
**Size**
  Size defines the size in bytes used by the binary or fixed partitioner.
  For binary this can be set to 1,2,4 or 8. By default 4 is chosen.
  For fixed this defines the size of a message. By default 1 is chosen.

Example
-------

.. code-block:: yaml

  - "consumer.Socket":
    Enable: true
    Address: "unix:///var/gollum.socket"
    Acknowledge: "OK"
    Partitioner: "ascii"
    Delimiter: ":"
    Offset: 1
    Stream:
      - "external"
      - "socket"
