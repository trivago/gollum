Writing streams
===============

When writing a new stream it is advisable to have a look at existing streams.
A good starting point is the Random stream.

Requirements
------------

All streams have to implement the "core/Stream" as well as the "core/Plugin" interface.
The most convenient way to do this is to derive from the "core/StreamBase" type as it will provide implementations of the most common methods required as well as message metrics.
In addition to this, every plugin has to register at the plugin registry to be available as a config option.
This is explained in the general :doc:`plugin section </examples/plugins>`.

StreamBase
------------

Streams deriving from "core/StreamBase" have to implement a custom method that has to be hooked to the "Distribute" callback during Configure().
This allows StreamBase to check and format the message before actually distributing it.
In addition to that a message count metric is updated.
The following example implements a stream that sends messages only to the first producer in the list.

.. code-block:: go

  func (stream *MyStream) myDistribute() {
    stream.StreamBase.Producers[0].Enqueue(msg)
  }

  func (stream *MyStream) Configure(conf core.PluginConfig) {
    if err := stream.StreamBase.Configure(conf); err != nil {
      return err
    }
    stream.StreamBase.Distribute = stream.myDistribute
    return nil
  }

Sending messages
----------------

Messages are sent directly to a producer by calling the Enqueue method.
This call may block as either the underlying channel is filled up completely or the producer plugin implemented Enqueue as a blocking method.

Streams that derive from StreamBase may also by paused.
In that case messages are not passed to the custom distributor function but to a temporary function.
These messages will be sent to the custom distributor function after the stream is resumed.
A Pause() call is normally done from producers that encounter a connection loss or an unavailable resource in general.
