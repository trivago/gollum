Installation
==================================

Latest Release
---------------

You can download a compressed pre-compiled binary from `github releases`_:

.. _github releases: https://github.com/trivago/gollum/releases

.. code-block:: bash

    # linux bases example
    curl -L https://github.com/trivago/gollum/releases/download/v0.4.5/gollum-0.4.5-Linux_x64.zip -o gollum.zip
    unzip -o gollum.zip
    chmod 0755 gollum

    ./gollum --help


From source
---------------

Installation from source requires the installation of the `go toolchain`_.

.. _go toolchain: http://golang.org/

Gollum need a least go version 1.7 or higher and supports the Go 1.5 vendor experiment that is automatically enabled when using the provided makefile.
With Go 1.7 and later you can also use **go build** directly without additional modifications.
Builds with Go 1.6 or earlier versions are not officially supported and might require additional steps and modifications.

.. code-block:: bash

    # checkout
    mkdir -p $(GOPATH)/src/github.com/trivago
    cd $(GOPATH)/src/github.com/trivago
    git clone git@github.com:trivago/gollum.git
    cd gollum

    # run tests and compile
    make test
    ./gollum --help

You can use the make file coming with gollum to trigger cross platform builds.
Make will produce ready to deploy .zip files with the corresponding platform builds inside the dist folder.


Build
`````````````

Building gollum is as easy as `make` or `go build`.
If you want to do cross platform builds use `make all` or specify one of the following platforms instead of "all":

:current: build for current OS (default)
:freebsd: build for FreeBSD
:linux:   build for Linux x64
:mac:     build for MacOS X
:pi:      build for Linux ARM
:win:     build for Windows
:debug:   build for current OS with debug compiler flags
:clean:   clean all artifacts created by the build process


Docker
---------------

The repository contains a `Dockerfile` which enables you to build and run gollum inside a Docker container.

.. code-block:: bash

    docker build -t trivago/gollum .
    docker run -it --rm trivago/gollum -c config/profile.conf -ps -ll 3


To use your own configuration you could run:

.. code-block:: bash

    docker run -it --rm -v /path/to/config.conf:/etc/gollum/gollum.conf:ro trivago/gollum -c /etc/gollum/gollum.conf
