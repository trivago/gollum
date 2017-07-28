Writing producers
=================

When writing a new producer it is advisable to have a look at existing producers.
A good starting point are the Console and File producers.

Requirements
------------

All producers have to implement the "core/Producer" as well as the "core/Plugin" interface.
The most convenient way to do this is to derive from the "core/ProducerBase" type as it will provide implementations of the most common methods required.
In addition to this, every plugin has to register at the plugin registry to be available as a config option.
This is explained in the general :doc:`plugin section </src/instructions/writingPlugins>`.

ProducerBase
------------

Producers deriving from core/ProducerBase have to implement the "Produce" method from the "core/Producer" interface.
In addition to that most plugins might also want to overload the "Configure" function from the "core/Plugin" interface.

The Produce() function will be called as a separate go routine and should provide two things.
1. Listen to the control channel
2. Listen to incoming messages

As Produce() is called as a separate go routine you can decide wether to spawn additional go routines to handle both tasks or to let Produce() handle everything.
ProducerBase gives you three convenience loop functions to handle control commands:

**ControlLoop**
  Will only listen to control messages and trigger the corresponding callbacks that can be registered during Configure.
  Stop control messages will cause this loop to end.

**MessageControlLoop**
  In addition to the functionality of ControlLoop this will also check for incoming messages.
  Messages from the internal message channel are passed to the given message handler.
  The log rotation callback can be set e.g. in the Configure method by using the SetRollBack function.
  Other possible callbacks functions are SetPrepareStopCallback and SetStopCallback.

**TickerMessageControlLoop**
  Gives you an additional callback that is triggered in regular intervals.

It is highly recommended to use at least one of these functions in your plugin implementation.
By doing this you can be sure that changes to message streaming and control handling are automatically used by your plugin after a Gollum update.
A typical produce function will look like this:

.. code-block:: go

  func (prod *MyProducer) close() {
    prod.CloseMessageChannel(prod.processData) // Close the internal channel and flush remaining messages
    prod.WorkerDone()                          // Signal that we're done now
  }

  func (prod *MyProducer) Configure(conf core.PluginConfig) error {
    prod.SetStopCallback(prod.close)  // Call close upon shutdown
    prod.SetRollCallback(prod.onRoll) // Call onRoll when SIG_HUP is sent to the process
  }

  func (prod *MyProducer) processData(msg core.Message) {
    // Do something with the message
  }

  func (prod *MyProducer) Produce(workers *sync.WaitGroup) {
    prod.AddMainWorker(workers)
    prod.MessageControlLoop(prod.processData)
  }

The framework will call the registered StopCallback function when the control loop receives a stop.
As the shutdown procedure needs to wait until all messages from this producers have been sent (to avoid data loss) at least one worker should always be registered.
The shutdown procedure will wait until all producer workers have finished before exiting.
As of this you have to make sure that all AddWorker calls are followed by a WorkerDone() call during shutdown.
If this does not happen the shutdown procedure will block.
If your producer sends messages to other producers you can manually set dependencies between receiving producers and this producer by using StreamRegistry.LinkDependencies.
DropStream dependencies are automatically added during startup.

Configuration
-------------

If your producer requires additonal configuration options you should implement the Configure method.
Please refer to the :doc:`Plugin documentation </src/instructions/writingPlugins>` for further details.

Working with slow services
--------------------------

Messages are passed to the producer one-by-one.
Certain services however might perform better when messages are not sent one-by-one but as a batch of messages.
Gollum gives you several tools to handle these kind of message batches.
A good example for this is the socket producer.
This producer takes advantage of the "core/MessageBatch" type.
This allows storing messages in a double-buffered queue and provides callback based methods to flush the queue asynchronously.
The following code illustrates a best practice approach on how to use the MessageBatch.
You may of course change details if required.

.. code-block:: go

  buffer := NewMessageBatch(8192)                // Hold up to 8192*2 messages (front and backbuffer)

  for {
    // Append the given message
    // - If the buffer is full call the sendBatch method and wait for flush
    // - If the producers is not active or if it is shutting down pass the message to prod.Drop

    buffer.AppendOrFlush(message, prod.sendBatch, prod.IsActiveOrStopping, prod.Drop)

    // ...

    if buffer.ReachedSizeThreshold(2048) {       // Check if at least 2 KB have been written
      buffer.Flush(prod.sendBatch)               // Send all buffered messages via sendBatch
      buffer.WaitForFlush()                      // Wait until done
    }
  }

Filtering messages
------------------

Producers are able to filter messages like streams do, too.
In contrast to streams messages are filtered before they are send to the internal message channel, i.e. before formatting.
As formatting is an implementation detail (and may also not happen) a plugin that needs filtering after formatting has too implement it by itself.

Formatting messages
-------------------

Messages are not automatically formatted when passed to the producer.
If you wish to enable producer based formatting you need to call ProducerBase.Format() at an appropriate point inside your plugin.
All producers deriving from ProducerBase - and that have called ProducerBase.Configure() - may have a formatter set and should thus provide this possibility.

Writing bare bone producers
---------------------------

Sometimes it might be useful not to derive from ProducerBase.
An example for this is the Null producer which is extremely lightweight.
If you decide to go this way please have a look at Gollum's GoDoc API documentation as well as the source of ConsumerBase.
