RegExp
======

This filter tries to match a regular expression on text based messages.
Messages that match the given expression will be passed, everything else is blocked.
Documentation on the used regular expression dialect can be found here http://golang.org/pkg/regexp/

Parameters
----------

**FilterExpression**
  Defines the expression to use when filtering messages. Messages matching this pattern are passed. Empty string by default.
**FilterExpressionNot**
  Defines the expression to use when filtering messages. Messages not matching this pattern are passed. Empty string by default.

Example
-------

.. code-block:: yaml

  - "stream.Broadcast":
    Filter: "filter.RegExp"
    FilterExpression: "^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+$"
    FilterExpressionNot: "foo.bar$"
