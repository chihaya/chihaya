# Chihaya

[![Coverage Status](https://coveralls.io/repos/github/chihaya/chihaya/badge.svg?branch=master)](https://coveralls.io/github/chihaya/chihaya?branch=master)
[![Build Status](https://api.travis-ci.org/chihaya/chihaya.svg?branch=master)](https://travis-ci.org/chihaya/chihaya)
[![Docker Repository on Quay.io](https://quay.io/repository/jzelinskie/chihaya/status "Docker Repository on Quay.io")](https://quay.io/repository/jzelinskie/chihaya)
[![GoDoc](https://godoc.org/github.com/chihaya/chihaya?status.svg)](https://godoc.org/github.com/chihaya/chihaya)
[![License](https://img.shields.io/badge/license-BSD-blue.svg)](https://en.wikipedia.org/wiki/BSD_licenses#2-clause_license_.28.22Simplified_BSD_License.22_or_.22FreeBSD_License.22.29)
[![IRC Channel](https://img.shields.io/badge/freenode-%23chihaya-blue.svg "IRC Channel")](http://webchat.freenode.net/?channels=chihaya)

Chihaya is an open source [BitTorrent tracker] written in [Go].

Differentiating features include:

- Protocol-agnostic and middleware-composed logic
- Low resource consumption and fast, asynchronous request processing
- Unified IPv4 and IPv6 [swarms]
- [YAML] configuration
- Optional metrics via [Prometheus]

[BitTorrent tracker]: http://en.wikipedia.org/wiki/BitTorrent_tracker
[Go]: https://golang.org
[swarms]: https://en.wikipedia.org/wiki/Glossary_of_BitTorrent_terms#Swarm
[YAML]: http://yaml.org
[Prometheus]: http://prometheus.io

## Getting Started

In order to compile the project, the latest stable version of Go and a working Go environment are required.

```sh
$ go get github.com/chihaya/chihaya
$ go install github.com/chihaya/chihaya/cmd/chihaya
```

## Development

Long-term discussion and bug reports are maintained via [GitHub Issues].
Code review is done via [GitHub Pull Requests].
Real-time discussion is done via [freenode IRC].

[GitHub Issues]: https://github.com/chihaya/chihaya/issues
[GitHub Pull Requests]: https://github.com/chihaya/chihaya/pulls
[freenode IRC]: http://webchat.freenode.net/?channels=chihaya

## Related projects

- [OpenTracker](http://erdgeist.org/arts/software/opentracker): a popular BitTorrent tracker written in C
- [Ocelot](https://github.com/WhatCD/Ocelot): a private BitTorrent tracker written in C++

## License

Chihaya is distributed under the 2-Clause BSD license that can be found in the `LICENSE` file.
