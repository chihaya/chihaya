## Deniability Middleware

This package provides the announce middleware `deniability` which inserts ghost peers into announce responses to achieve plausible deniability.

### Functionality

This middleware will choose random announces and modify the list of peers returned.
A random number of randomly generated peers will be inserted at random positions into the list of peers.
As soon as the list of peers exceeds `numWant`, peers will be replaced rather than inserted.

Note that if a response is picked for augmentation, both IPv4 and IPv6 peers will be modified, in case they are not empty.

Also note that the IP address for the generated peeer consists of bytes in the range [1,254].

### Configuration

This middleware provides the following parameters for configuration:

- `modify_response_probability` (float, >0, <= 1) indicates the probability by which a response will be augmented with random peers.
- `max_random_peers` (int, >0) sets an upper boundary (inclusive) for the amount of peers added.
- `prefix` (string, 20 characters at most) sets the prefix for generated peer IDs.
    The peer ID will be padded to 20 bytes using a random string of alphanumeric characters.
- `min_port` (int, >0, <=65535) sets a lower boundary for the port for generated peers.
- `max_port` (int, >0, <=65536, > `min_port`) sets an upper boundary for the port for generated peers.

An example config might look like this:

    chihaya:
      tracker:
        announce_middleware:
          - name: deniability
            config:
              modify_response_probability: 0.2
              max_random_peers: 5
              prefix: -AZ2060-
              min_port: 40000
              max_port: 60000

For more information about peer IDs and their prefixes, see [this wiki entry](https://wiki.theory.org/BitTorrentSpecification#peer_id).