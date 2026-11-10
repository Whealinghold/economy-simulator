package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// liveServer runs serve on a loopback listener the test opened itself on port 0, so
// there is no port to guess and no clash, and returns the base URL and a function that
// stops serve and returns its result. The kernel queues connections made while serve
// still solves the world, so a request needs no start-up poll and the test no sleep.
func liveServer(t *testing.T, seed uint64, state string) (base string, stop func() error) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() = %v, want nil", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serve(ctx, discardLogger(), l, seed, 0, state) }()
	var stopped bool
	stop = func() error {
		cancel()
		select {
		case err := <-done:
			stopped = true
			return err
		case <-time.After(60 * time.Second):
			t.Fatalf("serve did not return within 60 s of the stop")
			return nil
		}
	}
	t.Cleanup(func() {
		if !stopped {
			cancel()
			<-done
		}
		client.CloseIdleConnections()
	})
	return "http://" + l.Addr().String(), stop
}

// client has a timeout, so a server that stops answering fails the test instead of
// hanging it. The longest call is the 1 000-tick step, which is stopped long before.
var client = &http.Client{Timeout: 3 * time.Minute}

// do sends a request with the real client and returns the status and body.
func do(t *testing.T, method, url, body string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest(%s %s) = %v, want nil", method, url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("%s %s: read body: %v", method, url, err)
	}
	return resp.StatusCode, b
}

func tickOf(t *testing.T, b []byte) int64 {
	t.Helper()
	var v tickBody
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("decode %q: %v", b, err)
	}
	return v.Tick
}

// TestServe_LoopbackAddPlayerStepCheckpointRestoreAndShutdown verifies the whole
// server as a client sees it, over a real listener: a foreign Host is refused, the
// player is added, the world steps, a checkpoint is saved to the state file (the
// bytes answered equal the file, no temporary file stays), more steps run, a restore
// of the saved bytes puts the world back at that tick with the player and a new
// checkpoint equal to the saved one, and a stop makes serve return nil and close
// the port.
//
// Requirement: FR-INT-027
// Guard: architecture.md 1.3 and 12.3. serve, the allow-list, the atomic save and
// the clean stop are only wired together here; a unit test of each cannot see a
// wiring that drops one of them.
func TestServe_LoopbackAddPlayerStepCheckpointRestoreAndShutdown(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	dir := t.TempDir()
	state := filepath.Join(dir, "realm.ckpt")
	base, stop := liveServer(t, 21, state)

	req, _ := http.NewRequest(http.MethodGet, base+"/v1/world", nil)
	req.Host = "evil.example"
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET with a foreign Host: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("GET /v1/world with Host evil.example = %d, want 403", resp.StatusCode)
	}

	if code, b := do(t, http.MethodPost, base+"/v1/player", playerJSON); code != http.StatusCreated {
		t.Fatalf("POST /v1/player = %d %s, want 201", code, b)
	}
	if code, b := do(t, http.MethodPost, base+"/v1/player/directive", `{"bids":[{"commodity":"grain","quantity":1,"limit":1000}]}`); code != http.StatusOK && code != http.StatusBadRequest {
		t.Fatalf("POST /v1/player/directive = %d %s, want 200 or a 400 refusal", code, b)
	}
	code, b := do(t, http.MethodPost, base+"/v1/step?n=2", "")
	if code != http.StatusOK || tickOf(t, b) != 2 {
		t.Fatalf("POST /v1/step?n=2 = %d %s, want 200 at tick 2", code, b)
	}
	code, saved := do(t, http.MethodPost, base+"/v1/checkpoint?save=true", "")
	if code != http.StatusOK {
		t.Fatalf("POST /v1/checkpoint?save=true = %d, want 200", code)
	}
	if file, err := os.ReadFile(state); err != nil || !bytes.Equal(file, saved) {
		t.Errorf("state file: %d bytes, error %v, want the %d bytes answered", len(file), err, len(saved))
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("state directory holds %d entries after a save, want only the state file", len(entries))
	}
	if code, b = do(t, http.MethodPost, base+"/v1/step", ""); code != http.StatusOK || tickOf(t, b) != 3 {
		t.Fatalf("POST /v1/step = %d %s, want 200 at tick 3", code, b)
	}
	code, b = do(t, http.MethodPost, base+"/v1/restore", string(saved))
	if code != http.StatusOK || tickOf(t, b) != 2 || !strings.Contains(string(b), `"player":true`) {
		t.Fatalf("POST /v1/restore = %d %s, want 200 at tick 2 with the player", code, b)
	}
	if _, again := do(t, http.MethodPost, base+"/v1/checkpoint", ""); !bytes.Equal(again, saved) {
		t.Errorf("checkpoint after the restore differs from the saved one (%d vs %d bytes), want identical", len(again), len(saved))
	}
	if code, b = do(t, http.MethodPost, base+"/v1/restore", "not a checkpoint"); code != http.StatusBadRequest {
		t.Errorf("POST /v1/restore with garbage = %d %s, want 400", code, b)
	}

	addr := strings.TrimPrefix(base, "http://")
	if err := stop(); err != nil {
		t.Errorf("serve after the stop = %v, want nil", err)
	}
	if c, err := net.DialTimeout("tcp", addr, time.Second); err == nil {
		_ = c.Close()
		t.Errorf("port %s still accepts connections after the stop, want it closed", addr)
	}
}

// TestServe_ShutdownStopsAnInFlightStepAtATickBoundary verifies the drain: a 1 000-tick
// step is running when the process is told to stop; the step answers 503 ErrCancelled
// at a tick boundary instead of running to the end, the new-step token is seen taken
// meanwhile (429), and serve returns nil.
//
// Guard: architecture.md 12.3. Without BaseContext tied to the stop context the
// Shutdown would wait out the full step (about a minute) and then hit its 5 s limit.
func TestServe_ShutdownStopsAnInFlightStepAtATickBoundary(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	base, stop := liveServer(t, 22, "")
	type result struct {
		code int
		body []byte
	}
	got := make(chan result, 1)
	go func() {
		resp, err := client.Post(base+"/v1/step?n=1000", "application/json", nil)
		if err != nil {
			got <- result{-1, []byte(err.Error())}
			return
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		got <- result{resp.StatusCode, b}
	}()
	// A restore of garbage answers 429 only while the step holds the token (else 400,
	// changing nothing): that is the signal the step has started. Each probe is a round
	// trip, so the loop needs no sleep; it ends when the step is running or the
	// client timeout of 3 minutes fails a probe.
	for probing := true; probing; {
		select {
		case r := <-got:
			t.Fatalf("the long step answered %d %s before the stop", r.code, r.body)
		default:
		}
		code, _ := do(t, http.MethodPost, base+"/v1/restore", "x")
		probing = code != http.StatusTooManyRequests
	}
	start := time.Now()
	if err := stop(); err != nil {
		t.Errorf("serve after the stop = %v, want nil", err)
	}
	select {
	case r := <-got:
		if r.code != http.StatusServiceUnavailable || !strings.Contains(string(r.body), "ErrCancelled") {
			t.Errorf("in-flight step = %d %s, want 503 ErrCancelled", r.code, r.body)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("the in-flight step has no answer %v after the stop", time.Since(start))
	}
}

// TestServe_RefusesAnUnreadableStateFile verifies that serve returns the error and
// closes the listener it was given when the -state file is not a checkpoint.
//
// Guard: starting fresh over it would overwrite the file at the next save.
func TestServe_RefusesAnUnreadableStateFile(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	state := filepath.Join(t.TempDir(), "realm.ckpt")
	if err := os.WriteFile(state, []byte("not a checkpoint"), 0o600); err != nil {
		t.Fatalf("WriteFile() = %v, want nil", err)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() = %v, want nil", err)
	}
	err = serve(context.Background(), discardLogger(), l, 1, 0, state)
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		t.Errorf("serve over a state file that is no checkpoint = %v, want an error", err)
	}
}
