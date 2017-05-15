RegExp
======

This plugin allows filtering messages using regular expressions.


Parameters
----------

**FilterExpression**
  FilterExpression defines the regular expression used for matching the message payload.
  If the expression matches, the message is passed.
  FilterExpression is evaluated after FilterExpressionNot.

**FilterExpressionNot**
  FilterExpressionNot defines a negated regular expression used for matching the message payload.
  If the expression matches, the message is blocked.
  FilterExpressionNot is evaluated before FilterExpression.

Example
-------

.. code-block:: yaml

	- "stream.Broadcast":
	    Filter: "filter.RegExp"
	    FilterExpression: "\d+-.*"
	    FilterExpressionNot: "\d+-.*"
