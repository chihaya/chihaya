#chihaya [![Build Status](https://travis-ci.org/pushrax/chihaya.png?branch=master)](https://travis-ci.org/pushrax/chihaya)

chihaya is a high-performance [BitTorrent tracker](http://en.wikipedia.org/wiki/BitTorrent_tracker) written in the Go programming language. It isn't quite ready for prime-time just yet, but these are the features that it targets:

- Asynchronous and concurrent (multiplexes requests over all available threads)
- Low processing and memory footprint
- IPv6 support
- Storage interface that can be easily adapted to use any data store
- Scaling properties that directly correlate with the chosen data store's scaling properties


##installing

```sh
$ go install github.com/pushrax/chihaya
```

##configuring

Configuration is done in a JSON formatted file specified with the `-config` flag. One can start with [`example/config.json`](https://github.com/pushrax/chihaya/blob/master/example/config.json) as a base.


##implementing a new storage backend

The [`storage`](http://godoc.org/github.com/pushrax/chihaya/storage) package works similar to the standard library's [`database/sql`](http://godoc.org/database/sql) package. To write a new storage backend, create a new Go package that has an implementation of both the [`Storage`](http://godoc.org/github.com/pushrax/chihaya/storage#Storage) and the [`StorageDriver`](http://godoc.org/github.com/pushrax/chihaya/storage#StorageDriver) interfaces. Within your package define an [`init()`](http://golang.org/ref/spec#Program_execution) function that calls [`storage.Register(driverName, yourStorageDriver)`](http://godoc.org/github.com/pushrax/chihaya/storage#Register). After that, all you have to do is remember to add `import _ path/to/your/library` to the top of `main.go` and now config files will recognize your driver by name.


##contributing

If you're interested in contributing, please contact us in **#chihaya on [freenode](http://freenode.net/)**([webchat](http://webchat.freenode.net?channels=chihaya)) or post to the issue tracker. Please don't offer a pull request with no prior communication attempts (unless it's small), as it will most likely lead to confusion and time wasted for everyone. And remember: good gophers always use gofmt!asted for everyone. And remember: good gophers always use gofmt!
