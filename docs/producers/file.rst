File
====

This producers writes messages to a file or file based resource.
Folders in the file path will be created if necessary.
If configured, a log rotation can be triggered by sending a SIG_HUP.
You can use ``kill -1 $(cat gollum.pid)`` to achieve this. To create a pidfile you can start gollum with the -p option.


Parameters
----------

**Enable**
  Can either be true or false to enable or disable this producer.
**ID**
  Allows this producer to be found by other plugins by name.
  By default this is set to "" which does not register this producer.
**Stream**
  Defines either one or an aray of stream names this producer recieves messages from.
**DropToStream**
  Defines the stream used for messages that are dropped after a timeout (see ChannelTimeoutMs).
  By default this is _DROPPED_.
**Channel**
  Defines the number of messages that can be buffered by the internal channel.
  By default this is set to 8192.
**ChannelTimeoutMs**
  Defines a timeout in milliseconds for messages to wait if this producer's queue is full.

  - A timeout of -1 or lower will discard the message without notice.
  - A timeout of 0 will block until the queue is free. This is the default.
  - A timeout of 1 or higher will wait n milliseconds for the queues to become available again.
    If this does not happen, the message will be send to the _DROPPED_ stream that can be processed by the :doc:`Loopback </consumers/loopback>` consumer.

**FlushTimeoutSec**
  Sets the maximum number of seconds to wait before a flush is aborted during shutdown.
  By default this is set to 0, which does not abort the flushing procedure.
**Format**
  Defines a message formatter to use. :doc:`Format.Forward </formatters/forward>` by default.
**File**
  Sets the path to the log file to write.
  The wildcard character "*" can be used as a placeholder for the stream name.
  By default this is set to /var/prod/gollum.log.
**BatchMaxCount**
  Defines the maximum number of messages that can be buffered before a flush is mandatory.
  If the buffer is full and a flush is still underway or cannot be triggered out of other reasons, the producer will block.
**BatchFlushCount**
  Defines the number of messages to be buffered before they are written to disk.
  This setting is clamped to BatchMaxCount.
  By default this is set to BatchMaxCount / 2.
**BatchTimeoutSec**
  Defines the number of seconds to wait after a message before a flush is triggered.
  The timer is reset after each new message.
  By default this is set to 5.
 **FlushTimeoutSec**
  Sets the maximum number of seconds to wait before a flush is aborted during shutdown.
  By default this is set to 0, which does not abort the flushing procedure.
**Rotate**
  Set to true to enable log rotation. Disabled by default.
**RotateTimeoutMin**
  Defines a timeout in minutes that will cause a log rotation.
  Can be set in parallel with RotateSizeMB.
  By default this is set to 1440 (i.e. 1 Day).
  The timer is reset after a log rotation has been triggered by any event.
**RotateAt**
  Defines specific timestamp as in "HH:MM" when the log should be rotated.
  Hours must be given in 24h format.
  When left empty this setting is ignored. By default this setting is disabled.
**RotateSizeMB**
  Defines the maximum file size in MB that triggers a file rotate.
  Files can get bigger than this size. By default this is set to 1024.
**RotateTimestamp**
  Sets the timestamp added to the filename when file rotation is enabled.
  The format is based on Go's time.Format function and set to "2006-01-02_15" by default.
**RotatePruneCount**
  Removes old logfiles upon rotate so that only the given number of logfiles remain.
  Logfiles are located by the name defined by "File" and are pruned by date (followed by name).
  By default this is set to 0 which disables pruning.
**RotatePruneTotalSizeMB**
  Removes old logfiles upon rotate so that only the given number of MBs are used by logfiles.
  Logfiles are located by the name defined by "File" and are pruned by date (followed by name).
  By default this is set to 0 which disables pruning.
**Compress**
  Set to true to gzip a file after rotation.
  By default this is set to false.

Example
-------

.. code-block:: yaml

  - "producer.File":
    Enable: true
    Channel: 8192
    ChannelTimeoutMs: 100
    File: "/var/log/gollum/*/*.log"
    BatchSizeMaxKB: 16384
    BatchSizeByte: 4096
    BatchTimeoutSec: 2
    Rotate: true
    RotateTimeoutMin: 1440
    RotateSizeMB: 1024
    RotateAt: "00:00"
    RotatePruneCount: 0
    RotatePruneTotalSizeMB: 0
    Compress: true
    Stream: "*"
