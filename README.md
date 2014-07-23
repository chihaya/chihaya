# Chihaya [![Build Status](https://api.travis-ci.org/chihaya/chihaya.svg?branch=master)](https://travis-ci.org/chihaya/chihaya)

Chihaya is a high-performance [BitTorrent tracker](http://en.wikipedia.org/wiki/BitTorrent_tracker) written in the Go programming language. It is still heavily under development and the current `master` branch should probably not be used in production.

Features include:

- Low resource consumption, and fast, asynchronous request processing
- Full IPv6 support, including handling for dual-stacked peers
- Generic storage interfaces that are easily adapted to work with any database
- Full compatibility with what exists of the BitTorrent spec
- Extensive metrics for visibility into the tracker and swarm's performance
- Ability to group peers in local subnets to reduce backbone contention

## Using Chihaya

Chihaya can be ran as a public or private tracker and is intended to work with existing torrent-indexing web frameworks, such as [Gazelle], [Batter] and any others that spring up. Following the Unix way, it is built to perform one specific task: handling announces and scrapes. By cleanly separating the concerns between tracker and database, we can provide an interface that can be used by system that needs its functionality. See [below](#drivers) for more info.

[batter]: https://github.com/wafflesfm/batter
[gazelle]: https://github.com/whatcd/gazelle

### Installing

Chihaya requires Go 1.3+ to build. To install the Chihaya server, run:

```sh
$ go get github.com/chihaya/chihaya/cmd/chihaya
```

Make sure you have your `$GOPATH` set up correctly, and have `$GOPATH/bin` in your `$PATH`.
If you're new to Go, an overview of the directory structure can be found [here](http://golang.org/doc/code.html).

### Configuring

Configuration is done in a JSON formatted file specified with the `-config`
flag. An example configuration file can be found
[here](https://github.com/chihaya/chihaya/blob/master/example.json).

### Running the tests

```sh
$ cd $GOPATH/src/github.com/chihaya/chihaya
$ go test -v ./...
```

### Technical Details

See [the wiki](https://github.com/chihaya/chihaya/wiki) for a discussion of the design behind Chihaya.

## Drivers

Chihaya is designed to remain agnostic about the choice of data store for an
application, and it is straightforward to [implement a new driver]. However, there
are a number of drivers that will be directly supported "out of the box":

Tracker:

* memory
* [redis](https://github.com/chihaya/chihaya-redis)

Backend:

* noop (for public trackers)
* [gazelle (mysql)](https://github.com/chihaya/chihaya-gazelle)

To use an external driver, make your own package and call it something like `github.com/yourusername/chihaya`. Then, import Chihaya like so:

```go
package main

import (
	"github.com/chihaya/chihaya"

	_ "github.com/yourusername/chihaya-custom-backend" // Import any of your own drivers.
)

func main() {
	chihaya.Boot() // Start Chihaya normally.
}
```

Then, when you do `go install github.com/yourusername/chihaya`, your own drivers will be included in the binary.

[implement a new driver]: https://github.com/chihaya/chihaya/wiki/Implementing-a-driver

## Contributing

If you're interested in contributing, please contact us via IRC in **[#chihaya] on
[freenode]** or post to the GitHub issue tracker. Please don't write
massive patches with no prior communication, as it will most
likely lead to confusion and time wasted for everyone. However, small
unannounced fixes are always welcome!

[#chihaya]: http://webchat.freenode.net?channels=chihaya
[freenode]: http://freenode.net

And remember: good gophers always use gofmt!
