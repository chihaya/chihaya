# Chihaya [![Build Status](https://api.travis-ci.org/chihaya/chihaya.svg?branch=master)](https://travis-ci.org/chihaya/chihaya)

Chihaya is a high-performance [BitTorrent tracker] written in the Go
programming language. It is still heavily under development and the current
`master` branch should probably not be used in production
(unless you know what you're doing).

Features include:

- Public tracker feature-set with full compatibility with what exists of the BitTorrent spec
- Private tracker feature-set with compatibility for a [Gazelle]-like deployment (WIP)
- Low resource consumption, and fast, asynchronous request processing
- Full IPv6 support, including handling for dual-stacked peers
- Extensive metrics for visibility into the tracker and swarm's performance
- Ability to prioritize peers in local subnets to reduce backbone contention
- Pluggable backend driver that can coordinate with an external database

[BitTorrent tracker]: http://en.wikipedia.org/wiki/BitTorrent_tracker
[gazelle]: https://github.com/whatcd/gazelle

## When would I use Chihaya?

Chihaya is a meant for every kind of BitTorrent tracker deployment. Chihaya has
been used to replace instances of [opentracker] and also instances of [ocelot].
Chihaya handles torrent announces and scrapes in memory, but using a backend
driver, can also asynchronously provide deltas to maintain a set of persistent
data without throttling a database (this most useful for private tracker
use-cases).

[opentracker]: http://erdgeist.org/arts/software/opentracker
[ocelot]: https://github.com/WhatCD/Ocelot

## Building & Installing

Chihaya requires Go 1.4, [Godep], and a [Go environment] previously setup.

[Godep]: https://github.com/tools/godep
[Go environment]: https://golang.org/doc/code.html

```sh
$ export GOPATH=$PWD/chihaya
$ git clone github.com/chihaya/chihaya chihaya/src/github.com/chihaya/chihaya
$ godep go install chihaya/src/github.com/chihaya/cmd/chihaya
```

### Testing

Chihaya has end-to-end test coverage for announces in addition to unit tests for
isolated components. To run the tests, use:

```sh
$ cd $GOPATH/src/github.com/chihaya/chihaya
$ godep go test -v ./...
```

There is also a set of benchmarks for performance-critical sections of Chihaya.
These can be run similarly:

```sh
$ cd $GOPATH/src/github.com/chihaya/chihaya
$ godep go test -v ./... -bench .
```

## Configuration

Copy [`example_config.json`](https://github.com/chihaya/chihaya/blob/master/example_config.json)
to your choice of location, and update the values as required.
The available keys and their default values are as follows:

- `private_enabled: false` – if this is a private tracker
- `freeleech_enabled: false` – for private trackers, whether download stats should be counted for users
- `purge_inactive_torrents: true` – if torrents should be forgotten after some time
- `announce: "30m"` – the announce "interval" value sent to clients
- `min_announce: "15m"` – the announce "min_interval" value sent to clients
- `default_num_want: 50` – the default number of peers to return if the client has not specified
- `torrent_map_shards: 1` – number of torrent maps to use (leave this at 1 in general)
- `allow_ip_spoofing: true` – if peers are allowed to set their own IP, this must be enabled for dual-stack IP support
- `dual_stacked_peers: true` – if peers may have both an IPv4 and IPv6 address, otherwise only one IP per peer will be used
- `real_ip_header: ""` – optionally an HTTP header where the upstream IP is stored, for example `X-Forwarded-For` or `X-Real-IP`
- `respect_af: false` – if responses should only include peers of the same address family as the announcing peer
- `client_whitelist_enabled: false` – if peer IDs should be matched against the whitelist
- `client_whitelist: []` – list of peer ID prefixes to allow
- `http_listen_addr: ":6881"` – listen address for the HTTP server
- `http_request_timeout: "10s"`
- `http_read_timeout: "10s"`
- `http_write_timeout: "10s"`
- `http_listen_limit: 0`
- `driver: "noop"`
- `stats_buffer_size: 0`
- `include_mem_stats: true`
- `verbose_mem_stats: false`
- `mem_stats_interval: "5s"`
