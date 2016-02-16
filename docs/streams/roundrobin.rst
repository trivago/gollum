RoundRobin
==========

Messages will be sent to one of the producers attached to this stream.
Producers will be switched one-by-one.


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

Example
-------

.. code-block:: yaml

	- "stream.Foobar"
	    Enable: true
	    Stream: "streamToConfigure"
	    Formatter: "format.Forward"
	    Filter: "filter.All"
	    TimeoutMs: 0
