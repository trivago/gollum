Plugins
#######

The main components, `consumers`_, `routers`_, `producers`_, `filters`_ and `formatters`_
are build upon a plugin architecture.
This allows each component to be exchanged and configured individually with a different sets of options.

.. image:: /src/flow_800w.png

.. _consumers: ../gen/consumer/index.html
.. _routers:  ../gen/router/index.html
.. _producers: ../gen/producer/index.html
.. _filters: ../gen/filter/index.html
.. _formatters: ../gen/formatter/index.html

**Plugin types:**

.. toctree::
    :maxdepth: 1

    /src/gen/consumer/index
    /src/gen/producer/index
    /src/gen/router/index
    /src/gen/filter/index
    /src/gen/formatter/index


Aggregate plugins
==================

To simplify complex pipeline configs you are able to aggregate plugin configurations.
That means that all settings witch are defined in the aggregation scope will injected to each defined "sub-plugin".

To define an aggregation use the keyword **Aggregate** as plugin type.


Parameters
----------

**Plugins**

  List of plugins witch will instantiate and get the aggregate settings injected.



Examples
--------

In this example both consumers get the streams and modulator injected from the aggregation settings:

.. code-block:: yaml

     AggregatePipeline:
       Type: Aggregate
       Streams: console
       Modulators:
         - format.Envelope:
             Postfix: "\n"
       Plugins:
         consumerFoo:
           Type: consumer.File
           File: /tmp/foo.log
         consumerBar:
           Type: consumer.File
           File: /tmp/bar.log


This example shows a second use case to reuse server settings easier:

.. code-block:: yaml

     consumerConsole:
       Type: consumer.Console
       Streams: write

     kafka:
       Type: Aggregate
       Servers:
         - kafka0:9092
         - kafka1:9093
         - kafka2:9094

       Plugins:
         producer:
           Type: producer.Kafka
           Streams: write
           Compression: zip
           Topics:
             write: test

         consumer:
           Type: consumer.Kafka
           Streams: read
           Topic: test
           DefaultOffset: Oldest

     producerConsole:
       Type: producer.Console
       Streams: read