# Chihaya [![Build Status](https://travis-ci.org/pushrax/chihaya.png?branch=master)](https://travis-ci.org/pushrax/chihaya)

Chihaya is a high-performance [BitTorrent tracker](http://en.wikipedia.org/wiki/BitTorrent_tracker)
written in the Go programming language. It is still heavily under development and should not be used
in production. Some of the planned features include.

- Low resource consumption
- *Fast* request processing
- Maximum compatibility with what exists of the BitTorrent spec
- Correct IPv6 support
- A generic storage interfaces that is easily adapted to use any data store and web application
- Scaling properties that directly correlate with those of the chosen data stores

## Architecture

You are most likely looking to integrate Chihaya with a web application for organizing torrents
and managing a community. Chihaya was designed with this in mind, but also tries to remain
independent. Chihaya connects to two data stores. The first, known as "cache", is used between
Chihaya processes in order to keep up with fast changing data. The second, known as "storage",
is your web application's data store. Changes immediately take place in the cache, which is why
fast data stores are recommended. These changes are also collected and periodically applied to the
storage in order to avoid locking up your web application's data store.


## Installing

Make sure you have your $GOROOT and $GOPATH set up correctly and have your $GOBIN on your $PATH.

```sh
$ go install github.com/pushrax/chihaya
```

## Testing

```sh
$ export TESTCONFIGPATH=$GOPATH/src/chihaya/config/example.json
$ go get github.com/pushrax/chihaya
$ go test -v ./...
```

## Configuring

Configuration is done in a JSON formatted file specified with the `-config`
flag. An example configuration can be seen in the `exampleConfig` variable of
[`config/config_test.go`](https://github.com/pushrax/chihaya/blob/master/config/config_test.go).

## Default drivers

### Cache

Chihaya currently supports drivers for the following caches out of the box:

* [redis](http://redis.io)

### Storage

Chihaya currently supports drivers for the following storages out of the box:

* [batter-postgres](https://github.com/wafflesfm/batter)

## Custom drivers

Please read the documentation and understand these interfaces as there are
assumptions made about thread-safety. After you've implemented a new driver,
all you have to do is remember to add `import _ path/to/your/package` to the
top of `main.go` and the side effects from `init()` will globally register
your driver so that config package will recognize your driver by name.
If you're writing a driver for a popular data store, consider contributing it.

### Cache

The [`cache`] package is heavily inspired by the standard library's
[`database/sql`] package. To write a new cache backend, create a new Go
package that has an implementation of the [`Pool`], [`Tx`], and [`Driver`]
interfaces. Within that package, you must also define an [`init()`] that calls
[`cache.Register`].

[`cache`]: http://godoc.org/github.com/pushrax/chihaya/cache
[`database/sql`]: http://godoc.org/database/sql
[`Pool`]: http://godoc.org/github.com/pushrax/chihaya/cache#Pool
[`Tx`]: http://godoc.org/github.com/pushrax/chihaya/cache#Tx
[`Driver`]: http://godoc.org/github.com/pushrax/chihaya/cache#Driver
[`init()`]: http://golang.org/ref/spec#Program_execution
[`cache.Register`]: http://godoc.org/github.com/pushrax/chihaya/cache#Register

### Storage

TODO

## Contributing

If you're interested in contributing, please contact us in **[#chihaya] on
[freenode IRC]** or post to the GitHub issue tracker. Please don't offer
massive pull requests with no prior communication attempts as it will most
likely lead to confusion and time wasted for everyone. However, small
unannounced fixes are always welcome.

[#chihaya]: http://webchat.freenode.net?channels=chihaya
[freenode IRC]: http://freenode.net

And remember: good gophers always use gofmt!
