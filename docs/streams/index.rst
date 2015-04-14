Streams
############################

.. toctree::
	:maxdepth: 1

	broadcast
	roundrobin
	random
    
Streams manage the transfer of messages between  :doc:`consumers </consumers/index>` and :doc:`producers </producers/index>`.
Streams can act as a kind of proxy that may filter, modify and define the distribution algorithm of messages.
Streams can be referred to by cleartext names. This names are free to choose but there are several reserved names for internal or special purpose streams:

- **"\_GOLLUM\_"** is used for internal log messages
- **"\_DROPPED\_"** is used for messages that could not be sent, e.g. because of a channel timeout
- **"*"** is a placeholder for "all streams but the internal streams".
  In some cases "*" means "all streams" without exceptions. This is denoted in the corresponding documentations whenever this is the case.
