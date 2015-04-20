Round Robin
===========

This stream passes a message to one producer listening to the streams defined with the stream parameter.
The producers will be iterated in a round robin fashion.

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

Example
-------

.. code-block:: yaml

  - "stream.RoundRobin":
    Enable: true
    Filter: "filter.All"
    Formatter: "format.Forward"
    Stream:
        - "logs"
        - "console"
