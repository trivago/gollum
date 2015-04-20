Null
====

This producers discards all messages similar to a /dev/null.
Its main purpose is to profile consumers and/or streams.

Parameters
----------

**Enable**
  Can either be true or false to enable or disable this producer.
**Stream**
  Defines either one or an aray of stream names this producer recieves messages from.

Example
-------

.. code-block:: yaml

  - "producer.Null":
    Enable: true
    Stream:
        - "log"
        - "console"
