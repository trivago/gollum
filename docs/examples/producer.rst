Writing producers
=================

When writing a new producer it is advisable to have a look at existing producers.
A good starting point are the Console and File producers.

Requirements
------------

All producers have to implement the "core/Producer" as well as the "core/Plugin" interface.
The most convenient way to do this is to derive from the "core/ProducerBase" type as it will provide implementations of the most common methods required.
In addition to this, every plugin has to register at the plugin registry to be available as a config option.
This is explained in the general :doc:`plugin section </examples/plugins>`.

ProducerBase
------------

Producers deriving from core/ProducerBase have to implement the "Produce" method from the "core/Producer" interface.
In addition to that most plugins might also want to overload the "Configure" function from the "core/Plugin" interface.

The Produce() function will be called as a separate go routine and should do two things.
1. Listen to the control channel
2. Listen to incoming messages

As Produce() is called as a separate go routine you can decide wether to spawn additional go routines to handle both tasks or to let Produce() handle everything.
ProducerBase gives you two convenience loop functions to handle control commands:

**DefaultControlLoop**
  Will loop until a stop is recieved and can trigger a callback if a log rotation is requested (SIG_HUP is sent).
  Messages from the internal message channel are passed to the given message handler.

**TickerControlLoop**
  Gives you an additional callback that is triggered in regular intervals.

In contrast to the ConsumerBase loop methods these two method also handle the message loop.
As of this you don't need to spawn additional go routines.
In addition to that these two functions will automatically close and flush the message channel before they return.
A typical produce function will look like this:

.. code-block:: go

  func (prod *MyProducer) Close() {
    // Flush any internal structures and/or wait for messages to be written
    prod.WorkerDone()
  }

  func (prod *MyProducer) processData(msg core.Message) {
    // Process message
  }

  func (prod *MyProducer) Produce(workers *sync.WaitGroup) {
    prod.AddMainWorker(workers)
    prod.DefaultControlLoop(prod.processData, prod.rotate)
  }

The framework will call the Close() function when the default control loop exits, i.e. after a shutdown signal was sent.
As the shutdown procedure needs to wait until all messages from this producers have been sent (to avoid data loss) at least one worker should always be registered.
The shutdown procedure will wait until all producer workers have finished before exiting.
As of this you have to make sure that all AddWorker calls are followed by a WorkerDone() call during shutdown.
If this does not happen the shutdown procedure will block.
The Close() function is always called while the shutdown signal may not arrive due to a blocking message queue.
As of this it may happen that Close() is already running while the shutdown signal is processed by the producer.

A rotate() function passed as a callback to both loop functions.
This enables the producer to listen for log rotation requests.
If your producer does not need this you can pass nil instead.

Configuration
-------------

If your producer requires additonal configuration options you should implement the Configure method.
Please refer to the :doc:`Plugin documentation </examples/plugins>` for further details.

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
    if !buffer.Append(message)                   // Get a message from the channel and add it
      buffer.Flush(yourSendMethod)               // We failed to append. Maybe the queue is full?
      if !buffer.AppendOrBlock(message) {        // Try to append it and block while not possible
        prod.Drop(Message)                       // The buffer has been closed, drop the message
      }
    }
    // ...

    if buffer.ReachedSizeThreshold(2048) {       // Check if at least 2 KB have been written
      buffer.Flush(yourSendMethod)               // See API doc for success and error callbacks
      buffer.WaitForFlush()                      // Wait until done
    }
  }

Formatting messages
-------------------

Messages are not automatically formatted when passed to the producer.
If you wish to enable producer based formatting you need to call ProducerBase.Format() at an appropiate point inside your code.
All producers deriving from ProducerBase - and that have called ProducerBase.Configure() - may have a formatter set and should thus provide this possibility.

Writing bare bone producers
---------------------------

Sometimes it might be useful not to derive from ProducerBase.
An example for this is the Null producer which is extremely lightweight.
If you decide to go this way please have a look at Gollum's GoDoc API documentation as well as the source of ConsumerBase.
