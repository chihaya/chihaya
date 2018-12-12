# Redis Storage

This storage system separates chihaya from storage and stores all peer data in Redis to achieve HA.

## Use Case

When one chihaya instance is down, the Redis can continuily serve peer data through other chihaya instances.

## Configuration

```yaml
chihaya:
  storage:
    name: redis
    config:
      # The frequency which stale peers are removed.
      gc_interval: 14m

      # The frequency which metrics are pushed into a local Prometheus endpoint.
      prometheus_reporting_interval: 1s

      # The amount of time until a peer is considered stale.
      # To avoid churn, keep this slightly larger than `announce_interval`
      peer_lifetime: 16m

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

Seeders and Leechers for a particular InfoHash are stored with a redis hash structure, the infohash is used as hash key, peer key is field, last modified time is value.

All the InfoHashes (swarms) are also stored into redis hash, IP family is the key, infohash is field, last modified time is value.

Here is an example

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


In this case, prometheus will record two swarms, three seeders and one leecher.

So tree keys are used to record the count of swarms, seeders and leechers for each group (IPv4, IPv6).

```
- IPv4_infohash_count: 2
- IPv4_S_count: 3
- IPv4_L_count: 1
```

Note: IPv4_infohash_count has the different meaning with `memory` storage, it represents the number of infohashes reported by seeder.
