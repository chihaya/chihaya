chihaya
=======

[![Build Status](https://travis-ci.org/pushrax/chihaya.png?branch=master)](https://travis-ci.org/pushrax/chihaya)

chihaya is a high-performance [BitTorrent tracker](http://en.wikipedia.org/wiki/BitTorrent_tracker) written in the Go programming language.
It isn't quite ready for prime-time just yet, but these are the features that it'll have:

- Low processing and memory footprint
- IPv6 support
- Support for multiple storage backends
- Linear horizontal scalability (depending on the backends)


Installing
----------

    $ go install github.com/pushrax/chihaya


Configuration
-------------

Configuration is done in a JSON formatted file specified with the -config flag.
One can start with `example/config.json`, as a base. Check out GoDoc for more info.


Contributing
------------

If you want to make a smaller change, just go ahead and do it, and when you're
done send a pull request through GitHub. If there's a larger change you want to
make, it would be preferable to discuss it first via a GitHub issue or by
getting in touch on IRC. Always remember to gofmt your code!


Contact
-------

If you have any questions or want to contribute something, come say hi in the
IRC channel: **#chihaya on [freenode](http://freenode.net/)**
([webchat](http://webchat.freenode.net?channels=chihaya)).

