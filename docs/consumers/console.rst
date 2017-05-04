Console
=======

This consumer reads from stdin.
A message is generated after each newline character.


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

**Console**
  Console defines the pipe to read from.
  This can be "stdin" or the name of a named pipe that is created if not existing.
  The default is "stdin".

**Permissions**
  Permissions accepts an octal number string that contains the unix file permissions used when creating a named pipe.
  By default this is set to "0664".

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
	        - "foo"
	        - "bar"
	    Console: "stdin"
	    Permissions: "0664"
	    ExitOnEOF: false
