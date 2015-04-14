Socket
#############

The socket consumer listens to an arbitrary port.
Messages are read from a minimalistic, ASCII based protocol that is compatible to the :doc:`Runlength </formatters/runlength>` and  :doc:`Sequence </formatters/sequence>` formatter.
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
  When set to true, the socket consumer will send "OK\n" after recieving a message or batch of messages.
  Set to false by default.
**Runlength**
  When set to true the message parser expects a runlength at the start of the message. See :doc:`Runlength formatter </formatters/runlength>` for additional details.
  Set to false by default.
**Sequence**
  When set to true the message parser expects a sequence at the start of the message (after the runlength). See :doc:`Sequence formatter </formatters/sequence>` for additional details.
  Set to false by default.
**Delimiter**
  Defines a delimiter that marks the end of a message. Only valid if Runlength is set to true.
  Set to "\n" by default.

Example
-------

::

  - "consumer.Socket":
    Enable: true
    Address: "unix:///var/gollum.socket"
    Acknowledge: true
    Runlength: true
    Sequence: true
    Delimiter: "\n"
    Stream:
      - "external"
      - "socket"
