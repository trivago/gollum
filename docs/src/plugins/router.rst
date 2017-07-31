.. This file is included by docs/src/gen/router/index.rst

Routers
############################

Routers manage the transfer of messages between  :doc:`consumers </src/plugins/consumer>` and :doc:`producers </src/plugins/producer>` by `streams`.
Routers can act as a kind of proxy that may filter and define the distribution algorithm of messages.

The stream names can be referred to by cleartext names. This stream names are free to choose but there are several reserved names for internal or special purpose:

:_GOLLUM_:     is used for internal log messages
:\*:           is a placeholder for "all routers but the internal routers". In some cases "*" means "all routers" without exceptions. This is denoted in the corresponding documentations whenever this is the case.


**Basics router setups:**

.. image:: /src/router_800w.png

**List of available Router:**