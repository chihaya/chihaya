## Client Blacklisting/Whitelisting Middlewares

This package provides the announce middlewares `client_whitelist` and `client_blacklist` for blacklisting or whitelisting clients for announces.

### `client_blacklist`

The `client_blacklist` middleware uses all clientIDs stored in the `StringStore` to blacklist, i.e. block announces.

The clientID part of the peerID of an announce is matched against the `StringStore`, if it's contained within the `StringStore`, the announce is aborted.

### `client_whitelist`

The `client_whitelist` middleware uses all clientIDs stored in the `StringStore` to whitelist, i.e. allow announces.

The clientID part of the peerID of an announce is matched against the `StringStore`, if it's _not_ contained within the `StringStore`, the announce is aborted.

### Important things to notice

Both middlewares operate on announce requests only.

Both middlewares use the same `StringStore`.
It is therefore not advised to have both the `client_blacklist` and the `client_whitelist` middleware running.
(If you add clientID to the `StringStore`, it will be used for blacklisting and whitelisting.
If your store contains no clientIDs, no announces will be blocked by the blacklist, but all announces will be blocked by the whitelist.
If your store contains all clientIDs, no announces will be blocked by the whitelist, but all announces will be blocked by the blacklist.)