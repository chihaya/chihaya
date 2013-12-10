# Chihaya [![Build Status](https://travis-ci.org/chihaya/chihaya.png?branch=master)](https://travis-ci.org/chihaya/chihaya)

Chihaya is a high-performance [BitTorrent tracker](http://en.wikipedia.org/wiki/BitTorrent_tracker)
written in the Go programming language. It is still heavily under development and the current `master` branch
should not be used in production.

Planned features include:

- Light resource consumption
- Fast request processing, sparing the network from exorbitant connection counts
- Maximum compatibility with what exists of the BitTorrent spec
- Correct IPv6 support
- Generic storage interfaces that are easily adapted to work with any existing web application
- Scaling properties that directly correlate with those of the chosen data stores

### Technical Details

See [the wiki](https://github.com/chihaya/chihaya/wiki) for a discussion of the design behind Chihaya.

## Using Chihaya

Chihaya is intended to work with existing torrent indexing web frameworks, such as [Batter] and [Gazelle].
Following the Unix way, it is built to perform a specific task, and interface with any system that
needs its functionality. See [below](#drivers) for more info.

[batter]: https://github.com/wafflesfm/batter
[gazelle]: https://github.com/whatcd/gazelle

### Installing

Make sure you have your `$GOROOT` and `$GOPATH` set up correctly, and have your `$GOBIN` in your `$PATH`.

```sh
$ go get github.com/chihaya/chihaya
```

### Configuring

Configuration is done in a JSON formatted file specified with the `-config`
flag. An example configuration file can be found
[here](https://github.com/chihaya/chihaya/blob/master/example.json).

### Running the tests

```sh
$ cd $GOPATH/src/github.com/chihaya/chihaya
$ go test -v ./...
```

## Drivers

Chihaya is designed to remain agnostic about the choice of data store for an
application, and it is straightforward to [implement a new driver]. However, there
are a number of drivers that will be directly supported:

Tracker:

* mock (memory)
* [redis](https://github.com/chihaya/chihaya-redis)

Backend:

* mock (memory)
* [gazelle (mysql)](https://github.com/chihaya/chihaya-gazelle)

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
