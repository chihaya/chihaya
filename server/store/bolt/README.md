## `bolt` Store Implementations

This package provides implementations of the `store` APIs using [bolt].
They register with the store under the name `bolt`.
Currently only the `StringStore` is implemented.

[bolt]: https://github.com/boltdb/bolt

### Configuration

Bolt uses a single database file on disk, it is therefore necessary to specify which file a driver should use.
Note that every driver opens a new `bolt.DB`, which means that they cannot share the same file.

A typical configuration for the `StringStore` would look like this:

```yaml
string_store:
  name: bolt
  config:
    file: strings.db
```
