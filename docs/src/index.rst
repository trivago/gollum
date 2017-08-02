.. This is the actual root file for Gollum's documentation.


Welcome to Gollum's documentation!
==================================

.. image:: /src/gollum.png

What is Gollum?
---------------

Gollum is an n:m multiplexer that gathers messages from different sources and broadcasts them to a set of destinations.

Gollum originally started as a tool to **MUL**-tiplex **LOG**-files (read it backwards to get the name).
It quickly evolved to a one-way router for all kinds of messages, not limited to just logs.
Gollum is written in Go to make it scalable and easy to extend without the need to use a scripting language.

Terminology
-----------

The main components of Gollum are consumers, streams and producers. To explain these it helps imagineing to look at Gollum "from the outside".

:Message:   A single set of data passing over a `stream` is called a message.
:Metadata:  A optional part of messages. These can contain key/value pairs with additional information or content.
:Stream:    A stream defines a path between one or more `consumers`, `routers` and `producers`.
:Consumer:  The consumer create messages by "consuming" a specific data source. This can be everything like files, ports, external services and so on.
:Producer:  The producer processed receiving message and "produce" something with it. That can be writing to files or ports, sending to external services and so on.
:Router:    The router get and forward messages from specific source- to target-stream(s).
:Modulator: A modulator can be a `Filter` or `Formatter` which "modulates" a message.
:Formatter: A formatter can modulate the payload of a message like convert a plain-text to JSON.
:Filter:    A filter can inspect a message to decide wether to drop the message or to let it pass.

.. image:: /src/flow_800w.png

These main components, `consumers`, `routers`, `producers`, `filters` and `formatters`
are build upon a plugin architecture.
This allows each component to be exchanged and configured individually with a different sets of options.


Configuration
-------------

A Gollum configuration file is written in YAML and may contain any number of plugins.
Multiple plugins of the same type are possible, too.
The Gollum core does not make any assumption over the type of data you are processing.
Plugins however may do that. So it is up to the person configuring Gollum to ensure valid data is passed from consumers to producers.
Formatters can help to achieve this.




Table of contents
-----------------

.. toctree::
    :maxdepth: 2

    /src/instructions/index
    /src/plugins/index
    /src/examples/index
    /src/releaseNotes/index
    /src/license/index
