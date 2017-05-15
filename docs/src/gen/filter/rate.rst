Rate
====

This plugin blocks messages after a certain number of messages per second has been reached.


Parameters
----------

**RateLimitPerSec**
  RateLimitPerSec defines the maximum number of messages per second allowed to pass through this filter.
  By default this is set to 100.

**RateLimitDropToStream**
  RateLimitDropToStream is an optional stream messages are sent to when the limit is reached.
  By default this is disabled and set to "".

**RateLimitIgnore**
  RateLimitIgnore defines a list of streams that should not be affected by rate limiting.
  This is useful for e.g. producers listeing to "*".
  By default this list is empty.

Example
-------

.. code-block:: yaml

	    - "stream.Broadcast":
	        Filter: "filter.Rate"
	        RateLimitPerSec: 100
	        RateLimitDropToStream: ""
	        RateLimitIgnore:
	            - "foo"
