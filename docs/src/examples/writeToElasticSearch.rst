Write to Elasticsearch (ElasticSearch producer)
===============================================

Description
-----------

This example can be used for developing or testing the ElasticSearch
producer.

gollum config
-------------

With the following config gollum will create a ``console.consumer`` with
a ``ElasticSearch.producer``. All data which write to the console will
send to ElasticSearch.

This payload can be used for the configured setup:

.. code:: json

    {"user" : "olivere", "message" : "It's a Raggy Waltz"}

gollum >= v0.5.0
~~~~~~~~~~~~~~~~

.. code:: yaml

    consumerConsole:
        type: consumer.Console
        Streams: "write"

    producerElastic:
        Type: producer.ElasticSearch
        Streams: write
        User: elastic
        Password: changeme
        Servers:
            - http://127.0.0.1:9200
        Retry:
            Count: 3
            TimeToWaitSec: 5
        SetGzip: true
        StreamProperties:
            write:
                Index: twitter
                DayBasedIndex: true
                Type: tweet
                Mapping:
                    user: keyword
                    message: text
                Settings:
                    number_of_shards: 1
                    number_of_replicas: 1

This config example can also be found `here`_

ElasticSearch setup for docker
------------------------------

Here you find a docker-compose setup which works for the config example:

.. code:: yaml

    version: '2'
    services:
      elasticsearch1:
        image: docker.elastic.co/elasticsearch/elasticsearch:5.4.1
        container_name: elasticsearch1
        environment:
          - cluster.name=docker-cluster
          - bootstrap.memory_lock=true
          - "ES_JAVA_OPTS=-Xms512m -Xmx512m"
        ulimits:
          memlock:
            soft: -1
            hard: -1
        mem_limit: 1g
        volumes:
          - esdata1:/usr/share/elasticsearch/data
        ports:
          - 9200:9200
        networks:
          - esnet
      elasticsearch2:
        image: docker.elastic.co/elasticsearch/elasticsearch:5.4.1
        environment:
          - cluster.name=docker-cluster
          - bootstrap.memory_lock=true
          - "ES_JAVA_OPTS=-Xms512m -Xmx512m"
          - "discovery.zen.ping.unicast.hosts=elasticsearch1"
        ulimits:
          memlock:
            soft: -1
            hard: -1
        mem_limit: 1g
        volumes:
          - esdata2:/usr/share/elasticsearch/data
        networks:
          - esnet

    volumes:
      esdata1:
        driver: local
      esdata2:
        driver: local

    networks:
      esnet:

This docker-compose file can be run by:

.. code:: bash

    docker-compose -f docker-compose-elastic.yml up

.. _here: https://github.com/trivago/gollum/blob/master/config/console_elastic.conf