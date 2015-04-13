Sequence
#############

| This formatter prepends the internal sequence number of the message as "number:" to the message.
| Note that "number" is the actual ASCII representation of a number, not a binary representation.
| This formatter allows a nested formatter to further modify the message.

Parameters
----------

**SequenceDataFormatter**
  Defines an additional formatter applied before adding the sequence number. :doc:`Format.Forward </formatters/forward>` by default.

Example
-------

::

  - "stream.Broadcast":
    Formatter: "format.Sequence"
    SequenceDataFormatter: "format.Forward"
