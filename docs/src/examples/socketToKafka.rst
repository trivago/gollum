Socket to Kafka
===============

This example creates a unix domain socket /tmp/kafka.socket that
accepts the following protocol:
| ``<topic>:<message_base64>\n``

The message will be base64 decoded and written to the topic mentioned at
the start of the message. This example also allows shows how to apply
rate limiting per topic.

**Configuration (v0.4.x):**

.. code:: yaml

    # Socket accepts <topic>:<message_base64>
    - "consumer.Socket":
        Stream: "raw"
        Address: "unix:///tmp/kafka.socket"
        Permissions: "0777"

    # Stream "raw" to stream "<topic>" conversion
    # Decoding of <message_base64> to <message>
    - "stream.Broadcast":
        Stream: "raw"
        Formatter: "format.StreamRoute"
        StreamRouteDelimiter: ":"
        StreamRouteFormatter: "format.Base64Decode"

    # Listening to all streams as streams are generated at runtime
    # Use ChannelTimeoutMs to be non-blocking
    - "producer.Kafka":
        Stream: "*"
        Filter: "filter.Rate"
        RateLimitPerSec: 100
        ChannelTimeoutMs: 10
        Servers:
            - "kafka1:9092"
            - "kafka2:9092"
            - "kafka3:9092"