.. This file is included by docs/src/gen/router/index.rst

Routers
############################

Routers manage the transfer of messages between  :doc:`consumers </src/plugins/consumer>` and :doc:`producers </src/plugins/producer>`.
Routers can act as a kind of proxy that may filter, modify and define the distribution algorithm of messages.
Routers can be referred to by cleartext names. This names are free to choose but there are several reserved names for internal or special purpose routers:

- **"\_GOLLUM\_"** is used for internal log messages
- **"\_DROPPED\_"** is used for messages that could not be sent, e.g. because of a channel timeout
- **"*"** is a placeholder for "all routers but the internal routers".
  In some cases "*" means "all routers" without exceptions. This is denoted in the corresponding documentations whenever this is the case.
