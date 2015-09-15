Console
=======

This consumer listens to stdin.
When attached to a fuse, this consumer will stop accepting messages in case that fuse is burned.
See the `API documentation <http://gollum.readthedocs.org/en/latest/consumers/console.html>`_ for additional details.

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
**ExitOnEOF**
  Set to true to trigger an exit signal if StdIn is closed (e.g. happens when a pipe is closed).
  This is set to false by default.

Example
-------

.. code-block:: yaml

  - "consumer.Console":
    Enable: true
    Stream:
        - "stdin"
        - "console"
