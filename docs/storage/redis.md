# Redis Storage

This storage implementation separates Chihaya from its storage service.
If Chihaya goes down, peer data will still be available in Redis.
It is safe to configure multiple instances of Chihaya to use the same Redis instance.
If Redis goes down, peer data will be unavailable.
By clustering redis, you can have a service that is highly-available end-to-end.

This storage implementation is currently orders of magnitude slower than the in-memory implementation.

Note: IPv4_infohash_count has a different meaning compared to the `memory` storage: it represents the number of infohashes reported by seeder, meaning that infohashes without seeders are not counted.

## Use Case

Highly available chihaya instances: when one instance of Chihaya is down, other instances can continue serving peers from Redis.

## Example Configuration

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
