package server

import (
	"bytes"
	"github.com/kotokoko/chihaya/bufferpool"
	"github.com/kotokoko/chihaya/database"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"testing"
	"time"
)

var testUser database.User

//requires db to be correctly setup and for config.json to be valid
var testHandler = &httpHandler{db: &database.Database{}, startTime: time.Now()}
var dbInit = false

//works around db access
var passkey = "23456789123456789123456789123456"

//used for comparison benchmarking
var parseRequestURLTests = []struct {
	url           string
	expectedValid bool
}{
	{"http://foo.com", true},
	{"http://foo.com/", true},
	{"http://foo.com/path", true},
	{"/", true},
	{"//not.a.user@not.a.host/just/a/path", true},
	{"//not.a.user@%66%6f%6f.com/just/a/path/also", true},
	{"foo.html", true},
	{"../dir/", true},
	{"*", true},
	{"93 h23m\n32\"4&(^#%%  ", true},
	{"http:///foot.com/%MYhorse/;%HT&%  everywhere/(", true},
	{"", true},
}

func BenchmarkURLParse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, testURI := range parseRequestURLTests {
			url.Parse(testURI.url)
		}
	}
}

func BenchmarkURLParseReq(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, testURI := range parseRequestURLTests {
			url.ParseRequestURI(testURI.url)
		}
	}
}

func BenchmarkShortParseQuery(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, testURI := range parseRequestURLTests {
			parseQuery(testURI.url)
		}
	}
}

//parseQuery isn't expected to ever return any unique errors because the errors are generated from
// url.QueryUnescape
func TestShortParseQuery(t *testing.T) {
	for _, testURI := range parseRequestURLTests {
		_, errV := parseQuery(testURI.url)
		valid := errV == nil
		if valid != testURI.expectedValid {
			t.Errorf("Query validity mismatch. %v for %q != %v",
				testURI.expectedValid, testURI.url, valid)
		}
	}
}

//Some of the following tests are taken from url_test.go
// but with slightly different functionality
// *returns the last key value pair in the case of duplicates instead of the first
func TestQueryValues(t *testing.T) {
	qp, _ := parseQuery("http://x.com?foo=bar&bar=1&bar=2")
	v := qp.params
	if len(v) != 2 {
		t.Errorf("got %d keys in Query values, want 2", len(v))
	}
	if g, e := v["foo"], "bar"; g != e {
		t.Logf("v=%v", v)
		t.Errorf("Get(foo) = %q, want %q", g, e)
	}
	// Case sensitive:
	if g, e := v["Foo"], ""; g != e {
		t.Errorf("Get(Foo) = %q, want %q", g, e)
	}
	if g, e := v["bar"], "2"; g != e {
		t.Errorf("Get(bar) = %q, want %q", g, e)
	}
	if g, e := v["baz"], ""; g != e {
		t.Errorf("Get(baz) = %q, want %q", g, e)
	}
	delete(v, "bar")
	if g, e := v["bar"], ""; g != e {
		t.Errorf("second Get(bar) = %q, want %q", g, e)
	}
}

type parseTest struct {
	query string
	out   map[string]string
}

var parseTests = []parseTest{
	{
		query: "a=1&b=2",
		out:   map[string]string{"a": "1", "b": "2"},
	},
	{
		query: "a=1&a=2&a=banana",
		out:   map[string]string{"a": "banana"},
	},
	{
		query: "ascii=%3Ckey%3A+0x90%3E",
		out:   map[string]string{"ascii": "<key: 0x90>"},
	},
	{
		query: "a=1;b=2",
		out:   map[string]string{"a": "1", "b": "2"},
	},
	{
		query: "a=1&a=2;a=banana",
		out:   map[string]string{"a": "banana"},
	},
}

//Again, no support duplicate parameters
// last one in wins.
func TestParseQuery(t *testing.T) {
	for i, test := range parseTests {
		qa, err := parseQuery(test.query)
		form := qa.params
		if err != nil {
			t.Errorf("test %d: Unexpected error: %v", i, err)
			continue
		}
		if len(form) != len(test.out) {
			t.Errorf("test %d: len(form) = %d, want %d", i, len(form), len(test.out))
		}
		//This loop can also test for duplicate parsings in a multi-map,
		// parseQuery does not support duplicate parameters
		for k, evs := range test.out {
			vs, ok := form[k]
			if !ok {
				t.Errorf("test %d: Missing key %q", i, k)
				continue
			}
			if len(vs) != len(evs) {
				t.Errorf("test %d: len(form[%q]) = %d, want %d", i, k, len(vs), len(evs))
				continue
			}
			for j, ev := range evs {
				if v := vs[j]; rune(v) != ev {
					t.Errorf("test %d: form[%q][%d] = %q, want %q", i, k, j, v, ev)
				}
			}
		}
	}
}

const invalidPasskeyErrorStr = "passkey is invalid"
const missingPasskeyErrorStr = "Passkey not found"
const ipAddrErrorStr = "Failed to parse IP address"
const badRequestErrorStr = "Unknown action"

var respondErrorTests = []struct {
	url    string
	errStr string
}{
	{"http://bing.com", invalidPasskeyErrorStr},
	{"http://www.bing.com/stats", invalidPasskeyErrorStr},
	{"http://www.bing.com/8765431987643219876543219876543/error", invalidPasskeyErrorStr},
	{"http://www.bing.com/098765431987643219876543219876543/error", invalidPasskeyErrorStr},
	{"http://www.bing.com/98765431987643219876543219876543/error", missingPasskeyErrorStr},
	{"http://www.bing.com/02345678912345678912345678912345/error", missingPasskeyErrorStr},
	{"http://www.bing.com/2345678912345678912345678912345O/error", missingPasskeyErrorStr},
	{"http://www.bing.com/23456789123456789123456789123456/error", ipAddrErrorStr},
	{"http://www.bing.com/23456789123456789123456789123456/error&ipv6=2607:f0d0:1002:0051:0000:0000:0000:0004", ipAddrErrorStr},
	{"http://www.bing.com/23456789123456789123456789123456/error&ipv6=2607:f0d0:1002:51::4", ipAddrErrorStr},
	{"http://www.bing.com/23456789123456789123456789123456/error&ip=127.0.0.1", badRequestErrorStr},
	{"http://www.bing.com/23456789123456789123456789123456/error&ipv4=127.0.0.1", badRequestErrorStr},
	{"http://www.bing.com/23456789123456789123456789123456/error&ip=2607:f0d0:1002:51::4", badRequestErrorStr},
	{"http://www.bing.com/23456789123456789123456789123456/error&ip=2607:f0d0:1002:0051:0000:0000:0000:0004", badRequestErrorStr},
}

func TestRespondErrors(t *testing.T) {
	if !dbInit {
		testHandler.db.Init()
		dbInit = true
	}
	testHandler.db.Users[passkey] = &testUser
	usr, ex := testHandler.db.Users[passkey]
	if !ex {
		t.Errorf("Cannot mock user(%s) passkey %v", usr, passkey)
	}
	for _, reqText := range respondErrorTests {
		var resultBuf bytes.Buffer
		testReq, reqErr := http.NewRequest("GET", reqText.url, nil)
		if reqErr != nil {
			t.Errorf("http.NewRequest error=%v", reqErr)
		}
		d, _ := path.Split(testReq.URL.Path)
		testReq.URL.RawQuery = testReq.URL.Path
		params, pqe := parseQuery(testReq.URL.RawQuery)
		if pqe != nil {
			t.Fatal("ParseQuery error during respond test")
		}
		testHandler.respond(testReq, &resultBuf)
		if !strings.Contains(resultBuf.String(), reqText.errStr) {
			t.Logf("testReq.URL=%v", testReq.URL)
			t.Logf("URL.RawQuery=%v", testReq.URL.RawQuery)
			t.Logf("key=%v", d[1:33])
			t.Logf("params=%v", params)
			t.Errorf("Uncaught error: %s. Got %s instead", reqText.errStr, resultBuf.String())
		}
	}
}

func TestServeHTTP(t *testing.T) {
	testWriter := httptest.NewRecorder()
	if !dbInit {
		testHandler.db.Init()
		dbInit = true
	}
	testHandler.bufferPool = bufferpool.New(5, 5)
	testReq, _ := http.NewRequest("GET", "www.bing.com/stats", nil)
	testHandler.terminate = true
	testHandler.ServeHTTP(testWriter, testReq)
	if testWriter.Header().Get("Connection") != "" {
		t.Error("HTTP server not terminated")
	}

	testHandler.terminate = false
	testHandler.ServeHTTP(testWriter, testReq)
	if conHeader := testWriter.Header().Get("Connection"); conHeader != "close" {
		t.Errorf("Unexpected HTTP header value. Connection=%v", conHeader)
	}
	if strings.Contains(testWriter.Body.String(), "Uptime:") {
		t.Errorf("Unexpected HTTP response=%v", testWriter.Body)
	}
	testReq.URL.Path = "/stats"
	testHandler.ServeHTTP(testWriter, testReq)
	if !strings.Contains(testWriter.Body.String(), "Uptime:") {
		t.Errorf("Unexpected HTTP response=%v", testWriter.Body)
	}
	if !testWriter.Flushed {
		t.Error("HTTP response not flushed!")
	}
	if (testWriter.Code / 100) != 2 {
		t.Errorf("HTTP responce not OK. Code=%v", testWriter.Code)
	}
}

func BenchmarkRespondErrors_BestCase(b *testing.B) {
	b.StopTimer()
	if !dbInit {
		testHandler.db.Init()
		dbInit = true
	}
	testHandler.db.Users[passkey] = &testUser
	usr, ex := testHandler.db.Users[passkey]
	if !ex {
		b.Fatalf("Cannot mock user(%s) passkey %v", usr, passkey)
	}
	var resultBuf bytes.Buffer
	for i := 0; i < b.N; i++ {
		for j := 0; j < len(respondErrorTests); j++ {
			testReq, _ := http.NewRequest("GET", respondErrorTests[0].url, nil)
			b.StartTimer()
			testHandler.respond(testReq, &resultBuf)
			b.StopTimer()
		}
	}
}

func BenchmarkRespondErrors_WorstCase(b *testing.B) {
	b.StopTimer()
	if !dbInit {
		testHandler.db.Init()
		dbInit = true
	}
	testHandler.db.Users[passkey] = &testUser
	usr, ex := testHandler.db.Users[passkey]
	if !ex {
		b.Fatalf("Cannot mock user(%s) passkey %v", usr, passkey)
	}
	var resultBuf bytes.Buffer
	for i := 0; i < b.N; i++ {
		for j := 0; j < len(respondErrorTests); j++ {
			testReq, _ := http.NewRequest("GET", respondErrorTests[6].url, nil)
			b.StartTimer()
			testHandler.respond(testReq, &resultBuf)
			b.StopTimer()
		}
	}
}

func BenchmarkRespondErrors_All(b *testing.B) {
	b.StopTimer()
	if !dbInit {
		testHandler.db.Init()
		dbInit = true
	}
	testHandler.db.Users[passkey] = &testUser
	usr, ex := testHandler.db.Users[passkey]
	if !ex {
		b.Fatalf("Cannot mock user(%s) passkey %v", usr, passkey)
	}
	var resultBuf bytes.Buffer
	for i := 0; i < b.N; i++ {
		for _, reqText := range respondErrorTests {
			testReq, _ := http.NewRequest("GET", reqText.url, nil)
			b.StartTimer()
			testHandler.respond(testReq, &resultBuf)
			b.StopTimer()
		}
	}
}
