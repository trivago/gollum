Profiler
========

The profile consumer generates messages to test producers or the general infrastructure.
When attached to a fuse, this consumer will stop processing messages in case that fuse is burned.
See the `API documentation <http://gollum.readthedocs.org/en/latest/consumers/profiler.html>`_ for additional details.

Parameters
----------

**Enable**
  Can either be true or false to enable or disable this consumer.
**ID**
  Allows this consumer to be found by other plugins by name.
  By default this is set to "" which does not register this consumer.
**Fuse**
  Defines the name of the fuse this consumer is attached to.
  When left empty no fuse is attached. This is the default value.
**Stream**
  Defines either one or an aray of stream names this consumer sends messages to.
**Runs**
  Defines the number of messages to send per batch.
**Batches**
  Defines the number of profiling runs before automatically stopping Gollum.
**Characters**
  Defines a set of allowed characters when generating dummy strings.
  Characters are chosen randomly from this string.
**TemplateCount**
  The number of different message templates to generate.
**Message**
  Formatting string to generate messages from. This is compatible to standard fmt.Printf style formatters.
  The length attribute will be used to define the length of the data generated.
**DelayMs**
  Defines the number of milliseconds of sleep between messages.
  By default this is set to 0.

Example
-------

.. code-block:: yaml

  - "consumer.Profiler":
    Enable: true
    Runs: 100000
    Batches: 100
    Characters: "abcdefghijklmnopqrstuvwxyz .,!;:-_"
    TemplateCount: 20
    Message: "{name:\"%100s\", number: %2d, float: %4f}"
    Stream:
      - "profile"
      - "dummy"
