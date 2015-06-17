# bufferpool [![Build Status](https://secure.travis-ci.org/pushrax/bufferpool.png)](http://travis-ci.org/pushrax/bufferpool)

The bufferpool package implements a thread-safe pool of reusable, equally sized `byte.Buffer`s.
If you're allocating `byte.Buffer`s very frequently, you can use this to speed up your
program and take strain off the garbage collector.

## docs

[GoDoc](http://godoc.org/github.com/pushrax/bufferpool)
