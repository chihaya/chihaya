## The store Package

The `store` package offers a storage interface and middlewares sufficient to run a public tracker based on it.
Additionally, a modular API that exposes the store interface via HTTP is provided.

### Store Drivers/Architecture

TODO(mrd0ll4r): fill this in

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


#### Results

- `ContainedResult` is returned to indicate whether or not the requested resource is contained in the store:

```json
{
    "contained":false
}
```


#### Predefined Endpoints

- `PUT /ips/:ip` takes the given IPv4 or IPv6 address and adds it to the IP store.  
    An example of this would be `PUT /ip/1.2.3.4`, which would add the IPv4 address `1.2.3.4` to the IP store.  
    This method does not return a result.

- `DELETE /ips/:ip` takes the given IPv4 or IPv6 address and attempts to delete it from the store.
    If the IP address is not contained, an error and an HTTP code 404 will be returned.  
    It is important to note that networks and single addresses are generally distinct and never interact with each other, except when matching against the IP store.
    That means that, if the store contains a network `n`, which contains an IP address `a`, attempting to delete `a` will fail, because the IP store does not contain the single address `a`.  
    An example of this would be `DELETE /ip/1.2.3.4`, which would attempt to delete the IPv4 address `1.2.3.4` from the string store.  
    This method does not return a result.

- `GET /ips/:ip` takes the given IPv4 or IPv6 address and matches it against the store.  
    It is important to note that the address will be matched against both single-address entries and entire networks.  
    This method returns a `ContainedResult`.

- `PUT /networks/:network` takes the given network in CIDR notation and adds it to the IP store.  
    An example of this would be `PUT /network/1.2.3.4/24` which would add the network `1.2.3.0 - 1.2.3.255` to the IP store.  
    This method does not return a result.

- `DELETE /networks/:network` takes the given network in CIDR notation and attempts to delete it from the IP store.
    If the network is not contained, an error and an HTTP code 404 will be returned.  
    It is important to note, that this will _never_ remove single IP addresses from a larger network.
    If you added the address `1.2.3.4` previously, deleting the network `1.2.3.4/24` will return an error.  
    An example of this would be `DELETE /network/1.2.3.4/24` which would attempt to delete the network `1.2.3.0 - 1.2.3.255` from the IP store.  
    This method does not return a result.

- `PUT /strings/:string` takes the provided string, URLEscapes it and stores it in the string store.  
    It is important to note, that middlewares that depend on the string store often register API endpoints to make the interaction with the store easier.  
    An example of this would be `PUT /string/someString`, which would add `someString` to the string store.  
    This method does not return a result.

- `DELETE /strings/:string` takes the provided string, URLEscapes it and attempts to delete it from the string store.
    If the string is not contained, an error and an HTTP code 404 will be returned.  
    An example of this would be `DELETE /string/someString`, which would attempt to delete `someString` from the string store.  
    This method does not return a result.

- `GET /strings/:string` takes the provided string, URLEscapes it and matches it against the store.  
    This method returns a `ContainedResult`.


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





