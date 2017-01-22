# Deniability Middleware

This package provides the PreHook `deniability` which inserts ghost peers into announce responses and increases scrape counters to achieve plausible deniability.

## Functionality

### For Announces

This middleware will choose random announces and modify the list of peers returned.
A random number of randomly generated peers will be inserted at random positions into the list of peers.
As soon as the length of the list of peers exceeds `numWant`, peers will be replaced rather than inserted.

Also note that the IP address for the generated peeer consists of bytes in the range [1,254].
Whether IPv4 or IPv6 addresses are generated depends on the announcing Peer's IP.

Note that if a response is picked for augmentation, at least one Peer will be inserted.
There is one exception to this rule:
Otherwise empty reponse will not be augmented to make it more difficult to determine the prefixes used for generated Peers.

### For Scrapes

A scrape will randomly be chosen, based on the `modify_response_probability`.
If chosen, a number of seeders and leechers will be generated for every InfoHash of the scrape.

Note that there will be at least one peer added to every InfoHash, this can be either a seeder or a leecher.
As with Announces, the only exception to this rule are otherwise empty scrapes.

## Configuration

This middleware provides the following parameters for configuration:

- `modify_response_probability` (float, >0, <= 1) indicates the probability by which a response will be augmented.
- `max_random_peers` (int, >0) sets an upper boundary (inclusive) for the amount of peers added.
- `prefix` (string, 20 characters at most) sets the prefix for generated peer IDs.
    The peer ID will be padded to 20 bytes using a random string of numeric characters.
- `min_port` (int, >0, <=65535) sets a lower boundary for the port for generated peers.
- `max_port` (int, >0, <=65536, > `min_port`) sets an upper boundary for the port for generated peers.
- `parallelism` (int, >=0) determines the amount of parallelism this hook can operate with.
    The resulting theoretical upper limit of parallelism is `2^parallelism)`, assuming an even random distribution of infohashes.

An example config might look like this:

```yaml
chihaya:
  prehooks:
    - name: deniability
      config:
        modify_response_probability: 0.2
        max_random_peers: 5
        prefix: -AZ2060-
        min_port: 40000
        max_port: 60000
        parallelism: 8
```

For more information about peer IDs and their prefixes, see [this wiki entry](https://wiki.theory.org/BitTorrentSpecification#peer_id).
