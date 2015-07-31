CollectdToInflux09
==================

This formatter converts data produced by Collectd to InfluxDB 0.9.x format.
Trailing and leading commas are removed from the Collectd message beforehand.

Parameters
----------

**CollectdToInfluxFormatter**
  Defines the formatter applied before the conversion from Collectd to InfluxDB.
  Set to :doc:`Format.Forward </formatters/forward>` by default.

Example
-------

.. code-block:: yaml

  - "stream.Broadcast":
    Formatter: "format.CollectdToInflux09"
    CollectdToInflux09Formatter: "format.Forward"
