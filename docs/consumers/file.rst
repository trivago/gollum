File
====

This consumer reads from a file.
If the file is part of a log rotation the file can be reopened by sending a SIG_HUP.
You can use ``kill -1 $(cat gollum.pid)`` to achieve this. To create a pidfile you can start gollum with the -p option.


Parameters
----------

**Enable**
  Can either be true or false to enable or disable this consumer.
**Stream**
  Defines either one or an aray of stream names this consumer sends messages to.
**File**
  Defines the file to read from.
**DefaultOffset**
  Defines the offset inside the file to start reading from if no OffsetFile is defined or found. Valid values are "newest" and "oldest" while the former is the default value.
**OffsetFile**
  Defines a file to store the current offset to. The offset is updated in intervalls and can be used to continue reading after a restart.
**Delimiter**
  Defines a string that marks the end of a message.
  Standard escape characters like "\r", "\n" and "\t" are allowed.
  The default values is "/n".

Offsets
-------

**Oldest**
  Start reading at the end of the file if no offset file is set and/or found.
**Neweset**
  Start reading at the beginning of the file if no offset file is set and/or found.

Example
-------

.. code-block:: yaml

  - "consumer.File":
    Enable: true
    File: "test.txt"
    DefaultOffset: "Oldest"
    OffsetFile: "/tmp/test.progress"
    Delimiter: "\n"
    Stream:
        - "stdin"
        - "console"
