Random
======

This stream passes a message to one random producer listening to the streams defined with the stream parameter.

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

Example
-------

.. code-block:: yaml

  - "stream.Random":
    Enable: true
    Filter: "filter.All"
    Formatter: "format.Forward"
    Stream:
        - "logs"
        - "console"
