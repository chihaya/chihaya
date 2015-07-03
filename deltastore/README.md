# Implementing a Driver

The `deltastore` package is meant to provide announce deltas to a slower and more consistent database, such as the one powering a torrent-indexing website. Implementing a driver is heavily inspired by the standard library's [`database/sql`] package: simply create a package that implements the `Driver` and `Conn` interfaces and calls `Register` in an [`init()`]. Please note that `Conn` must be thread-safe. A great place to start is to read the `nop` driver which comes out-of-the-box with Chihaya and is meant to be used for public trackers.

[`init()`]: http://golang.org/ref/spec#Program_execution
[`database/sql`]: http://godoc.org/database/sql

## Creating a binary with your own driver

Chihaya is designed to be extended. If you require more than the drivers provided out-of-the-box, you are free to create your own and then produce your own custom Chihaya binary. To create this binary, simply create your own main package, import your custom drivers, then call [`chihaya.Boot`] from main.

[`chihaya.Boot`]: http://godoc.org/github.com/chihaya/chihaya

### Example

```go
package main

import (
	"github.com/chihaya/chihaya"

  // Import any of your own drivers.
	_ "github.com/yourusername/chihaya-custom-backend"
)

func main() {
  // Start Chihaya normally.
	chihaya.Boot()
}
```
