chihaya
=======

[![Build Status](https://travis-ci.org/kotokoko/chihaya.png?branch=master)](https://travis-ci.org/kotokoko/chihaya)

Due to the many inconsistencies AB has with Gazelle, Chihaya is not ready for general use. Currently the way Chihaya finds out about new and deleted data is by polling the database server, which is highly inefficent and introduces a race condition when data is deleted from the source (due to `INSERT INTO x ON DUPLICATE KEY UPDATE` being used). Once [Batter](https://github.com/wafflesfm/batter) is ready for use, Chihaya will be updated to use a pubsub architecture for loading these data changes.

Compiling
---------

`go get` to fetch dependencies, `go build` to compile.

Configuration
-------------

Configuration is done in `config.json`, which you'll need to create by copying `config.json.example`. See [config/config.go](https://github.com/kotokoko/chihaya/blob/master/config/config.go) for a description of each configuration value.

Running
-------

`./chihaya` to run normally, `./chihaya -profile` to generate pprof data for analysis.

