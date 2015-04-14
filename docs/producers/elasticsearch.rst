ElasticSearch
#############

| This producers writes messages to ElasticSearch.
| ElasticSearch expects messages to be JSON encoded so the configuration should assure that messages are arriving as or are converted into a valid JSON format.
| You can use the :doc:`JSON formatter </formatters/json>` to convert messages to valid JSON.

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
**Connections**
  Defines the number of simultaneous connections allowed to an ElasticSearch server.
  This is set to 6 by default.
**RetrySec**
  Defines the time in seconds after which a failed dataset will be transmitted again.
  By default this is set to 5.
**TTL**
  Defines the TTL set for each ElasticSearch message.
  By default this is set to an empty string which means no TTL.
**Port**
  Defines the ElasticSearch port, wich has to be the same for all servers.
  By default this is set to 9200.
**Domain**
  Defines the ElasticSearch domain setting.
  If no value is set here the first server from the Servers setting is used to derive the domain.
**User**
  Sets the username passed when sending data. Set to an empty string by default.
**Password**
  Sets the password passed when sending data. Set to an empty string by default.
**BatchSizeByte**
  Defines the minimum size in bytes required to trigger a batch send.
  By default this is set to 32768 (32KB).
**BatchMaxCount**
  Defines the minimum number of documents required to trigger batch send.
  By default this is set to 256.
**BatchTimeoutSec**
  Defines the time in seconds after which a batch send will be triggered.
  By default this is set to 5.
**DayBasedIndex**
  Set to true to append the date of the message to the index as in "<index>_YYYY-MM-DD".
  By default this is set to false.
**Index**
  Maps a stream to a specific ElasticSearch index.
  If you define a mapping on "*" all streams that do not have a specific mapping will go to this index (including internal streams).
  By default a mapping "*" to "default" is used.
  If no explicit mapping to "*" is set this mapping is preserved, i.e. streams without a mapping will use the "default" index.
**Type**
  Maps a stream to a specific ElasticSearch type.
  This behaves like the index map and is used to assign a "_type" to an elasticsearch message.
  By default the type "log" is used.
**Servers**
  Defines a list of servers to connect to.

Example
-------

::

  - "producer.ElasticSearch":
    Enable: true
    Channel: 8192
    ChannelTimeoutMs: 100
    Connections: 10
    RetrySec: 5
    TTL: "1d"
    Port: 9200
    Domain: "local"
    User: "root"
    Password: "root"
    BatchSizeByte: 65535
    BatchMaxCount: 512
    BatchTimeoutSec: 5
    DayBasedIndex: false
    Index:
      "console" : "default"
      "_GOLLUM_"  : "default"
    Type:
      "console" : "log"
      "_GOLLUM_"  : "gollum"
    Servers:
      - "localhost"
    Stream:
        - "log"
        - "console"
