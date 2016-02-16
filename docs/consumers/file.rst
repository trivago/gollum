File
====

The file consumer allows to read from files while looking for a delimiter that marks the end of a message.
If the file is part of e.g. a log rotation the file consumer can be set to a symbolic link of the latest file and (optionally) be told to reopen the file by sending a SIGHUP.
A symlink to a file will automatically be reopened if the underlying file is changed.
When attached to a fuse, this consumer will stop accepting messages in case that fuse is burned.


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

**Fuse**
  Fuse defines the name of a fuse to observe for this consumer.
  Producer may "burn" the fuse when they encounter errors.
  Consumers may react on this by e.g. closing connections to notify any writing services of the problem.
  Set to "" by default which disables the fuse feature for this consumer.
  It is up to the consumer implementation to react on a broken fuse in an appropriate manner.

**File**
  File is a mandatory setting and contains the file to read.
  The file will be read from beginning to end and the reader will stay attached until the consumer is stopped.
  I.e. appends to the attached file will be recognized automatically.

**DefaultOffset**
  DefaultOffset defines where to start reading the file.
  Valid values are "oldest" and "newest".
  If OffsetFile is defined the DefaultOffset setting will be ignored unless the file does not exist.
  By default this is set to "newest".

**OffsetFile**
  OffsetFile defines the path to a file that stores the current offset inside the given file.
  If the consumer is restarted that offset is used to continue reading.
  By default this is set to "" which disables the offset file.

**Delimiter**
  Delimiter defines the end of a message inside the file.
  By default this is set to "\n".

Example
-------

.. code-block:: yaml

- "consumer.File":
    Enable: true
    ID: ""
    Fuse: ""
    Stream:
        - "foo"
        - "bar"
    File: "/var/run/system.log"
    DefaultOffset: "Newest"
    OffsetFile: ""
    Delimiter: "\n"
