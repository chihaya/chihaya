This page describes how data is stored in Redis and design/implementation details. This schema is still a [work in progress](https://github.com/pushrax/chihaya/tree/redis_driver), with partial tests and benchmarks. The interface to Redis must follow the [cache](../blob/master/cache/cache.go) specification.

## Model storage
Following the [architecture](./Architecture), the Redis cache stores [models]( ../blob/master/models/models.go) using Redis's Sets and Hashes. The peer cache is designed to be shared among multiple tracker instances.

### models.Torrent
Each torrent is a [Redis Hash type](http://redis.io/commands#hash), keyed by the concatenation of "torrent:" with the torrent's Infohash. Each hash field corresponds to the data members of the torrent with the exception of seeders and leechers map(see below)

### models.Peer
Each peer is stored as a Redis hash, with keyed by the concatenation of peer.ID, user.ID and torrent.ID. This does not store if the peer is a seeder or leecher on the torrent.

The seeder and leecher statuses are saved in [Redis sets](http://redis.io/commands#set), with the set name keyed by the concatenation of a prefix to differentiate seeders("seeders:") from leechers("leechers:") and the corresponding torrent's ID. The set contains the hash key of the peer.

### models.User
Users are stored as [Redis Hash type](http://redis.io/commands#hash), keyed by the concatenation of "user:" and a passkey. The hash fields contain the user's data.

## Shared tracker memory
In addition to caching frequently written data, the peer cache also stores a white-list of torrent clients as a [Redis sets](http://redis.io/commands#set)

***

# Concurrency concerns
As multiple trackers will be updating the peer cache simultaneously, it is important to make each cache action atomic. This will prevent the peer cache from becoming inconsistent _(e.g. peers without torrents)_ or incorrect (write-after-write hazards). 
Luckily, Redis processes commands serially (using only a single thread) and all methods on Redis data types are atomic. The only problems then stem from updating multiple related types (peers and torrents), and local tracker copies of data in the peer cache.

## Consistency
For example, the cache.FindTorrent method returns all of the peers associated with the torrent. This requires at least 2 commands; one to find the hash keys of the peers, and another to return the peer's data. If, in between the execution of the first and second commands, a seeder is removed, the peer's hash look-up will return an error because a requested peer no longer exists.

Any pair of methods that: (return a torrent or multiple peers) and (modify the existence of a peer) can encounter the aforementioned issue.

### Lowered expectations or 'Eventual consistency'
Looking up a list of peers could allow for failed look-ups. This approach would make the torrent-peer relationships eventually consistent, but special care would need to be taken in controlling write access to Redis to make sure that the failed look-ups are only indicative of temporary inconsistencies.

## Local copies
This issue lies mostly with the interface. As local copies of models cannot be used as truth without a method to keep them updated. Preventing read-modfiy-writeback scenarios on local copies outside of the cache interface can be difficult, as the peer cache interface consumer would need to be aware of the cache's asynchronous writes.

### No writing local copies
Instead of complicating the peer cache interface ( which is a second level cache already ) with local cache that needs to be kept in sync, the tracker could be prevented from performing destructive writes to the peer cache. This would leverage Redis's atomic in-memory modification commands instead of relying on complex Lua scripts. The peer information is overwritten by design from Announce, making the local copy the truth when it is written to the peer cache.

# Benchmarks

The results below were created by running `go test -bench . -benchmem ` in the cache/redis directory.
To prevent errors caused by quickly creating and releasing many connections to the Redis server, run `echo 1 > /proc/sys/net/ipv4/tcp_tw_reuse` or equivalent command for your operating system to prevent running out TCP sockets.

[Redis self-benchmark](https://gist.github.com/cpb8010/6431820)

Benchmark machine specs:
* CPU: PhenomII X3 720 @ 3.0Ghz
* Memory: 8GB DDR3 @ 1333Mhz
* Redis: Local instance, Version 2.6.14 (00000000/0) 64 bit
* OS: OpenSuse 12.3 x64


| Benchmark Name | Iterations  | Avg. function run-time | Avg. bytes processed | Avg. memory allocations |
|:-------------- | -----------:| ---------------------:| --------------------:| -----------------------:|
| BenchmarkSuccessfulFindUser |    50000 |      36802 ns/op |     1483 B/op |       49 allocs/op |
| BenchmarkFailedFindUser |   100000 |      25297 ns/op |      131 B/op |        4 allocs/op |
| BenchmarkSuccessfulFindTorrent |     5000 |     343198 ns/op |    15621 B/op |      465 allocs/op |
| BenchmarkFailFindTorrent |   100000 |      24897 ns/op |      131 B/op |        4 allocs/op |
| BenchmarkSuccessfulClientWhitelisted |   100000 |      24983 ns/op |       82 B/op |        4 allocs/op |
| BenchmarkFailClientWhitelisted |   100000 |      24520 ns/op |       82 B/op |        4 allocs/op |
| BenchmarkRecordSnatch |    50000 |      53286 ns/op |      293 B/op |        8 allocs/op |
| BenchmarkMarkActive |   100000 |      26108 ns/op |      147 B/op |        4 allocs/op |
| BenchmarkAddSeeder |    50000 |      72068 ns/op |     1865 B/op |       33 allocs/op |
| BenchmarkRemoveSeeder |    50000 |      55899 ns/op |      326 B/op |       17 allocs/op |
| BenchmarkSetSeeder |    50000 |      45467 ns/op |     1320 B/op |       25 allocs/op |
| BenchmarkIncrementSlots |   100000 |      27531 ns/op |      146 B/op |        4 allocs/op |
| BenchmarkLeecherFinished |    50000 |      75805 ns/op |     1984 B/op |       37 allocs/op |
| BenchmarkRemoveLeecherAddSeeder |    10000 |     127241 ns/op |     2219 B/op |       50 allocs/op |