RegExp
======



Parameters
----------

**FilterExpression**
  FilterExpression defines the regular expression used for matching the message payload.
  If the expression matches, the message is passed.

**FilterExpressionNot**
  FilterExpressionNot defines a negated regular expression used for matching the message payload.
  If the expression matches, the message is blocked.

Example
-------

.. code-block:: yaml

	    - "stream.Broadcast":
	        Filter: "filter.RegExp"
	        FilterExpression: "\d+-.*"
	        FilterExpressionNot: "\d+-.*"
