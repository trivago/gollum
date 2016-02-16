Base64Encode
============

Base64Encode is a formatter that encodes a message as base64.


Parameters
----------

**Base64Dictionary**
  Base64Dictionary defines the 64-character base64 lookup dictionary to use.
  When left empty a dictionary as defined by RFC4648 is used.
  This is the default.

**Base64DataFormatter**
  Base64DataFormatter defines a formatter that is applied before the base64 encoding takes place.
  By default this is set to "format.Forward" .

Example
-------

.. code-block:: yaml

- "stream.Broadcast":
    Formatter: "format.Base64Encode"
    Base64Formatter: "format.Forward"
    Base64Dictionary: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz01234567890+/"
