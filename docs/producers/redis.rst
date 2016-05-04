Redis
=====

This producer sends data to a redis server.
Different redis storage types and database indexes are supported.
This producer does not implement support for redis 3.0 cluster.


Parameters
----------

**Enable**
  Enable switches the consumer on or off.
  By default this value is set to true.

**ID**
  ID allows this producer to be found by other plugins by name.
  By default this is set to "" which does not register this producer.

**Channel**
  Channel sets the size of the channel used to communicate messages.
  By default this value is set to 8192.

**ChannelTimeoutMs**
  ChannelTimeoutMs sets a timeout in milliseconds for messages to wait if this producer's queue is full.
  A timeout of -1 or lower will drop the message without notice.
  A timeout of 0 will block until the queue is free.
  This is the default.
  A timeout of 1 or higher will wait x milliseconds for the queues to become available again.
  If this does not happen, the message will be send to the retry channel.

**ShutdownTimeoutMs**
  ShutdownTimeoutMs sets a timeout in milliseconds that will be used to detect a blocking producer during shutdown.
  By default this is set to 3 seconds.
  If processing a message takes longer to process than this duration, messages will be dropped during shutdown.

**Stream**
  Stream contains either a single string or a list of strings defining the message channels this producer will consume.
  By default this is set to "*" which means "listen to all streams but the internal".

**DropToStream**
  DropToStream defines the stream used for messages that are dropped after a timeout (see ChannelTimeoutMs).
  By default this is _DROPPED_.

**Formatter**
  Formatter sets a formatter to use.
  Each formatter has its own set of options which can be set here, too.
  By default this is set to format.Forward.
  Each producer decides if and when to use a Formatter.

**Filter**
  Filter sets a filter that is applied before formatting, i.e. before a message is send to the message queue.
  If a producer requires filtering after formatting it has to define a separate filter as the producer decides if and where to format.

**Fuse**
  Fuse defines the name of a fuse to burn if e.g. the producer encounteres a lost connection.
  Each producer defines its own fuse breaking logic if necessary / applyable.
  Disable fuse behavior for a producer by setting an empty  name or a FuseTimeoutSec <= 0.
  By default this is set to "".

**FuseTimeoutSec**
  FuseTimeoutSec defines the interval in seconds used to check if the fuse can be recovered.
  Note that automatic fuse recovery logic depends on each producer's implementation.
  By default this setting is set to 10.

**Address**
  Address stores the identifier to connect to.
  This can either be any ip address and port like "localhost:6379" or a file like "unix:///var/redis.socket".
  By default this is set to ":6379".
  This producer does not implement a fuse breaker.

**Database**
  Database defines the redis database to connect to.
  By default this is set to 0.

**Key**
  Key defines the redis key to store the values in.
  By default this is set to "default".

**Storage**
  Storage defines the type of the storage to use.
  Valid values are: "hash", "list", "set", "sortedset", "string".
  By default this is set to "hash".

**FieldFormatter**
  FieldFormatter defines an extra formatter used to define an additional field or score value if required by the storage type.
  If no field value is required this value is ignored.
  By default this is set to "format.Identifier".

**FieldAfterFormat**
  FieldAfterFormat will send the formatted message to the FieldFormatter if set to true.
  If this is set to false the message will be send to the FieldFormatter before it has been formatted.
  By default this is set to false.

**KeyFormatter**
  KeyFormatter defines an extra formatter used to allow generating the key from a message. 
  If this value is set the "Key" field will be ignored. 
  By default this field is not used.

**KeyAfterFormat**
  KeyAfterFormat will send the formatted message to the keyFormatter if set to true. 
  If this is set to false the message will be send to the keyFormatter before it has been formatted. 
  By default this is set to false.

Example
-------

.. code-block:: yaml

	- "producer.Redis":
	    Enable: true
	    ID: ""
	    Channel: 8192
	    ChannelTimeoutMs: 0
	    ShutdownTimeoutMs: 3000
	    Formatter: "format.Forward"
	    Filter: "filter.All"
	    DropToStream: "_DROPPED_"
	    Fuse: ""
	    FuseTimeoutSec: 5
	    Stream:
	        - "foo"
	        - "bar"
	    Address: ":6379"
	    Database: 0
	    Key: "default"
	    Storage: "hash"
	    FieldFormatter: "format.Identifier"
	    FieldAfterFormat: false
	    KeyFormatter: "format.Forward"
	    KeyAfterFormat: false
