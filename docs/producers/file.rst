File
====

This producers writes messages to a file or file based resource.
If configured, a log rotation can be triggered by sending a SIG_HUP.
You can use ``kill -1 $(cat gollum.pid)`` to achieve this. To create a pidfile you can start gollum with the -p option.


Parameters
----------

**Enable**
  Can either be true or false to enable or disable this producer.
**Stream**
  Defines either one or an aray of stream names this producer recieves messages from.
**Channel**
  Defines the number of messages that can be buffered by the internal channel.
  By default this is set to 8192.
**ChannelTimeoutMs**
  Defines a timeout in milliseconds for messages to wait if this producer's queue is full.

  - A timeout of -1 or lower will discard the message without notice.
  - A timeout of 0 will block until the queue is free. This is the default.
  - A timeout of 1 or higher will wait n milliseconds for the queues to become available again.
    If this does not happen, the message will be send to the _DROPPED_ stream that can be processed by the :doc:`Loopback </consumers/loopback>` consumer.

**Format**
  Defines a message formatter to use. :doc:`Format.Forward </formatters/forward>` by default.
**File**
  Sets the path to the log file to write.
  By default this is set to /var/prod/gollum.log.
**BatchSizeMaxKB**
  Defines the internal file buffer size in KB.
  This producers allocates a front- and a backbuffer of this size.
  If the frontbuffer is filled up completely a flush is triggered and the frontbuffer becomes available for writing again.
  Messages larger than BatchSizeMaxKB are rejected.
  By default this is set to 8192 (8MB)
**BatchSizeByte**
  Defines the number of bytes to be buffered before a flush is triggered.
  By default this is set to 8192 (8KB).
**BatchTimeoutSec**
  Defines the number of seconds to wait after a message before a flush is triggered.
  The timer is reset after each new message.
  By default this is set to 5.
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
    File: "/var/log/gollum.log"
    BatchSizeMaxKB: 16384
    BatchSizeByte: 4096
    BatchTimeoutSec: 2
    Rotate: true
    RotateTimeoutMin: 1440
    RotateSizeMB: 1024
    RotateAt: "00:00"
    Compress: true
    Stream:
        - "log"
        - "console"
