# Implementing a Driver

The [`backend`] package is meant to provide announce deltas to a slower and more consistent database, such as the one powering a torrent-indexing website. Implementing a backend driver is heavily inspired by the standard library's [`database/sql`] package: simply create a package that implements the [`backend.Driver`] and [`backend.Conn`] interfaces and calls [`backend.Register`] in it's [`init()`]. Please note that [`backend.Conn`] must be thread-safe. A great place to start is to read the [`no-op`] driver which comes out-of-the-box with Chihaya and is meant to be used for public trackers.

[`init()`]: http://golang.org/ref/spec#Program_execution
[`database/sql`]: http://godoc.org/database/sql
[`backend`]: http://godoc.org/github.com/chihaya/chihaya/backend
[`backend.Register`]: http://godoc.org/github.com/chihaya/chihaya/backend#Register
[`backend.Driver`]: http://godoc.org/github.com/chihaya/chihaya/backend#Driver
[`backend.Conn`]: http://godoc.org/github.com/chihaya/chihaya/backend#Conn
[`no-op`]: http://godoc.org/github.com/chihaya/chihaya/backend/noop

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
