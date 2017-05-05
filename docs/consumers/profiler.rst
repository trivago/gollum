Profiler
========

The profiler plugin generates Runs x Batches messages and send them to the configured streams as fast as possible.
This consumer can be used to profile producers and/or configurations.


Parameters
----------

**Enable**
  Enable switches the consumer on or off.
  By default this value is set to true.

**ID**
  ID allows this consumer to be found by other plugins by name.
  By default this is set to "" which does not register this consumer.

**Stream**
  Stream contains either a single string or a list of strings defining the message channels this consumer will produce.
  By default this is set to "*" which means only producers set to consume "all streams" will get these messages.

**Runs**
  Runs defines the number of messages per batch.
  By default this is set to 10000.

**Batches**
  Batches defines the number of measurement runs to do.
  By default this is set to 10.

**TemplateCount**
  TemplateCount defines the number of message templates to be generated.
  A random message template will be chosen when a message is sent.
  Templates are generated in advance.
  By default this is set to 10.

**Characters**
  Characters defines the characters to be used in generated strings.
  By default these are "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890 ".

**Message**
  Message defines a go format string to be used to generate the message payloads.
  The length of the values generated will be deducted from the format size parameter.
  I.e. "%200d" will generate a digit between 0 and 200, "%10s" will generate a string with 10 characters, etc.
  By default this is set to "%256s".

**DelayMs**
  DelayMs defines the number of milliseconds of sleep between messages.
  By default this is set to 0.

**KeepRunning**
  KeepRunning can be set to true to disable automatic shutdown of gollum after profiling is done.
  This can be used to e.g. read metrics after a profile run.
  By default this is set to false.

Example
-------

.. code-block:: yaml

	- "consumer.Profile":
	    Enable: true
	    ID: ""
	    Stream:
	        - "foo"
	        - "bar"
	    Runs: 10000
	    Batches: 10
	    TemplateCount: 10
	    Characters: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890"
	    Message: "%256s"
	        DelayMs: 0
	    KeepRunning: false
