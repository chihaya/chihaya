# Frontends

A _Frontend_ is a component of Chihaya that serves a BitTorrent tracker on one protocol.
The frontend accepts, parses and sanitizes requests, passes them to the _Logic_ and writes responses to _Clients_.

This documentation first gives a high-level overview of Frontends and later goes into implementation specifics.
Users of Chihaya are expected to just read the first part - developers should read both.

## Functionality

A Frontend serves one protocol, for example HTTP ([BEP 3]) or UDP ([BEP 15]).
It listens for requests and usually answers each of them with one response, a basic overview of the control flow is:

1. Read the request.
2. Parse the request.
3. Have the Logic handle the request. This calls a series of `PreHooks`.
4. Send a response to the Client.
5. Process the request and response through `PostHooks`.

## Available Frontends

Chihaya ships with frontends for HTTP(S) and UDP.
The HTTP frontend uses Go's `http` package.
The UDP frontend implements [opentracker-style] IPv6, contrary to the specification in [BEP 15].

## Implementing a Frontend

This part is intended for developers.

### Implementation Specifics

A frontend should serve only one protocol.
It may serve that protocol on multiple transports or networks, if applicable.
An example of that is the `http` Frontend, operating both on HTTP and HTTPS.

The typical control flow of handling announces, in more detail, is:

1. Read the request.
2. Parse the request, if invalid go to 9.
3. Validate/sanitize the request, if invalid go to 9.
4. If the request is protocol-specific, handle, respond, and go to 8.
5. Pass the request to the `TrackerLogic`'s `HandleAnnounce` or `HandleScrape` method, if an error is returned go to 9.
6. Send the response to the Client.
7. Pass the request and response to the `TrackerLogic`'s `AfterAnnounce` or `AfterScrape` method.
8. Finish, accept next request.
9. For invalid requests or errors during processing: Send an error response to the client. 
    This step may be skipped for suspected denial-of-service attacks.
    The error response may contain information about the cause of the error.
    Only errors where the Client is at fault should be explained, internal server errors should be returned without explanation. 
    Then finish, and accept the next request.

#### Configuration

The frontend must be configurable using a single, exported struct.
The struct must have YAML annotations.
The struct must implement `log.Fielder` to be logged on startup.

#### Metrics

Frontends may provide runtime metrics, such as the number of requests or their duration.
Metrics must be reported using [Prometheus].

A frontend should provide at least the following metrics:
- The number of valid and invalid requests handled
- The average time it takes to handle a single request.
    This request timing should be made optional using a config entry.

Requests should be separated by type, i.e. Scrapes, Announces, and other protocol-specific requests.
If the frontend serves multiple transports or networks, metrics for them should be separable.

It is recommended to publish one Prometheus `HistogramVec` with:
- A name like `chihaya_PROTOCOL_response_duration_milliseconds`
- A value holding the duration in milliseconds of the reported request
- Labels for:
    - `action` (= `announce`, `scrape`, ...)
    - `address_family` (= `Unknown`, `IPv4`, `IPv6`, ...), if applicable
     - `error` (= A textual representation of the error encountered during processing.)
    Because `error` is expected to hold the textual representation of any error that occurred during the request, great care must be taken to ensure all error messages are static.
    `error` must not contain any information directly taken from the request, e.g. the value of an invalid parameter.
    This would cause this dimension of prometheus to explode, which slows down prometheus clients and reporters.

#### Error Handling

Frontends should return `bittorrent.ClientError`s to the Client.
Frontends must not return errors that are not a `bittorrent.ClientError` to the Client.
A message like `internal server error` should be used instead.

#### Request Sanitization

The `TrackerLogic` expects sanitized requests in order to function properly.

The `bittorrent` package provides the `SanitizeAnnounce` and `SanitizeScrape` functions to sanitize Announces and Scrapes, respectively.
This is the minimal required sanitization, every `AnnounceRequest` and `ScrapeRequest` must be sanitized this way.

Note that the `AnnounceRequest` struct contains booleans of the form `XProvided`, where `X` denotes an optional parameter of the BitTorrent protocol.
These should be set according to the values received by the Client.

#### Contexts

All methods of the `TrackerLogic` interface expect a `context.Context` as a parameter.
After a request is handled by `HandleAnnounce` without errors, the populated context returned must be used to call `AfterAnnounce`.
The same applies to Scrapes.
This way, a PreHook can communicate with a PostHook by setting a context value.

[BEP 3]: http://bittorrent.org/beps/bep_0003.html
[BEP 15]: http://bittorrent.org/beps/bep_0015.html
[Prometheus]: https://prometheus.io/
[opentracker-style]: http://opentracker.blog.h3q.com/2007/12/28/the-ipv6-situation/