# Chihaya [![Build Status](https://api.travis-ci.org/chihaya/chihaya.svg?branch=master)](https://travis-ci.org/chihaya/chihaya)

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

Chihaya is a meant for every kind of BitTorrent tracker deployment. Chihaya has
been used to replace instances of [opentracker] and also instances of [ocelot].
Chihaya handles torrent announces and scrapes in memory, but using a backend
driver, can also asynchronously provide deltas to maintain a set of persistent
data without throttling a database (this most useful for private tracker
use-cases).

[opentracker]: http://erdgeist.org/arts/software/opentracker
[ocelot]: https://github.com/WhatCD/Ocelot

## Building & Installing

Chihaya requires Go 1.4, [Godep], and a [Go environment] previously setup.

[Godep]: https://github.com/tools/godep
[Go environment]: https://golang.org/doc/code.html

```sh
$ export GOPATH=$PWD/chihaya
$ git clone github.com/chihaya/chihaya chihaya/src/github.com/chihaya/chihaya
$ godep go install chihaya/src/github.com/chihaya/cmd/chihaya
```

## Testing

Chihaya has end-to-end test coverage for announces in addition to unit tests for
isolated components. To run the tests, use:

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
