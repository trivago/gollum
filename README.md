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

* [read the docs - http://gollum.readthedocs.org/en/latest/](http://gollum.readthedocs.org/en/latest/)
* [godoc pages - https://godoc.org/github.com/trivago/gollum](https://godoc.org/github.com/trivago/gollum)


## Installation

Gollum is tested and packaged to run on FreeBSD, Debian, Ubuntu, Windows and MacOS. Download Gollum and get started now.

https://github.com/trivago/gollum/releases

[Installation Instructions](http://gollum.readthedocs.io/en/latest/index.html)


## Get Gollum Support and Help

**gitter Chat**: If you can't find your answer in the documentation or have other questions you can reach us on [gitter](https://gitter.im/trivago/gollum?utm_source=share-link&utm_medium=link&utm_campaign=share-link), too.

**Reporting Issues**: To report an issue with Gollum, please create an Issue here on github: https://github.com/trivago/gollum/issues


## License

This project is released under the terms of the [Apache 2.0 license](http://www.apache.org/licenses/LICENSE-2.0).



# OLD - HAVE TO MOVE


### Build



There are also supplementary targets for make:

* `clean` clean all artifacts created by the build process
* `test` run unittests
* `vendor` install [Glide](https://github.com/Masterminds/glide) and update all dependencies
* `aws` build for Linux x64 and generate an [Elastic Beanstalk](https://aws.amazon.com/de/elasticbeanstalk/) package

If you want to use native plugins (contrib/native) or self provided you will have to enable the corresponding imports in the file `contrib_loader.go`. You can copy the `contrib_loader.go.dist` file here and activate the plugins you want to use.

Doing so will disable the possibility to do cross-platform builds for most users.
Please check also the requirements for each plugin.





## ???
***This is a DEVELOPMENT branch.***
Please read the list of [breaking changes](https://github.com/trivago/gollum/wiki/Breaking050) from 0.4.x to 0.5.0.

Writing a custom plugin does not require you to change any additional code besides your new plugin file.


