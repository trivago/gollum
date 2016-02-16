StreamName
==========

StreamName is a formatter that prefixes a message with the StreamName.


Parameters
----------

**StreamNameFormatter**
  StreamNameFormatter defines the formatter for the data transferred as message.
  By default this is set to "format.Envelope".

**StreamNameHistory**
  StreamNameHistory can be set to true to not use the current but the previous stream name.
  This can be usefull to e.g. get the name of the stream messages were dropped from.
  By default this is set to false.

**StreamNameSeparator**
  StreamNameSeparator sets the separator character placed after the stream name.
  This is set to " " by default.

Example
-------

.. code-block:: yaml

	    - "stream.Broadcast":
	        Formatter: "format.StreamName"
	        StreamNameFormatter: "format.Envelope"
