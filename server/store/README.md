## The store Package

The `store` package offers a storage interface and middlewares sufficient to run a public tracker based on it.
Additionally, a modular API that exposes the store interface via HTTP is provided.

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

### The API

Every API response returns a JSON encoded `response`:

```json
{
    "ok":false,
    "error":"<nil>/<a string describing the error>",
    "result":{}
}
```

The `ok` field indicates whether the call was executed successfully.
If `ok=false`, the `error` field will contain a string explaining the error that occurred during the execution.
The `result` field is independent of the value `ok` has and contains a result of the call performed.

The HTTP API is composed of two parts:
- A set of predefined endpoints
- Endpoints defined by middlewares

#### Access restriction

The easiest way to restrict access to the API is by setting the address to bind to localhost.
If it is however desired to expose the API and still have access restriction, the store config allows to set the parameter `api_key`.
If non-empty, the same key will be expected for every request made.
It must be provided in at least one of these two ways:

- The value of the `X-API-Key` HTTP header
- A query parameter called `apikey`

A mismatch will immediately fail the request and return a 403 status code.

#### Types

- `ContainedResult` is returned to indicate whether or not the requested resource is contained in the store:

    ```json
    {
        "contained":false
    }
    ```

- `CountResult` is returned when the number of elements in a collection is requested:

    ```json
    {
        "count":1
    }
    ```

- `PeersResult` contains two sets of `Peer`s: one for IPv4 and one for IPv6:

    ```json
    {
        "peers4":[
            {
                "id":"00112233445566778899aabbccddeeff00112233",
                "ip":"10.12.14.16",
                "port":12345
            }
        ],
        "peers6":[]
    }
    ```

- `Peer` is a JSON encoding of a single peer:

    ```json
    {
        "id":"00112233445566778899aabbccddeeff00112233",
        "ip":"10.12.14.16",
        "port":12345
    }
    ```

    This type is used both as a result and as a parameter to some endpoints.  
    All `Peer`s returned by the API have their ID encoded as a hexadecimal string.
    `Peer`s that are used as a parameter can have their ID encoded as either a hexadecimal string, a base32 string or just a 20-byte JSON string.

- `DualStackedPeer` is a type encapsulating two `Peer`s: one for IPv4 and one for IPv6:

    ```json
    {
        "peer4":{
            "id":"00112233445566778899aabbccddeeff00112233",
            "ip":"10.12.14.16",
            "port":12345
        },
        "peer6":{
            "id":"112233445566778899aabbccddeeff0011223344",
            "ip":"6464:aAbB:1234::6464",
            "port":54321
        }
    }
    ```

    This type is used as a parameter.

#### Predefined Endpoints

- `PUT /ips/:ip` takes the given IPv4 or IPv6 address and adds it to the IP store.  
    An example of this would be `PUT /ips/1.2.3.4`, which would add the IPv4 address `1.2.3.4` to the IP store.  
    This method does not return a result.

- `DELETE /ips/:ip` takes the given IPv4 or IPv6 address and attempts to delete it from the store.
    If the IP address is not contained, an error and an HTTP code 404 will be returned.  
    It is important to note that networks and single addresses are generally distinct and never interact with each other, except when matching against the IP store.
    That means that, if the store contains a network `n`, which contains an IP address `a`, attempting to delete `a` will fail, because the IP store does not contain the single address `a`.  
    An example of this would be `DELETE /ips/1.2.3.4`, which would attempt to delete the IPv4 address `1.2.3.4` from the string store.  
    This method does not return a result.

- `GET /ips/:ip` takes the given IPv4 or IPv6 address and matches it against the store.  
    It is important to note that the address will be matched against both single-address entries and entire networks.  
    This method returns a `ContainedResult`.

- `PUT /networks/:network` takes the given network in CIDR notation and adds it to the IP store.  
    An example of this would be `PUT /networks/1.2.3.4/24` which would add the network `1.2.3.0 - 1.2.3.255` to the IP store.  
    This method does not return a result.

- `DELETE /networks/:network` takes the given network in CIDR notation and attempts to delete it from the IP store.
    If the network is not contained, an error and an HTTP code 404 will be returned.  
    It is important to note, that this will _never_ remove single IP addresses from a larger network.
    If you added the address `1.2.3.4` previously, deleting the network `1.2.3.4/24` will return an error.  
    An example of this would be `DELETE /networks/1.2.3.4/24` which would attempt to delete the network `1.2.3.0 - 1.2.3.255` from the IP store.  
    This method does not return a result.

- `PUT /strings/:string` takes the provided string, URLEscapes it and stores it in the string store.  
    It is important to note, that middlewares that depend on the string store often register API endpoints to make the interaction with the store easier.  
    An example of this would be `PUT /strings/someString`, which would add `someString` to the string store.  
    This method does not return a result.

- `DELETE /strings/:string` takes the provided string, URLEscapes it and attempts to delete it from the string store.
    If the string is not contained, an error and an HTTP code 404 will be returned.  
    An example of this would be `DELETE /strings/someString`, which would attempt to delete `someString` from the string store.  
    This method does not return a result.

- `GET /strings/:string` takes the provided string, URLEscapes it and matches it against the store.  
    This method returns a `ContainedResult`.

- `GET /peers/seeders/:infohash` returns all seeders for the given infohash.  
    The infohash can be provided as a hexadecimal string, a base32 string or URLEncoded.  
    This method returns a `PeersResult`.

- `PUT /peers/seeders/:infohash` adds or replaces a given `Peer` as a seeder to the swarm for the given infohash.  
    The infohash can be provided as a hexadecimal string, a base32 string or URLEncoded.
    The `Peer` is to be provided as a JSON object in the body of the request.  
    This method does not return a result.

- `DELETE /peers/seeders/:infohash` deletes a given `Peer` from the set of seeders of the swarm for the given infohash.  
    The infohash can be provided as a hexadecimal string, a base32 string or URLEncoded.
    The `Peer` is to be provided as a JSON object in the body of the request.  
    This method does not return a result.

- `COUNT /peers/seeders/:infohash` or `GET /peers/numSeeders/:infohash` returns the number of seeders in the swarm for the given infohash.  
    The infohash can be provided as a hexadecimal string, a base32 string or URLEncoded.  
    This method returns a `CountResult`.

- `GET /peers/leechers/:infohash` returns all leechers for the given infohash.  
    The infohash can be provided as a hexadecimal string, a base32 string or URLEncoded.  
    This method returns a `PeersResult`.

- `PUT /peers/leechers/:infohash` adds or replaces a given `Peer` as a leecher to the swarm for the given infohash.  
    The infohash can be provided as a hexadecimal string, a base32 string or URLEncoded.
    The `Peer` is to be provided as a JSON object in the body of the request.  
    This method does not return a result.

- `DELETE /peers/leechers/:infohash` deletes a given `Peer` from the set of leechers of the swarm for the given infohash.  
    The infohash can be provided as a hexadecimal string, a base32 string or URLEncoded.
    The `Peer` is to be provided as a JSON object in the body of the request.  
    This method does not return a result.

- `COUNT /peers/leechers/:infohash` or `GET /peers/numLeechers/:infohash` returns the number of leechers in the swarm for the given infohash.  
    The infohash can be provided as a hexadecimal string, a base32 string or URLEncoded.  
    This method returns a `CountResult`.

- `POST /peers/graduateLeecher/:infohash` graduates a given `Peer` from a leecher to a seeder in the swarm for the given infohash.  
    The infohash can be provided as a hexadecimal string, a base32 string or URLEncoded.  
    The `Peer` is to be provided as a JSON object in the body of the request.  
    This method does not return a result.

- `GET /peers/announce/:infohash?seeder=<true/false>` returns a set of peers from the swarm for the given infohash.  
    The infohash can be provided as a hexadecimal string, a base32 string or URLEncoded.  
    The `seeder` parameter describes whether the caller is a seeder or a leecher in the swarm.
    This method takes a `DualStackedPeer` parameter, to be JSON encoded in the body of the request.  
    This method returns a `PeersResult`.


#### Endpoints Defined By Middlewares

The store offers a set of methods to _register_ and _activate_ endpoints for the API.
A package that offers a middleware that depends on the store should, if it's desired, register API methods to interact with the store in a way that is fitting to the middleware.
In the middleware constructors of that package, these methods should be activated _once_.

A boilerplate for the whole process would look like this:

```go
package middleware

import (
    "net/http"
    
    "github.com/chihaya/chihaya/server/store"
    "github.com/chihaya/chihaya/tracker"
)

func init() {
    tracker.RegisterAnnounceMiddleware("a", a)

    store.RegisterNoResponseHandler(http.MethodPut, pathPutSomething, handlePutSomething)
    store.RegisterHandler(http.MethodDelete, pathDeleteSomething, handleDeleteSomething)
}

const pathPutSomething = "/some/:thing"
const pathDeleteSomething = "/some/:thing"

var routesActivated sync.Once

func activateRoutes() {
    store.ActivateRoute(http.MethodPut, pathPutSomething)
    store.ActivateRoute(http.MethodDelete, pathDeleteSomething)
}

func a(next tracker.AnnounceHandler) tracker.AnnounceHandler {
    routesActivated.Do(activateRoutes)

    return func(cfg *chihaya.TrackerConfig, req *chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) error {
        // Do middleware stuff before the rest of the chain.
        return next(cfg, req, resp)
        // Do middleware stuff after the rest of the chain.
    }
}

func handlePutSomething(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
    // handle API request
    return http.StatusOK, nil
}

func handleDeleteSomething(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, interface{}, error) {
    // handle API request and return a result or nil
    return http.StatusOK, nil, nil
}
```

This allows the API endpoints to be dependent on the config, which loads the middlewares, instead of package imports.

All handler functions registered with the store will be wrapped in a recovery handler and a logging handler.
It is idiomatic to panic in case of an internal error.
The panic will be logged and an HTTP status 500, an appropriate error and no result will be returned to the caller.
If on the other hand an error is returned, that error and the status code will be returned as-is to the caller.
