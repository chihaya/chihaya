## Swarm Interaction Middleware

This package provides the announce middleware that modifies peer data stored in the `store` package.

### `store_swarm_interaction`

The `store_swarm_interaction` middleware updates the data stored in the `peerStore` based on the announce.

### Important things to notice

It is recommended to have this middleware run before the `store_response` middleware.
The `store_response` middleware assumes the store to be already updated by the announce.