# Configuration

Chihaya's behaviour is customized by setting up a JSON configuration file.
Available keys are as follows:

##### `httpListenAddr`

    type: string
    default: "localhost:6881"

The listen address for the HTTP server. If only a port is specified, the tracker will listen on all interfaces. If left empty, the tracker will not run a HTTP endpoint.

##### `httpRequestTimeout`

    type: duration
    default: "4s"

The duration to allow outstanding requests to survive before forcefully terminating them.

##### `httpReadTimeout`

    type: duration
    default: "4s"

The maximum duration before timing out read of the request.

##### `httpWriteTimeout`

    type: duration
    default: "4s"

The maximum duration before timing out write of the request.

##### `httpListenLimit`

    type: integer
    default: 0

Limits the number of outstanding requests. Set to `0` to disable.

##### `udpListenAddr`

    type: string
    default: "localhost:6881"

Then listen address for the UDP server. If only a port is specified, the tracker will listen on all interfaces. If left empty, the tracker will not run a UDP endpoint.

##### `createOnAnnounce`

    type: bool
    default: true

Whether to register new torrents with the tracker when any client announces (`true`), or to return an error if the torrent doesn't exist (`false`).

##### `purgeInactiveTorrents`

    type: bool
    default: true

If torrents should be forgotten when there are no active peers.

##### `announce`

    type: duration
    default: "30m"

The announce `interval` value sent to clients. This specifies how long clients should wait between regular announces.

##### `minAnnounce`

    type: duration
    default: "30m"

The announce `min_interval` value sent to clients. This theoretically specifies the minimum allowed time between announces, but most clients don't really respect it.

##### `defaultNumWant`

    type: integer
    default: 50

The default maximum number of peers to return if the client has not requested a specific number.

##### `allowIPSpoofing`

    type: bool
    default: true

Whether peers are allowed to set their own IP via the various supported methods or if these are ignored. This must be enabled for dual-stack IP support, since there is no other way to determine both IPs of a peer otherwise.

##### `dualStackedPeers`

    type: bool
    default: true

True if peers may have both an IPv4 and IPv6 address, otherwise only one IP per peer will be used.

##### `realIPHeader`

    type: string
    default: blank

An optional HTTP header indicating the upstream IP, for example `X-Forwarded-For` or `X-Real-IP`. Use this when running the tracker behind a reverse proxy.

##### `respectAF`

    type: bool
    default: false

Whether responses should only include peers of the same address family as the announcing peer, or if peers of any family may be returned (i.e. both IPv4 and IPv6).

##### `clientWhitelistEnabled`

    type: bool
    default: false

Enables the peer ID whitelist.

##### `clientWhitelist`

    type: array of strings
    default: []

List of peer ID prefixes to allow if `client_whitelist_enabled` is set to true.

##### `torrentMapShards`

    type: integer
    default: 1

Number of internal torrent maps to use. Leave this at 1 in general, however it can potentially improve performance when there are many unique torrents and few peers per torrent.

##### `reapInterval`

    type: duration
    default: "60s"

Interval at which a search for inactive peers should be performed.

##### `reapRatio`

    type: float64
    default: 1.25

Peers will be rated inactive if they haven't announced for `reapRatio * minAnnounce`.

##### `apiListenAddr`

    type: string
    default: "localhost:6880"

The listen address for the HTTP API. If only a port is specified, the tracker will listen on all interfaces. If left empty, the tracker will not run the HTTP API.

##### `apiRequestTimeout`

    type: duration
    default: "4s"

The duration to allow outstanding requests to survive before forcefully terminating them.

##### `apiReadTimeout`

    type: duration
    default: "4s"

The maximum duration before timing out read of the request.

##### `apiWriteTimeout`

    type: duration
    default: "4s"

The maximum duration before timing out write of the request.

##### `apiListenLimit`

    type: integer
    default: 0

Limits the number of outstanding requests. Set to `0` to disable.

##### `driver`

    type: string
    default: "noop"

Sets the backend driver to load. The included `"noop"` driver provides no functionality.

##### `statsBufferSize`

    type: integer
    default: 0

The size of the event-queues for statistics.

##### `includeMemStats`

    type: bool
    default: true

Whether to include information about memory in the statistics.

##### `verboseMemStats`

    type: bool
    default: false

Whether the information about memory should be verbose.

##### `memStatsInterval`

    type: duration
    default: "5s"

Interval at which to collect statistics about memory.


##### `jwkSetURI`

    type: string
    default: ""

If this string is not empty, then the tracker will attempt to use JWTs to validate infohashes before announces. The format for the JSON at this endpoint can be found at [the RFC for JWKs](https://tools.ietf.org/html/draft-ietf-jose-json-web-key-41#page-10) with the addition of an "issuer" key. Simply stated, this feature requires two fields at this JSON endpoint: "keys" and "issuer". "keys" is a list of JWKs that can be used to validate JWTs and "issuer" should match the "iss" claim in the JWT. The lifetime of a JWK is based upon standard HTTP caching headers and falls back to 5 minutes if no cache headers are provided.


#### `jwkSetUpdateInterval`

    type: duration
    default: "5m"

The interval at which keys are updated from JWKSetURI. Because the fallback lifetime for keys without cache headers is 5 minutes, this value should never be below 5 minutes unless you know your jwkSetURI has caching headers.

#### `jwtAudience`

    type: string
    default: ""

The audience claim that is used to validate JWTs.
