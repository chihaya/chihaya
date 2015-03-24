# Chihaya

[![GoDoc](https://godoc.org/github.com/chihaya/chihaya?status.svg)](https://godoc.org/github.com/chihaya/chihaya)
[![Build Status](https://api.travis-ci.org/chihaya/chihaya.svg?branch=master)](https://travis-ci.org/chihaya/chihaya)
[![Docker Repository on Quay.io](https://quay.io/repository/jzelinskie/chihaya/status "Docker Repository on Quay.io")](https://quay.io/repository/jzelinskie/chihaya)

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

Chihaya requires 64-bit Go 1.4, [Godep], and a [Go environment] previously set up.

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

##### `http_listen_addr`

    type: string
    default: ":6881"
    
The listen address for the HTTP server. If only a port is specified, the tracker will listen on all interfaces.

##### `private_enabled`

    type: bool
    default: true

Whether this is a public or private tracker.

##### `create_on_announce`

    type: bool
    default: true

Whether to register new torrents with the tracker when any client announces (`true`), or to return an error if the torrent doesn't exist (`false`). This should be set to `false` for private trackers in most cases.

##### `purge_inactive_torrents`

    type: bool
    default: true

If torrents should be forgotten when there are no active peers. This should be set to `false` for private trackers.

##### `announce`

    type: duration
    default: "30m"

The announce `interval` value sent to clients. This specifies how long clients should wait between regular announces.

##### `min_announce`

    type: duration
    default: "30m"

The announce `min_interval` value sent to clients. This theoretically specifies the minimum allowed time between announces, but most clients don't really respect it.

##### `default_num_want`

    type: integer
    default: 50

The default maximum number of peers to return if the client has not requested a specific number.

##### `allow_ip_spoofing`

    type: bool
    default: true

Whether peers are allowed to set their own IP via the various supported methods or if these are ignored. This must be enabled for dual-stack IP support, since there is no other way to determine both IPs of a peer otherwise.

##### `dual_stacked_peers`

    type: bool
    default: true

True if peers may have both an IPv4 and IPv6 address, otherwise only one IP per peer will be used.

##### `real_ip_header`

    type: string
    default: blank

An optional HTTP header indicating the upstream IP, for example `X-Forwarded-For` or `X-Real-IP`. Use this when running the tracker behind a reverse proxy.

##### `respect_af`

    type: bool
    default: false

Whether responses should only include peers of the same address family as the announcing peer, or if peers of any family may be returned (i.e. both IPv4 and IPv6).

##### `client_whitelist_enabled`

    type: bool
    default: false

Enables the peer ID whitelist.

##### `client_whitelist`

    type: array of strings
    default: []

List of peer ID prefixes to allow if `client_whitelist_enabled` is set to true.

##### `freeleech_enabled`

    type: bool
    default: false

For private trackers only, whether download stats should be counted or ignored for users.

##### `torrent_map_shards`

    type: integer
    default: 1

Number of internal torrent maps to use. Leave this at 1 in general, however it can potentially improve performance when there are many unique torrents and few peers per torrent.

- `http_request_timeout: "10s"`
- `http_read_timeout: "10s"`
- `http_write_timeout: "10s"`
- `http_listen_limit: 0`
- `driver: "noop"`
- `stats_buffer_size: 0`
- `include_mem_stats: true`
- `verbose_mem_stats: false`
- `mem_stats_interval: "5s"`
