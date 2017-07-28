Writing filters
===============

Requirements
------------

All filters have to implement the "core/Filter" as well as the "core/Plugin" interface.
In addition to this, every plugin has to register at the plugin registry to be available as a config option.
This is explained in the general :doc:`plugin section </src/instructions/writingPlugins>`.

Attention
---------

Filters are called in a multithreaded context, so you have to make sure that any internal state is secured by either a mutex or by using atomic functions.

Filtering messages
------------------

The Accept method is fairly simple to implement.
If the methods returns true the message is passed. If the method returns false the message is rejected.
You can inspect the message in question from the parameter passed to the accept method.
The following example filter will reject all messages that have no content:

.. code-block:: go

  func (filter *MyFilter) Accepts(msg core.Message) bool {
    return len(msg.Data) > 0
  }
