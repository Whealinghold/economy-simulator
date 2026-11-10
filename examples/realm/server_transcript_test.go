package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"testing"
)

// transcriptScript is a fixed session: look at the world, add the player, bid for
// grain, step, read the result, shock a harvest and a road, refuse a few requests.
var transcriptScript = []struct{ method, target, body string }{
	{"GET", "/v1/world", ""},
	{"GET", "/v1/market?region=crownlands&commodity=grain", ""},
	{"POST", "/v1/player", playerJSON},
	{"GET", "/v1/player", ""},
	{"POST", "/v1/step?n=3", ""},
	{"POST", "/v1/player/directive", `{"intents":[{"commodity":"grain","side":"bid","quantity":1,"limit":5000}]}`},
	{"POST", "/v1/step", ""},
	{"GET", "/v1/events", ""},
	{"GET", "/v1/player", ""},
	{"POST", "/v1/shock/harvest", `{"shocks":[{"region":"highmarch","recipe":"grow_grain","bp":2500}]}`},
	{"POST", "/v1/shock/road", `{"roads":[{"from":"crownlands","to":"highmarch","closed":true}]}`},
	{"GET", "/v1/world", ""},
	{"POST", "/v1/checkpoint", ""},
	{"GET", "/v1/market?region=atlantis&commodity=grain", ""},
	{"POST", "/v1/step?n=1001", ""},
	{"POST", "/v1/restore", "not a checkpoint"},
}

// TestServer_Transcript verifies that a fixed script of requests against the handler
// gives the same responses every time: the responses, concatenated, equal
// testdata/transcript.golden. The checkpoint is recorded by its size and tick, as it
// is binary.
//
// Guard: the example is deterministic (architecture.md 7.4). A route that read the clock, iterated a map
// into a response or leaked an address would show as a diff here. The file is
// rewritten only with -update, after the cause of a difference is understood
// (testing.md 7.2).
func TestServer_Transcript(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	s := newTestServer(t, 1, "")
	var out bytes.Buffer
	for _, step := range transcriptScript {
		fmt.Fprintf(&out, "> %s %s\n", step.method, step.target)
		if step.body != "" {
			fmt.Fprintf(&out, "> %s\n", step.body)
		}
		code, b := call(t, s, step.method, step.target, step.body)
		if step.target == "/v1/checkpoint" {
			fmt.Fprintf(&out, "< %d %d bytes\n\n", code, len(b))
			continue
		}
		fmt.Fprintf(&out, "< %d\n%s\n", code, b)
	}
	const golden = "testdata/transcript.golden"
	if *update {
		if err := os.WriteFile(golden, out.Bytes(), 0o644); err != nil {
			t.Fatalf("write %s: %v", golden, err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read %s: %v (run with -update to record it)", golden, err)
	}
	if !bytes.Equal(out.Bytes(), want) {
		t.Errorf("transcript differs from %s (%d and %d bytes); first difference at byte %d", golden, out.Len(), len(want), firstDiff(out.Bytes(), want))
	}
	if code, _ := call(t, s, "GET", "/v1/world", ""); code != http.StatusOK {
		t.Errorf("GET /v1/world after the script = %d, want 200", code)
	}
}

// firstDiff returns the index of the first byte where a and b differ.
func firstDiff(a, b []byte) int {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}
