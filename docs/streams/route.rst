Route
=====

This stream routes messages to other streams by using a 1:n mapping table.

Parameters
----------

**Enable**
    Can either be true or false to enable or disable this stream configuration.
**Filter**
    Defines a message filter to use. :doc:`Filter.All </filters/all>` by default.
**Format**
    Defines a message formatter to use. :doc:`Format.Forward </formatters/forward>` by default.
**Stream**
    Defines the stream to configure.
**Routes**
    Routes defines a 1:n stream remapping.
    Messages are reassigned to all of stream(s) in this list.
    If no route is set messages are forwarded on the incoming stream.
    When routing to multiple streams, the incoming stream has to be listed explicitly to be used.

Example
-------

.. code-block:: yaml

  - "stream.Route":
    Enable: true
    Filter: "filter.All"
    Formatter: "format.Forward"
    Stream: "data"
    Routes:
        - "db1"
        - "db2"
        - "data"
