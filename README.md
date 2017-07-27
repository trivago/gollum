# Gollum - n:m multiplexer

[![GoDoc](https://godoc.org/github.com/trivago/gollum?status.svg)](https://godoc.org/github.com/trivago/gollum)
[![Documentation Status](https://readthedocs.org/projects/gollum/badge/?version=latest)](http://gollum.readthedocs.org/en/latest/)
[![Go Report Card](https://goreportcard.com/badge/github.com/trivago/gollum)](https://goreportcard.com/report/github.com/trivago/gollum)
[![Build Status](https://travis-ci.org/trivago/gollum.svg?branch=v0.4.3dev)](https://travis-ci.org/trivago/gollum)
[![Coverage Status](https://coveralls.io/repos/github/trivago/gollum/badge.svg?branch=master)](https://coveralls.io/github/trivago/gollum?branch=master)
[![Gitter](https://badges.gitter.im/trivago/gollum.svg)](https://gitter.im/trivago/gollum?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)
[![Apache V2 License](http://img.shields.io/badge/license-Apache%20V2-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0)

![Gollum](docs/src/gollum.png)

Gollum is an n:m multiplexer that gathers messages from different sources and broadcasts them to a set of destinations.

## Documentation

A how-to-use documentation can be found on [read the docs](http://gollum.readthedocs.org/en/latest/).

There is also a go documentation available under [godoc pages](https://godoc.org/github.com/trivago/gollum), which is more relevant for developers.

The documentation is generated from the plugin source code. So if you feel that something is missing a look into the code may help and feel free to contribute.

If you can't find your answer in the documentation or have other questions you can reach us on [gitter](https://gitter.im/trivago/gollum?utm_source=share-link&utm_medium=link&utm_campaign=share-link), too.


## Installation

### Latest Release

You can download the a compressed pre-compiled binary from [github releases](https://github.com/trivago/gollum/releases)

```bash
# linux bases example
curl -L https://github.com/trivago/gollum/releases/download/v0.4.5/gollum-0.4.5-Linux_x64.zip -o gollum.zip
unzip -o gollum.zip
chmod 0755 gollum

./gollum --help
```

### From source

Installation from source requires the installation of the [Go toolchain](http://golang.org/).

Gollum need a least go version 1.7 or higher and supports the Go 1.5 vendor experiment that is automatically enabled when using the provided makefile.
With Go 1.7 and later you can also use `go build` directly without additional modifications.
Builds with Go 1.6 or earlier versions are not officially supported and might require additional steps and modifications.

```bash
# checout
git clone git@github.com:trivago/gollum.git
cd gollum

# compile
make
./gollum --help
```

You can use the make file coming with gollum to trigger cross platform builds.
Make will produce ready to deploy .zip files with the corresponding platform builds inside the dist folder.


## Usage

By default you start gollum with your config file of your defined pipeline.

Configuration files are written in the YAML format and have to be loaded via command line switch.
Each plugin has a different set of configuration options which are currently described in the plugin itself, i.e. you can find examples in the [Wiki](https://github.com/trivago/gollum/wiki).

```sh
# starts a gollum process
gollum -c path/to/your/config.yaml
```

Here is a minimal console example to run gollum:

```sh
echo \
{StdIn: {Type: consumer.Console, Streams: console}, StdOut: {Type: producer.Console, Streams: console}} \
> example_conf.yaml

gollum -c example_conf.yaml -ll 3
```

### Commandline

```sh
./gollum -h

Usage: gollum [OPTIONS]

Options:
-h, -help         	default: false	Print this help message.
-v, -version      	default: false	Print version information and quit.
-r, -report       	default: false	Print detailed version report and quit.
-l, -list         	default: false	Print plugin information and quit.
-c, -config       	default:      	Use a given configuration file.
-tc, -testconfig  	default:      	Test the given configuration file and exit.
-ll, -loglevel    	default: 1    	Set the loglevel [0-3] as in {0=Errors, 1=+Warnings, 2=+Notes, 3=+Debug}.
-n, -numcpu       	default: 0    	Number of CPUs to use. Set 0 for all CPUs.
-p, -pidfile      	default:      	Write the process id into a given file.
-m, -metrics      	default: 0    	Address to use for metric queries. Disabled by default.
-pc, -profilecpu  	default:      	Write CPU profiler results to a given file.
-pm, -profilemem  	default:      	Write heap profile results to a given file.
-ps, -profilespeed	default: false	Write msg/sec measurements to log.
-tr, -trace       	default:      	Write trace results to a given file.
```

## Building

### Mac OS X

The easiest way to install go is by using homebrew:

```bash
brew install go
```

### Linux

Download Go from the [golang website](https://golang.org/dl/) and unzip it to e.g. /usr/local/go.
You have to set the GOROOT environment variable to the folder you chose:
`export GOROOT=/usr/local/go`

### Prerequisites

If you do not already have a GOPATH set up you need to create one.
The location is free of choice, we prefer to put it into each users home folder:

```bash
mkdir -p ~/go
export GOPATH=$(HOME)/go
```

You can download gollum via `go get github.com/trivago/gollum` or clone it directly into your GOPATH.
If you choose this way you need to download your dependencies directly from that folder

```bash
mkdir -p $(GOPATH)/src/github.com/trivago
cd $(GOPATH)/src/github.com/trivago
git clone https://github.com/trivago/gollum.git
cd gollum
```

### Build

Building gollum is as easy as `make` or `go build`.
When using Go 1.5 make sure to enable the go vendor experiment by setting `export GO15VENDOREXPERIMENT=1` or use `make`.
If you want to do cross platform builds use `make all` or specify one of the following platforms instead of "all":

* `current` build for current OS (default)
* `freebsd` build for FreeBSD
* `linux` build for Linux x64
* `mac` build for MacOS X
* `pi` build for Linux ARM
* `win` build for Windows

There are also supplementary targets for make:

* `clean` clean all artifacts created by the build process
* `test` run unittests
* `vendor` install [Glide](https://github.com/Masterminds/glide) and update all dependencies
* `aws` build for Linux x64 and generate an [Elastic Beanstalk](https://aws.amazon.com/de/elasticbeanstalk/) package

If you want to use native plugins (contrib/native) or self provided you will have to enable the corresponding imports in the file `contrib_loader.go`. You can copy the `contrib_loader.go.dist` file here and activate the plugins you want to use.

Doing so will disable the possibility to do cross-platform builds for most users.
Please check also the requirements for each plugin.

### Dockerfile

The repository contains a `Dockerfile` which enables you to build and run gollum inside a Docker container.

```bash
docker build -t trivago/gollum .
docker run -it --rm trivago/gollum -c config/profile.conf -ps -ll 3
```

To use your own configuration you could run:

```bash
docker run -it --rm -v /path/to/config.conf:/etc/gollum/gollum.conf:ro trivago/gollum -c /etc/gollum/gollum.conf
```

## Best practice

### Managing own plugins in a seperate git repository

You can add a own plugin module by simple using `git submodule`:

```bash
git submodule add -f https://github.com/YOUR_NAMESPACE/YOUR_REPO.git contrib/namespace
```

The by git created `.gitmodules` will be ignored by the gollum repository.

To activate your plugin you need to create a `contrib_loader.go` to be able to compile gollum with your own provided plugins.
You can copy the existing `contrib_loader.go.dist` to `contrib_loader.go` and update the import path to your package:

```bash
cp contrib_loader.go.dist contrib_loader.go
# open contrib_loader.go with an editor
# update package path
make current
```

## Debugging

If you want to use [Delve](https://github.com/derekparker/delve) for debugging you need to build gollum with some additional flags:

```bash
go build -ldflags='-s -linkmode=internal' -gcflags='-N -l'
```

With this debug build you are able to start a [Delve](https://github.com/derekparker/delve) remote debugger:

```bash
# for the gollum arguments pls use this format: ./gollum -- -c my/config.conf
dlv --listen=:2345 --headless=true --api-version=2 --log exec ./gollum -- -c testing/configs/test_router.conf -ll 3
```

## License

This project is released under the terms of the [Apache 2.0 license](http://www.apache.org/licenses/LICENSE-2.0).

## TODO
***This is a DEVELOPMENT branch.***
Please read the list of [breaking changes](https://github.com/trivago/gollum/wiki/Breaking050) from 0.4.x to 0.5.0.

Writing a custom plugin does not require you to change any additional code besides your new plugin file.


To test gollum you can make a local profiler run with a predefined configuration:

```bash
gollum -c config/profile.conf -ps -ll 3
```

By default this test profiles the theoretic maximum throughput of 256 Byte messages.
You can enable different producers in that config to test the write performance of these producers, too.