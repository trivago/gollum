[![GoDoc](https://godoc.org/github.com/trivago/gollum?status.svg)](https://godoc.org/github.com/trivago/gollum)
[![Documentation Status](https://readthedocs.org/projects/gollum/badge/?version=latest)](http://gollum.readthedocs.org/en/latest/)
[![Go Report Card](https://goreportcard.com/badge/github.com/trivago/gollum)](https://goreportcard.com/report/github.com/trivago/gollum)
[![Build Status](https://travis-ci.org/trivago/gollum.svg?branch=v0.4.3dev)](https://travis-ci.org/trivago/gollum)
[![Coverage Status](https://coveralls.io/repos/github/trivago/gollum/badge.svg?branch=master)](https://coveralls.io/github/trivago/gollum?branch=master)
[![Gitter](https://badges.gitter.im/trivago/gollum.svg)](https://gitter.im/trivago/gollum?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)
[![Apache V2 License](http://img.shields.io/badge/license-Apache%20V2-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0)

![Gollum](docs/src/gollum.png)

# What is Gollum?

Gollum is an n:m multiplexer that gathers messages from different sources and broadcasts them to a set of destinations.

## Gollum Documentation

How-to-use, installation instructions, getting started guides, and in-depth plugin documentation.

* [read the docs](http://gollum.readthedocs.org/en/latest/)
* [godoc pages](https://godoc.org/github.com/trivago/gollum)


## Installation

Gollum is tested and packaged to run on FreeBSD, Debian, Ubuntu, Windows and MacOS. Download Gollum and get started now.

https://github.com/trivago/gollum/releases

[Installation Instructions](http://gollum.readthedocs.io/en/latest/index.html)


## Get Gollum Support and Help

*gitter chat*: If you can't find your answer in the documentation or have other questions you can reach us on [gitter](https://gitter.im/trivago/gollum?utm_source=share-link&utm_medium=link&utm_campaign=share-link), too.

*Reporting Issues*: To report an issue with Gollum, please create an Issue here on github: https://github.com/trivago/gollum/issues




# OLD - HAVE TO MOVE


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