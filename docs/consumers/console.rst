Console
=======

This consumer listens to stdin.
When attached to a fuse, this consumer will stop accepting messages in case that fuse is burned.
See the `API documentation <http://gollum.readthedocs.org/en/latest/consumers/console.html>`_ for additional details.

Parameters
----------

**Enable**
    Enable switches the consumer on or off. By default this value is set to true.

**ID** 
    ID allows this consumer to be found by other plugins by name. 
    By default this is set to "" which does not register this consumer.

**Stream**
    Stream contains either a single string or a list of strings defining the message channels this consumer will produce. 
    By default this is set to "*" which means only producers set to consume "all streams" will get these messages.

**Fuse**
    Fuse defines the name of a fuse to observe for this consumer. 
    Producer may "burn" the fuse when they encounter errors. 
    Consumers may react on this by e.g. closing connections to notify any writing services of the problem.
    Set to "" by default which disables the fuse feature for this consumer.
    It is up to the consumer implementation to react on a broken fuse in an appropriate manner.

**ExitOnEOF**
    ExitOnEOF can be set to true to trigger an exit signal if StdIn is closed (e.g. when a pipe is closed).
    This is set to false by default.

Example
-------

.. code-block:: yaml

  - "consumer.Console":
    Enable: true
    ID: ""
    Stream:
        - "stdin"
        - "console"
    Fuse: ""
    ExitOnEof: false
