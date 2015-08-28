Redis
=====

This producers stores messages on a redis server.

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
**Filter**
  Defines a message filter to apply before formatting. :doc:`Filter.All </filters/all>` by default.
**Address**
  Defines the redis server address to connect to.
  This can either be any ip address and port like "localhost:6379" or a file
  like "unix:///var/redis.socket". By default this is set to ":6379".
**Database**
  Defines the redis database index to connect to.
  By default this is set to 0.
**Key**
  Defines the redis key to store all values to.
  By default this is set to "default".
**Storage**
  Defines the type of the storage to use.
  Valid values are: "hash", "list", "set", "sortedset" and "string".
  By default this is set to "hash".
**FieldFormat**
  Defines an extra formatter to define the field or score value if required by the selected storage type.
  If no field value is required this value is ignored.
  By default this is set to :doc:`"format.Identifier" </formatters/identifier>`.
**FieldAfterFormat**
  When set to true the message passed to FieldFormat has been formatted by the producer's formatter first.
  If set to false FieldFormat will use the message as passed to the formatter.
  By default this is set to false.

Example
-------

.. code-block:: yaml

  - "producer.Redis":
    Enable: true
    Channel: 8192
    ChannelTimeoutMs: 100
    Address: "127.0.0.1:6379"
    Database: 0
    Key: "gollum"
    Storage: "hash"
    FieldFormat: "format.Identifier"
    FieldAfterFormat: true
    Stream:
        - "persist"
        - "_GOLLUM_"
