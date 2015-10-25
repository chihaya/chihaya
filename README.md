# Chihaya

[![GoDoc](https://godoc.org/github.com/chihaya/chihaya?status.svg)](https://godoc.org/github.com/chihaya/chihaya)
[![License](https://img.shields.io/badge/license-BSD-blue.svg)](https://en.wikipedia.org/wiki/BSD_licenses#2-clause_license_.28.22Simplified_BSD_License.22_or_.22FreeBSD_License.22.29)
[![Build Status](https://api.travis-ci.org/chihaya/chihaya.svg?branch=master)](https://travis-ci.org/chihaya/chihaya)
[![Docker Repository on Quay.io](https://quay.io/repository/jzelinskie/chihaya/status "Docker Repository on Quay.io")](https://quay.io/repository/jzelinskie/chihaya)

Chihaya is a high-performance [BitTorrent tracker] written in the Go
programming language. It is still heavily under development and the current
`master` branch should probably not be used in production
(unless you know what you're doing).

Features include:

- Public tracker feature-set with full compatibility with what exists of the BitTorrent spec
- Private tracker feature-set with compatibility for a [Gazelle]-like deployment (WIP)
- Low resource consumption, and fast, asynchronous request processing
- Full IPv6 support, including handling for dual-stacked peers
- Extensive metrics for visibility into the tracker and swarm's performance
- Ability to prioritize peers in local subnets to reduce backbone contention
- Pluggable backend driver that can coordinate with an external database

[BitTorrent tracker]: http://en.wikipedia.org/wiki/BitTorrent_tracker
[gazelle]: https://github.com/whatcd/gazelle

## When would I use Chihaya?

Chihaya is a eventually meant for every kind of BitTorrent tracker deployment.
Chihaya has been used to replace instances of [opentracker] and also instances of [ocelot].
Chihaya handles torrent announces and scrapes in memory.
However, using a backend driver, Chihaya can also asynchronously provide deltas to maintain a set of persistent data without throttling a database.
This is particularly useful behavior for private tracker use-cases.

[opentracker]: http://erdgeist.org/arts/software/opentracker
[ocelot]: https://github.com/WhatCD/Ocelot

## Running Chihaya

### Configuration

Copy [`example_config.json`] to your choice of location, and update the values as required.
An explanation of the available keys can be found in [CONFIGURATION.md].

[`example_config.json`]: https://github.com/chihaya/chihaya/blob/master/example_config.json
[CONFIGURATION.md]: https://github.com/chihaya/chihaya/blob/master/CONFIGURATION.md

### Docker

```sh
$ docker pull quay.io/jzelinskie/chihaya:latest
$ docker run -p 6881:6881 -v $DIR_WITH_CONFIG:/config:ro quay.io/jzelinskie/chihaya:latest
```

## Developing Chihaya

### Building & Installing

Chihaya requires Go 1.5+ (preferrably the latest stable Go), [Godep], and a [Go environment] previously set up.

[Godep]: https://github.com/tools/godep
[Go environment]: https://golang.org/doc/code.html

```sh
$ export GOPATH=$PWD/chihaya
$ git clone https://github.com/chihaya/chihaya.git chihaya/src/github.com/chihaya/chihaya
$ cd chihaya/src/github.com/chihaya/chihaya/cmd/chihaya/
$ godep restore
$ go install github.com/chihaya/chihaya/cmd/chihaya
```

### Testing

Chihaya has end-to-end test coverage for announces in addition to unit tests for isolated components.
To run the tests, use:

```sh
$ cd $GOPATH/src/github.com/chihaya/chihaya
$ godep go test -v ./...
```

There is also a set of benchmarks for performance-critical sections of Chihaya.
These can be run similarly:

```sh
$ cd $GOPATH/src/github.com/chihaya/chihaya
$ godep go test -v ./... -bench .
```

### Contributing

See [CONTRIBUTING.md] for guidelines to contributing to the project.
Feel free to make issues or ask questions.
Our maintainers are also always idle in #chihaya on freenode.

[CONTRIBUTING.md]: https://github.com/chihaya/chihaya/blob/master/CONTRIBUTING.md
