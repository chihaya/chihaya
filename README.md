# Chihaya

[![Build Status](https://api.travis-ci.org/chihaya/chihaya.svg?branch=master)](https://travis-ci.org/chihaya/chihaya)
[![Docker Repository on Quay.io](https://quay.io/repository/jzelinskie/chihaya/status "Docker Repository on Quay.io")](https://quay.io/repository/jzelinskie/chihaya)
[![Go Report Card](https://goreportcard.com/badge/github.com/chihaya/chihaya)](https://goreportcard.com/report/github.com/chihaya/chihaya)
[![GoDoc](https://godoc.org/github.com/chihaya/chihaya?status.svg)](https://godoc.org/github.com/chihaya/chihaya)
![Lines of Code](https://tokei.rs/b1/github/chihaya/chihaya)
[![License](https://img.shields.io/badge/license-BSD-blue.svg)](https://en.wikipedia.org/wiki/BSD_licenses#2-clause_license_.28.22Simplified_BSD_License.22_or_.22FreeBSD_License.22.29)
[![IRC Channel](https://img.shields.io/badge/freenode-%23chihaya-blue.svg "IRC Channel")](http://webchat.freenode.net/?channels=chihaya)

**Note:** The master branch may be in an unstable or even broken state during development.
Please use [releases] instead of the master branch in order to get stable binaries.

Chihaya is an open source [BitTorrent tracker] written in [Go].

Differentiating features include:

- HTTP and UDP protocols
- IPv4 and IPv6 support
- Pre/Post middlware hooks
- [YAML] configuration
- Metrics via [Prometheus]
- High Availability via [Redis]
- Kubernetes deployment via [Helm]

[releases]: https://github.com/chihaya/chihaya/releases
[BitTorrent tracker]: https://en.wikipedia.org/wiki/BitTorrent_tracker
[Go]: https://golang.org
[YAML]: https://yaml.org
[Prometheus]: https://prometheus.io
[Redis]: https://redis.io
[Helm]: https://helm.sh

## Why Chihaya?

Chihaya is built for developers looking to integrate BitTorrent into a preexisting production environment.
Chihaya's pluggable architecture and middleware framework offers a simple and flexible integration point that abstracts the BitTorrent tracker protocols.
The most common use case for Chihaya is enabling peer-to-peer cloud software deployments.

## Development

### Contributing

Long-term discussion and bug reports are maintained via [GitHub Issues].
Code review is done via [GitHub Pull Requests].
Real-time discussion is done via [freenode IRC].

For more information read [CONTRIBUTING.md].

[GitHub Issues]: https://github.com/chihaya/chihaya/issues
[GitHub Pull Requests]: https://github.com/chihaya/chihaya/pulls
[freenode IRC]: http://webchat.freenode.net/?channels=chihaya
[CONTRIBUTING.md]: https://github.com/chihaya/chihaya/blob/master/CONTRIBUTING.md

### Getting Started

#### Building from HEAD

In order to compile the project, the [latest stable version of Go] and knowledge of a [working Go environment] are required.

```sh
$ git clone git@github.com:chihaya/chihaya.git
$ cd chihaya
$ go build ./cmd/chihaya
$ ./chihaya --help
```

[latest stable version of Go]: https://golang.org/dl
[working Go environment]: https://golang.org/doc/code.html

#### Docker

Docker containers are available for [HEAD] and [stable] releases.

[HEAD]: https://quay.io/jzelinskie/chihaya-git
[stable]: https://quay.io/jzelinskie/chihaya

#### Testing

The following will run all tests and benchmarks.
Removing `-bench` will just run unit tests.

```sh
$ go test -bench $(go list ./...)
```

The Chihaya executable contains a command to end-to-end test a BitTorrent tracker.
See

```sh
$ chihaya --help
``` 

### Configuration

Configuration of Chihaya is done via one YAML configuration file.
The `dist/` directory contains an example configuration file.
Files and directories under `docs/` contain detailed information about configuring middleware, storage implementations, architecture etc.

## Related projects

- [BitTorrent.org](https://github.com/bittorrent/bittorrent.org): a static website containing the BitTorrent spec and all BEPs
- [OpenTracker](http://erdgeist.org/arts/software/opentracker): a popular BitTorrent tracker written in C
- [Ocelot](https://github.com/WhatCD/Ocelot): a private BitTorrent tracker written in C++
