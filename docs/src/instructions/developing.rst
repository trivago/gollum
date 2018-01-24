Developing
==================================

Testing
---------------

Gollum provides unit-, integrations- and a couple of linter tests which also runs regulary on travis-ci_.

.. _travis-ci: https://travis-ci.org/trivago/gollum

You can run the test by:

.. code-block:: bash

    # run tests
    make test

    # run unit-test only
    make unit

    # run integration-test only
    make integration

Here an overview of all provided tests by the Makefile:

:make test:        Run `go vet`, `golint`, `gofmt` and `go test`
:make unit:        Run `go test -tags unit`
:make integration: Run `go test -tags integration`
:make vet:         Run `go vet`
:make lint:        Run `golint`
:make fmt-check:   Run `gofmt -l`
:make ineffassign: Install and run `ineffassign`_

.. _ineffassign: https://github.com/gordonklaus/ineffassign

Debugging
---------------

If you want to use Delve_ for debugging you need to build gollum with some additional flags.
You can use the predefined make command `make debug`:

.. code-block:: bash

    # build for current OS with debug compiler flags
    make debug

    # or go build
    # go build -ldflags='-s -linkmode=internal' -gcflags='-N -l'


With this debug build you are able to start a Delve_ remote debugger:

.. code-block:: bash

    # for the gollum arguments pls use this format: ./gollum -- -c my/config.conf
    dlv --listen=:2345 --headless=true --api-version=2 --log exec ./gollum -- -c testing/configs/test_router.conf -ll 3


.. _Delve: https://github.com/derekparker/delve


Profiling
---------------

To test Gollum you can use the internal `profiler consumer`_ and the `benchmark producer`_.

- The `profiler consumer`_ allows you to create automatically messages with random payload.
- The `benchmark producer`_ is able to measure processed messages

By some optional parameters you can get further additional information:

:-ps:     Profile the processed message per second.
:-ll 3:   Set the log level to `debug`
:-m 8080: Activate metrics endpoint on port 8080


.. _profiler consumer: ../gen/consumer/profiler.html
.. _benchmark producer: ../gen/producer/benchmark.html


Here a simple config example how you can setup a `profiler consumer`_ with a `benchmark producer`_.
By default this test profiles the theoretic maximum throughput of 256 Byte messages:

.. code-block:: yaml

    Profiler:
        Type: consumer.Profiler
        Runs: 100000
        Batches: 100
        Message: "%256s"
        Streams: profile
        KeepRunning: true

    Benchmark:
        Type: producer.Benchmark
        Streams: profile


.. code-block:: bash

    # start Gollum for profiling
    gollum -ps -ll 3 -m 8080 -c config/profile.conf

    # get metrics
    nc -d 127.0.0.1 8080 | python -m json.tool


You can enable different producers in that config to test the write performance of these producers, too.


Dependencies
---------------

To handle external go-packages and -libraries Gollum use dep_. Like in other go projects the `vendor`
is also checked in on github.com. All dependencies can be found in the Gopkg.toml_ file.

To update the external dependencies we provide also a make command:

.. code-block:: bash

    # update external dependencies
    make update-vendor

.. _dep: https://github.com/golang/dep
.. _Gopkg.toml: https://github.com/golang/dep/blob/master/docs/Gopkg.toml.md