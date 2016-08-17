# Chihaya

[![Build Status](https://api.travis-ci.org/chihaya/chihaya.svg?branch=master)](https://travis-ci.org/chihaya/chihaya)
[![Docker Repository on Quay.io](https://quay.io/repository/jzelinskie/chihaya/status "Docker Repository on Quay.io")](https://quay.io/repository/jzelinskie/chihaya)
[![GoDoc](https://godoc.org/github.com/chihaya/chihaya?status.svg)](https://godoc.org/github.com/chihaya/chihaya)
[![License](https://img.shields.io/badge/license-BSD-blue.svg)](https://en.wikipedia.org/wiki/BSD_licenses#2-clause_license_.28.22Simplified_BSD_License.22_or_.22FreeBSD_License.22.29)
[![IRC Channel](https://img.shields.io/badge/freenode-%23chihaya-blue.svg "IRC Channel")](http://webchat.freenode.net/?channels=chihaya)

**Note:** The master branch may be in an unstable or even broken state during development.
Please use [releases] instead of the master branch in order to get stable binaries.

Chihaya is an open source [BitTorrent tracker] written in [Go].

Differentiating features include:

- Protocol-agnostic middleware
- HTTP and UDP frontends
- IPv4 and IPv6 support
- [YAML] configuration
- Metrics via [Prometheus]

[releases]: https://github.com/chihaya/chihaya/releases
[BitTorrent tracker]: http://en.wikipedia.org/wiki/BitTorrent_tracker
[Go]: https://golang.org
[YAML]: http://yaml.org
[Prometheus]: http://prometheus.io

## Production Use

### Facebook

[Facebook] uses BitTorrent to deploy new versions of their software.
In order to optimize the flow of traffic within their datacenters, Chihaya is configured to prefer peers within the same subnet.
Because Facebook organizes their network such that server racks are allocated IP addresses in the same subnet, the vast majority of deployment traffic never impacts the congested areas of their network.

[Facebook]: https://facebook.com

### CoreOS

[Quay] is a container registry that offers the ability to download containers via BitTorrent in order to speed up large or geographically distant deployments.
Announce URLs from Quay's torrent files contain a [JWT] in order to allow Chihaya to verify that an infohash was approved by the registry.
By verifying the infohash, Quay can be sure that only their content is being shared by their tracker.

[Quay]: https://quay.io
[JWT]: https://jwt.io

## Development

### Getting Started

In order to compile the project, the [latest stable version of Go] and a [working Go environment] are required.

```sh
$ go get -t -u github.com/chihaya/chihaya
$ go install github.com/chihaya/chihaya/cmd/chihaya
```

[latest stable version of Go]: https://golang.org/dl
[working Go environment]: https://golang.org/doc/code.html

### Contributing

Long-term discussion and bug reports are maintained via [GitHub Issues].
Code review is done via [GitHub Pull Requests].
Real-time discussion is done via [freenode IRC].

For more information read [CONTRIBUTING.md].

[GitHub Issues]: https://github.com/chihaya/chihaya/issues
[GitHub Pull Requests]: https://github.com/chihaya/chihaya/pulls
[freenode IRC]: http://webchat.freenode.net/?channels=chihaya
[CONTRIBUTING.md]: https://github.com/chihaya/chihaya/blob/master/CONTRIBUTING.md

## Related projects

- [BitTorrent.org](https://github.com/bittorrent/bittorrent.org): a static website containing the BitTorrent spec and all BEPs
- [OpenTracker](http://erdgeist.org/arts/software/opentracker): a popular BitTorrent tracker written in C
- [Ocelot](https://github.com/WhatCD/Ocelot): a private BitTorrent tracker written in C++

