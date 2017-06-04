# Memory Subnet Storage

This storage system stores all peer data ephemerally in memory and prioritizes peers in the same subnet.

## Use Case

When the network being used for BitTorrent traffic is organized such that IP address can be mapped to physical location, this storage will encourage peers to transfer data between physically closer peers.

## Configuration

```yaml
chihaya:
  storage:
    name: memorybysubnet
    config:
      # The frequency which stale peers are removed.
      gc_interval: 14m

      # The frequency which metrics are pushed into a local Prometheus endpoint.
      prometheus_reporting_interval: 1s

      # The amount of time until a peer is considered stale.
      # To avoid churn, keep this slightly larger than `announce_interval`
      peer_lifetime: 16m

      # The number of partitions data will be divided into in order to provide a
      # higher degree of parallelism.
      shard_count: 1024

      # The number of bits that are used to mask IPv4 peers' addresses such that peers with the same mask are returned first from announces.
      preferred_ipv4_subnet_mask_bits_set: 8

      # The number of bits that are used to mask IPv6 peers' addresses such that peers with the same mask are returned first from announces.
      preferred_ipv6_subnet_mask_bits_set: 16
```

## Implementation

The implementation of this storage strives to remain as similar to the `memory` storage system as possible.

Seeders and Leechers for a particular InfoHash are organized into maps by subnet (and then mapped to their last announce time):

```go
type swarm struct {
	seeders  map[peerSubnet]map[serializedPeer]int64
	leechers map[peerSubnet]map[serializedPeer]int64
}
```

This causes the allocation and maintenance overhead of many extra maps.
Expect approximately a x2 slowdown in performance compared to `memory`.
