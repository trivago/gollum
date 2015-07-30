Base64Encode
============

This formatter encodes messages as Base64.

Parameters
----------

**Base64Dictionary**
  Defines the 64-character base64 lookup dictionary to use.
  When left empty a dictionary as defined by RFC4648 is used.
  This is the default.

**Base64Formatter**
  Defines a formatter that is applied before the base64 encoding takes place.
  By default this is set to :doc:`Format.Forward </formatters/forward>`.

Example
-------

.. code-block:: yaml

  - "stream.Broadcast":
    Formatter: "format.Base64Encode"
    Base64Formatter: "format.Forward"
    Base64Dictionary: "abcdefghijklmnopqrstufvxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890+/"
