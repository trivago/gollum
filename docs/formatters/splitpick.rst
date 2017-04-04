SplitPick
=========

SplitPick separates value of messages according to a specified delimiter and returns the given indexed message.
The index are zero based.


Parameters
----------

**By**
  	By default, SplitPickIndex is 0.
  	By default, SplitPickDelimiter is ":".

Example
-------

.. code-block:: yaml

	- "stream.Broadcast":
	    Formatter: "format.SplitPick"
	        SplitPickIndex: 0
	        SplitPickDelimiter: ":"
