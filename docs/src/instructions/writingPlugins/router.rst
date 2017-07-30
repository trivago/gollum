Writing routers
===============

When writing a new router it is advisable to have a look at existing routers.
A good starting point is the Random router.

Requirements
------------

All routers have to implement the "core/Router" as well as the "core/Plugin" interface.
The most convenient way to do this is to derive from the "core/RouterBase" type as it will provide implementations of the most common methods required as well as message metrics.
In addition to this, every plugin has to register at the plugin registry to be available as a config option.
This is explained in the general :doc:`plugin section </src/instructions/writingPlugins>`.

RouterBase
------------

Routers deriving from "core/RouterBase" have to implement a custom method that has to be hooked to the "Distribute" callback during Configure().
This allows RouterBase to check and format the message before actually distributing it.
In addition to that a message count metric is updated.
The following example implements a router that sends messages only to the first producer in the list.

.. code-block:: go

  func (router *MyRouter) myDistribute() {
    router.RouterBase.Producers[0].Enqueue(msg)
  }

  func (router *MyRouter) Configure(conf core.PluginConfig) {
    if err := router.RouterBase.Configure(conf); err != nil {
      return err
    }
    router.RouterBase.Distribute = router.myDistribute
    return nil
  }

Sending messages
----------------

Messages are sent directly to a producer by calling the Enqueue method.
This call may block as either the underlying channel is filled up completely or the producer plugin implemented Enqueue as a blocking method.

Routers that derive from RouterBase may also by paused.
In that case messages are not passed to the custom distributor function but to a temporary function.
These messages will be sent to the custom distributor function after the router is resumed.
A Pause() call is normally done from producers that encounter a connection loss or an unavailable resource in general.
