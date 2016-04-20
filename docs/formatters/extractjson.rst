ExtractJSON
===========

ExtractJSON is a formatter that extracts a single value from a JSON message.


Parameters
----------

**ExtractJSONDataFormatter**
  ExtractJSONDataFormatter formatter that will be applied before the field is extracted.
  Set to format.Forward by default.

**ExtractJSONField**
  ExtractJSONField defines the field to extract.
  This value is empty by default.
  If the field does not exist an empty string is returned.

**ExtractJSONTrimValues**
  ExtractJSONTrimValues will trim whitspaces from the value if enabled.
  Enabled by default.

**ExtractJSONPrecision**
  ExtractJSONPrecision defines the floating point precision of number values.
  By default this is set to 0 i.e. all decimal places will be omitted.

Example
-------

.. code-block:: yaml

	- "stream.Broadcast":
	    Formatter: "format.ExtractJSON"
	    ExtractJSONdataFormatter: "format.Forward"
	    ExtractJSONField: ""
	    ExtractJSONTrimValues: true
	    ExtractJSONPrecision: 0
