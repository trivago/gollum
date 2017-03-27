Sample
======

This plugin blocks messages after a certain number of messages per second has been reached.


Parameters
----------

**SampleRatePerGroup**
  SampleRatePerGroup defines how many messages are passed through the filter in each group.
  By default this is set to 1.

**SampleGroupSize**
  SampleGroupSize defines how many messages make up a group.
  Messages over SampleRatePerGroup within a group are dropped.
  By default this is set to 1.

**SampleDropToStream**
  SampleDropToStream is an optional stream messages are sent to when they are sampled.
  By default this is disabled and set to "".

**SampleRateIgnore**
  SampleRateIgnore defines a list of streams that should not be affected by sampling.
  This is useful for e.g. producers listeing to "*".
  By default this list is empty.

Example
-------

.. code-block:: yaml

    - "stream.Broadcast":
	      Filter: "filter.Sample"
	      SampleRatePerGroup: 1
	      SampleGroupSize: 1
	      SampleDropToStream: ""
	      SampleRateIgnore:
	          - "foo"
