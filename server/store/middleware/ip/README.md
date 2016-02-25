## IP Blacklisting/Whitelisting Middlewares

This package provides the announce middlewares `IPBlacklist` and `IPWhitelist` for blacklisting or whitelisting IP addresses and networks for announces.

### `IPBlacklist`

The `IPBlacklist` middleware uses all IP addresses and networks stored in the `IPStore` to blacklist, i.e. block announces.

Both the IPv4 and the IPv6 addresses contained in the announce are matched against the `IPStore`.
If one or both of the two are contained in the `IPStore`, the announce will be rejected _completely_.

### `IPWhitelist`

The `IPWhitelist` middleware uses all IP addresses and networks stored in the `IPStore` to whitelist, i.e. allow announces.

If present, both the IPv4 and the IPv6 addresses contained in the announce are matched against the `IPStore`.
Only if all IP address that are present in the announce are also present in the `IPStore` will the announce be allowed, otherwise it will be rejected _completely_.

### Important things to notice

Both middlewares operate on announce requests only.
The middlewares will check the IPv4 and IPv6 IPs a client announces to the tracker against an `IPStore`.
Normally the IP address embedded in the announce is the public IP address of the machine the client is running on.
Note however, that a client can override this behaviour by specifying an IP address in the announce itself.
_This middleware does not (dis)allow announces coming from certain IP addresses, but announces containing certain IP addresses_.
Always keep that in mind.

Both middlewares use the same `IPStore`.
It is therefore not advised to have both the `IPBlacklist` and the `IPWhitelist` middleware running.
(If you add an IP address or network to the `IPStore`, it will be used for blacklisting and whitelisting.
If your store contains no addresses, no announces will be blocked by the blacklist, but all announces will be blocked by the whitelist.
If your store contains all addresses, no announces will be blocked by the whitelist, but all announces will be blocked by the blacklist.)