chihaya
=======

Don't expect this to work with your database right out of the box; it uses a different schema than gazelle.

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
