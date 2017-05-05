Firehose
========

This producer sends data to an AWS Firehose stream.


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

**Firehose**
  Firehose defines the stream to read from.
  By default this is set to "default".

**Region**
  Region defines the amazon region of your firehose stream.
  By default this is set to "eu-west-1".

**Endpoint**
  Endpoint defines the amazon endpoint for your firehose stream.
  By default this is set to "firehose.eu-west-1.amazonaws.com".

**CredentialType**
  CredentialType defines the credentials that are to be used when connecting to firehose.
  This can be one of the following: environment, static, shared, none.
  Static enables the parameters CredentialId, CredentialToken and CredentialSecret shared enables the parameters CredentialFile and CredentialProfile.
  None will not use any credentials and environment will pull the credentials from environmental settings.
  By default this is set to none.

**BatchMaxMessages**
  BatchMaxMessages defines the maximum number of messages to send per batch.
  By default this is set to 500.

**RecordMaxMessages**
  RecordMaxMessages defines the maximum number of messages to join into a firehose record.
  By default this is set to 500.

**RecordMessageDelimiter**
  RecordMessageDelimiter defines the string to delimit messages within a firehose record.
  By default this is set to "\n".

**SendTimeframeMs**
  SendTimeframeMs defines the timeframe in milliseconds in which a second batch send can be triggered.
  By default this is set to 1000, i.e. one send operation per second.

**BatchTimeoutSec**
  BatchTimeoutSec defines the number of seconds after which a batch is flushed automatically.
  By default this is set to 3.

**StreamMapping**
  StreamMapping defines a translation from gollum stream to firehose stream name.
  If no mapping is given the gollum stream name is used as firehose stream name.

Example
-------

.. code-block:: yaml

	- "producer.Firehose":
	    Enable: true
	    ID: ""
	    Channel: 8192
	    ChannelTimeoutMs: 0
	    ShutdownTimeoutMs: 3000
	    Formatter: "format.Forward"
	    Filter: "filter.All"
	    DropToStream: "_DROPPED_"
	    Stream:
	        - "foo"
	        - "bar"
	    Region: "eu-west-1"
	    Endpoint: "firehose.eu-west-1.amazonaws.com"
	    CredentialType: "none"
	    CredentialId: ""
	    CredentialToken: ""
	    CredentialSecret: ""
	    CredentialFile: ""
	    CredentialProfile: ""
	    BatchMaxMessages: 500
	    RecordMaxMessages: 1
	    RecordMessageDelimiter: "\n"
	    SendTimeframeSec: 1
	    BatchTimeoutSec: 3
	    StreamMapping:
	        "*" : "default"
