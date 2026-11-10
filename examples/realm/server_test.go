package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	economysim "github.com/Whealinghold/economy-sim"
)

// playerJSON adds the traveller of the tests to the crownlands: money and some grain
// and bread.
const playerJSON = `{"region":"crownlands","role":"traveller","money":5000,"goods":{"grain":10,"bread":2}}`

// serverTicks is the length of a run of the server tests: the 5 000-household world
// is about ten times slower under the race detector.
func serverTicks(normal, race int64) int64 {
	if raceEnabled {
		return race
	}
	return normal
}

// step posts /v1/step?n= and returns the tick after it.
func step(t testing.TB, h http.Handler, n int64) int64 {
	t.Helper()
	var b stepBody
	callJSON(t, h, http.MethodPost, fmt.Sprintf("/v1/step?n=%d", n), "", http.StatusOK, &b)
	if b.Completed != n {
		t.Fatalf("POST /v1/step?n=%d completed %d ticks, want %d", n, b.Completed, n)
	}
	return b.Tick
}

// price reads the price of a commodity in a region.
func price(t testing.TB, h http.Handler, region, commodity string) money {
	t.Helper()
	var m marketView
	callJSON(t, h, http.MethodGet, "/v1/market?region="+region+"&commodity="+commodity, "", http.StatusOK, &m)
	return m.Price
}

// TestServer_PlayerRoundTrip verifies the player's whole loop through HTTP: add the
// player, leave a directive that bids for grain, step once, and read the fill from
// /v1/events and the new balance from /v1/player. The directive changes nothing until
// the step runs (one tick of latency), and, not being sticky, buys nothing in the
// step after it.
//
// Requirement: FR-INT-027
// Guard: architecture.md 1.3. The public API alone is enough to run a player; if a
// call this test needs were missing, the example could not be written.
func TestServer_PlayerRoundTrip(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	s := newTestServer(t, 1, "")
	var added addedBody
	callJSON(t, s, http.MethodPost, "/v1/player", playerJSON, http.StatusCreated, &added)
	step(t, s, 3) // a few ticks give grain a published price
	p := price(t, s, "crownlands", "grain")
	if p.Value < 1 {
		t.Fatalf("grain price in the crownlands = %d, want a published price", p.Value)
	}

	var before playerView
	callJSON(t, s, http.MethodGet, "/v1/player", "", http.StatusOK, &before)
	var quiet eventsView
	callJSON(t, s, http.MethodGet, "/v1/events", "", http.StatusOK, &quiet)
	// The player's fallback strategy trades by itself, so the directive holds every
	// commodity: the one bid below is then the player's only order. One unit at three
	// times the market price: the limit is far above any ask, so it fills.
	limit := 3*p.Value + 1
	directive := fmt.Sprintf(`{"hold":%s,"intents":[{"commodity":"grain","side":"bid","quantity":1,"limit":%d}]}`, holdAll(t), limit)
	var accepted acceptedBody
	callJSON(t, s, http.MethodPost, "/v1/player/directive", directive, http.StatusOK, &accepted)
	if accepted.AppliesToTick != before.Tick {
		t.Errorf("directive applies to tick %d, want the next step, which runs at tick %d", accepted.AppliesToTick, before.Tick)
	}

	var idle eventsView
	callJSON(t, s, http.MethodGet, "/v1/events", "", http.StatusOK, &idle)
	var mid playerView
	callJSON(t, s, http.MethodGet, "/v1/player", "", http.StatusOK, &mid)
	if len(idle.Events) != len(quiet.Events) || mid.Money.Value != before.Money.Value {
		t.Errorf("after the directive: %d events (was %d), money %d (was %d), want no change (a directive acts in the next step)", len(idle.Events), len(quiet.Events), mid.Money.Value, before.Money.Value)
	}

	next := step(t, s, 1)
	var ev eventsView
	callJSON(t, s, http.MethodGet, fmt.Sprintf("/v1/events?since=%d", before.Tick), "", http.StatusOK, &ev)
	var trades []eventView
	for _, e := range ev.Events {
		if e.Kind == "trade" {
			trades = append(trades, e)
		}
	}
	if len(trades) != 1 {
		t.Fatalf("trades since tick %d = %d (events %+v), want the one bid filled", before.Tick, len(trades), ev.Events)
	}
	trade := trades[0]
	if trade.Side != "bid" || trade.Commodity != "grain" || trade.Quantity != 1 || trade.Price == nil || trade.Price.Value > limit {
		t.Errorf("trade = %+v, want one unit of grain bought at no more than %d", trade, limit)
	}
	var after playerView
	callJSON(t, s, http.MethodGet, "/v1/player", "", http.StatusOK, &after)
	if want := before.Money.Value - trade.Price.Value; after.Money.Value != want || after.Tick != next {
		t.Errorf("after the step: money %d at tick %d, want %d (%d less the %d paid) at tick %d", after.Money.Value, after.Tick, want, before.Money.Value, trade.Price.Value, next)
	}

	step(t, s, 1)
	callJSON(t, s, http.MethodGet, fmt.Sprintf("/v1/events?since=%d", next), "", http.StatusOK, &ev)
	for _, e := range ev.Events {
		if e.Kind == "order" && e.Side == "bid" && e.Commodity == "grain" && e.Price != nil && e.Price.Value == limit {
			t.Errorf("order %+v in the step after the directive, want none at its limit: a directive is for one step", e)
		}
	}
}

// holdAll returns the JSON list of every commodity name of the pack.
func holdAll(t testing.TB) string {
	t.Helper()
	cfg, err := packOnce()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	ids := make([]string, 0, len(cfg.Commodities))
	for _, c := range cfg.Commodities {
		ids = append(ids, fmt.Sprintf("%q", c.ID))
	}
	return "[" + strings.Join(ids, ",") + "]"
}

// TestServer_ShocksChangeThePriceAndTheClearSetsItBack verifies that a failed harvest
// moves the staple price against an unshocked twin of the same seed, that the world
// lists the shock, and that setting the factor back to 10 000 empties the slot.
//
// Requirement: FR-INT-040, FR-INT-048
func TestServer_ShocksChangeThePriceAndTheClearSetsItBack(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	shocked, twin := newTestServer(t, 2, ""), newTestServer(t, 2, "")
	failed := `{"shocks":[{"region":"highmarch","recipe":"grow_grain","bp":0},{"region":"eastvale","recipe":"grow_grain","bp":0},{"region":"southfen","recipe":"grow_grain","bp":0}]}`
	var slots slotsView
	callJSON(t, shocked, http.MethodPost, "/v1/shock/harvest", failed, http.StatusOK, &slots)
	if len(slots.Harvest) != 3 {
		t.Fatalf("slots after the shock = %+v, want 3 harvest factors", slots.Harvest)
	}
	ticks := serverTicks(60, 30)
	step(t, shocked, ticks)
	step(t, twin, ticks)
	got, want := price(t, shocked, "highmarch", "grain"), price(t, twin, "highmarch", "grain")
	// The price moves, not necessarily up: with no harvest the growers leave, and the
	// demand they carried goes with them. Only the difference is the contract.
	if got.Value == want.Value {
		t.Errorf("grain price in the highmarch after %d ticks of failed harvest = %d, want a price different from the twin's %d", ticks, got.Value, want.Value)
	}

	clear := strings.ReplaceAll(failed, `"bp":0`, `"bp":10000`)
	callJSON(t, shocked, http.MethodPost, "/v1/shock/harvest", clear, http.StatusOK, &slots)
	if len(slots.Harvest) != 0 {
		t.Errorf("slots after clearing = %+v, want none", slots.Harvest)
	}
	var w worldView
	callJSON(t, shocked, http.MethodGet, "/v1/world", "", http.StatusOK, &w)
	if len(w.Harvest) != 0 {
		t.Errorf("GET /v1/world harvest after clearing = %+v, want none", w.Harvest)
	}
}

// TestServer_ClosedRoadReportsTheEdge verifies that a closed road appears in
// /v1/world and that a checkpoint and a restore keep it.
//
// Requirement: FR-INT-041, FR-INT-070
func TestServer_ClosedRoadReportsTheEdge(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, 3, "")
	cfg, err := packOnce()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	edge := cfg.Edges[0]
	body := fmt.Sprintf(`{"roads":[{"from":%q,"to":%q,"closed":true}]}`, edge.From, edge.To)
	var slots slotsView
	callJSON(t, s, http.MethodPost, "/v1/shock/road", body, http.StatusOK, &slots)
	want := RoadSpec{From: edge.From, To: edge.To, Closed: true, FactorBP: 10_000}
	if len(slots.Roads) != 1 || slots.Roads[0] != want {
		t.Fatalf("slots after closing = %+v, want [%+v]", slots.Roads, want)
	}
	code, saved := call(t, s, http.MethodPost, "/v1/checkpoint", "")
	if code != http.StatusOK {
		t.Fatalf("POST /v1/checkpoint = %d, want 200", code)
	}
	other := newTestServer(t, 3, "")
	var restored restoredBody
	callJSON(t, other, http.MethodPost, "/v1/restore", string(saved), http.StatusOK, &restored)
	var w worldView
	callJSON(t, other, http.MethodGet, "/v1/world", "", http.StatusOK, &w)
	if len(w.Roads) != 1 || w.Roads[0] != want {
		t.Errorf("GET /v1/world roads after the restore = %+v, want [%+v]", w.Roads, want)
	}
}

// TestServer_CheckpointMidRunResumesBitIdentical verifies the exit criterion of the
// server: a checkpoint taken over HTTP mid-run and restored into a second server
// continues exactly as the first, with the same snapshot and the same checkpoint
// bytes after the same further ticks, and the player survives.
//
// Requirement: FR-CHK-057, FR-REF-032
func TestServer_CheckpointMidRunResumesBitIdentical(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	n := serverTicks(200, 8)
	a := newTestServer(t, 7, "")
	callJSON(t, a, http.MethodPost, "/v1/player", playerJSON, http.StatusCreated, new(addedBody))
	step(t, a, n)
	code, mid := call(t, a, http.MethodPost, "/v1/checkpoint", "")
	if code != http.StatusOK {
		t.Fatalf("POST /v1/checkpoint = %d, want 200", code)
	}
	step(t, a, n)

	b := newTestServer(t, 7, "")
	var restored restoredBody
	callJSON(t, b, http.MethodPost, "/v1/restore", string(mid), http.StatusOK, &restored)
	if !restored.Player || restored.Tick != n {
		t.Errorf("restore = {player %t, tick %d}, want {true, %d}", restored.Player, restored.Tick, n)
	}
	step(t, b, n)

	sa, err := a.game.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	sb, err := b.game.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	ja, _ := json.Marshal(sa)
	jb, _ := json.Marshal(sb)
	if !bytes.Equal(ja, jb) {
		t.Errorf("snapshot JSON at tick %d after the restore differs from the uninterrupted run (%d and %d bytes)", sa.Tick, len(jb), len(ja))
	}
	_, ca := call(t, a, http.MethodPost, "/v1/checkpoint", "")
	_, cb := call(t, b, http.MethodPost, "/v1/checkpoint", "")
	if !bytes.Equal(ca, cb) {
		t.Errorf("checkpoint at tick %d after the restore differs from the uninterrupted run (%d and %d bytes)", sa.Tick, len(cb), len(ca))
	}
}

// TestServer_ConcurrentReadsAreSerialised verifies that requests from many
// goroutines, while another steps the world, neither race nor panic and each get a
// complete answer. It is meaningful under -race.
//
// Guard: architecture.md 12.1. The world is single-threaded; if the server called it
// without the lock, the race detector would report it here.
func TestServer_ConcurrentReadsAreSerialised(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	s := newTestServer(t, 4, "")
	callJSON(t, s, http.MethodPost, "/v1/player", playerJSON, http.StatusCreated, new(addedBody))
	targets := []string{"/v1/world", "/v1/market?region=crownlands&commodity=grain", "/v1/region?region=crownlands", "/v1/player", "/v1/events"}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for _, target := range []string{targets[i%len(targets)], targets[(i+1)%len(targets)]} {
				if code, b := call(t, s, http.MethodGet, target, ""); code != http.StatusOK {
					t.Errorf("GET %s = %d (%s), want 200", target, code, bytes.TrimSpace(b))
				}
			}
		}()
	}
	close(start)
	step(t, s, 2)
	wg.Wait()
}

// TestServer_BadInputIsRefused verifies that a name the pack does not define, a
// negative or huge number, a malformed or oversized body, a directive with no player
// and a restore of garbage each get a 4xx with the name of the sentinel, and that
// none of them changes the tick, the player or a byte of the checkpoint.
//
// Requirement: FR-INT-002, FR-CHK-016
func TestServer_BadInputIsRefused(t *testing.T) {
	t.Parallel()
	s := sharedServer(t)
	_, before := call(t, s, http.MethodPost, "/v1/checkpoint", "")
	garbage := string(make([]byte, headerLen)) + "not a checkpoint"
	huge := strings.Repeat(" ", maxJSONBody+1)
	tests := []struct {
		name, method, target, body string
		status                     int
		sentinel                   string
	}{
		{"market of an unknown region", "GET", "/v1/market?region=atlantis&commodity=grain", "", 404, "ErrUnknownName"},
		{"market of an unknown commodity", "GET", "/v1/market?region=crownlands&commodity=mithril", "", 404, "ErrUnknownName"},
		{"market with no query", "GET", "/v1/market", "", 404, "ErrUnknownName"},
		{"unknown region", "GET", "/v1/region?region=atlantis", "", 404, "ErrUnknownName"},
		{"no player yet", "GET", "/v1/player", "", 404, "ErrNoPlayer"},
		{"events since a negative tick", "GET", "/v1/events?since=-1", "", 400, "ErrBadInput"},
		{"events since a word", "GET", "/v1/events?since=soon", "", 400, "ErrBadInput"},
		{"player in an unknown region", "POST", "/v1/player", `{"region":"atlantis","role":"traveller"}`, 404, "ErrUnknownName"},
		{"player of an unknown role", "POST", "/v1/player", `{"region":"crownlands","role":"wizard"}`, 404, "ErrUnknownName"},
		{"player with an unknown commodity", "POST", "/v1/player", `{"region":"crownlands","role":"traveller","goods":{"mithril":1}}`, 404, "ErrUnknownName"},
		{"player with negative money", "POST", "/v1/player", `{"region":"crownlands","role":"traveller","money":-1}`, 400, "ErrBadInput"},
		{"player with a negative quantity", "POST", "/v1/player", `{"region":"crownlands","role":"traveller","goods":{"grain":-5}}`, 400, "ErrBadInput"},
		{"player with more money than the bound", "POST", "/v1/player", `{"region":"crownlands","role":"traveller","money":9223372036854775807}`, 400, "ErrOutOfBounds"},
		{"player with more grain than the bound", "POST", "/v1/player", `{"region":"crownlands","role":"traveller","goods":{"grain":9223372036854775807}}`, 400, "ErrOutOfBounds"},
		{"player in the overseas node", "POST", "/v1/player", `{"region":"overseas","role":"traveller"}`, 400, "ErrOutsideNode"},
		{"player with a number past int64", "POST", "/v1/player", `{"region":"crownlands","role":"traveller","money":99999999999999999999}`, 400, "ErrBadInput"},
		{"player with an unknown field", "POST", "/v1/player", `{"region":"crownlands","role":"traveller","wings":2}`, 400, "ErrBadInput"},
		{"player body is not JSON", "POST", "/v1/player", `region=crownlands`, 400, "ErrBadInput"},
		{"player body is empty", "POST", "/v1/player", ``, 400, "ErrBadInput"},
		{"player body has a second value", "POST", "/v1/player", `{"region":"crownlands","role":"traveller"} {}`, 400, "ErrBadInput"},
		{"player body past the cap", "POST", "/v1/player", huge, 413, "ErrBodyTooLarge"},
		{"directive with no player", "POST", "/v1/player/directive", `{"hold":["grain"]}`, 404, "ErrNoPlayer"},
		{"harvest in an unknown region", "POST", "/v1/shock/harvest", `{"shocks":[{"region":"atlantis","recipe":"grow_grain","bp":0}]}`, 404, "ErrUnknownName"},
		{"harvest of an unknown recipe", "POST", "/v1/shock/harvest", `{"shocks":[{"region":"highmarch","recipe":"conjure","bp":0}]}`, 404, "ErrUnknownName"},
		{"harvest factor below zero", "POST", "/v1/shock/harvest", `{"shocks":[{"region":"highmarch","recipe":"grow_grain","bp":-1}]}`, 400, "ErrOutOfBounds"},
		{"harvest factor past the bound", "POST", "/v1/shock/harvest", `{"shocks":[{"region":"highmarch","recipe":"grow_grain","bp":9000000000000}]}`, 400, "ErrOutOfBounds"},
		{"road between unknown regions", "POST", "/v1/shock/road", `{"roads":[{"from":"atlantis","to":"highmarch"}]}`, 404, "ErrUnknownName"},
		{"road that is not defined", "POST", "/v1/shock/road", `{"roads":[{"from":"highmarch","to":"highmarch"}]}`, 400, "ErrUnknownEdge"},
		{"road factor below zero", "POST", "/v1/shock/road", `{"roads":[{"from":"crownlands","to":"highmarch","factor_bp":-3}]}`, 400, "ErrOutOfBounds"},
		{"step of a word", "POST", "/v1/step?n=many", "", 400, "ErrBadInput"},
		{"step of a negative count", "POST", "/v1/step?n=-1", "", 400, "ErrBadInput"},
		{"step past the cap", "POST", "/v1/step?n=1001", "", 400, "ErrBadInput"},
		{"step past int64", "POST", "/v1/step?n=99999999999999999999", "", 400, "ErrBadInput"},
		{"restore of garbage", "POST", "/v1/restore", garbage, 400, "ErrCheckpointCorrupt"},
		{"restore of fewer bytes than the header", "POST", "/v1/restore", "abc", 400, "ErrBadInput"},
		{"restore of nothing", "POST", "/v1/restore", "", 400, "ErrBadInput"},
		{"restore past the cap", "POST", "/v1/restore", strings.Repeat("x", maxRestoreBody+1), 413, "ErrBodyTooLarge"},
		{"save with no state path", "POST", "/v1/checkpoint?save=true", "", 400, "ErrNoState"},
		{"a path no route has", "GET", "/v2/world", "", 404, "ErrUnknownName"},
		{"a method no route has", "DELETE", "/v1/world", "", 404, "ErrUnknownName"},
	}
	// The group returns only when its parallel subtests are done, so the check after it
	// sees the server after every refusal.
	t.Run("refusals", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.target != "/v1/restore" { // restores share one token: two at once would answer 429
					t.Parallel()
				}
				var e errorBody
				callJSON(t, s, tt.method, tt.target, tt.body, tt.status, &e)
				if e.Error != tt.sentinel {
					t.Errorf("%s %s: error = %q, want %q", tt.method, tt.target, e.Error, tt.sentinel)
				}
				if e.Message == "" {
					t.Errorf("%s %s: empty message", tt.method, tt.target)
				}
			})
		}
	})
	t.Run("nothing changed", func(t *testing.T) {
		var w worldView
		callJSON(t, s, http.MethodGet, "/v1/world", "", http.StatusOK, &w)
		if w.Tick != 0 {
			t.Errorf("tick after the refused requests = %d, want 0", w.Tick)
		}
		_, after := call(t, s, http.MethodPost, "/v1/checkpoint", "")
		if !bytes.Equal(before, after) {
			t.Errorf("checkpoint after the refused requests differs from before (%d and %d bytes)", len(after), len(before))
		}
	})
}

// TestServer_FormatsPricesWithThePreset verifies that prices go out in the ladder of
// the RPG preset beside their minor units: 1 460 is 1 sp 4 cp and 12 345 is
// 1 gp 2 sp 3 cp, and a market response carries the text of its own value.
//
// Requirement: FR-CUR-017
func TestServer_FormatsPricesWithThePreset(t *testing.T) {
	t.Parallel()
	s := sharedServer(t)
	for _, tt := range []struct {
		value int64
		text  string
	}{
		{1_460, "1 sp 4 cp"}, // 1 460 = 1 x 1 000 + 4 x 100 + 60, and the 60 under a copper is dropped
		{12_345, "1 gp 2 sp 3 cp"},
		{100, "1 cp"},
	} {
		if got := s.amount(tt.value); got.Value != tt.value || got.Text != tt.text {
			t.Errorf("amount(%d) = %+v, want text %q", tt.value, got, tt.text)
		}
	}
	m := price(t, s, "crownlands", "grain")
	if want := s.money.Format(m.Value); m.Text != want {
		t.Errorf("market price %d has text %q, want %q", m.Value, m.Text, want)
	}
}

// TestServer_StepRunsTheTicksAskedAndStopsWhenCancelled verifies that /v1/step with
// no n runs one tick, and that a request whose context is already done runs none and
// answers without a 4xx.
//
// Guard: architecture.md 12.3. A step stops at a tick boundary when its context is done;
// a server that ignored the client's cancellation would run a 1 000-tick request to the end.
func TestServer_StepRunsTheTicksAskedAndStopsWhenCancelled(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	s := newTestServer(t, 5, "")
	var b stepBody
	callJSON(t, s, http.MethodPost, "/v1/step", "", http.StatusOK, &b)
	if b.Tick != 1 || b.Completed != 1 {
		t.Errorf("POST /v1/step = {tick %d, completed %d}, want {1, 1}", b.Tick, b.Completed)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/v1/step?n=3", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() = %v, want nil", err)
	}
	rec := &recorder{}
	s.ServeHTTP(rec, req)
	if rec.code != http.StatusServiceUnavailable || s.game.Tick() != 1 {
		t.Errorf("cancelled step = status %d at tick %d, want 503 at tick 1", rec.code, s.game.Tick())
	}
}

// recorder is the smallest http.ResponseWriter that keeps the status.
type recorder struct {
	code int
	hdr  http.Header
	buf  bytes.Buffer
}

func (r *recorder) Header() http.Header {
	if r.hdr == nil {
		r.hdr = http.Header{}
	}
	return r.hdr
}
func (r *recorder) Write(b []byte) (int, error) { return r.buf.Write(b) }
func (r *recorder) WriteHeader(code int)        { r.code = code }

// TestServer_StateFileSavesAndRestoresAtStart verifies the -state file: a checkpoint
// with save=true is written to it and equals the bytes answered, a server started
// over it resumes at that tick with the player, and a file that is no checkpoint
// stops the start instead of being written over.
//
// Guard: the state file is the one thing the example writes. If a save wrote in place, a
// crash mid-write would leave a truncated checkpoint that the next start refuses.
func TestServer_StateFileSavesAndRestoresAtStart(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	path := filepath.Join(t.TempDir(), "realm.ckpt")
	a := newTestServer(t, 6, path)
	callJSON(t, a, http.MethodPost, "/v1/player", playerJSON, http.StatusCreated, new(addedBody))
	step(t, a, 2)
	code, body := call(t, a, http.MethodPost, "/v1/checkpoint?save=true", "")
	if code != http.StatusOK {
		t.Fatalf("POST /v1/checkpoint?save=true = %d, want 200", code)
	}
	if file, err := os.ReadFile(path); err != nil || !bytes.Equal(file, body) {
		t.Errorf("state file: %d bytes, error %v, want the %d bytes answered", len(file), err, len(body))
	}
	b := newTestServer(t, 99, path)
	var p playerView
	callJSON(t, b, http.MethodGet, "/v1/player", "", http.StatusOK, &p)
	if p.Tick != 2 {
		t.Errorf("a server started over the state file is at tick %d, want 2", p.Tick)
	}
	if err := os.WriteFile(path, []byte("not a checkpoint at all"), 0o600); err != nil {
		t.Fatalf("WriteFile() = %v, want nil", err)
	}
	cfg, _ := packOnce()
	if _, err := newServer(cfg, 1, path, nil); err == nil {
		t.Errorf("newServer over a state file that is no checkpoint = nil, want an error")
	}
}

// TestServer_PaceStepsOncePerTickOfTheClock verifies that Pace steps the world once
// for each value of the clock it is given, and returns when its context is done. The
// test is the clock, so it neither sleeps nor reads the time.
//
// Guard: testing.md 10.2. The pacing loop is the one place the example meets time; it
// must stay driven by a channel the test owns.
func TestServer_PaceStepsOncePerTickOfTheClock(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	s := newTestServer(t, 8, "")
	ctx, cancel := context.WithCancel(context.Background())
	clock := make(chan time.Time)
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Pace(ctx, clock)
	}()
	// The third send returns only when Pace is back at its select, so two steps are done.
	for range 3 {
		clock <- time.Time{}
	}
	cancel()
	<-done
	if tick := s.game.Tick(); tick < 2 || tick > 3 {
		t.Errorf("tick after three clock values = %d, want 2 or 3 (the third may be cancelled)", tick)
	}
}

// TestEventRing_KeepsTheLastEventsOfThePlayerOnly verifies the ring: it records the
// orders, rejections and trades of the watched agent only, from the tick asked, drops
// the oldest past ringCap, and records nothing before it watches anyone.
//
// Guard: the ring is the observer's side of the loop, called on the stepping goroutine;
// an event of another agent in it would show the player other households' trades.
func TestEventRing_KeepsTheLastEventsOfThePlayerOnly(t *testing.T) {
	t.Parallel()
	r := &eventRing{}
	r.Observe(economysim.OrderPlaced{Tick: 1, Agent: 5})
	if got := r.since(0); len(got) != 0 {
		t.Fatalf("events before watch() = %d, want 0", len(got))
	}
	r.watch(5, true)
	r.Observe(economysim.OrderPlaced{Tick: 1, Agent: 6})
	r.Observe(economysim.TradeExecuted{Tick: 1, Buyer: 6, Seller: 7})
	r.Observe(economysim.OrderRejected{Tick: 2, Agent: 5, Reason: "no_funds"})
	r.Observe(economysim.TradeExecuted{Tick: 3, Buyer: 5, Seller: 5, Price: 4, Quantity: 2})
	r.Observe(economysim.TickEnded{Tick: 3})
	var kinds []string
	for _, e := range r.since(2) {
		kinds = append(kinds, fmt.Sprintf("%d:%s:%s", e.Tick, e.Kind, e.Side))
	}
	if want := "2:rejected:bid 3:trade:bid 3:trade:ask"; strings.Join(kinds, " ") != want {
		t.Errorf("events since 2 = %v, want %q (a trade with the player on both sides is two events)", kinds, want)
	}
	for i := range ringCap {
		r.Observe(economysim.OrderPlaced{Tick: int64(10 + i), Agent: 5})
	}
	all := r.since(0)
	if len(all) != ringCap || all[0].Tick != 10 {
		t.Errorf("after %d more events the ring holds %d from tick %d, want %d from tick 10", ringCap, len(all), all[0].Tick, ringCap)
	}
	r.reset()
	if got := r.since(0); len(got) != 0 {
		t.Errorf("events after reset() = %d, want 0", len(got))
	}
}

// TestServer_DirectiveByNameReachesTheWorld verifies the parts of a directive the
// round trip does not use: a job, a discard, a hold and a sticky flag are accepted
// by name, and a name or a side the pack does not know is refused.
//
// Requirement: FR-INT-025, FR-INT-026, FR-INT-028
func TestServer_DirectiveByNameReachesTheWorld(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	s := newTestServer(t, 9, "")
	callJSON(t, s, http.MethodPost, "/v1/player", playerJSON, http.StatusCreated, new(addedBody))
	ok := `{"job":{"no_job":true},"discards":[{"commodity":"bread","quantity":1}],"hold":["grain"],"sticky":true}`
	callJSON(t, s, http.MethodPost, "/v1/player/directive", ok, http.StatusOK, new(acceptedBody))
	for name, body := range map[string]string{
		"unknown recipe":    `{"job":{"recipe":"conjure"}}`,
		"unknown commodity": `{"intents":[{"commodity":"mithril","side":"bid","quantity":1,"limit":5}]}`,
		"unknown hold":      `{"hold":["mithril"]}`,
		"unknown discard":   `{"discards":[{"commodity":"mithril","quantity":1}]}`,
		"unknown side":      `{"intents":[{"commodity":"grain","side":"swap","quantity":1,"limit":5}]}`,
		"too many items":    `{"hold":[` + strings.Repeat(`"grain",`, maxDirectiveItems) + `"grain"]}`,
	} {
		var e errorBody
		code, b := call(t, s, http.MethodPost, "/v1/player/directive", body)
		if err := json.Unmarshal(b, &e); err != nil {
			t.Errorf("%s: decode %q: %v", name, b, err)
		}
		if code < 400 || code >= 500 || e.Error == "" {
			t.Errorf("%s: status %d error %q, want a 4xx with a sentinel name", name, code, e.Error)
		}
	}
	step(t, s, 1) // the sticky directive is a valid one: the step runs
}

// TestServer_RegionAndMarketAreReadByName verifies the two reads of a place: the
// region lists its households, roles and markets, and the market row of a commodity
// agrees with the market route.
//
// Requirement: FR-INT-053, FR-INT-054
func TestServer_RegionAndMarketAreReadByName(t *testing.T) {
	t.Parallel()
	s := sharedServer(t)
	var r regionStateView
	callJSON(t, s, http.MethodGet, "/v1/region?region=crownlands", "", http.StatusOK, &r)
	if r.Households < 1 || len(r.Roles) == 0 || len(r.Markets) == 0 {
		t.Fatalf("region = %d households, %d roles, %d markets, want all above zero", r.Households, len(r.Roles), len(r.Markets))
	}
	var w worldView
	callJSON(t, s, http.MethodGet, "/v1/world", "", http.StatusOK, &w)
	if len(w.ConfigHash) != 64 || len(w.Regions) != 6 {
		t.Errorf("world = hash of %d characters and %d regions, want 64 and 6", len(w.ConfigHash), len(w.Regions))
	}
	for _, row := range r.Markets {
		if row.Commodity != "grain" {
			continue
		}
		if m := price(t, s, "crownlands", "grain"); m != row.Price {
			t.Errorf("market row price %+v, GET /v1/market price %+v, want equal", row.Price, m)
		}
	}
}

// TestServer_RefusesAWriteFromAnotherOrigin verifies that a POST carrying an Origin
// header of another host is refused with 403 ErrForbidden and changes nothing, while
// the same Origin as the Host, no Origin and a GET from another origin go through.
//
// Guard: a web page in the user's browser can send a simple POST to a server on
// localhost; without the check it could step the world or add the player.
func TestServer_RefusesAWriteFromAnotherOrigin(t *testing.T) {
	t.Parallel()
	s := sharedServer(t)
	for _, tt := range []struct {
		name, method, origin string
		status               int
	}{
		{"another host", "POST", "http://evil.example", http.StatusForbidden},
		{"not a URL", "POST", "http://[::1", http.StatusForbidden},
		{"the same host", "POST", "http://example.com", http.StatusOK},
		{"no origin", "POST", "", http.StatusOK},
		{"a read from another host", "GET", "http://evil.example", http.StatusOK},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// example.com is the Host httptest.NewRequest sets; an empty shock list is a
			// write that changes nothing and, unlike a step, never answers 429.
			target, body := "/v1/shock/harvest", `{"shocks":[]}`
			if tt.method == "GET" {
				target, body = "/v1/world", ""
			}
			req := httptest.NewRequest(tt.method, target, strings.NewReader(body))
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()
			s.ServeHTTP(rec, req)
			if rec.Code != tt.status {
				t.Errorf("%s %s with Origin %q = status %d, want %d", tt.method, target, tt.origin, rec.Code, tt.status)
			}
		})
	}
}

// TestServer_StepsDoNotQueue verifies that a step asked for while another one runs
// is refused with 429 ErrBusy at once, and runs again when the first is done.
//
// Guard: each step holds the world for up to maxStepTicks ticks; queued steps would
// starve every other route for the sum of their lengths.
func TestServer_StepsDoNotQueue(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, 1, "")
	s.stepper <- struct{}{} // a step in progress
	var e errorBody
	callJSON(t, s, http.MethodPost, "/v1/step?n=1", "", http.StatusTooManyRequests, &e)
	if e.Error != "ErrBusy" {
		t.Errorf("POST /v1/step while one runs: error = %q, want ErrBusy", e.Error)
	}
	<-s.stepper
	var ok stepBody
	callJSON(t, s, http.MethodPost, "/v1/step?n=0", "", http.StatusOK, &ok)
}

// TestServer_AllowHostsRefusesOtherHosts verifies that once AllowHosts is called, a
// request whose Host is not on the list gets 403 ErrForbidden whatever the method,
// and a listed host (with or without a port, bracketed or not) goes through.
//
// Guard: a DNS-rebinding page sends an Origin equal to its Host, so the Origin check
// alone does not stop it; the Host list does.
func TestServer_AllowHostsRefusesOtherHosts(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, 1, "")
	s.AllowHosts("localhost", "::1")
	for _, tt := range []struct {
		host   string
		status int
	}{
		{"localhost:8080", http.StatusOK},
		{"localhost", http.StatusOK},
		{"[::1]:8080", http.StatusOK},
		{"[::1]", http.StatusOK},
		{"attacker.example:8080", http.StatusForbidden},
		{"", http.StatusForbidden},
	} {
		req := httptest.NewRequest("GET", "/v1/world", nil)
		req.Host = tt.host
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		if rec.Code != tt.status {
			t.Errorf("GET /v1/world with Host %q = status %d, want %d", tt.host, rec.Code, tt.status)
		}
	}
}

// TestServer_PanicIsAJSON500 verifies that a handler that panics answers 500 with a
// JSON error and leaves the server usable.
//
// Guard: net/http would otherwise drop the connection with no status, and the
// contract is that every failure is a status and a JSON body.
func TestServer_PanicIsAJSON500(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, 1, "")
	s.mux = http.NewServeMux()
	s.mux.HandleFunc("/boom", func(http.ResponseWriter, *http.Request) { panic("boom") })
	var e errorBody
	callJSON(t, s, http.MethodGet, "/boom", "", http.StatusInternalServerError, &e)
	if e.Error != "ErrInternal" {
		t.Errorf("GET /boom: error = %q, want ErrInternal", e.Error)
	}
}

// TestServer_RestoreRefusedWhileAStepRuns verifies that a restore asked for while a
// step runs is refused with 429 ErrBusy, like a second step.
//
// Guard: a restore rebuilds the world under the write lock; repeated ones would
// starve every other route.
func TestServer_RestoreRefusedWhileAStepRuns(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, 1, "")
	s.stepper <- struct{}{}
	var e errorBody
	callJSON(t, s, http.MethodPost, "/v1/restore", "x", http.StatusTooManyRequests, &e)
	if e.Error != "ErrBusy" {
		t.Errorf("POST /v1/restore while a step runs: error = %q, want ErrBusy", e.Error)
	}
}
