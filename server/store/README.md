## The store Package

The `store` package offers a storage interface and middlewares sufficient to run a public tracker based on it.

### Architecture

The store consists of three parts:
- A set of interfaces, tests based on these interfaces and the store logic, unifying these interfaces into the store
- Drivers, implementing the store interfaces and
- Middleware that depends on the store

The store interfaces are `IPStore`, `PeerStore` and `StringStore`.
During runtime, each of them will be implemented by a driver.
Even though all different drivers for one interface provide the same functionality, their behaviour can be very different.
For example: The memory implementation keeps all state in-memory - this is very fast, but not persistent, it loses its state on every restart.
A database-backed driver on the other hand could provide persistence, at the cost of performance.

The pluggable design of Chihaya allows for the different interfaces to use different drivers.
For example: A typical use case of the `StringStore` is to provide blacklists or whitelists for infohashes/client IDs/....
You'd typically want these lists to be persistent, so you'd choose a driver that provides persistence.
The `PeerStore` on the other hand rarely needs to be persistent, as all peer state will be restored after one announce interval.
You'd therefore typically choose a very performant but non-persistent driver for the `PeerStore`.

### Testing

The main store package also contains a set of tests and benchmarks for drivers.
Both use the store interfaces and can work with any driver that implements these interfaces.
The tests verify that the driver behaves as specified by the interface and its documentation.
The benchmarks can be used to compare performance of a wide range of operations on the interfaces.

This makes it very easy to implement a new driver:
All functions that are part of the store interfaces can be tested easily with the tests that come with the store package.
Generally the memory implementation can be used as a guideline for implementing new drivers.

Both benchmarks and tests require a clean state to work correctly.
All of the test and benchmark functions therefore take a `*DriverConfig` as a parameter, this should be used to configure the driver in a way that it provides a clean state for every test or benchmark.
For example: Imagine a file-based driver that achieves persistence by storing its state in a file.
It must then be possible to provide the location of this file in the `'DriverConfig`, so that every different benchmark gets to work with a new file.

Most benchmarks come in two flavors: The "normal" version and the "1K" version.
A normal benchmark uses the same value over and over again to benchmark one operation.
A 1K benchmark uses a different value from a set of 1000 values for every iteration, this can show caching effects, if the driver uses them.
The 1K benchmarks require a little more computation to select the values and thus typically yield slightly lower results even for a "perfect" cache, i.e. the memory implementation.
