CollectdToInflux08
==================

InfluxDB 0.8.x compatible JSON data.
Trailing and leading commas are removed from the Collectd message beforehand.


Parameters
----------

**CollectdToInfluxFormatter**
  CollectdToInfluxFormatter defines the formatter applied before the conversion from Collectd to InfluxDB.
  By default this is set to format.Forward.

Example
-------

.. code-block:: yaml

	    - "stream.Broadcast":
	        Formatter: "format.CollectdToInflux08"
	        CollectdToInfluxFormatter: "format.Forward"
