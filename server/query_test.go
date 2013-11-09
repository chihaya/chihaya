package server

import (
	"net/url"
	"testing"
)

var (
	baseAddr     = "https://www.subdomain.tracker.com:80/"
	testInfoHash = "01234567890123456789"
	testPeerId   = "-TEST01-6wfG2wk6wWLc"

	ValidAnnounceArguments = []url.Values{
		url.Values{"info_hash": {testInfoHash}, "peer_id": {testPeerId}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}},
		url.Values{"info_hash": {testInfoHash}, "peer_id": {testPeerId}, "ip": {"192.168.0.1"}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}},
		url.Values{"info_hash": {testInfoHash}, "peer_id": {testPeerId}, "ip": {"192.168.0.1"}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}, "numwant": {"28"}},
		url.Values{"info_hash": {testInfoHash}, "peer_id": {testPeerId}, "ip": {"192.168.0.1"}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}, "event": {"stopped"}},
		url.Values{"info_hash": {testInfoHash}, "peer_id": {testPeerId}, "ip": {"192.168.0.1"}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}, "event": {"started"}, "numwant": {"13"}},
		url.Values{"info_hash": {testInfoHash}, "peer_id": {testPeerId}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}, "no_peer_id": {"1"}},
		url.Values{"info_hash": {testInfoHash}, "peer_id": {testPeerId}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}, "compact": {"0"}, "no_peer_id": {"1"}},
		url.Values{"info_hash": {testInfoHash}, "peer_id": {testPeerId}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}, "compact": {"0"}, "no_peer_id": {"1"}, "key": {"peerKey"}},
		url.Values{"info_hash": {testInfoHash}, "peer_id": {testPeerId}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}, "compact": {"0"}, "no_peer_id": {"1"}, "key": {"peerKey"}, "trackerid": {"trackerId"}},
		url.Values{"info_hash": {testInfoHash}, "peer_id": {"%3Ckey%3A+0x90%3E"}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}, "compact": {"0"}, "no_peer_id": {"1"}, "key": {"peerKey"}, "trackerid": {"trackerId"}},
		url.Values{"info_hash": {testInfoHash}, "peer_id": {"%3Ckey%3A+0x90%3E"}, "compact": {"1"}},
	}

	InvalidQueries = []string{
		baseAddr + "announce/?" + "info_hash=%0%a",
	}
)

func mapArrayEqual(boxed map[string][]string, unboxed map[string]string) bool {
	if len(boxed) != len(unboxed) {
		return false
	}
	for mapKey, mapVal := range boxed {
		// Always expect box to hold only one element
		if len(mapVal) != 1 {
			return false
		}
		if ub_mapVal, eleExists := unboxed[mapKey]; !eleExists || mapVal[0] != ub_mapVal {
			return false
		}
	}
	return true
}

func TestValidQueries(t *testing.T) {
	for parseIndex, parseVal := range ValidAnnounceArguments {
		parsedQueryObj, err := parseQuery(baseAddr + "announce/?" + parseVal.Encode())
		if err != nil {
			t.Error(err)
		}
		if !mapArrayEqual(parseVal, parsedQueryObj.Params) {
			t.Errorf("Incorrect parse at item %d.\n Expected=%v\n Recieved=%v\n", parseIndex, parseVal, parsedQueryObj.Params)
		}
	}
}

func TestInvalidQueries(t *testing.T) {
	for parseIndex, parseStr := range InvalidQueries {
		parsedQueryObj, err := parseQuery(parseStr)
		if err == nil {
			t.Error("Should have produced error ", parseIndex)
		}
		if parsedQueryObj != nil {
			t.Error("Should be nil after error ", parsedQueryObj, parseIndex)
		}
	}
}

func BenchmarkParseQuery(b *testing.B) {
	for bCount := 0; bCount < b.N; bCount++ {
		for parseIndex, parseStr := range ValidAnnounceArguments {
			parsedQueryObj, err := parseQuery(baseAddr + "announce/?" + parseStr.Encode())
			if err != nil {
				b.Error(err, parseIndex)
				b.Log(parsedQueryObj)
			}
		}
	}
}

func BenchmarkURLParseQuery(b *testing.B) {
	for bCount := 0; bCount < b.N; bCount++ {
		for parseIndex, parseStr := range ValidAnnounceArguments {
			parsedQueryObj, err := url.ParseQuery(baseAddr + "announce/?" + parseStr.Encode())
			if err != nil {
				b.Error(err, parseIndex)
				b.Log(parsedQueryObj)
			}
		}
	}
}
