package bittorrent

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/chihaya/chihaya/pkg/log"
)

// Params is used to fetch (optional) request parameters from an Announce.
// For HTTP Announces this includes the request path and parsed query, for UDP
// Announces this is the extracted path and parsed query from optional URLData
// as specified in BEP41.
//
// See ParseURLData for specifics on parsing and limitations.
type Params interface {
	// String returns a string parsed from a query. Every key can be
	// returned as a string because they are encoded in the URL as strings.
	String(key string) (string, bool)

	// RawPath returns the raw path from the request URL.
	// The path returned can contain URL encoded data.
	// For a request of the form "/announce?port=1234" this would return
	// "/announce".
	RawPath() string

	// RawQuery returns the raw query from the request URL, excluding the
	// delimiter '?'.
	// For a request of the form "/announce?port=1234" this would return
	// "port=1234"
	RawQuery() string
}

// ErrKeyNotFound is returned when a provided key has no value associated with
// it.
var ErrKeyNotFound = errors.New("query: value for the provided key does not exist")

// ErrInvalidInfohash is returned when parsing a query encounters an infohash
// with invalid length.
var ErrInvalidInfohash = ClientError("provided invalid infohash")

// ErrInvalidQueryEscape is returned when a query string contains invalid
// escapes.
var ErrInvalidQueryEscape = ClientError("invalid query escape")

// QueryParams parses a URL Query and implements the Params interface with some
// additional helpers.
type QueryParams struct {
	path       string
	query      string
	params     map[string]string
	infoHashes []InfoHash
}

type routeParamsKey struct{}

// RouteParamsKey is a key for the context of a request that
// contains the named parameters from the http router.
var RouteParamsKey = routeParamsKey{}

// RouteParam is a type that contains the values from the named parameters
// on the route.
type RouteParam struct {
	Key   string
	Value string
}

// RouteParams is a collection of RouteParam instances.
type RouteParams []RouteParam

// ByName returns the value of the first RouteParam that matches the given
// name. If no matching RouteParam is found, an empty string is returned.
// In the event that a "catch-all" parameter is provided on the route and
// no value is matched, an empty string is returned. For example: a route of
// "/announce/*param" matches on "/announce/". However, ByName("param") will
// return an empty string.
func (rp RouteParams) ByName(name string) string {
	for _, p := range rp {
		if p.Key == name {
			return p.Value
		}
	}
	return ""
}

// ParseURLData parses a request URL or UDP URLData as defined in BEP41.
// It expects a concatenated string of the request's path and query parts as
// defined in RFC 3986. As both the udp: and http: scheme used by BitTorrent
// include an authority part the path part must always begin with a slash.
// An example of the expected URLData would be "/announce?port=1234&uploaded=0"
// or "/?auth=0x1337".
// HTTP servers should pass (*http.Request).RequestURI, UDP servers should
// pass the concatenated, unchanged URLData as defined in BEP41.
//
// Note that, in the case of a key occurring multiple times in the query, only
// the last value for that key is kept.
// The only exception to this rule is the key "info_hash" which will attempt to
// parse each value as an InfoHash and return an error if parsing fails. All
// InfoHashes are collected and can later be retrieved by calling the InfoHashes
// method.
//
// Also note that any error that is encountered during parsing is returned as a
// ClientError, as this method is expected to be used to parse client-provided
// data.
func ParseURLData(urlData string) (*QueryParams, error) {
	var path, query string

	queryDelim := strings.IndexAny(urlData, "?")
	if queryDelim == -1 {
		path = urlData
	} else {
		path = urlData[:queryDelim]
		query = urlData[queryDelim+1:]
	}

	q, err := parseQuery(query)
	if err != nil {
		return nil, ClientError(err.Error())
	}
	q.path = path
	return q, nil
}

// parseQuery parses a URL query into QueryParams.
// The query is expected to exclude the delimiting '?'.
func parseQuery(query string) (q *QueryParams, err error) {
	// This is basically url.parseQuery, but with a map[string]string
	// instead of map[string][]string for the values.
	q = &QueryParams{
		query:      query,
		infoHashes: nil,
		params:     make(map[string]string),
	}

	for query != "" {
		key := query
		if i := strings.IndexAny(key, "&;"); i >= 0 {
			key, query = key[:i], key[i+1:]
		} else {
			query = ""
		}
		if key == "" {
			continue
		}
		value := ""
		if i := strings.Index(key, "="); i >= 0 {
			key, value = key[:i], key[i+1:]
		}
		key, err = url.QueryUnescape(key)
		if err != nil {
			// QueryUnescape returns an error like "invalid escape: '%x'".
			// But frontends record these errors to prometheus, which generates
			// a lot of time series.
			// We log it here for debugging instead.
			log.Debug("failed to unescape query param key", log.Err(err))
			return nil, ErrInvalidQueryEscape
		}
		value, err = url.QueryUnescape(value)
		if err != nil {
			// QueryUnescape returns an error like "invalid escape: '%x'".
			// But frontends record these errors to prometheus, which generates
			// a lot of time series.
			// We log it here for debugging instead.
			log.Debug("failed to unescape query param value", log.Err(err))
			return nil, ErrInvalidQueryEscape
		}

		if key == "info_hash" {
			if len(value) != 20 {
				return nil, ErrInvalidInfohash
			}
			q.infoHashes = append(q.infoHashes, InfoHashFromString(value))
		} else {
			q.params[strings.ToLower(key)] = value
		}
	}

	return q, nil
}

// String returns a string parsed from a query. Every key can be returned as a
// string because they are encoded in the URL as strings.
func (qp *QueryParams) String(key string) (string, bool) {
	value, ok := qp.params[key]
	return value, ok
}

// Uint64 returns a uint parsed from a query. After being called, it is safe to
// cast the uint64 to your desired length.
func (qp *QueryParams) Uint64(key string) (uint64, error) {
	str, exists := qp.params[key]
	if !exists {
		return 0, ErrKeyNotFound
	}

	val, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return 0, err
	}

	return val, nil
}

// InfoHashes returns a list of requested infohashes.
func (qp *QueryParams) InfoHashes() []InfoHash {
	return qp.infoHashes
}

// RawPath returns the raw path from the parsed URL.
func (qp *QueryParams) RawPath() string {
	return qp.path
}

// RawQuery returns the raw query from the parsed URL.
func (qp *QueryParams) RawQuery() string {
	return qp.query
}
