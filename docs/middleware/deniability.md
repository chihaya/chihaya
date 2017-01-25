# Deniability Middleware

This package provides the PreHook `deniability` which inserts ghost peers into announce responses to achieve plausible deniability.

## Functionality

### For Announces

This middleware will choose random announces and modify the list of peers returned.
A random number *k* of randomly generated peers will be inserted at random positions into the list of peers.
Peers will only be inserted until there is only space for one more Peer or *k* Peers have been inserted. 

Also note that the IP address for the generated peeer consists of bytes in the range [1,254].
Whether IPv4 or IPv6 addresses are generated depends on the announcing Peer's IP.

Note that if a response is picked for augmentation, at least one Peer will be inserted.

### For Scrapes

Scrapes are not altered.

## Configuration

This middleware provides the following parameters for configuration:

- `modify_response_probability` (float, >0, <= 1) indicates the probability by which a response will be augmented.
- `max_random_peers` (int, >0) sets an upper boundary (inclusive) for the amount of peers added.
- `prefix` (string, 20 characters at most) sets the prefix for generated peer IDs.
    The peer ID will be padded to 20 bytes using a random string of numeric characters.
- `min_port` (int, >0, <=65535) sets a lower boundary for the port for generated peers.
- `max_port` (int, >0, <=65536, > `min_port`) sets an upper boundary for the port for generated peers.

An example config might look like this:

```yaml
chihaya:
  prehooks:
    - name: deniability
      config:
        modify_response_probability: 0.01
        max_random_peers: 5
        prefix: OP1011-
        min_port: 10000
        max_port: 60000
```


For more information about peer IDs and their prefixes, see [this wiki entry](https://wiki.theory.org/BitTorrentSpecification#peer_id).
