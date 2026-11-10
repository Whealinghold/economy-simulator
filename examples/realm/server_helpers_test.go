package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// newTestServer builds a server over the shared configuration. state is the -state
// path ("" for none).
func newTestServer(t testing.TB, seed uint64, state string) *Server {
	t.Helper()
	s, err := buildServer(seed, state)
	if err != nil {
		t.Fatalf("newServer(seed %d) = %v, want nil", seed, err)
	}
	return s
}

func buildServer(seed uint64, state string) (*Server, error) {
	cfg, err := packOnce()
	if err != nil {
		return nil, err
	}
	return newServer(cfg, seed, state, nil)
}

var sharedOnce = sync.OnceValues(func() (*Server, error) { return buildServer(1, "") })

// sharedServer is one server for the tests that only read or are refused, so they
// do not each build a world. A test that steps, adds the player or shocks the world
// builds its own.
func sharedServer(t testing.TB) *Server {
	t.Helper()
	s, err := sharedOnce()
	if err != nil {
		t.Fatalf("shared server: %v", err)
	}
	return s
}

// call sends one request to the handler and returns the status and the body.
func call(t testing.TB, h http.Handler, method, target, body string) (int, []byte) {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	b, err := io.ReadAll(rec.Result().Body)
	if err != nil {
		t.Fatalf("read the response of %s %s: %v", method, target, err)
	}
	return rec.Code, b
}

// callJSON is call that decodes the body into v and fails on a status other than
// want.
func callJSON(t testing.TB, h http.Handler, method, target, body string, want int, v any) {
	t.Helper()
	code, b := call(t, h, method, target, body)
	if code != want {
		t.Fatalf("%s %s = status %d (%s), want %d", method, target, code, bytes.TrimSpace(b), want)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("%s %s: decode %q: %v", method, target, b, err)
	}
}

// discardLogger is a logger that drops everything.
func discardLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }
