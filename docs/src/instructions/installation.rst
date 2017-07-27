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
    git clone git@github.com:trivago/gollum.git
    cd gollum

    # compile
    make test
    ./gollum --help

You can use the make file coming with gollum to trigger cross platform builds.
Make will produce ready to deploy .zip files with the corresponding platform builds inside the dist folder.