Usage
==================================

Commandline
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
-n, -numcpu         Number of CPUs to use. Set 0 for all CPUs (respects cgroup limits).
-p, -pidfile        Write the process id into a given file.
-m, -metrics        Address to use for metric queries. Disabled by default.
-hc, -healthcheck   Listening address ([IP]:PORT) to use for healthcheck HTTP endpoint. Disabled by default.
-pc, -profilecpu    Write CPU profiler results to a given file.
-pm, -profilemem    Write heap profile results to a given file.
-ps, -profilespeed  Write msg/sec measurements to log.
-pt, -profiletrace 	Write profile trace results to a given file.
-t, -trace          Write message trace results _TRACE_ stream.


Running Gollum
--------------

By default you start Gollum with your config file of your defined pipeline.

Configuration files are written in the YAML format and have to be loaded via command line switch.
Each plugin has a different set of configuration options which are currently described in the plugin itself, i.e. you can find examples in the `github wiki`_.

.. _github wiki: https://github.com/trivago/gollum/wiki

.. code-block:: bash

    # starts a gollum process
    gollum -c path/to/your/config.yaml


Here is a minimal console example to run Gollum:

.. code-block:: bash

    # create a minimal config
    echo \
    {StdIn: {Type: consumer.Console, Streams: console}, StdOut: {Type: producer.Console, Streams: console}} \
    > example_conf.yaml

    # starts a gollum process
    gollum -c example_conf.yaml -ll 3