# Chihaya [![Build Status](https://travis-ci.org/pushrax/chihaya.png?branch=master)](https://travis-ci.org/pushrax/chihaya)

Chihaya is a high-performance [BitTorrent tracker](http://en.wikipedia.org/wiki/BitTorrent_tracker)
written in the Go programming language. It is still heavily under development and should not be used
in production. Some of the planned features include.

- Low resource consumption
- *Fast* request processing
- Maximum compatibility with what exists of the BitTorrent spec
- Correct IPv6 support
- A generic storage interface that is easily adapted to use any data store
- Scaling properties that directly correlate with those of the chosen data store

## Architecture

You are most likely looking to integrate Chihaya with a web application for organizing torrents
and managing a community. Chihaya was designed with this in mind, but also tries to remain
independent. Chihaya has its own data store that needs to be bootstrapped with data from your
web application. ZeroMQ is used to publish changes to this data. Your web application must
subscribe to this stream, collect these changes, and apply them (usually in a batch fashion).
The only caveat to this architecture is that when a torrent is added or deleted your web
application needs to update both its own data store and Chihaya's.


## Installing

Make sure you have your $GOROOT and $GOPATH set up correctly and have your $GOBIN on your $PATH.
You'll also need to install ZeroMQ with your favourite package manager. Next, you'll need to
"go get" the correct version of the gozmq library that corresponds to your system's version.
For example, these are the steps you'd use to install on Ubuntu 12.04 LTS:

```sh
$ sudo apt-get install libzmq-dev
$ go get -tags zmq_2_1 github.com/alecthomas/gozmq
$ go install github.com/pushrax/chihaya
```

## Configuring

Configuration is done in a JSON formatted file specified with the `-config`
flag. An example configuration can be seen in the `exampleConfig` variable of
[`config/config_test.go`](https://github.com/pushrax/chihaya/blob/master/config/config_test.go).

## Default storage drivers

Chihaya currently supports the following data stores out of the box:

* [redis](http://redis.io)

## Custom storage drivers

The [`storage`] package is heavily inspired by the standard library's
[`database/sql`] package. To write a new storage backend, create a new Go
package that has an implementation of the [`Pool`], [`Tx`], and [`Driver`]
interfaces. Within that package, you must also define an [`init()`] that calls
[`storage.Register`].

[`storage`]: http://godoc.org/github.com/pushrax/chihaya/storage
[`database/sql`]: http://godoc.org/database/sql
[`Pool`]: http://godoc.org/github.com/pushrax/chihaya/storage#Pool
[`Tx`]: http://godoc.org/github.com/pushrax/chihaya/storage#Tx
[`Driver`]: http://godoc.org/github.com/pushrax/chihaya/storage#Driver
[`init()`]: http://golang.org/ref/spec#Program_execution
[`storage.Register`]: http://godoc.org/github.com/pushrax/chihaya/storage#Register

Please read the documentation and understand these interfaces as there are
assumptions made about thread-safety. After you've implemented a new driver,
all you have to do is remember to add `import _ path/to/your/library` to the
top of any file in your project (preferably `main.go`) and the side effects from
`func init()` will globally register your driver so that config package will recognize
your driver by name. If you're writing a driver for a popular data store, consider
contributing it.


## Contributing

If you're interested in contributing, please contact us in **[#chihaya] on
[freenode]** or post to the GitHub issue tracker. Please don't offer
massive pull requests with no prior communication attempts as it will most
likely lead to confusion and time wasted for everyone. However, small
unannounced fixes are always welcome.

[#chihaya]: http://webchat.freenode.net?channels=chihaya
[freenode]: http://freenode.net

And remember: good gophers always use gofmt!
