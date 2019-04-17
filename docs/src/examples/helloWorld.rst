Hello World Examples
====================

Hello World
-----------

This example sets up a simple console consumer and producer that will simply echo everything you type back to the console.
As messages have no new line appended by default an envelope formatter is used to add one before writing to console.
Make sure to start Gollum with `gollum -ll 3` to see all log messages.

::

  'StdIn':
    Type: 'consumer.Console'
    Streams: 'console'
  
  'StdOut':
    Type: 'producer.Console'
    Streams: 'console'
    Modulators:
    - 'format.Envelope': {}


Loadbalancer
------------

This example extends the Hello World example by introducing a route configuration.
All messages from the console consumer will be sent to a round robin loadbalancer that will forward messages to one of the two attached producers.
Make sure to start Gollum with `gollum -ll 3` to see all log messages.

::

  'StdIn':
    Type: 'consumer.Console'
    Streams: 'console'

  'loadbalancer':
    Type: 'router.RoundRobin'
    Stream: 'console'

  'StdOut1':
    Type: 'producer.Console'
    Streams: 'console'
    Modulators:
    - 'format.Envelope':
        Prefix: '1: '

  'StdOut2':
    Type: 'producer.Console'
    Streams: 'console'
    Modulators:
    - 'format.Envelope':
        Prefix: '2: '

When you remove the router from the config you will see each message to reach both producers.

Hello World filtered
--------------------

This example extends the previous example by setting up a filter to only echo sentences that end with the word "gollum".
A regular expression filter is used to achieve this.
Note that this filter does not apply to standard log messages.
Make sure to start Gollum with `gollum -ll 3` to see all log messages.

::

  'StdIn':
    Type: 'consumer.Console'
    Streams: 'console'

  'loadbalancer':
    Type: 'router.RoundRobin'
    Stream: 'console'
    Filters:
    - 'filter.RegExp':
        FilterExpression: ".*gollum$"

  'StdOut1':
    Type: 'producer.Console'
    Streams: 'console'
    Modulators:
    - 'format.Envelope':
        Prefix: '1: '

  'StdOut2':
    Type: 'producer.Console'
    Streams: 'console'
    Modulators:
    - 'format.Envelope':
        Prefix: '2: '

You can also attach filters to the modulators section of a consumer or a producer.
Please note that routers can filter but not modify messages.

Hello World splitter
--------------------

This example extends the first example by introducing a stream split.
This time we will print the console output twice, encoded as XML and as JSON.
Make sure to start Gollum with `gollum -ll 3` to see all log messages.

::

  'StdIn':
    Type: 'consumer.Console'
    Streams: 'console'
  
  'StdOutXML':
    Type: 'producer.Console'
    Streams: 'console'
    Modulators:
    - 'format.Envelope':
        Prefix: '<msg>'
        Postfix: '</msg>\n'

  'StdOutJSON':
    Type: 'producer.Console'
    Streams: 'console'
    Modulators:
    - 'format.Envelope':
        Prefix: '{"msg":"'
        Postfix: '"}\n'

You can also do this in a slightly different way by utilizing two streams.
When doing this you can filter or route both streams differently.
In this extended example, every second example will output only JSON.

::

  'StdIn':
    Type: 'consumer.Console'
    Streams: 
    - 'consoleJSON'
    - 'consoleXML'

  'xmlFilter':
    Type: 'router.Broadcast'
    Stream: 'consoleXML'
    Filters:
    - 'filter.Sample': {}
  
  'StdOutXML':
    Type: 'producer.Console'
    Streams: 'consoleXML'
    Modulators:
    - 'format.Envelope':
        Prefix: '<msg>'
        Postfix: '</msg>\n'

  'StdOutJSON':
    Type: 'producer.Console'
    Streams: 'consoleJSON'
    Modulators:
    - 'format.Envelope':
        Prefix: '{"msg":"'
        Postfix: '"}\n'

Chat server
-----------

This example requires two Gollum instances to run.
The first one acts as the "chat client" while the second one acts as the "chat server".
Messages entered on the client will be sent to the server using runlength encoding.
When the message reaches the server, it will be decoded and written to the console.
If the server does not respond, the message will be sent to the fallback and displayed as an error.
Make sure to start Gollum with `gollum -ll 3` to see all log messages.

**Client**
::

  'StdIn':
    Type: 'consumer.Console'
    Streams: 'console'

  'SocketOut':
    Type: 'producer.Socket'
    Streams: 'console'
    Address: ':5880'
    Acknowledge: 'OK'
    FallbackStream: 'failed'
    Modulators: 
    - 'format.Runlength': {}

  'Failed':
    Type: 'producer.Console'
    Streams: 'failed'
    Modulators:
    - 'format.Envelope':
        Prefix: 'Failed to sent: '

**Server**
::

  'SocketIn':
    Type: 'consumer.Socket'
    Streams: 'socket'
    Address: ":5880"
    Acknowledge: 'OK'
    Partitioner: 'ascii'
    Delimiter: ':'
  
  'StdOut':
    Type: 'producer.Console'
    Streams: 'socket'
    Modulators:
    - 'format.Envelope': {}

Profiling
---------

This configuration will test Gollum for its theoretic maximum message throughput.
You can of course modify this example to test e.g. file producer performance.
Make sure to start Gollum with `gollum -ll 3 -ps` to see all log messages as well as intermediate profiling results.

::

'Profiler':
  Type: 'consumer.Profiler'
  Streams: 'profile'
  Runs: 100000
  Batches: 100
  Characters: 'abcdefghijklmnopqrstuvwxyz .,!;:-_'
  Message: '%256s'
  KeepRunning: false
  ModulatorRoutines: 0

'Benchmark':
  Type: 'producer.Benchmark'
  Streams: 'profile'

