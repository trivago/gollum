Examples
========

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
