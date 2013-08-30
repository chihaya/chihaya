# Chihaya [![Build Status](https://travis-ci.org/pushrax/chihaya.png?branch=master)](https://travis-ci.org/pushrax/chihaya)

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

See [the wiki](https://github.com/pushrax/chihaya/wiki) for a discussion of the design behind Chihaya.

## Using Chihaya
### Installing

Make sure you have your `$GOROOT` and `$GOPATH` set up correctly, and have your `$GOBIN` in your `$PATH`.

```sh
$ go get github.com/pushrax/chihaya
```

### Configuring

Configuration is done in a JSON formatted file specified with the `-config`
flag. An example configuration file can be found
[here](https://github.com/pushrax/chihaya/blob/master/config/example.json).

### Running the tests

```sh
$ export TESTCONFIGPATH=$GOPATH/src/chihaya/config/example.json
$ go get github.com/pushrax/chihaya
$ go test -v ./...
```

## Default drivers

Chihaya is designed to remain agnostic about the choice of data store for an
application, and it is straightforward to [implement a new driver]. However, there
are a number of directly supported drivers:

Cache:

* [redis](http://redis.io) — allows for multiple tracker instances to run at the same time for the same swarm
* memory — only a single instance can run, but it requires no extra setup

Storage:

* [batter-postgres](https://github.com/wafflesfm/batter)
* [gazelle-mysql](https://github.com/whatcd/gazelle)

[implement a new driver]: https://github.com/pushrax/chihaya/wiki/Implementing-a-driver


## Contributing

If you're interested in contributing, please contact us via IRC in **[#chihaya] on
[freenode]** or post to the GitHub issue tracker. Please don't write
massive patches with no prior communication, as it will most
likely lead to confusion and time wasted for everyone. However, small
unannounced fixes are always welcome!

[#chihaya]: http://webchat.freenode.net?channels=chihaya
[freenode]: http://freenode.net

And remember: good gophers always use gofmt!
