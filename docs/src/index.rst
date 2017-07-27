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

:Message: A single set of data passing over a `stream` is called a message.
:Stream: A stream defines a path between one or more `consumers`, `routers` and `producers`.
:Consumer: The consumer create messages by "consuming" a specific data source. This can be everything like files, ports, external services and so on.
:Producer: The producer processed receiving message and "produce" something with it. That can be writing to files or ports, sending to external services and so on.
:Router:  The router get and forward messages from specific source- to target-stream(s).
:Modulator: A modulator can be a `Filter` or `Formatter` which "modulate" a message.
:Formatter: A formatter can modulate the payload of a message like convert a plain-text to JSON.
:Filter: A filter can inspect a message to decide wether to drop the message or to let it pass.

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

Running Gollum
--------------

Gollum goes into an infinte loop once started.
You can shutdown gollum by sending a SIG_INT, i.e. Ctrl+C, SIG_TERM or SIG_KILL.

Gollum has several commandline options that can be accessed by starting Gollum without any paramters:

-h, -help           Print this help message.
-v, -version        Print version information and quit.
-r, -runtime        Print runtime information and quit.
-l, -list           Print plugin information and quit.
-c, -config         Use a given configuration file.
-tc, -testconfig    Test the given configuration file and exit.
-ll, -loglevel      Set the loglevel [0-3] as in {0=Error, 1=+Warning, 2=+Info, 3=+Debug}.
-lc, -log-colors    Use Logrus's "colored" log format. One of "never", "auto" (default), "always"
-n, -numcpu         Number of CPUs to use. Set 0 for all CPUs.
-p, -pidfile        Write the process id into a given file.
-m, -metrics        Address to use for metric queries. Disabled by default.
-hc, -healthcheck   Listening address ([IP]:PORT) to use for healthcheck HTTP endpoint. Disabled by default.
-pc, -profilecpu    Write CPU profiler results to a given file.
-pm, -profilemem    Write heap profile results to a given file.
-ps, -profilespeed  Write msg/sec measurements to log.
-tr, -trace       	Write trace results to a given file.


Table of contents
-----------------

.. toctree::
    :maxdepth: 2

    /src/plugins/index
    /src/examples/index
    /src/license/index
