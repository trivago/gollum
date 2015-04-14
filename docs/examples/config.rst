Configuration
=============

Hello World
-----------

This example sets up a simple console consumer and producer that will simply echo everything you type back to the console.
As messages have no new line appended by default a Envelope formatter is used to add one before writing to console.
Make sure to start Gollum with `gollum -ll 3` to see all log messages.

::

  - "consumer.Console":
      Stream: "console"

  - "producer.Console":
      Formatter: "format.Envelope"
      Stream:
          - "*"
          - "_GOLLUM_"

Hello World XML
---------------

This example extends the Hello World example by introducing a stream configuration.
All messages from the console consumer will be enveloped into a XML tag, while the log messages are not.
Make sure to start Gollum with `gollum -ll 3` to see all log messages.

::

  - "consumer.Console":
      Stream: "console"

  - "stream.Broadcast":
      Formatter: "format.Envelope"
      Prefix: "<message>"
      Postfix: "</message>"
      Stream: "console"

  - "producer.Console":
      Formatter: "format.Envelope"
      Stream:
          - "*"
          - "_GOLLUM_"

Hello World filtered
--------------------

This example extends the previous exmaples by setting up a filter to only echo sentences that end with the word "gollum".
A regular expression filter is used to achieve this.
Note that this filter does not apply to standard log messages.
Make sure to start Gollum with `gollum -ll 3` to see all log messages.

::

  - "consumer.Console":
      Stream: "console"

  - "stream.Broadcast":
      Filter: "filter.RegExp"
      FilterExpression: "gollum$"
      Stream: "console"

  - "producer.Console":
      Formatter: "format.Envelope"
      Stream:
          - "*"
          - "_GOLLUM_"

Hello World splitter
--------------------

This example extends the Hello World example by introducing another stream configuration.
This time we will print the console output twice, encoded as XML and as JSON.
Make sure to start Gollum with `gollum -ll 3` to see all log messages.

::

  - "consumer.Console":
      Stream:
          - "consoleXML"
          - "consoleJSON"

  - "stream.Broadcast":
      Formatter: "format.Envelope"
      Prefix: "<message>"
      Postfix: "</message>"
      Stream: "consoleXML"

  - "stream.Broadcast":
      Formatter: "format.Envelope"
      Prefix: "{message:\""
      Postfix: "\"}"
      Stream: "consoleJSON"

  - "producer.Console":
      Formatter: "format.Envelope"
      Stream:
          - "*"
          - "_GOLLUM_"

Chat server
-----------

This example requires two Gollum instances to run.
The first one acts as the "chat client" while the second one acts as the "chat server".
Messages entered on the client will be sent to the server using runlength encoding where they are written to a log file and to console.
The logfile will write a standard timestamp before each message while the console will just print the message.
Both servers have a standard console producer attached to print log messages to console aswell.
Make sure to start Gollum with `gollum -ll 3` to see all log messages.

**Client**
::

  - "consumer.Console":
      Stream: "client"

  - "producer.Socket":
      Address: ":5880"
      Formatter: "format.Runlength"
      Acknowledge: true
      Stream: "client"

  - "producer.Console":
      Formatter: "format.Envelope"
      Stream: "_GOLLUM_"

**Server**
::

  - "consumer.Socket":
      Runlength: true
      Acknowledge: true
      Address: ":5880"
      Stream: "server"

  - "producer.File":
      Formatter: "format.Timestamp"
      TimestampFormatter: "format.Envelope"
      File: "chat.log"
      Stream: "server"

  - "producer.Console":
      Formatter: "format.Envelope"
      Stream:
        - "*"
        - "_GOLLUM_"

Profiling
---------

This configuration will test Gollum for its theoretic maximum message throughput.
You can of course modify this example to test e.g. file producer performance.
Make sure to start Gollum with `gollum -ll 3 -ps` to see all log messages as well as intermediate profiling results.

::

  - "consumer.Profiler":
      Runs: 100000
      Batches: 100
      Characters: "abcdefghijklmnopqrstuvwxyz ABCDEFGHIJKLMNOPQRSTUFVXYZ 0123456789 .,!;:-_"
      Message: "%256s"
      Stream: "profile"

  - "producer.Null":
      Stream: "profile"

  - "producer.Console":
      Formatter: "format.Envelope"
      Stream: "_GOLLUM_"
