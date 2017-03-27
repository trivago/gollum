Any
===

This plugin blocks messages after a certain number of messages per second has been reached.


Parameters
----------

**AnyFilter**
  AnyFilter defines a list of filters that should be checked before dropping a message.
  Filters are checked in order, and if the message passes then no further filters are checked.
  By default this list is empty.

Example
-------

.. code-block:: yaml

	- "stream.Broadcast":
	    Filter: "filter.Any"
	    AnyFilter:
	        - "filter.JSON"
	        - "filter.RegEx"
