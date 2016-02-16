Base64Decode
============

Base64Decode is a formatter that decodes a base64 message.
If a message is not or only partly base64 encoded an error will be logged and the decoded part is returned.
RFC 4648 is expected.


Parameters
----------

**Base64Dictionary**
  Base64Dictionary defines the 64-character base64 lookup dictionary to use.
  When left empty a dictionary as defined by RFC4648 is used.
  This is the default.

**Base64DataFormatter**
  Base64DataFormatter defines a formatter that is applied before the base64 decoding takes place.
  By default this is set to "format.Forward" .

Example
-------

.. code-block:: yaml

	    - "stream.Broadcast":
	        Formatter: "format.Base64Decode"
	        Base64Formatter: "format.Forward"
	        Base64Dictionary: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz01234567890+/"
