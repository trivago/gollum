JSON
#############

This filter tries to parse a text based message as JSON.
If the message is not valiud JSON, the message is blocked.
You can define whitelists and blacklists to check the contents of JSON fields by using regular expressions.

- FilterReject defines a blacklist. If the given field does **not** exist or the given expression does **not match**, the message is passed.
- FilterAccept defines a whitelist. If the given field does **not** exist or the given expression **matches**, the message is passed.

If multiple rules apply to one dataset all rules must pass for a message to get passed.

If your JSON dataset contains nested fields or arrays you can use a simplified path notation to access these fields.
Nested fields are accesed by using a forward slash "/" as a delimiter.
Array values can be accessed by using square brackets "[]", i.e. standard array notation.
If an array value contains a nested structure the path starts directly after the bracket without an additional forward slash.

Parameters
----------

**FilterReject**
  Blacklist filters that must match to block a message. Empty by default.

**FilterAccept**
  Blacklist filters that must match to pass a message. Empty by default.

Example
-------

::

  - "stream.Broadcast":
    Filter: "filter.JSON"
    FilterReject:
      "command" : "state\d\..*"
    FilterAccept:
      "args/results[0]value" : "true"
      "args/results[1]" : "true"
      "command" : "state\d\..*"
