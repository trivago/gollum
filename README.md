[![Documentation Status](https://readthedocs.org/projects/gollum/badge/?version=latest)](http://gollum.readthedocs.org/en/latest/)
[![GoDoc](https://godoc.org/github.com/trivago/gollum?status.svg)](https://godoc.org/github.com/trivago/gollum)
[![Gitter](https://badges.gitter.im/trivago/gollum.svg)](https://gitter.im/trivago/gollum?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)
[![Go Report Card](https://goreportcard.com/badge/github.com/trivago/gollum)](https://goreportcard.com/report/github.com/trivago/gollum)
[![Build Status](https://travis-ci.org/trivago/gollum.svg?branch=master)](https://travis-ci.org/trivago/gollum)
[![Coverage Status](https://coveralls.io/repos/github/trivago/gollum/badge.svg?branch=master)](https://coveralls.io/github/trivago/gollum?branch=master)
[![Homebrew](https://img.shields.io/badge/dynamic/json.svg?url=https://formulae.brew.sh/api/formula/gollum.json&query=$.versions.stable&label=homebrew)](https://formulae.brew.sh/formula/gollum)

![Gollum](docs/src/gollum.png)

> [!CAUTION]
> This repository has been archived as of October 2025.  
> If you wish to update or extend this project, please create a fork.

# What is Gollum?

Gollum is an **n:m multiplexer** that gathers messages from different sources and broadcasts them to a set of destinations.

Gollum originally started as a tool to **MUL**-tiplex **LOG**-files (read it backwards to get the name).
It quickly evolved to a one-way router for all kinds of messages, not limited to just logs.
Gollum is written in Go to make it scalable and easy to extend without the need to use a scripting language.

Please not that this project is not related to the [Gollum Wiki](https://github.com/gollum/gollum).
Follow the link above if you came here looking for it.

## Gollum Documentation

How-to-use, installation instructions, getting started guides, and in-depth plugin documentation:

* [read the docs user documentation](http://gollum.readthedocs.io/en/latest/)
* [godoc pages for go developers](https://godoc.org/github.com/trivago/gollum)


## Installation

Gollum is tested and packaged to run on FreeBSD, Debian, Ubuntu, Windows and MacOS. Download Gollum and get started now.

* [Installation Instructions](http://gollum.readthedocs.io/en/latest/src/instructions/installation.html)
* [Releases on github.com](https://github.com/trivago/gollum/releases)


## Get Gollum Support and Help

**gitter**: If you can't find your answer in the documentation or have other questions you can reach us on [gitter](https://gitter.im/trivago/gollum?utm_source=share-link&utm_medium=link&utm_campaign=share-link), too.

**Reporting Issues**: To report an issue with Gollum, please create an Issue here on github: https://github.com/trivago/gollum/issues


## License

This project is released under the terms of the [Apache 2.0 license](http://www.apache.org/licenses/LICENSE-2.0).
