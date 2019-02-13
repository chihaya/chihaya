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

### Production Use

#### Facebook

[Facebook] uses BitTorrent to deploy new versions of their software.
In order to optimize the flow of traffic within their datacenters, Facebook uses an alternative Storage driver for Chihaya that is configured to prefer peers within the same subnet.
Because Facebook organizes their network such that server racks are allocated IP addresses in the same subnet, the vast majority of deployment traffic never impacts the congested areas of their network.

[Facebook]: https://facebook.com

#### Red Hat

[Quay] is a container registry that offers the ability to download containers via BitTorrent in order to speed up large or geographically distant deployments.
Announce URLs from Quay's torrent files contain a [JWT] in order to allow Chihaya to verify that an infohash was approved by the registry.
By verifying the infohash, Quay can restrict their tracker to only sharing their own content.

[Quay]: https://quay.io
[JWT]: https://jwt.io

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

**NOTE:** Building in this fashion will download the latest version of all dependencies, which may have introduced breaking changes.
To produce a build with safe versions of dependencies, follow the instructions for reproducible builds.


```sh
$ mkdir chihaya && export GOPATH=$PWD/chihaya
$ go get -t -u github.com/chihaya/chihaya/...
$ $GOPATH/bin/chihaya --help
```

[latest stable version of Go]: https://golang.org/dl
[working Go environment]: https://golang.org/doc/code.html

#### Reproducible Builds

Reproducible builds are handled by using [dep] to vendor dependencies.

```sh
$ mkdir chihaya && export GOPATH=$PWD/chihaya
$ git clone git@github.com:chihaya/chihaya.git $GOPATH/src/github.com/chihaya/chihaya
$ cd $GOPATH/src/github.com/chihaya/chihaya
$ dep ensure
$ go install github.com/chihaya/chihaya/cmd/...
$ $GOPATH/bin/chihaya --help
```

[dep]: https://github.com/golang/dep

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

## Related projects

- [BitTorrent.org](https://github.com/bittorrent/bittorrent.org): a static website containing the BitTorrent spec and all BEPs
- [OpenTracker](http://erdgeist.org/arts/software/opentracker): a popular BitTorrent tracker written in C
- [Ocelot](https://github.com/WhatCD/Ocelot): a private BitTorrent tracker written in C++
