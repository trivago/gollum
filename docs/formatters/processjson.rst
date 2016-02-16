ProcessJSON
===========

ProcessJSON is a formatter that allows modifications to fields of a given JSON message.
The message is modified and returned again as JSON.


Parameters
----------

**ProcessJSONDataFormatter**
  ProcessJSONDataFormatter formatter that will be applied before ProcessJSONDirectives are processed.

**ProcessJSONDirectives**
  ProcessJSONDirectives defines the action to be applied to the json payload.
  Directives are processed in order of appearance.
  The directives have to be given in the form of key:operation:parameters, where operation can be one of the following: - split:<string>:{<key>:<key>:...} Split the value by a string and set the resulting array elements to the given fields in order of appearance.
  - replace:<old>:<new> replace a given string in the value with a new one - trim:<characters> remove the given characters (not string!) from the start and end of the value - rename:<old>:<new> rename a given field - timestamp:<read>:<write> read a timestamp and transform it into another format.

**ProcessJSONTrimValues**
  ProcessJSONTrimValues will trim whitspaces from all values if enabled.
  Enabled by default.

Example
-------

.. code-block:: yaml

- "stream.Broadcast":
    Formatter: "format.processJSON"
    ProcessJSONDataFormatter: "format.Forward"
    ProcessJSONDirectives:
        - "host:split: :host:@timestamp"
        - "@timestamp:time:20060102150405:2006-01-02 15\\:04\\:05"
        - "error:replace:°:\n"
        - "text:trim: \t"
        - "foo:rename:bar"
    ProcessJSONTrimFields: true
