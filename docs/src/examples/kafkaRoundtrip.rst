Kafka roundtrip
===============

This example can be used for developing or testing kafka consumers and
producers.

gollum config
-------------

With the following config gollum will create a ``console.consumer`` with
a ``kafka.producer`` and a ``kafka.consumer`` with a
``console.producer``. All data which write to the console will send to
kafka. The second ``kafka.consumer`` will read all data from kafka and
send it back to your console by the ``console.producer``:

gollum >= v0.5.0
~~~~~~~~~~~~~~~~

.. code:: yaml

    consumerConsole:
        type: consumer.Console
        Streams: "write"

    producerKafka:
        type: producer.Kafka
        Streams: "write"
        Compression: "zip"
        Topics:
            "write" : "test"
        Servers:
            - kafka0:9092
            - kafka1:9093
            - kafka2:9094

    consumerKafka:
        type: consumer.Kafka
        Streams: "read"
        Topic: "test"
        DefaultOffset: "Oldest"
        MaxFetchSizeByte: 100
        Servers:
            - kafka0:9092
            - kafka1:9093
            - kafka2:9094

    producerConsole:
        type: producer.Console
        Streams: "read"
        Modulators:
            - format.Envelope:
                Postfix: "\n"

This config example can also be found `here`_

kafka setup for docker
----------------------

Here you find a docker-compose setup which works for the gollum config
example.

/etc/hosts entry
~~~~~~~~~~~~~~~~

You need a valid ``/etc/hosts`` entry to be able to use the set
hostnames:

::

    # you can not use 127.0.0.1 or localhost here
    <YOUR PUBLIC IP> kafka0 kafka1 kafka2

docker-compose file
~~~~~~~~~~~~~~~~~~~

.. code:: yaml

    zookeeper:
      image: wurstmeister/zookeeper
      ports:
        - "2181:2181"
        - "2888:2888"
        - "3888:3888"
    kafkaone:
      image: wurstmeister/kafka:0.10.0.0
      ports:
        - "9092:9092"
      links:
        - zookeeper:zookeeper
      volumes:
        - /var/run/docker.sock:/var/run/docker.sock
      environment:
        KAFKA_ADVERTISED_HOST_NAME: kafka0
        KAFKA_ZOOKEEPER_CONNECT: "zookeeper"
        KAFKA_BROKER_ID: "21"
        KAFKA_CREATE_TOPICS: "test:1:3,Topic2:1:1:compact"
    kafkatwo:
      image: wurstmeister/kafka:0.10.0.0
      ports:
        - "9093:9092"
      links:
        - zookeeper:zookeeper
      volumes:
        - /var/run/docker.sock:/var/run/docker.sock
      environment:
        KAFKA_ADVERTISED_HOST_NAME: kafka1
        KAFKA_ZOOKEEPER_CONNECT: "zookeeper"
        KAFKA_BROKER_ID: "22"
        KAFKA_CREATE_TOPICS: "test:1:3,Topic2:1:1:compact"
    kafkathree:
      image: wurstmeister/kafka:0.10.0.0
      ports:
        - "9094:9092"
      links:
        - zookeeper:zookeeper
      volumes:
        - /var/run/docker.sock:/var/run/docker.sock
      environment:
        KAFKA_ADVERTISED_HOST_NAME: kafka2
        KAFKA_ZOOKEEPER_CONNECT: "zookeeper"
        KAFKA_BROKER_ID: "23"
        KAFKA_CREATE_TOPICS: "test:1:3,Topic2:1:1:compact"

This docker-compose file can be run by

.. code:: bash

    docker-compose -f docker-compose-kafka.yml -p kafka010 up

.. _here: https://github.com/trivago/gollum/blob/master/config/kafka_roundtrip.conf