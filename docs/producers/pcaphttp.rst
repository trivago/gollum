PcapHTTPConsumer
================

NOTICE: This producer is not included in standard builds.
To enable it you need to trigger a custom build with native plugins enabled.
This plugin utilizes libpcap to listen for network traffic and reassamble http requests from it.
As it uses a CGO based library it will break cross platform builds (i.e. you will have to compile it on the correct platform).


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

**Interface**
  Interface defines the network interface to listen on.
  By default this is set to eth0, get your specific value from ifconfig.

**Filter**
  Filter defines a libpcap filter for the incoming packages.
  You can filter for specific ports, portocols, ips, etc.
  The documentation can be found here: http://www.tcpdump.org/manpages/pcap-filter.7.txt (manpage).
  By default this is set to listen on port 80 for localhost packages.

**Promiscuous**
  Promiscuous switches the network interface defined by Interface into promiscuous mode.
  This is required if you want to listen for all packages coming from the network, even those that were not meant for the ip bound to the interface you listen on.
  Enabling this can increase your CPU load.
  This setting is enabled by default.

**TimeoutMs**
  TimeoutMs defines a timeout after which a tcp session is considered to have dropped, i.e. the (remaining) packages will be discarded.
  Every incoming packet will restart the timer for the specific client session.
  By default this is set to 3000, i.e. 3 seconds.

Example
-------

.. code-block:: yaml

	- "native.PcapHTTPConsumer":
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
	    Enable: true
	    Interface: eth0
	    Filter: "dst port 80 and dst host 127.0.0.1"
	    Promiscuous: true
	    TimeoutMs: 3000
