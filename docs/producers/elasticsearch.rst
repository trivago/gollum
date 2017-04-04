ElasticSearch
=============

The ElasticSearch producer sends messages to elastic search using the bulk http API.
This producer uses a fuse breaker when cluster health reports a "red" status or the connection is down.


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
  Fuse defines the name of a fuse to burn if e.g. the producer encounters a lost connection.
  Each producer defines its own fuse breaking logic if necessary / applyable.
  Disable fuse behavior for a producer by setting an empty  name or a FuseTimeoutSec <= 0.
  By default this is set to "".

**FuseTimeoutSec**
  FuseTimeoutSec defines the interval in seconds used to check if the fuse can be recovered.
  Note that automatic fuse recovery logic depends on each producer's implementation.
  By default this setting is set to 10.

**RetrySec**
  RetrySec denotes the time in seconds after which a failed dataset will be transmitted again.
  By default this is set to 5.

**Connections**
  Connections defines the number of simultaneous connections allowed to a elasticsearch server.
  This is set to 6 by default.

**TTL**
  TTL defines the TTL set in elasticsearch messages.
  By default this is set to "" which means no TTL.

**DayBasedIndex**
  DayBasedIndex can be set to true to append the date of the message to the index as in "<index>_YYYY-MM-DD".
  By default this is set to false.

**Servers**
  Servers defines a list of servers to connect to.
  The first server in the list is used as the server passed to the "Domain" setting.
  The Domain setting can be overwritten, too.

**Port**
  Port defines the elasticsearch port, which has to be the same for all servers.
  By default this is set to 9200.

**User**
  User and Password can be used to pass credentials to the elasticsearch server.
  By default both settings are empty.

**Index**
  Index maps a stream to a specific index.
  You can define the wildcard stream (*) here, too.
  If set all streams that do not have a specific mapping will go to this stream (including _GOLLUM_).
  If no category mappings are set the stream name is used.

**Type**
  Type maps a stream to a specific type.
  This behaves like the index map and is used to assign a _type to an elasticsearch message.
  By default the type "log" is used.

**DataTypes**
  DataTypes allows to define elasticsearch type mappings for indexes that are being created by this producer (e.g. day based indexes).
  You can define mappings per index.

**Settings**
  Settings allows to define elasticsearch index settings for indexes that are being created by this producer (e.g. day based indexes).
  You can define settings per index.

**BatchSizeByte**
  BatchSizeByte defines the size in bytes required to trigger a flush.
  By default this is set to 32768 (32KB).

**BatchMaxCount**
  BatchMaxCount defines the number of documents required to trigger a flush.
  By default this is set to 256.

**BatchTimeoutSec**
  BatchTimeoutSec defines the time in seconds after which a flush will be triggered.
  By default this is set to 5.

Example
-------

.. code-block:: yaml

	- "producer.ElasticSearch":
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
	    Connections: 6
	    RetrySec: 5
	    TTL: ""
	    DayBasedIndex: false
	    User: ""
	    Password: ""
	    BatchSizeByte: 32768
	    BatchMaxCount: 256
	    BatchTimeoutSec: 5
	    Port: 9200
	    Servers:
	        - "localhost"
	    Index:
	        "console" : "console"
	        "_GOLLUM_"  : "_GOLLUM_"
	    Settings:
	        "console":
	            "number_of_shards": 1
	    DataTypes:
	        "console":
	            "source": "ip"
	    Type:
	        "console" : "log"
	        "_GOLLUM_"  : "log"
