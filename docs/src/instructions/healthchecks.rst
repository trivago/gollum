Health checks
=============

Gollum provide optional http endpoints for health checks.

To activate the health check endpoints you need to start the gollum process with the `"-hc <address:port>"` option_.
If gollum is running with the `"-hc"` option you are able to request different http endpoints
to get global- and plugin health status.

.. _option: http://gollum.readthedocs.io/en/latest/src/instructions/usage.html#commandline


.. code-block:: bash

    # start gollum on host with health check endpoints
    gollum -hc 8080 -c /my/config/file.conf


Endpoints
---------

**/_ALL_**

Request:

.. code-block:: bash

    curl -i 127.0.0.1:8080/_ALL_

Response:

.. code-block:: text

    HTTP/1.1 200 OK
    Date: Fri, 04 Aug 2017 16:03:22 GMT
    Content-Length: 191
    Content-Type: text/plain; charset=utf-8

    /pluginID-A/pluginState 200 ACTIVE: Active
    /pluginID-B/pluginState 200 ACTIVE: Active
    /pluginID-C/pluginState 200 ACTIVE: Active
    /pluginID-D/pluginState 200 ACTIVE: Active
    /_PING_ 200 PONG


**/_PING_**

Request:

.. code-block:: bash

    curl -i 127.0.0.1:8080/_PING_

Response:

.. code-block:: text

    HTTP/1.1 200 OK
    Date: Fri, 04 Aug 2017 15:46:34 GMT
    Content-Length: 5
    Content-Type: text/plain; charset=utf-8

    PONG

**/<PLUGIN_ID>/pluginState**

Request:

.. code-block:: bash

    # example request with active `producer.benchmark`
    curl -i 127.0.0.1:8080/pluginID-A/pluginState

Response:

.. code-block:: text

    HTTP/1.1 200 OK
    Date: Fri, 04 Aug 2017 15:47:45 GMT
    Content-Length: 15
    Content-Type: text/plain; charset=utf-8

    ACTIVE: Active