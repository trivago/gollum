Writing formatters
==================

Requirements
------------

All filters have to implement the "core/Formatter" as well as the "core/Plugin" interface.
In addition to this, every plugin has to register at the plugin registry to be available as a config option.
This is explained in the general :doc:`plugin section </src/instructions/writingPlugins>`.

Attention
---------

Formatters are called in a multithreaded context, so you have to make sure that any internal state is secured by either a mutex or by using atomic functions.

Transforming messages
---------------------

The Format method is fairly simple to implement.
It accepts the message to modify and returns the new content plus the stream the message should be sent to.
The message itself cannot be changed directly.
The following example adds a newline to each message:

.. code-block:: go

  func (format *MyFormatter) Format(msg core.Message) ([]byte, core.MessageStreamID) {
    return append(msg.Data, '\n'), msg.StreamID
  }
