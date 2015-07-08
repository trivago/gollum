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
    Defines either one or an aray of stream names this configuration applies to.
**Routes**
    Routes defines a 1:n stream remapping.
    Messages reaching the Route stream are reassigned to the given stream(s).
    If no Route is set the message will be broadcasted to all producers attached to this stream.

Example
-------

.. code-block:: yaml

  - "stream.Route":
    Enable: true
    Filter: "filter.All"
    Formatter: "format.Forward"
    Stream:
      - "data"
      - "_DROPPED_"
    Routes:
      "_DROPPED_": "myStream"
      "data":
        - "db1"
        - "db2"
