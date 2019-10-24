# Redis Storage

This storage implementation separates Chihaya from its storage service.
Chihaya achieves HA by storing all peer data in Redis.
Multiple instances of Chihaya can use the same redis instance concurrently.
The storage service can get HA by clustering.
If one instance of Chihaya goes down, peer data will still be available in Redis.

The HA of storage service is not considered here.
In case Redis runs as a single node, peer data will be unavailable if the node is down.
You should consider setting up a Redis cluster for Chihaya in production.

This storage implementation is currently orders of magnitude slower than the in-memory implementation.

## Use Case

When one instance of Chihaya is down, other instances can continue serving peers from Redis.

## Configuration

```yaml
chihaya:
  storage:
    name: redis
    config:
      # The frequency which stale peers are removed.
      # This balances between
      # - collecting garbage more often, potentially using more CPU time, but potentially using less memory (lower value)
      # - collecting garbage less frequently, saving CPU time, but keeping old peers long, thus using more memory (higher value).
      gc_interval: 3m

      # The interval at which metrics about the number of infohashes and peers
      # are collected and posted to Prometheus.
      prometheus_reporting_interval: 1s

      # The amount of time until a peer is considered stale.
      # To avoid churn, keep this slightly larger than `announce_interval`
      peer_lifetime: 31m

      # The address of redis storage.
      redis_broker: "redis://pwd@127.0.0.1:6379/0"

      # The timeout for reading a command reply from redis.
      redis_read_timeout: 15s

      # The timeout for writing a command to redis.
      redis_write_timeout: 15s

      # The timeout for connecting to redis server.
      redis_connect_timeout: 15s
```

## Implementation

Seeders and Leechers for a particular InfoHash are stored within a redis hash.
The InfoHash is used as key, _peer keys_ are the fields, last modified times are values.
Peer keys are derived from peers and contain Peer ID, IP, and Port.
All the InfoHashes (swarms) are also stored in a redis hash, with IP family as the key, infohash as field, and last modified time as value.

Here is an example:

```
- IPv4
  - IPv4_S_<infohash 1>: <modification time>
  - IPv4_L_<infohash 1>: <modification time>
  - IPv4_S_<infohash 2>: <modification time>
- IPv4_S_<infohash 1>
  - <peer 1 key>: <modification time>
  - <peer 2 key>: <modification time>
- IPv4_L_<infohash 1>
  - <peer 3 key>: <modification time>
- IPv4_S_<infohash 2>
  - <peer 3 key>: <modification time>
```


In this case, prometheus would record two swarms, three seeders, and one leecher.
These three keys per address family are used to record the count of swarms, seeders, and leechers.

```
- IPv4_infohash_count: 2
- IPv4_S_count: 3
- IPv4_L_count: 1
```

Note: IPv4_infohash_count has a different meaning compared to the `memory` storage:
It represents the number of infohashes reported by seeder, meaning that infohashes without seeders are not counted.
