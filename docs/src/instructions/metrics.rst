Metrics
=======

Gollum provide various metrics which can be used for monitoring or controlling.


Collecting metrics
------------------

To collect metrics you need to start the gollum process with the `"-m <address:port>"` option_.
If gollum is running with the `"-m"` option you are able to get all collected metrics by a tcp request in
json format.

.. _option: http://gollum.readthedocs.io/en/latest/src/instructions/usage.html#commandline


Example request:

.. code-block:: bash

    # start gollum on host
    gollum -m 8080 -c /my/config/file.conf

    # get metrics by curl and prettify response by python
    curl 127.0.0.1:8080 | python -m json.tool

    # alternative by netcat
    nc -d 127.0.0.1 8080 | python -m json.tool

Example response

.. code-block:: json

    {
        "Consumers": 3,
        "GoMemoryAllocated": 21850200,
        "GoMemoryGCEnabled": 1,
        "GoMemoryNumObjects": 37238,
        "GoRoutines": 20,
        "GoVersion": 10803,
        "Messages:Discarded": 0,
        "Messages:Discarded:AvgPerSec": 0,
        "Messages:Enqueued": 13972236,
        "Messages:Enqueued:AvgPerSec": 1764931,
        "Messages:Routed": 13972233,
        "Messages:Routed:AvgPerSec": 1764930,
        "Plugins:ActiveWorkers": 3,
        "Plugins:State:Active": 4,
        "Plugins:State:Dead": 0,
        "Plugins:State:Initializing": 0,
        "Plugins:State:PrepareStop": 0,
        "Plugins:State:Stopping": 0,
        "Plugins:State:Waiting": 0,
        "ProcessStart": 1501855102,
        "Producers": 1,
        "Routers": 2,
        "Routers:Fallback": 2,
        "Stream:profile:Messages:Discarded": 0,
        "Stream:profile:Messages:Discarded:AvgPerSec": 0,
        "Stream:profile:Messages:Routed": 13972239,
        "Stream:profile:Messages:Routed:AvgPerSec": 1768878,
        "Version": 500
    }


Metrics overview
----------------


Global metrics
``````````````

**Consumers**

  Number of current active consumers.

**GoMemoryAllocated**

  Current allocated memory in bytes.

**GoMemoryGCEnabled**

  Indicates that GC is enabled.

**GoMemoryNumObjects**

  The number of allocated heap objects.

**GoRoutines**

  The number of active go routines.

**GoVersion**

  The golang version in number format.

**Messages:Discarded**

  The count of discarded messages over all.

**Messages:Discarded:AvgPerSec**

  The average of discarded messages from the last seconds.

**Messages:Enqueued**

  The count of enqueued messages over all.

**Messages:Enqueued:AvgPerSec**

  The average of enqueued messages from the last seconds.

**Messages:Routed**

  The count of routed messages over all.

**Messages:Routed:AvgPerSec**

  The average of routed messages from the last seconds.

**Plugins:ActiveWorkers**

  Number of active worker (plugin) processes.

**Plugins:State:<STATE>**

  Number of plugins in specific `states`. The following `states` can possible for plugins:

  * Active
  * Dead
  * Initializing
  * PrepareStop
  * Stopping
  * Waiting

**ProcessStart**

  Timestamp of the process start time.

**Producers**

  Number of current active producers.

**Routers**

  Number of current active routers.

**Routers:Fallback**

  Number of current active "fallback" (auto created) routers.

**Version**

  Gollum version as numeric value.



Stream based metrics
````````````````````

**Stream:<STREAM_NAME>:Messages:Discarded**

  The count of discarded messages for a specific stream.

**Stream:<STREAM_NAME>:Messages:Discarded:AvgPerSec**

  The average of discarded messages from the last seconds for a specific stream.

**Stream:<STREAM_NAME>:Messages:Routed**

  The count of routed messages for a specific stream.

**Stream:<STREAM_NAME>:Messages:Routed:AvgPerSec**

  The average of routed messages from the last seconds for a specific stream.