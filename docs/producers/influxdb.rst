InfluxDB
=======

This producers writes messages to InfluxDB.
Messages passed to this consumer must either be converted to or already in InfluxDB compatible format.
InfluxDB 0.8.x and 0.9.x are supported.

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
**Host**
  Defines the host (and port) of the InfluxDB server.
  Defaults to "localhost:8086".
**User**
  Defines the InfluxDB username to use to login.
  If this name is left empty credentials are assumed to be disabled. Defaults to empty.
**Password**
  Defines the user's password.
  Defaults to empty.
**Database**
  Sets the InfluxDB database to write to.
  Database names go through time.Format before use.
  By default this is is set to "default".
**RetentionPolicy**
  This setting correlates to the InfluxDB retention policy setting.
  This is left empty by default (no retention policy used)
**UseVersion08**
  Set to true when writing data to InfluxDB 0.8.x.
  By default this is set to false.
**BatchMaxCount**
  Defines the maximum number of messages that can be buffered before a flush is mandatory.
  If the buffer is full and a flush is still underway or cannot be triggered out of other reasons, the producer will block.
  By default this is set to 8192.
**BatchFlushCount**
  Defines the number of messages to be buffered before they are written to InfluxDB.
  This setting is clamped to BatchMaxCount.
  By default this is set to BatchMaxCount / 2.
**BatchTimeoutSec**
  Defines the maximum number of seconds to wait after the last message arrived before a batch is flushed automatically.
  By default this is set to 5.

Example
-------

.. code-block:: yaml

  - "producer.InfluxDB":
    Enable: true
    Channel: 8192
    ChannelTimeoutMs: 100
    Host: "localhost:8086"
    User: ""
    Password: ""
    Database: "default"
    UseVersion08: false
    RetentionPolicy: ""
    BatchMaxCount: 8192
    BatchFlushCount: 4096
    BatchTimeoutSec: 5
    Stream:
        - "performance"
