# Chihaya [![Build Status](https://travis-ci.org/pushrax/chihaya.png?branch=master)](https://travis-ci.org/pushrax/chihaya)

Chihaya is a high-performance [BitTorrent tracker](http://en.wikipedia.org/wiki/BitTorrent_tracker)
written in the Go programming language. It isn't quite ready for prime-time
just yet, but these are the features it targets:

- Low resource consumption
- *Fast* request processing
- A generic storage interface that is easily adapted to use any data store
- Scaling properties that directly correlate with those of the chosen data store
- IPv6 support
- Maximum compatibility with what exists of the BitTorrent spec


## Installing

First, you'll need to install libzmq with your favourite package manager. Then,

```sh
$ go install github.com/pushrax/chihaya
```

## Configuring

Configuration is done in a JSON formatted file specified with the `-config`
flag. An example configuration can be seen in the `exampleConfig` variable of
[`config/config_test.go`](https://github.com/pushrax/chihaya/blob/master/config/config_test.go).

## Default storage drivers

Chihaya currently supports the following drivers out of the box:

* [redis](http://redis.io)

## Custom storage drivers

The [`storage`] package is heavily inspired by the standard library's
[`database/sql`] package. To write a new storage backend, create a new Go
package that has an implementation of the [`DS`], [`Tx`], and [`Driver`]
interfaces. Within that package, you must also define an [`init()`] that calls
[`storage.Register`].

[`storage`]: http://godoc.org/github.com/pushrax/chihaya/storage
[`database/sql`]: http://godoc.org/database/sql
[`DS`]: http://godoc.org/github.com/pushrax/chihaya/storage#DS
[`Tx`]: http://godoc.org/github.com/pushrax/chihaya/storage#Tx
[`Driver`]: http://godoc.org/github.com/pushrax/chihaya/storage#Driver
[`init()`]: http://golang.org/ref/spec#Program_execution
[`storage.Register`]: http://godoc.org/github.com/pushrax/chihaya/storage#Register

Please read the documentation and understand these interfaces as there are
assumptions made about thread-safety. After you've implemented a new driver,
all you have to do is remember to add `import _ path/to/your/library` to the
top of any file (preferably `main.go`) and the side effects from `func init()`
will globally register your driver so that config files will recognize your
driver by name. If you're writing a driver for a popular data store, consider
contributing it.


## Contributing

If you're interested in contributing, please contact us in **#chihaya on
[freenode]** ([webchat]) or post to the issue tracker. Please don't offer
massive pull requests with no prior communication attempts as it will most
likely lead to confusion and time wasted for everyone.  However, small
unannounced fixes are always welcome.

[freenode]: http://freenode.net
[webchat]: http://webchat.freenode.net?channels=chihaya

And remember: good gophers always use gofmt!
