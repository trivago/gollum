Performance tests
=================

History
-------

All tests were executed by calling ``time gollum -c profile.conf -ll 1``.

+------------------+------------+---------+--------+------+-----------+ 
| test             | ver        | user    | sys    | cpu  | msg/sec   |
+==================+============+=========+========+======+===========+ 
| Raw pipeline     | 0.6.0      | 11,45s  | 3,13s  | 200% | 1.316.153 |
+------------------+------------+---------+--------+------+-----------+ 
|                  | 0.5.0      | 11,85s  | 3,69s  | 173% | 1.116.071 |
+------------------+------------+---------+--------+------+-----------+ 
|                  | 0.4.6      | 10,48s  | 3,01s  | 178% | 1.320.132 |
+------------------+------------+---------+--------+------+-----------+ 
| Basic formatting | 0.6.0      | 37,63s  | 4,01s  | 520% | 1.173.945 |
+------------------+------------+---------+--------+------+-----------+ 
|                  | 0.5.0      | 39,70s  | 6,09s  | 532% | 1.163.602 |
+------------------+------------+---------+--------+------+-----------+ 
|                  | 0.4.6 [#]_ | 21,84s  | 5,78s  | 206% | 746.881   |
+------------------+------------+---------+--------+------+-----------+ 
| 8 consumers      | 0.6.0      | 325,18s | 18,32s | 511% | 1.137.784 |
+------------------+------------+---------+--------+------+-----------+  
|                  | 0.5.0      | 344,33s | 28,24s | 673% | 1.446.157 |
+------------------+------------+---------+--------+------+-----------+  
|                  | 0.4.6      | 319,44s | 72,22s | 574% | 1.173.536 |
+------------------+------------+---------+--------+------+-----------+ 
| JSON pipeline    | 0.6.0      | 14,98s  | 4,77s  | 173% | 78.377    |
+------------------+------------+---------+--------+------+-----------+ 
|                  | 0.5.0      | 28,23s  | 6,33s  | 138% | 40.033    |
+------------------+------------+---------+--------+------+-----------+ 
|                  | 0.4.6      | 28,30s  | 6,30s  | 150% | 43.400    |
+------------------+------------+---------+--------+------+-----------+ 

.. [#] this version does not use paralell formatting

v0.6.0
------

JSON pipeline
``````````````
| Intel Core i7-7700HQ CPU @ 2.80GHz, 16 GB RAM
| go1.12.3 darwin/amd64

 * 14,98s user 
 * 4,77s system 
 * 173% cpu 
 * 11,388s total
 * 73.846 msg/sec

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
            - format.JSON: {}
            - format.Move:
                Source: "test"
                Target: "foobar"
            - format.Delete:
                Target: "bar"
            - format.SplitToFields:
                Source: "foo"
                Delimiter: "|"
                Fields: ["foo1","foo2"]
            - format.Copy:
                Source: "thisisquitealongstring"

    "Benchmark":
        Type: "producer.Benchmark"
        Streams: "profile"

v0.5.0
------

Raw pipeline
````````````
| Intel Core i7-4770HQ CPU @ 2.20GHz, 16 GB RAM
| go1.8.3 darwin/amd64

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
| Intel Core i7-4770HQ CPU @ 2.20GHz, 16 GB RAM
| go1.8.3 darwin/amd64
| Please note that from this version on formatting is done in parallel.

 * 39,70s user 
 * 6,09s system 
 * 532% cpu 
 * 8,594s total
 * 1.163.602 msg/sec

.. code:: yaml

    "Profiler":
        Type: "consumer.Profiler"
        Runs: 100000
        Batches: 100
        Characters: "abcdefghijklmnopqrstuvwxyz .,!;:-_"
        Message: "%256s"
        Streams: "profile"
        KeepRunning: false
        ModulatorRoutines: 4
        Modulators:
            - format.Envelope
            - format.Timestamp

    "Benchmark":
        Type: "producer.Benchmark"
        Streams: "profile"


8 consumers with formatting
```````````````````````````
| Intel Core i7-4770HQ CPU @ 2.20GHz, 16 GB RAM
| go1.8.3 darwin/amd64

 * 344,33s user 
 * 28,24s system 
 * 673% cpu 
 * 55,319s total
 * 1.446.157 msg/sec

.. code:: yaml

     "Profiler":
        Type: Aggregate
        Runs: 100000
        Batches: 100
        Characters: "abcdefghijklmnopqrstuvwxyz .,!;:-_"
        Message: "%256s"
        Streams: "profile"
        KeepRunning: false
        ModulatorRoutines: 0
        Modulators:
            - format.Envelope
            - format.Timestamp
        Plugins:
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
| Intel Core i7-4770HQ CPU @ 2.20GHz, 16 GB RAM
| go1.8.3 darwin/amd64

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
| Intel Core i7-4770HQ CPU @ 2.20GHz, 16 GB RAM
| go1.8.3 darwin/amd64

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
| Intel Core i7-4770HQ CPU @ 2.20GHz, 16 GB RAM
| go1.8.3 darwin/amd64

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
| Intel Core i7-4770HQ CPU @ 2.20GHz, 16 GB RAM
| go1.8.3 darwin/amd64

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
| Intel Core i7-4770HQ CPU @ 2.20GHz, 16 GB RAM
| go1.8.3 darwin/amd64

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
