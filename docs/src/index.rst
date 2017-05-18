.. This is the actual root file for Gollum's documentation.


Welcome to Gollum's documentation!
==================================

.. image:: /src/gollum.png

What is Gollum?
---------------

Gollum originally started as a tool to **MUL**-tiplex **LOG**-files (read it backwards to get the name).
It quickly evolved to a one-way router for all kinds of messages, not limited to just logs.
Gollum is written in Go to make it scaleable and easy to extend without the need to use a scripting language.

Terminology
-----------

The main components of Gollum are consumers, streams and producers. To explain these it helps imagineing to look at Gollum "from the outside".

- A consumer "consumes" message, i.e. it reads data from some external service or e.g. listens to a port.
- A producer "produces" messages, i.e. it writes data to some external service or e.g. to disk.
- A stream defines a path between one or more consumers and one or more producers.
- A single set of data passing over a stream is called a message.

.. image:: /src/flow.png

These main components, consumers, producers and streams are build upon a plugin architecture.
This allows each component to be exchanged and configured individually.
Every plugin has a different sets of options.
Streams for example may define filters that can inspect a message to decide wether to drop the message or to let it pass.
Producers and streams may use formatters to modify a message's content and e.g. convert a plain-text log to JSON.
Filters and Formatters are plugins, too, but can only be configured in context of another plugin like a stream.
As of this they are called "nested plugins". These plugins have access to all configuration options of their "host" plugin.


Configuration
-------------

A Gollum configuration file is written in YAML and may contain any number of plugins.
Multiple plugins of the same type are possible, too.
The Gollum core does not make any assumption over the type of data you are processing.
Plugins however may do that. So it is up to the person configuring Gollum to ensure valid data is passed from consumers to producers.
Formatters can help to achieve this.

Running Gollum
--------------

Gollum goes into an infinte loop once started.
You can shutdown gollum by sending a SIG_INT, i.e. Ctrl+C, SIG_TERM or SIG_KILL.
Gollum has several commandline options that can be accessed by starting Gollum without any paramters:

**-c, --config=""**
   Use a given configuration file.
**-h, --help**
  Print this help message.
**-ll, --loglevel=0**
  Set the loglevel [0-3]. Higher levels produce more messages.
**-m, --metrics=0**
  Port to use for metric queries. Set 0 to disable.
**-n, --numcpu=0**
  Number of CPUs to use. Set 0 for all CPUs.
**-p, --pidfile=""**
  Write the process id into a given file.
**-pc, --profilecpu=""**
  Write CPU profiler results to a given file.
**-pm, --profilemem=""**
  Write heap profile results to a given file.
**-ps, --profilespeed**
  Write msg/sec measurements to log.
**-r, --report**
  Print detailed version report and exit.
**-tc, --testconfig=""**
  Test a given configuration file and exit.
**-tr, --trace**
  Write trace results to a given file.
**-v, --version**
  Print version information and exit.

Table of contents
-----------------

.. toctree::
    :maxdepth: 2

    /src/plugins/index
    /src/examples/index
    /src/license/index
