Loopback
#############

The loopback consumer feeds messages from the Gollum back into Gollum.
These typically are messages dropped by a producer because they could not be delivered on first try.

Parameters
----------

**Enable**
  Can either be true or false to enable or disable this consumer.
**Channel**
  Defines the size of the internal loopback channel.
**Routes**
  Defines a stream to stream(s) mapping for incoming messages.
  Dropped messages typically arrive from the \_DROPPED\_ stream.

Example
-------

::

  - "consumer.Loopback":
    Enable: true
    Channel: 8192
    Routes:
        "_DROPPED_": "myStream"
        "myStream":
            - "myOtherStream"
            - "myOtherStream2"
