Route
=====

Messages will be routed to all streams configured.
Each target stream can hold another stream configuration, too, so this is not directly sending to the producers attached to the target streams.


Parameters
----------

**Enable**
  Enable can be set to false to disable this stream configuration but leave it in the config for future use.
  Set to true by default.

**Stream**
  Stream defines the stream to configure.
  This is a mandatory setting and has no default value.

**Formatter**
  Formatter defines the first formatter to apply to the messages passing through this stream.
  By default this is set to "format.Forward".

**Filter**
  Filter defines the filter to apply to the messages passing through this stream.
  By default this is et to "filter.All".

**TimeoutMs**
  TimeoutMs defines an optional timeout that can be used to wait for producers attached to this stream to unblock.
  This setting overwrites the corresponding producer setting for this (and only this) stream.

**Routes**
  Routes defines a 1:n stream remapping.
  Messages are reassigned to all of stream(s) in this list.
  If no route is set messages are forwarded on the incoming stream.
  When routing to multiple streams, the incoming stream has to be listed explicitly to be used.

Example
-------

.. code-block:: yaml

	- "stream.Route":
	    Enable: true
	    Stream: "streamToConfigure"
	    Formatter: "format.Forward"
	    Filter: "filter.All"
	    TimeoutMs: 0
	    Routes:
	        - "foo"
	        - "bar"
