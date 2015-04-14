Console
#############

This consumer listens to stdin.

Parameters
----------

**Enable**
    Can either be true or false to enable or disable this consumer.
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
