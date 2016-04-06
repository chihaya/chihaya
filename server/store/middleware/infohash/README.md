## Infohash Blacklisting/Whitelisting Middlewares

This package provides the middleware `infohash_blacklist` and `infohash_whitelist` for blacklisting or whitelisting infohashes.
It also provides the configurable scrape middleware `infohash_blacklist` and `infohash_whitelist` for blacklisting or whitelisting infohashes.

### `infohash_blacklist`

#### For Announces

The `infohash_blacklist` middleware uses all infohashes stored in the `StringStore` with the `PrefixInfohash` prefix to blacklist, i.e. block announces.

#### For Scrapes

The configurable `infohash_blacklist` middleware uses all infohashes stored in the `StringStore` with the `PrefixInfohash` prefix to blacklist scrape requests.

The scrape middleware has two modes of operation: _Block_ and _Filter_.

- _Block_ will drop a scrape request if it contains a blacklisted infohash.
- _Filter_ will filter all blacklisted infohashes from a scrape request, potentially leaving behind an empty scrape request.
    **IMPORTANT**: This mode **does not work with UDP servers**.

See the configuration section for information about how to configure the scrape middleware.

### `infohash_whitelist`

#### For Announces

The `infohash_blacklist` middleware uses all infohashes stored in the `StringStore` with the `PrefixInfohash` prefix to whitelist, i.e. allow announces.

#### For Scrapes

The configurable `infohash_blacklist` middleware uses all infohashes stored in the `StringStore` with the `PrefixInfohash` prefix to whitelist scrape requests.

The scrape middleware has two modes of operation: _Block_ and _Filter_.

- _Block_ will drop a scrape request if it contains a non-whitelisted infohash.
- _Filter_ will filter all non-whitelisted infohashes from a scrape request, potentially leaving behind an empty scrape request.
    **IMPORTANT**: This mode **does not work with UDP servers**.

See the configuration section for information about how to configure the scrape middleware.

### Important things to notice

Both blacklist and whitelist middleware use the same `StringStore`.
It is therefore not advised to have both the `infohash_blacklist` and the `infohash_whitelist` announce or scrape middleware running.
(If you add an infohash to the `StringStore`, it will be used for blacklisting and whitelisting.
If your store contains no infohashes, no announces/scrapes will be blocked by the blacklist, but all will be blocked by the whitelist.
If your store contains all addresses, no announces/scrapes will be blocked by the whitelist, but all will be blocked by the blacklist.)

Also note that the announce and scrape middleware both use the same `StringStore`.
It is therefore not possible to use different infohashes for black-/whitelisting on announces and scrape requests.

### Configuration

The scrape middleware is configurable.

The configuration uses a single required parameter `mode` to determine the mode of operation for the middleware.
An example configuration might look like this:

    chihaya:
      tracker:
        scrape_middleware:
          - name: infohash_blacklist
            config:
              mode: block

`mode` accepts two values: `block` and `filter`.

**IMPORTANT**: The `filter` mode **does not work with UDP servers**.