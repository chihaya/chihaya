package bittorrent

import (
	"net/url"
	"testing"
)

var (
	testPeerID = "-TEST01-6wfG2wk6wWLc"

	ValidAnnounceArguments = []url.Values{
		{},
		{"peer_id": {testPeerID}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}},
		{"peer_id": {testPeerID}, "ip": {"192.168.0.1"}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}},
		{"peer_id": {testPeerID}, "ip": {"192.168.0.1"}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}, "numwant": {"28"}},
		{"peer_id": {testPeerID}, "ip": {"192.168.0.1"}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}, "event": {"stopped"}},
		{"peer_id": {testPeerID}, "ip": {"192.168.0.1"}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}, "event": {"started"}, "numwant": {"13"}},
		{"peer_id": {testPeerID}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}, "no_peer_id": {"1"}},
		{"peer_id": {testPeerID}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}, "compact": {"0"}, "no_peer_id": {"1"}},
		{"peer_id": {testPeerID}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}, "compact": {"0"}, "no_peer_id": {"1"}, "key": {"peerKey"}},
		{"peer_id": {testPeerID}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}, "compact": {"0"}, "no_peer_id": {"1"}, "key": {"peerKey"}, "trackerid": {"trackerId"}},
		{"peer_id": {"%3Ckey%3A+0x90%3E"}, "port": {"6881"}, "downloaded": {"1234"}, "left": {"4321"}, "compact": {"0"}, "no_peer_id": {"1"}, "key": {"peerKey"}, "trackerid": {"trackerId"}},
		{"peer_id": {"%3Ckey%3A+0x90%3E"}, "compact": {"1"}},
		{"peer_id": {""}, "compact": {""}},
	}

	InvalidQueries = []string{
		"/announce?" + "info_hash=%0%a",
	}

	// See https://github.com/ProtocolONE/chihaya/issues/334.
	shouldNotPanicQueries = []string{
		"/annnounce?" + "info_hash=" + testPeerID + "&a",
		"/annnounce?" + "info_hash=" + testPeerID + "&=b?",
	}
)

func mapArrayEqual(boxed map[string][]string, unboxed map[string]string) bool {
	if len(boxed) != len(unboxed) {
		return false
	}

	for mapKey, mapVal := range boxed {
		// Always expect box to hold only one element
		if len(mapVal) != 1 || mapVal[0] != unboxed[mapKey] {
			return false
		}
	}

	return true
}

func TestParseEmptyURLData(t *testing.T) {
	parsedQuery, err := ParseURLData("")
	if err != nil {
		t.Fatal(err)
	}
	if parsedQuery == nil {
		t.Fatal("Parsed query must not be nil")
	}
}

func TestParseValidURLData(t *testing.T) {
	for parseIndex, parseVal := range ValidAnnounceArguments {
		parsedQueryObj, err := ParseURLData("/announce?" + parseVal.Encode())
		if err != nil {
			t.Fatal(err)
		}

		if !mapArrayEqual(parseVal, parsedQueryObj.params) {
			t.Fatalf("Incorrect parse at item %d.\n Expected=%v\n Received=%v\n", parseIndex, parseVal, parsedQueryObj.params)
		}

		if parsedQueryObj.path != "/announce" {
			t.Fatalf("Incorrect path, expected %q, got %q", "/announce", parsedQueryObj.path)
		}
	}
}

func TestParseInvalidURLData(t *testing.T) {
	for parseIndex, parseStr := range InvalidQueries {
		parsedQueryObj, err := ParseURLData(parseStr)
		if err == nil {
			t.Fatal("Should have produced error", parseIndex)
		}

		if parsedQueryObj != nil {
			t.Fatal("Should be nil after error", parsedQueryObj, parseIndex)
		}
	}
}

func TestParseShouldNotPanicURLData(t *testing.T) {
	for _, parseStr := range shouldNotPanicQueries {
		ParseURLData(parseStr)
	}
}

func BenchmarkParseQuery(b *testing.B) {
	announceStrings := make([]string, 0)
	for i := range ValidAnnounceArguments {
		announceStrings = append(announceStrings, ValidAnnounceArguments[i].Encode())
	}
	b.ResetTimer()
	for bCount := 0; bCount < b.N; bCount++ {
		i := bCount % len(announceStrings)
		parsedQueryObj, err := parseQuery(announceStrings[i])
		if err != nil {
			b.Error(err, i)
			b.Log(parsedQueryObj)
		}
	}
}

func BenchmarkURLParseQuery(b *testing.B) {
	announceStrings := make([]string, 0)
	for i := range ValidAnnounceArguments {
		announceStrings = append(announceStrings, ValidAnnounceArguments[i].Encode())
	}
	b.ResetTimer()
	for bCount := 0; bCount < b.N; bCount++ {
		i := bCount % len(announceStrings)
		parsedQueryObj, err := url.ParseQuery(announceStrings[i])
		if err != nil {
			b.Error(err, i)
			b.Log(parsedQueryObj)
		}
	}
}
