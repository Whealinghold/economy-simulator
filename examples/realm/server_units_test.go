package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestNameAt_FallsBackToTheDecimalID verifies the name lookup: an ID the pack lists
// gives its name, and one it does not (negative or past the end) gives the decimal ID.
//
// Guard: a response must never fail on a lookup. If nameAt indexed unchecked, a world
// that returned an ID the pack lacks would panic the handler instead of showing "7".
func TestNameAt_FallsBackToTheDecimalID(t *testing.T) {
	t.Parallel()
	list := []string{"a", "b"}
	tests := []struct {
		name string
		i    int
		want string
	}{
		{"first", 0, "a"},
		{"last", 1, "b"},
		{"past the end", 2, "2"},
		{"negative", -1, "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := nameAt(list, tt.i); got != tt.want {
				t.Errorf("nameAt(%v, %d) = %q, want %q", list, tt.i, got, tt.want)
			}
		})
	}
	if got := nameAt(nil, 0); got != "0" {
		t.Errorf("nameAt(nil, 0) = %q, want %q", got, "0")
	}
}

// TestClip_CutsAtMaxMessage verifies that text up to maxMessage bytes is kept whole
// and longer text is cut to exactly maxMessage bytes.
//
// Guard: error messages echo client-sent names; without the cut a long name inflates
// every error reply and log line.
func TestClip_CutsAtMaxMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want int
	}{
		{"empty", "", 0},
		{"at the limit", strings.Repeat("x", maxMessage), maxMessage},
		{"one over", strings.Repeat("x", maxMessage+1), maxMessage},
		{"far over", strings.Repeat("x", 10*maxMessage), maxMessage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := len(clip(tt.in)); got != tt.want {
				t.Errorf("len(clip(%d bytes)) = %d, want %d", len(tt.in), got, tt.want)
			}
		})
	}
}

// TestHostname_StripsPortAndCase verifies the Host comparison key: port removed,
// brackets of an IPv6 literal removed, lower case.
//
// Guard: the Host allow-list defends against DNS rebinding; a key that kept the port
// or the case would refuse a legitimate client or accept a spoofed spelling.
func TestHostname_StripsPortAndCase(t *testing.T) {
	t.Parallel()
	tests := []struct{ in, want string }{
		{"localhost", "localhost"},
		{"LocalHost:8080", "localhost"},
		{"127.0.0.1:1", "127.0.0.1"},
		{"[::1]:8080", "::1"},
		{"[::1]", "::1"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := hostname(tt.in); got != tt.want {
				t.Errorf("hostname(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestFail_ErrorTable verifies the error table: every sentinel of refusals, also
// wrapped, answers its own status and name, and an error the table does not list
// answers 422 ErrRefused.
//
// Guard: the table is the HTTP contract of every refusal. A reordered or dropped row
// would turn a 404 or 429 into a 422 for clients that branch on the status.
func TestFail_ErrorTable(t *testing.T) {
	t.Parallel()
	s := &Server{log: nil}
	s.log = discardLogger()
	for _, r := range refusals {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			s.fail(rec, fmt.Errorf("wrapped: %w", r.err))
			if rec.Code != r.status {
				t.Errorf("fail(%v) status = %d, want %d", r.err, rec.Code, r.status)
			}
			if !bytes.Contains(rec.Body.Bytes(), []byte(`"error":"`+r.name+`"`)) {
				t.Errorf("fail(%v) body = %s, want error name %q", r.err, rec.Body.Bytes(), r.name)
			}
		})
	}
	t.Run("unlisted refusal", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		s.fail(rec, errors.New("something else"))
		if rec.Code != http.StatusUnprocessableEntity || !bytes.Contains(rec.Body.Bytes(), []byte(`"ErrRefused"`)) {
			t.Errorf("fail(unlisted) = %d %s, want 422 ErrRefused", rec.Code, rec.Body.Bytes())
		}
	})
}

// TestReply_UnencodableValueIsA500 verifies that a value JSON cannot encode answers
// 500 with nothing of it sent as a 2xx.
//
// Guard: the body is encoded before the header goes out; writing the header first
// would send a 200 with a truncated body.
func TestReply_UnencodableValueIsA500(t *testing.T) {
	t.Parallel()
	s := &Server{log: discardLogger()}
	rec := httptest.NewRecorder()
	s.reply(rec, http.StatusOK, make(chan int))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("reply(chan) status = %d, want 500", rec.Code)
	}
}

// TestWriteAtomic_WritesReplacesAndLeavesNoTemporaryFile verifies the save: a new file
// has the bytes, a second save replaces them whole, a missing directory is an error
// that creates nothing, and no temporary file stays in the directory.
//
// Guard: a crash or a failed save must leave the previous state file intact and no
// stray temp files; saving in place would leave a truncated checkpoint.
func TestWriteAtomic_WritesReplacesAndLeavesNoTemporaryFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "realm.ckpt")
	for _, want := range []string{"first checkpoint, longer", "second"} {
		if err := writeAtomic(path, []byte(want)); err != nil {
			t.Fatalf("writeAtomic(%q) = %v, want nil", want, err)
		}
		if got, err := os.ReadFile(path); err != nil || string(got) != want {
			t.Errorf("state file = %q (error %v), want %q", got, err, want)
		}
	}
	if err := writeAtomic(filepath.Join(dir, "missing", "x.ckpt"), []byte("x")); err == nil {
		t.Errorf("writeAtomic into a missing directory = nil, want an error")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() = %v, want nil", err)
	}
	if len(entries) != 1 {
		t.Errorf("directory holds %d entries after the saves, want only the state file", len(entries))
	}
	// A directory in the way of the rename: the save fails and removes its temp file.
	blocked := filepath.Join(dir, "blocked")
	if err := os.Mkdir(blocked, 0o700); err != nil {
		t.Fatalf("Mkdir() = %v, want nil", err)
	}
	if err := os.WriteFile(filepath.Join(blocked, "keep"), nil, 0o600); err != nil {
		t.Fatalf("WriteFile() = %v, want nil", err)
	}
	if err := writeAtomic(blocked, []byte("x")); err == nil {
		t.Errorf("writeAtomic over a non-empty directory = nil, want an error")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 2 {
		t.Errorf("directory holds %d entries after the failed rename, want 2 (no temp file left)", len(entries))
	}
}

// TestRestoreOptions_AddsObserverAndAuditOnlyWhenAsked verifies that the option list
// grows by one for an observer and by one for the audit, and by none for neither.
//
// Guard: the pack's strategies must always be registered (a restore without them
// refuses the merchants); the other two are opt-in.
func TestRestoreOptions_AddsObserverAndAuditOnlyWhenAsked(t *testing.T) {
	t.Parallel()
	base := len(RestoreOptions(nil, false))
	if base == 0 {
		t.Fatalf("RestoreOptions(nil, false) is empty, want the strategy registry")
	}
	if got := len(RestoreOptions(&eventRing{}, false)); got != base+1 {
		t.Errorf("RestoreOptions(observer, false) = %d options, want %d", got, base+1)
	}
	if got := len(RestoreOptions(nil, true)); got != base+1 {
		t.Errorf("RestoreOptions(nil, audit) = %d options, want %d", got, base+1)
	}
	if got := len(RestoreOptions(&eventRing{}, true)); got != base+2 {
		t.Errorf("RestoreOptions(observer, audit) = %d options, want %d", got, base+2)
	}
}

var _ economysim.Observer = (*eventRing)(nil)

// TestServer_FailedSaveIs500AndKeepsThePathOut verifies that a checkpoint that cannot
// be written answers 500 ErrSaveFailed without the state path in the body.
//
// Guard: a full or unwritable disk is the server's fault, not a refused request (422),
// and the path is the operator's business, so it goes to the log only.
func TestServer_FailedSaveIs500AndKeepsThePathOut(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "missing-dir")
	s := newTestServer(t, 1, filepath.Join(dir, "x.ckpt"))
	var e errorBody
	callJSON(t, s, http.MethodPost, "/v1/checkpoint?save=true", "", http.StatusInternalServerError, &e)
	if e.Error != "ErrSaveFailed" || strings.Contains(e.Message, dir) {
		t.Errorf("save into a missing directory: error %q message %q, want ErrSaveFailed without %q", e.Error, e.Message, dir)
	}
}

// TestServer_AllowHostsIsCaseInsensitive verifies that a mixed-case allowed host
// matches a request in any case.
//
// Guard: hostname lowercases the request side, so the list must too.
func TestServer_AllowHostsIsCaseInsensitive(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, 1, "")
	s.AllowHosts("MyHost")
	req := httptest.NewRequest("GET", "/v1/world", nil)
	req.Host = "myhost:80"
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET /v1/world with Host myhost after AllowHosts(MyHost) = %d, want 200", rec.Code)
	}
}

// TestServer_AbortHandlerPanicPropagates verifies that http.ErrAbortHandler is not
// turned into a 500 reply.
//
// Guard: net/http uses that panic to drop a connection; answering it would write a
// second status onto a half-sent response.
func TestServer_AbortHandlerPanicPropagates(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, 1, "")
	s.mux = http.NewServeMux()
	s.mux.HandleFunc("/abort", func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) })
	defer func() {
		if p := recover(); p != http.ErrAbortHandler {
			t.Errorf("GET /abort recovered %v, want http.ErrAbortHandler", p)
		}
	}()
	s.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/abort", nil))
}

// TestGame_SlotsListsShocksByName verifies that Slots returns the harvest factors and
// roads in force by name, the same rows Info carries.
func TestGame_SlotsListsShocksByName(t *testing.T) {
	t.Parallel()
	g := newTestGame(t, 1)
	h, r, err := g.Slots()
	info, ierr := g.Info()
	if err != nil || ierr != nil {
		t.Fatalf("Slots() error %v, Info() error %v, want nil", err, ierr)
	}
	if !reflect.DeepEqual(h, info.Harvest) || !reflect.DeepEqual(r, info.Roads) {
		t.Errorf("Slots() = %v, %v, want Info's %v, %v", h, r, info.Harvest, info.Roads)
	}
}

// TestServe_NegativePaceIsRefused verifies that serve refuses a negative pace and
// closes the listener.
//
// Guard: a negative pace used to mean "off" silently, which hides a typo.
func TestServe_NegativePaceIsRefused(t *testing.T) {
	t.Parallel()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() = %v, want nil", err)
	}
	if err := serve(context.Background(), discardLogger(), l, 1, -time.Second, ""); !errors.Is(err, ErrBadInput) {
		t.Errorf("serve with pace -1s = %v, want ErrBadInput", err)
	}
	if _, err := l.Accept(); err == nil {
		t.Errorf("Accept() after serve refused = nil, want the listener closed")
	}
}
