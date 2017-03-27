Http
====

This consumer opens up an HTTP 1.1 server and processes the contents of any incoming HTTP request.
When attached to a fuse, this consumer will return error 503 in case that fuse is burned.


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
  Address stores the host and port to bind to.
  This is allowed be any ip address/dns and port like "localhost:5880".
  By default this is set to ":80".

**ReadTimeoutSec**
  ReadTimeoutSec specifies the maximum duration in seconds before timing out the HTTP read request.
  By default this is set to 3 seconds.

**WithHeaders**
  WithHeaders can be set to false to only read the HTTP body instead of passing the whole HTTP message.
  By default this setting is set to true.

**Htpasswd**
  Htpasswd can be set to the htpasswd formatted file to enable HTTP BasicAuth.

**BasicRealm**
  BasicRealm can be set for HTTP BasicAuth.

**Certificate**
  Certificate defines a path to a root certificate file to make this consumer handle HTTPS connections.
  Left empty by default (disabled).
  If a Certificate is given, a PrivateKey must be given, too.

**PrivateKey**
  PrivateKey defines a path to the private key used for HTTPS connections.
  Left empty by default (disabled).
  If a Certificate is given, a PrivatKey must be given, too.

Example
-------

.. code-block:: yaml

	- "consumer.Http":
	    Enable: true
	    ID: ""
	    Fuse: ""
	    Stream:
	        - "foo"
	        - "bar"
	    Address: ":80"
	    ReadTimeoutSec: 3
	    WithHeaders: true
	    Htpasswd: ""
	    BasicRealm: ""
	    Certificate: ""
	    PrivateKey: ""
