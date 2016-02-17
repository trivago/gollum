File
====

The file producer writes messages to a file.
This producer also allows log rotation and compression of the rotated logs.
Folders in the file path will be created if necessary.
This producer does not implement a fuse breaker.


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

**File**
  File contains the path to the log file to write.
  The wildcard character "*" can be used as a placeholder for the stream name.
  By default this is set to /var/log/gollum.log.

**FileOverwrite**
  FileOverwrite enables files to be overwritten instead of appending new data to it.
  This is set to false by default.

**Permissions**
  Permissions accepts an octal number string that contains the unix file permissions used when creating a file.
  By default this is set to "0664".

**FolderPermissions**
  FolderPermissions accepts an octal number string that contains the unix file permissions used when creating a folder.
  By default this is set to "0755".

**BatchMaxCount**
  BatchMaxCount defines the maximum number of messages that can be buffered before a flush is mandatory.
  If the buffer is full and a flush is still underway or cannot be triggered out of other reasons, the producer will block.
  By default this is set to 8192.

**BatchFlushCount**
  BatchFlushCount defines the number of messages to be buffered before they are written to disk.
  This setting is clamped to BatchMaxCount.
  By default this is set to BatchMaxCount / 2.

**BatchTimeoutSec**
  BatchTimeoutSec defines the maximum number of seconds to wait after the last message arrived before a batch is flushed automatically.
  By default this is set to 5.

**FlushTimeoutSec**
  FlushTimeoutSec sets the maximum number of seconds to wait before a flush is aborted during shutdown.
  By default this is set to 0, which does not abort the flushing procedure.

**Rotate**
  Rotate if set to true the logs will rotate after reaching certain thresholds.
  By default this is set to false.

**RotateTimeoutMin**
  RotateTimeoutMin defines a timeout in minutes that will cause the logs to rotate.
  Can be set in parallel with RotateSizeMB.
  By default this is set to 1440 (i.e. 1 Day).

**RotateAt**
  RotateAt defines specific timestamp as in "HH:MM" when the log should be rotated.
  Hours must be given in 24h format.
  When left empty this setting is ignored.
  By default this setting is disabled.

**RotateSizeMB**
  RotateSizeMB defines the maximum file size in MB that triggers a file rotate.
  Files can get bigger than this size.
  By default this is set to 1024.

**RotateTimestamp**
  RotateTimestamp sets the timestamp added to the filename when file rotation is enabled.
  The format is based on Go's time.Format function and set to "2006-01-02_15" by default.

**RotatePruneCount**
  RotatePruneCount removes old logfiles upon rotate so that only the given number of logfiles remain.
  Logfiles are located by the name defined by "File" and are pruned by date (followed by name).
  By default this is set to 0 which disables pruning.

**RotatePruneAfterHours**
  RotatePruneAfterHours removes old logfiles that are older than a given number of hours.
  By default this is set to 0 which disables pruning.

**RotatePruneTotalSizeMB**
  RotatePruneTotalSizeMB removes old logfiles upon rotate so that only the given number of MBs are used by logfiles.
  Logfiles are located by the name defined by "File" and are pruned by date (followed by name).
  By default this is set to 0 which disables pruning.

**RotateZeroPadding**
  RotateZeroPadding sets the number of leading zeros when rotating files with an existing name.
  Setting this setting to 0 won't add zeros, every other number defines the number of leading zeros to be used.
  By default this is set to 0.

**Compress**
  Compress defines if a rotated logfile is to be gzip compressed or not.
  By default this is set to false.

Example
-------

.. code-block:: yaml

	- "producer.File":
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
	    File: "/var/log/gollum.log"
	    FileOverwrite: false
	    Permissions: "0664"
	    FolderPermissions: "0755"
	    BatchMaxCount: 8192
	    BatchFlushCount: 4096
	    BatchTimeoutSec: 5
	    FlushTimeoutSec: 0
	    Rotate: false
	    RotateTimeoutMin: 1440
	    RotateSizeMB: 1024
	    RotateAt: ""
	    RotateTimestamp: "2006-01-02_15"
	    RotatePruneCount: 0
	    RotatePruneAfterHours: 0
	    RotatePruneTotalSizeMB: 0
	    Compress: false
