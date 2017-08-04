Performance tests
=================

History
-------

+------------------+-------+---------+--------+------+-----------+ 
| test             | ver   | user    | sys    | cpu  | msg/sec   |
+==================+=======+=========+========+======+===========+ 
| Raw pipeline     | 0.5.0 | 11,85s  | 3,69s  | 173% | 1.116.071 |
+------------------+-------+---------+--------+------+-----------+ 
|                  | 0.4.6 | 10,48s  | 3,01s  | 178% | 1.320.132 |
+------------------+-------+---------+--------+------+-----------+ 
| Basic formatting | 0.5.0 | 46,30s  | 7,06s  | 591% | 1.108.278 |
+------------------+-------+---------+--------+------+-----------+ 
|                  | 0.4.6 | 21,84s  | 5,78s  | 206% | 746.881   |
+------------------+-------+---------+--------+------+-----------+ 
| 8 consumers      | 0.5.0 | 344,33s | 28,24s | 673% | 1.446.157 |
+------------------+-------+---------+--------+------+-----------+ 
|                  | 0.4.6 | 319,44s | 72,22s | 574% | 1.173.536 |
+------------------+-------+---------+--------+------+-----------+ 
| JSON pipeline    | 0.5.0 | 28,23s  | 6,33s  | 138% | 40.033    |
+------------------+-------+---------+--------+------+-----------+ 
|                  | 0.4.6 | 28,30s  | 6,30s  | 150% | 43.400    |
+------------------+-------+---------+--------+------+-----------+ 

v0.5.0
------

Raw pipeline
````````````
Intel Core i7-4770HQ CPU @ 2.20GHz, 16 GB RAM

 * 11,85s user 
 * 3,69s system 
 * 173% cpu 
 * 8,960s total
 * 1.116.071 msg/sec

.. code:: yaml

    "Profiler":
        Type: "consumer.Profiler"
        Runs: 100000
        Batches: 100
        Characters: "abcdefghijklmnopqrstuvwxyz .,!;:-_"
        Message: "%256s"
        Streams: "profile"
        KeepRunning: false
        ModulatorRoutines: 0

    "Benchmark":
        Type: "producer.Benchmark"
        Streams: "profile"


Basic formatting
`````````````````
Intel Core i7-4770HQ CPU @ 2.20GHz, 16 GB RAM

 * 20,48s user 
 * 6,11s system 
 * 175% cpu 
 * 15,150s total
 * 660.066 msg/sec

.. code:: yaml

    "Profiler":
        Type: "consumer.Profiler"
        Runs: 100000
        Batches: 100
        Characters: "abcdefghijklmnopqrstuvwxyz .,!;:-_"
        Message: "%256s"
        Streams: "profile"
        KeepRunning: false
        ModulatorRoutines: 8
        Modulators:
            - format.Envelope
            - format.Timestamp

    "Benchmark":
        Type: "producer.Benchmark"
        Streams: "profile"


 * 46,30s user 
 * 7,06s system 
 * 591% cpu 
 * 9,023s total
 * 1.108.278 msg/sec

.. code:: yaml

    "Profiler":
        Type: "consumer.Profiler"
        Runs: 100000
        Batches: 100
        Characters: "abcdefghijklmnopqrstuvwxyz .,!;:-_"
        Message: "%256s"
        Streams: "profile"
        KeepRunning: false
        ModulatorRoutines: 8
        Modulators:
            - format.Envelope
            - format.Timestamp

    "Benchmark":
        Type: "producer.Benchmark"
        Streams: "profile"


8 consumers with formatting
```````````````````````````
Intel Core i7-4770HQ CPU @ 2.20GHz, 16 GB RAM

 * 344,33s user 
 * 28,24s system 
 * 673% cpu 
 * 55,319s total
 * 1.446.157 msg/sec

.. code:: yaml

     "Profiler":
        Type: aggregate
        Runs: 100000
        Batches: 100
        Characters: "abcdefghijklmnopqrstuvwxyz .,!;:-_"
        Message: "%256s"
        Streams: "profile"
        KeepRunning: false
        ModulatorRoutines: 8
        Modulators:
            - format.Envelope
            - format.Timestamp
        Aggregate:
            P01:
                Type: "consumer.Profiler"
            P02:
                Type: "consumer.Profiler"
            P03:
                Type: "consumer.Profiler"
            P04:
                Type: "consumer.Profiler"
            P05:
                Type: "consumer.Profiler"
            P06:
                Type: "consumer.Profiler"
            P07:
                Type: "consumer.Profiler"
            P08:
                Type: "consumer.Profiler"

    "Benchmark":
        Type: "producer.Benchmark"
        Streams: "profile"


JSON pipeline
``````````````
Intel Core i7-4770HQ CPU @ 2.20GHz, 16 GB RAM

 * 28,23s user 
 * 6,33s system 
 * 138% cpu 
 * 24,979s total
 * 40.033 msg/sec

.. code:: yaml

    "Profiler":
        Type: consumer.Profiler
        Runs: 10000
        Batches: 100
        Characters: "abcdefghijklmnopqrstuvwxyz .,!;:-_"
        Message: "{\"test\":\"%64s\",\"foo\":\"%32s|%32s\",\"bar\":\"%64s\",\"thisisquitealongstring\":\"%64s\"}"
        Streams: "profile"
        KeepRunning: false
        ModulatorRoutines: 0
        Modulators:
            - format.ProcessJSON:
                Directives:
                    - "test:rename:foobar"
                    - "bar:remove"
                    - "foo:split:|:foo1:foo2"
            - format.ExtractJSON:
                Field: thisisquitealongstring

    "Benchmark":
        Type: "producer.Benchmark"
        Streams: "profile"


v0.4.6
------

Raw pipeline
````````````
Intel Core i7-4770HQ CPU @ 2.20GHz, 16 GB RAM

 * 10,48s user 
 * 3,01s system 
 * 178% cpu 
 * 7,575s total
 * 1.320.132 msg/sec

.. code:: yaml

    - "consumer.Profiler":
        Runs: 100000
        Batches: 100
        Characters: "abcdefghijklmnopqrstuvwxyz .,!;:-_"
        Message: "{\"test\":\"%64s\",\"foo\":\"%32s|%32s\",\"bar\":\"%64s\",\"thisisquitealongstring\":\"%64s\"}"
        Stream: "profile"
        KeepRunning: false

    - "producer.Benchmark":
        Stream: "profile"


Basic formatting
`````````````````
Intel Core i7-4770HQ CPU @ 2.20GHz, 16 GB RAM
 
 * 21,84s user 
 * 5,78s system 
 * 206% cpu 
 * 13,389s total
 * 746.881 msg/sec

.. code:: yaml

    - "consumer.Profiler":
        Runs: 100000
        Batches: 100
        Characters: "abcdefghijklmnopqrstuvwxyz .,!;:-_"
        Message: "%256s"
        Stream: "profile"
        KeepRunning: false

    - "stream.Broadcast":
        Stream: "profile"
        Formatter: format.Timestamp
        TimestampFormatter: format.Envelope

    - "producer.Benchmark":
        Stream: "profile"


8 consumers with formatting
```````````````````````````
Intel Core i7-4770HQ CPU @ 2.20GHz, 16 GB RAM

 * 319,44s user 
 * 72,22s system 
 * 574% cpu 
 * 68,17s total
 * 1.173.536 msg/sec

.. code:: yaml

    - "consumer.Profiler":
        Instances: 8
        Runs: 100000
        Batches: 100
        Characters: "abcdefghijklmnopqrstuvwxyz .,!;:-_"
        Message: "%256s"
        Stream: "profile"
        KeepRunning: false

    - "stream.Broadcast":
        Stream: "profile"
        Formatter: format.Timestamp
        TimestampFormatter: format.Envelope

    - "producer.Benchmark":
        Stream: "profile"

JSON pipeline
``````````````
Intel Core i7-4770HQ CPU @ 2.20GHz, 16 GB RAM

 * 28,30s user 
 * 6,30s system 
 * 150% cpu 
 * 23,041s total
 * 43.400 msg/sec

.. code:: yaml

    - "consumer.Profiler":
        Runs: 10000
        Batches: 100
        Characters: "abcdefghijklmnopqrstuvwxyz .,!;:-_"
        Message: "%256s"
        Stream: "profile"
        KeepRunning: false

    - "stream.Broadcast":
        Stream: "profile"
        Formatter: format.ExtractJSON
        ExtractJSONdataFormatter: format.ProcessJSON
        ProcessJSONDirectives:
            - "test:rename:foobar"
            - "bar:remove"
            - "foo:split:|:foo1:foo2"
        ExtractJSONField: thisisquitealongstring

    - "producer.Benchmark":
        Stream: "profile"
