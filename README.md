chihaya
=======

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
		"unix": "/var/run/mysqld/mysqld.sock",
		"encoding": "utf8"
	},

	"addr": ":34000"
}
```

Either specify `unix` to use a unix socket or `host` and `port` to use a tcp socket. `addr` specifies the address to bind the server to.

Running
-------

`./chihaya` to run normally, `./chihaya -profile` to generate pprof data for analysis.
