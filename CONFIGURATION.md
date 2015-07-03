# Configuration

Chihaya's behaviour is customized by setting up a JSON configuration file.
Available keys are as follows:

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
- `stats_buffer_size: 0`
- `include_mem_stats: true`
- `verbose_mem_stats: false`
- `mem_stats_interval: "5s"`

