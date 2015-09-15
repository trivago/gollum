Proxy
=====

The proxy consumer reads messages directly as-is from a given socket.
Messages are separated from the stream by using a specific paritioner method.
In combination with a compatible proxy producer like the :doc:`Proxy Producer </producers/proxy>` this can be used to create a proxy style two-way communication over Gollum.
When attached to a fuse, this consumer will stop accepting new connections and close all existing connections in case that fuse is burned.
See the `API documentation <http://gollum.readthedocs.org/en/latest/consumers/proxy.html>`_ for additional details.

Parameters
----------

**Enable**
  Can either be true or false to enable or disable this consumer.
**ID**
  Allows this consumer to be found by other plugins by name.
  By default this is set to "" which does not register this consumer.
**Fuse**
  Defines the name of the fuse this consumer is attached to.
  When left empty no fuse is attached. This is the default value.
**Stream**
  Defines either one or an aray of stream names this consumer sends messages to.
**Address**
  Defines the protocol, address/DNS and port to listen to.
  The protocol can either be "socket://" for unix domain or "tcp://" for TCP. UDP sockets cannot be used.
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

  - "consumer.Proxy":
    Enable: true
    Address: "unix:///var/gollum.socket"
    Partitioner: "ascii"
    Delimiter: ":"
    Offset: 1
    Stream:
      - "external"
      - "socket"
