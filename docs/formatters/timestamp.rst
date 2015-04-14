Timestamp
#############

This formatter prepends a custom timestamp to the message.
The timestamp format is based upon Go's timestamp formatter. See Go's `documentation <http://golang.org/pkg/time/#pkg-constants>`_.
This formatter allows a nested formatter to further modify the message.

Parameters
----------

**TimestampDataFormatter**
  Defines an additional formatter applied before adding the timestamp. :doc:`Format.Forward </formatters/forward>` by default.

**Timestamp**
  Defines the format of the timestamp. "2006-01-02T15:04:05.000 MST " by default.

Example
-------

::

  - "stream.Broadcast":
    Formatter: "format.Sequence"
    Timestamp: "[2006-01-02 15:04:05]"
    TimestampDataFormatter: "format.Forward"
