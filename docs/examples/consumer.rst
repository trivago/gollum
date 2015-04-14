Writing consumers
=================

When starting to write a consumer its probably a good idea to have a look at already existing consumers.
A good starting point is the console consumer as it is very lightweight.
If you plan to write a special purpose consumer you should place it into "contrib/yourCompanyName".
Consumers that can be used for general purpose should be placed into the "consumer" package.

To enable a contrib plugin you will need to extend the file "contrib/loader.go".
Add an anonymous import to the list of imports like this:

.. code-block:: go

  import (
    _ "./yourCompanyName"                                 // this is ok for local extensions
    _ "github.com/trivago/gollum/contrib/yourCompanyName" // if you plan to contribute
  )

Requirements
------------

All consumers have to implement the "core/Consumer" as well as the "core/Plugin" interface.
The most convenient way to do this is to derive from the "core/ConsumerBase" type as this will provide common implementations of most methods required.
In addition to this, every plugin has to register at the plugin registry to be available as a config option.

.. code-block:: go

  import (
    "github.com/trivago/gollum/core"
    "github.com/trivago/gollum/shared"
  	"sync"                                    // Required for the Consume() implementation
  )

  type MyConsumer struct {
    core.ConsumerBase                         // Derive from ConsumerBase
  }

  func init() {
    shared.RuntimeType.Register(MyConsumer{}) // Register the new plugin type
  }

ConsumerBase
------------

Consumers deriving from "core/ConsumerBase" have to implement the "Consume" method from the "core/Consumer" interface.
In addition to that most plugins might also want to overload the "Configure" function from the "core/Plugin" interface.

The Consume() function will be called as a separate go routine and should do two things.
1. Listen to the control channel
2. Process incoming data

As Consume() is called as a separate go routine you can decide wether to spawn additional go routines to handle one or both tasks.
ConsumerBase gives you two convenience loop functions to handle:

**DefaultControlLoop**
  Will loop until a stop is recieved and can trigger a callback if a log rotation is requested (SIG_HUP is sent).

**TickerControlLoop**
  Gives you an additional callback that is triggered in regular intervals.

Both loops only cover the handling of control messages and are blocking.
As of their blocking nature you will probably spawn a separate go routine handling your messages when using these loops.
A typical consume function will look like this:

.. code-block:: go

  func (cons *MyConsumer) close() {
    cons.WorkerDone()
  }

  func (cons *MyConsumer) Consume(workers *sync.WaitGroup) {
    cons.AddMainWorker(workers)          // New go routine = new worker
    defer cons.close()                   // Make sure WorkerDone is called by using defer
    go cons.readData()                   // Run until close is called
    cons.DefaultControlLoop(cons.rotate) // Blocks
  }

This function will call the close() function when the default control loop exits, i.e. when a shutdown is requested.
As we want to run a new go routine we also add a new worker. As this is the first worker we use AddMainWorker().
Additional workers can be added by using AddWorker().
This enables the shutdown routine to wait until all consumers have properly stopped.
However - to avoid a hang during shutdown, make sure that all workers added are properly closed during the shutdown sequence.

After we made sure all workers are registered, the core function readData() is called as a separate go routine.
This is necessary as the DefaultControlLoop will block Consume() until a shutdown is requested.
Last but not least the rotate() function passed as a callback to DefaultControlLoop().
This enables the consumer to listen for log rotation requests.
If your consumer does not need this you can pass nil instead.


Configuration
-------------

If your consumer requires additonal configuration options you should implement the Configure method.
If you derived from ConsumerBase It is advisable to call Configure() of ConsumerBase before checking your configuration options.
There are several convenience functions in the PluginConfig type that makes it easy to obtain configuration values and setting default values.
Please refer to Gollum's GoDoc API documentation for more details on this.

.. code-block:: go

  func (cons *MyConsumer) Configure(conf core.PluginConfig) error {
    err := cons.ConsumerBase.Configure(conf)
    if err != nil {
      return err
    }
    // ... read custom options ...
    return nil
  }

Sending messages
----------------

Messages can be sent by using either the Enqueue() or EnqueueCopy() method.
Both function will make sure that the message is sent to all streams and the correct stream ID is set.
The function Enqueue() will reference the data you pass to it, while EnqueueCopy() will copy the data to the new message.
The latter will allow you to e.g. safely recycle internal buffers without changing messages that are not processed by all producers, yet.

Both methods expect a sequence number to be passed.
This sequence number is meant to be a runtime unique ID that may allow future checks on duplicate messages.
The most common sequence number is an incrementing 64-bit integer.

.. code-block:: go

  func (cons *MyConsumer) readData() {
    var data []byte                   // read data into this from somewhere
    cons.Enqueue(data, cons.sequence) // this call may block
    cons.sequence++                   // Or use atomic.AddUint64(cons.sequence, 1) to avoid races
  }

Writing bare bone consumers
---------------------------

Sometimes it might be useful not to derive from ConsumerBase.
An example for this is the Null consumer which is extremely lightweight.
If you decide to go this way please have a look at Gollum's GoDoc API documentation as well as the source of ConsumerBase.
