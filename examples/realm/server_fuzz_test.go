package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

// fuzzServer is the one server of a fuzzing process. Its step cap is 1: a valid
// /v1/step then costs one tick, and the cap itself is exercised by a request above it
// (the same code as the cap of 1 000, which TestServer_BadInputIsRefused checks).
var fuzzServer = sync.OnceValues(func() (*Server, error) {
	s, err := buildServer(1, "")
	if s != nil {
		s.maxStep = 1
	}
	return s, err
})

// FuzzServerRequests verifies that the router never panics and never answers 5xx for
// any method, path, query and body, that a refused request leaves the tick as it was,
// and that a JSON route answers valid JSON. The requests are built by hand, not by
// httptest.NewRequest, which panics on a method or target that is not valid HTTP:
// the server must meet those too.
//
// Requirement: FR-INT-002, FR-CHK-016
// Guard: the server takes untrusted input. The files of testdata/fuzz/ pin a step
// count past the cap, a restore of bytes with a header and no checkpoint, and a JSON
// body followed by more data; none crashed, they are the shapes a regression would
// bring back. A crash found later is added with a Fix: line naming it.
func FuzzServerRequests(f *testing.F) {
	player := `{"region":"crownlands","role":"traveller","money":5000,"goods":{"grain":10}}`
	f.Add("GET", "/v1/world", "", []byte(nil))
	f.Add("GET", "/v1/market", "region=crownlands&commodity=grain", []byte(nil))
	f.Add("GET", "/v1/region", "region=highmarch", []byte(nil))
	f.Add("GET", "/v1/events", "since=3", []byte(nil))
	f.Add("GET", "/v1/events", "since=-1", []byte(nil))
	f.Add("POST", "/v1/player", "", []byte(player))
	f.Add("POST", "/v1/player", "", []byte(`{"region":"atlantis"}`))
	f.Add("POST", "/v1/player/directive", "", []byte(`{"intents":[{"commodity":"grain","side":"bid","quantity":1,"limit":50}]}`))
	f.Add("POST", "/v1/player/directive", "", []byte(`{"intents":[{"commodity":"grain","side":"ask","quantity":-1,"limit":0}],"job":{"recipe":"bake"}}`))
	f.Add("POST", "/v1/shock/harvest", "", []byte(`{"shocks":[{"region":"highmarch","recipe":"grow_grain","bp":0}]}`))
	f.Add("POST", "/v1/shock/road", "", []byte(`{"roads":[{"from":"crownlands","to":"highmarch","closed":true,"factor_bp":99999999999}]}`))
	f.Add("POST", "/v1/step", "n=1", []byte(nil))
	f.Add("POST", "/v1/step", "n=1001", []byte(nil))
	f.Add("POST", "/v1/step", "n=-1", []byte(nil))
	f.Add("POST", "/v1/checkpoint", "save=true", []byte(nil))
	f.Add("POST", "/v1/restore", "", append(make([]byte, headerLen), "garbage"...))
	f.Add("BREW", "/v1/../v1//world", "%zz", []byte("x"))
	f.Fuzz(func(t *testing.T, method, path, query string, body []byte) {
		s, err := fuzzServer()
		if err != nil {
			t.Fatalf("build the server: %v", err)
		}
		before := s.game.Tick()
		req := &http.Request{
			Method: method, URL: &url.URL{Path: path, RawQuery: query}, Header: http.Header{},
			Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body)),
		}
		req.RequestURI = req.URL.RequestURI()
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		if rec.Code >= http.StatusInternalServerError {
			t.Fatalf("%s %q ? %q (%d body bytes) = status %d (%s), want below 500", method, path, query, len(body), rec.Code, bytes.TrimSpace(rec.Body.Bytes()))
		}
		if rec.Code >= http.StatusBadRequest && s.game.Tick() != before {
			t.Errorf("%s %q ? %q: status %d moved the tick from %d to %d, want a refused request to change nothing", method, path, query, rec.Code, before, s.game.Tick())
		}
		if strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") && !json.Valid(rec.Body.Bytes()) {
			t.Errorf("%s %q ? %q: body %q is not valid JSON", method, path, query, rec.Body.Bytes())
		}
	})
}
