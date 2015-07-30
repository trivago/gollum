Hostname
========

Prefixes a message with the hostname.

Parameters
----------

**HostnameDataFormatter**
  Defines the formatter that is applied before adding the hostname.
  By default this is set to :doc:`Format.Forward </formatters/forward>`.

**HostnameSeparator**
  Sets the separator character placed after the hostname.
  This is set to " " by default.

Example
-------

.. code-block:: yaml

  - "stream.Broadcast":
    Formatter: "format.Hostname"
    HostnameFormatter: "format.Forward"
    HostnameSeparator: " "
