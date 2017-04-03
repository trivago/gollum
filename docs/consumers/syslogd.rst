Syslogd
=======

The syslogd consumer accepts messages from a syslogd compatible socket.
When attached to a fuse, this consumer will stop the syslogd service in case that fuse is burned.


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

**Fuse**
  Fuse defines the name of a fuse to observe for this consumer.
  Producer may "burn" the fuse when they encounter errors.
  Consumers may react on this by e.g. closing connections to notify any writing services of the problem.
  Set to "" by default which disables the fuse feature for this consumer.
  It is up to the consumer implementation to react on a broken fuse in an appropriate manner.

**Address**
  Address defines the protocol, host and port or socket to bind to.
  This can either be any ip address and port like "localhost:5880" or a file like "unix:///var/gollum.socket".
  By default this is set to "udp://0.0.0.0:514".
  The protocol can be defined along with the address, e.g. "tcp://..." but this may be ignored if a certain protocol format does not support the desired transport protocol.

**Format**
  Format defines the syslog standard to expect for message encoding.
  Three standards are currently supported, by default this is set to "RFC6587".
   * RFC3164 (https://tools.ietf.org/html/rfc3164) udp only. 
   * RFC5424 (https://tools.ietf.org/html/rfc5424) udp only. 
   * RFC6587 (https://tools.ietf.org/html/rfc6587) tcp or udp. 

Example
-------

.. code-block:: yaml

	- "consumer.Syslogd":
	    Enable: true
	    ID: ""
	    Fuse: ""
	    Stream:
	        - "foo"
	        - "bar"
	    Address: "udp://0.0.0.0:514"
	    Format: "RFC6587"
