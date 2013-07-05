#chihaya [![Build Status](https://travis-ci.org/pushrax/chihaya.png?branch=master)](https://travis-ci.org/pushrax/chihaya)

chihaya is a high-performance [BitTorrent tracker](http://en.wikipedia.org/wiki/BitTorrent_tracker) written in the Go programming language. It isn't quite ready for prime-time just yet, but these are the features that it targets:

- Requests are multiplexed over all available threads (1 goroutine per request)
- Low processing and memory footprint
- IPv6 support
- Generic storage interface that can be easily adapted to use any data store
- Scaling properties that directly correlate with the chosen data store's scaling properties


##installing

```sh
$ go install github.com/pushrax/chihaya
```

##configuring

Configuration is done in a JSON formatted file specified with the `-config` flag. An example configuration can be seen in the `exampleConfig` variable of [`config/config_test.go`](https://github.com/pushrax/chihaya/blob/master/config/config_test.go).

##out of the box drivers

Chihaya currently supports the following drivers out of the box:

* [redis](http://redis.io)

##implementing custom storage

The [`storage`](http://godoc.org/github.com/pushrax/chihaya/storage) package is heavily inspired by the standard library's [`database/sql`](http://godoc.org/database/sql) package. To write a new storage backend, create a new Go package that has an implementation of the [`DS`](http://godoc.org/github.com/pushrax/chihaya/storage#DS), [`Tx`](http://godoc.org/github.com/pushrax/chihaya/storage#Tx), and [`Driver`](http://godoc.org/github.com/pushrax/chihaya/storage#Driver) interfaces. Within that package, you must also define an [`func init()`](http://golang.org/ref/spec#Program_execution) that calls [`storage.Register("driverName", &myDriver{})`](http://godoc.org/github.com/pushrax/chihaya/storage#Register). Please read the documentation and understand these interfaces as there are assumptions about thread-safety. After you've implemented a new driver, all you have to do is remember to add `import _ path/to/your/library` to the top of any file (preferably `main.go`) and the side effects from `func init()` will globally register your driver so that config files will recognize your driver by name. If you're writing a driver for a popular data store, consider contributing it.


##contributing

If you're interested in contributing, please contact us in **#chihaya on [freenode](http://freenode.net/)**([webchat](http://webchat.freenode.net?channels=chihaya)) or post to the issue tracker. Please don't offer massive pull requests with no prior communication attempts as it will most likely lead to confusion and time wasted for everyone. However, small unannounced fixes are welcome.

And remember: good gophers always use gofmt!
