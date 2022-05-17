# LBRY Tracker

The LBRY tracker is a server that helps peers find each other. It was forked from [Chihaya](https://github.com/chihaya/chihaya), an open-source [BitTorrent tracker](https://en.wikipedia.org/wiki/BitTorrent_tracker).


## Installation and Usage

### Building from HEAD

In order to compile the project, the [latest stable version of Go] and knowledge of a [working Go environment] are required.

```sh
git clone git@github.com:lbryio/tracker.git
cd tracker
go build ./cmd/chihaya
./chihaya --help
```

[latest stable version of Go]: https://golang.org/dl
[working Go environment]: https://golang.org/doc/code.html

### Testing

The following will run all tests and benchmarks.
Removing `-bench` will just run unit tests.

```sh
go test -bench $(go list ./...)
```

The tracker executable contains a command to end-to-end test a BitTorrent tracker.
See

```sh
tracker --help
```

### Configuration

Configuration of the tracker is done via one YAML configuration file.
The `dist/` directory contains an example configuration file.
Files and directories under `docs/` contain detailed information about configuring middleware, storage implementations, architecture etc.

## Contributing

Contributions to this project are welcome, encouraged, and compensated. For more details, please check [this](https://lbry.tech/contribute) link.

## License

LBRY's code changes are MIT licensed, and the upstream Chihaya code is licensed under a BSD 2-Clause license. For the full license, see [LICENSE](LICENSE).

## Security

We take security seriously. Please contact security@lbry.com regarding any security issues. [Our PGP key is here](https://lbry.com/faq/pgp-key) if you need it.

## Contact

The primary contact for this project is [@shyba](mailto:vshyba@lbry.com).
