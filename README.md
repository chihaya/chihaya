chihaya
=======

Due to the many inconsistencies AB has with Gazelle, Chihaya is not ready for general use. Currently the way Chihaya finds out about new and deleted data is by polling the database server, which is highly inefficent and introduces a race condition when data is deleted from the source (due to `INSERT INTO x ON DUPLICATE KEY UPDATE` being used). Once [Batter](https://github.com/wafflesfm/batter) is ready for use, Chihaya will be updated to use a pubsub architecture for loading these data changes.

Compiling
---------

`go get` to fetch dependencies, `go build` to compile.

Configuration
-------------

Timing configuration is currently hardcoded in `config/config.go`. Edit that and recompile.

Database configuration is done in `config.json`, which you'll need to create with the following format:

```json
{
	"database": {
		"username": "user",
		"password": "pass",
		"database": "database",
		"proto": "unix",
		"addr": "/var/run/mysqld/mysqld.sock",
		"encoding": "utf8"
	},

	"addr": ":34000"
}
```

`addr` specifies the address to bind the server to. Possible values for `database.proto` are `unix` and `tcp`.

Running
-------

`./chihaya` to run normally, `./chihaya -profile` to generate pprof data for analysis.
