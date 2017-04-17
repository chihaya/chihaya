### Terminology

- swarm: the entire group of peers using the tracker
- client: the physical torrent client software running on a computer
- peer: a single client operating on a single torrent
- user: a single logical source of peers, usually correlated with a single client
- database: the primary source of site data; the canonical source of statistics
- cache: the shared memory of all instances of the tracker, usually a key-value store of some kind
- passkey: a user's API key, which identifies them and authenticates them simultaneously
- peer_id: a randomly generated identifier from the torrent client, prefixed by the client's "user agent"
- info_hash: the 20-byte SHA-1 hash of the data within a torrent file

### Data

BitTorrent trackers provide a very specific function: store peers and distribute data about those peers to other peers. Thus, the only data "owned" by a tracker is the set of peers active in its swarm. However, in the case of a private, authenticated swarm like Chihaya provides, it is necessary for the tracker to also know about a set of users and torrents that are controlled by another entity, and for statistics to be sent back to that controlling entity.

Since there is a clear separation of the source of truth for each of these types of data, Chihaya makes them distinct internally. All interaction with the peer set is done through the `peers` package, and all interaction with the user and torrent sets is done through the `storage` package (naming tbd).

The issue arises that the primary storage of users and torrents—often a SQL database—is rather expensive to read from. The solution is obviously to use a cache, but doing so presents the problem of how to handle invalidating the cache efficiently when there is such a high volume of writes. However, due to the specifics of the data being read and written, the tracker can cheat a little:

| Type      | Data read                        | Data written                 |
|:--------- |:-------------------------------- |:---------------------------- |
| Peers     | connection info, usage stats     | connection info, usage stats |
| Users     | passkey, statistic multipliers   | usage stats                  |
| Torrents  | info_hash, statistic multipliers | usage stats                  |

Since the tracker never actually needs to read the data it writes to users and torrents, it ends up being safe to write directly to the primary storage and read directly out of the cache without actually caring about updating both simultaneously. Thus, storage drivers don't need to worry about making sure that subsequent reads will contain data written by the tracker—i.e. it doesn't have to care about cache invalidation **(not strictly correct, we need to discuss this)**. It also means that there is only one way to load any particular resource, and that usage of any cache is completely left up to the storage driver.

----------------

#### Database storage

Information from the database is considered to be slow-changing:

passkey → User
 - ID
 - Statistic multipliers

info_hash → Torrent
 - ID
 - Status
 - Statistic multipliers

#### Tracker data

peer_id → Peer (stored per torrent, since peer_ids are unique for each peer for a torrent)
 - User/Torrent IDs
 - Statistic totals (to calculate deltas, as the protocol specifies the client should send totals)
 - Timestamps (for pruning inactive peers)
 - Connection info