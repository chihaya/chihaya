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

Seeders and Leechers for a particular swarm are stored within a redis hash each.
The keys are derived from the InfoHash, _peer keys_ are the fields, last modified times are values.
Peer keys are derived from peers and contain Peer ID, IP, and Port.
Additionally, a count of peers per swarms is also kept under a hash for each address family.

Here is an example:

```
- IPv4_swarm_counts
  - <20-byte infohash 1>: 3
  - <20-byte infohash 2>: 1
- s<20-byte infohash 1><IPv4 marker byte><0x00 (seeder marker)>
  - <peer 1 key>: <modification time>
  - <peer 2 key>: <modification time>
- s<20-byte infohash 1><IPv4 marker byte><0x01 (seeder marker)>
  - <peer 3 key>: <modification time>
- s<20-byte infohash 2><IPv4 marker byte><0x00 (seeder marker)>
  - <peer 3 key>: <modification time>
```


In this case, prometheus would record two swarms, two seeders, and two leechers.
These two keys per address family are used to record the count of seeders and leechers:

```
- IPv4_seeder_count: 2
- IPv4_leecher_count: 2
```

Most interactions with redis are implemented using scripts.

The GoDoc of the `storage/redis` package may have more information.
