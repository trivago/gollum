Console
=======

This consumer listens to stdin.

Parameters
----------

**Enable**
  Can either be true or false to enable or disable this consumer.
**ID**
  Allows this consumer to be found by other plugins by name.
  By default this is set to "" which does not register this consumer.
**ExitOnEOF**
  Set to true to trigger an exit signal if StdIn is closed (e.g. happens when a pipe is closed).
  This is set to false by default.
**Stream**
  Defines either one or an aray of stream names this consumer sends messages to.

Example
-------

.. code-block:: yaml

  - "consumer.Console":
    Enable: true
    Stream:
        - "stdin"
        - "console"
