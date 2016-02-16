JSON
====

Note that this filter is quite expensive due to JSON marshaling and regexp testing of every message passing through it.


Parameters
----------

**FormatReject**
  FormatReject defines fields that will cause a message to be rejected if the given regular expression matches.
  Rejects are checked before Accepts.
  Field paths can be defined in a format accepted by shared.MarshalMap.Path.

**FormatAccept**
  FormatAccept defines fields that will cause a message to be rejected if the given regular expression does not match.
  Field paths can be defined in a format accepted by shared.MarshalMap.Path.

Example
-------

.. code-block:: yaml

	- "stream.Broadcast":
	    Filter: "filter.JSON"
	    FilterReject:
	        "command" : "state\d\..*"
	    FilterAccept:
	        "args/results[0]value" : "true"
	        "args/results[1]" : "true"
	        "command" : "state\d\..*"
