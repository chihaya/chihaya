# Chihaya [![Build Status](https://api.travis-ci.org/chihaya/chihaya.svg?branch=master)](https://travis-ci.org/chihaya/chihaya)

Chihaya is a high-performance [BitTorrent tracker](http://en.wikipedia.org/wiki/BitTorrent_tracker) written in the Go programming language. It is still heavily under development and the current `master` branch should probably not be used in production (unless you know what you're doing).

Features include:

- Public tracker feature-set with full compatibility with what exists of the BitTorrent spec
- Private tracker feature-set with compatibility for a [Gazelle]-like deployment (WIP)
- Low resource consumption, and fast, asynchronous request processing
- Full IPv6 support, including handling for dual-stacked peers
- Extensive metrics for visibility into the tracker and swarm's performance
- Ability to prioritize peers in local subnets to reduce backbone contention
- Pluggable backend driver that can coordinate with an external database

[gazelle]: https://github.com/whatcd/gazelle

## When would I use Chihaya?

Chihaya is a meant for every kind of BitTorrent tracker deployment. Chihaya has been used to replace instances of [opentracker] and also instances of [ocelot]. Chihaya handles torrent announces and scrapes in memory, but using a backend driver, can also asynchronously provide deltas to maintain a set of persistent data without throttling a database (this most useful for private tracker use-cases).

[opentracker]: http://erdgeist.org/arts/software/opentracker
[ocelot]: https://github.com/WhatCD/Ocelot

## Building & Installing

Chihaya requires Go 1.3, [Godep], and a [Go environment] previously setup.

[Godep]: https://github.com/tools/godep
[Go environment]: https://golang.org/doc/code.html

```sh
$ export GOPATH=$PWD/chihaya
$ git clone github.com/chihaya/chihaya chihaya/src/github.com/chihaya/chihaya
$ godep go install chihaya/src/github.com/chihaya/cmd/chihaya
```

## Testing

Chihaya has end-to-end test coverage for announces in addition to unit tests for isolated components. To run the tests, use:

```sh
$ cd $GOPATH/src/github.com/chihaya/chihaya
$ godep go test -v ./...
```

There is also a set of benchmarks for performance-critical sections of Chihaya. These can be run similarly:

```sh
$ cd $GOPATH/src/github.com/chihaya/chihaya
$ godep go test -v ./... -bench .
```

## Implementing a Driver

The [`backend`] package is meant to provide announce deltas to a slower and more consistent database, such as the one powering a torrent-indexing website. Implementing a backend driver is heavily inspired by the standard library's [`database/sql`] package: simply create a package that implements the [`backend.Driver`] and [`backend.Conn`] interfaces and calls [`backend.Register`] in it's [`init()`]. Please note that [`backend.Conn`] must be thread-safe. A great place to start is to read the [`no-op`] driver which comes out-of-the-box with Chihaya and is meant to be used for public trackers.

[`init()`]: http://golang.org/ref/spec#Program_execution
[`database/sql`]: http://godoc.org/database/sql
[`backend`]: http://godoc.org/github.com/chihaya/chihaya/backend
[`backend.Register`]: http://godoc.org/github.com/chihaya/chihaya/backend#Register
[`backend.Driver`]: http://godoc.org/github.com/chihaya/chihaya/backend#Driver
[`backend.Conn`]: http://godoc.org/github.com/chihaya/chihaya/backend#Conn
[`no-op`]: http://godoc.org/github.com/chihaya/chihaya/backend/noop

### Creating a binary with your own driver

Chihaya is designed to be extended. If you require more than the drivers provided out-of-the-box, you are free to create your own and then produce your own custom Chihaya binary. To create this binary, simply create your own main package, import your custom drivers, then call [`chihaya.Boot`] from main.

[`chihaya.Boot`]: http://godoc.org/github.com/chihaya/chihaya

#### Example

```go
package main

import (
	"github.com/chihaya/chihaya"

  // Import any of your own drivers.
	_ "github.com/yourusername/chihaya-custom-backend"
)

func main() {
  // Start Chihaya normally.
	chihaya.Boot()
}
```

## Contributing

The project follows idiomatic [Go conventions] for style. If you're interested in contributing, please contact us via IRC in **[#chihaya] on [freenode]** or post to the GitHub issue tracker. Please don't write massive patches with no prior communication, as it will most likely lead to confusion and time wasted for everyone. However, small unannounced fixes are always welcome!

[#chihaya]: http://webchat.freenode.net?channels=chihaya
[freenode]: http://freenode.net
[Go conventions]: https://github.com/jzelinskie/conventions
