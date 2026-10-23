package ledger

import (
	"bytes"
	"errors"
	"math"
	"math/rand/v2"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
)

// freshLike returns a ledger as the root builds one for a restore: the
// configuration of the rig, with the boundary accounts of g's ledger opened
// in their order and nothing else.
func freshLike(tb testing.TB, g *rig) *Ledger {
	tb.Helper()
	l, err := New(rigConfig(g.l.maxMoney))
	if err != nil {
		tb.Fatalf("New: %v", err)
	}
	for _, a := range g.l.accounts[:g.l.boundaries()] {
		if _, err := l.OpenBoundary(a.reason, a.thing); err != nil {
			tb.Fatalf("OpenBoundary(%d, %d): %v", a.reason, a.thing, err)
		}
	}
	return l
}

// saved returns the bytes WriteState writes for l.
func saved(l *Ledger) []byte {
	w := codec.NewWriter(nil)
	l.WriteState(w)
	return w.Out()
}

// read restores b into l as a world at tick would, and returns the result of
// ReadState.
func read(l *Ledger, b []byte, tick int64, construction bool) error {
	return l.ReadState(codec.NewReader(b), tick, construction)
}

// normal returns the fingerprint of l with the differences that a restore does
// not carry removed: the hints are exact, the construction bucket is read-only
// history that a world restored after its first step does not have, and an
// empty slice is nil.
func normal(l *Ledger) *Ledger {
	l.RebuildHints()
	f := fingerprint(l)
	if !f.construction {
		f.bucket, f.wearBucket = make([]int64, len(f.bucket)), make([]int64, len(f.wearBucket))
	}
	for i := range f.accounts {
		if len(f.accounts[i].lots) == 0 {
			f.accounts[i].lots = nil
		}
	}
	if len(f.claims) == 0 {
		f.claims = nil
	}
	if len(f.relocs) == 0 {
		f.relocs = nil
	}
	return f
}

// settle puts a scripted rig at a tick boundary: no reservation is held and a
// world that has left construction has cleared its totals, as the world's
// P8.6 does.
func settle(tb testing.TB, s *scripter) {
	tb.Helper()
	l := s.g.l
	if err := s.releaseAll(); err != nil {
		tb.Fatalf("releaseAll: %v", err)
	}
	if !l.InConstruction() && !l.cleared {
		if err := s.endTick(); err != nil {
			tb.Fatalf("endTick: %v", err)
		}
	}
}

// TestState_RoundTripAfterScript verifies, over 20 seeds of the scripted
// ledger, that the state written at a tick boundary reads back into a fresh
// ledger of the same configuration as the same state (the fingerprint, with
// the hints rebuilt), that the restored ledger passes the audit, and that
// writing it again gives the same bytes. The seeds include worlds in the
// construction phase, worlds past it, and worlds with open claims, closed
// accounts and calls after the last tick's end.
//
// Requirement: FR-CHK-004, FR-CHK-005, FR-CHK-046, FR-CHK-047
func TestState_RoundTripAfterScript(t *testing.T) {
	t.Parallel()
	var inConstruction, past, claims, closed, pending int
	for seed := uint64(1); seed <= 20; seed++ {
		g := newRig(t, scriptMaxMoney)
		g.fund(t)
		// Every fourth seed stops early, still in the construction phase.
		steps := 150 + 40*int(seed)
		if seed%4 == 0 {
			steps = 8
		}
		s := newScripter(g, &randChooser{r: rand.New(rand.NewPCG(seed, 11)), left: steps})
		if err := s.run(); err != nil {
			t.Fatalf("seed %d: script: %v", seed, err)
		}
		settle(t, s)
		l := g.l
		if seed%2 == 0 { // a closed account, which the script may not have made
			id, err := l.Open(KindEscrow, 0, 1)
			if err == nil {
				err = l.Close(id)
			}
			if err != nil {
				t.Fatalf("seed %d: open and close: %v", seed, err)
			}
		}
		b := saved(l)
		fresh := freshLike(t, g)
		if err := read(fresh, b, l.Tick(), l.InConstruction()); err != nil {
			t.Fatalf("seed %d: ReadState: %v", seed, err)
		}
		if err := fresh.Audit(); err != nil {
			t.Errorf("seed %d: Audit of the restored ledger: %v", seed, err)
		}
		if want, got := normal(l), normal(fresh); !reflect.DeepEqual(want, got) {
			t.Errorf("seed %d: restored ledger differs from the original", seed)
		}
		if again := saved(fresh); !bytes.Equal(b, again) {
			t.Errorf("seed %d: writing the restored ledger gave %d bytes that differ from the %d written first", seed, len(again), len(b))
		}
		if l.InConstruction() {
			inConstruction++
		} else {
			past++
		}
		if len(l.claims) > 0 {
			claims++
		}
		if slices.ContainsFunc(l.accounts, func(a account) bool { return a.closed }) {
			closed++
		}
		if slices.ContainsFunc(l.totals, func(v int64) bool { return v != 0 }) {
			pending++
		}
	}
	// Guard: a round trip over seeds that never reach a case proves nothing
	// about it.
	if inConstruction == 0 || past == 0 || claims == 0 || closed == 0 || pending == 0 {
		t.Errorf("the 20 seeds reached construction %d, past it %d, claims %d, closed accounts %d, pending totals %d times, want each above 0", inConstruction, past, claims, closed, pending)
	}
}

// tiny returns a ledger of two regions, three commodities (one spoils with a
// shelf life of 3, one wears with a durability of 5, one does neither), bounds
// of 1000 and 100 and room for the given number of claims, with no account.
func tiny(tb testing.TB, maxClaims int) *Ledger {
	tb.Helper()
	l, err := New(Config{
		Commodities: []Commodity{{ShelfLife: 3}, {Durability: 5}, {}},
		Regions:     2,
		MaxMoney:    1000,
		MaxQuantity: 100,
		MaxClaims:   maxClaims,
	})
	if err != nil {
		tb.Fatalf("New: %v", err)
	}
	return l
}

// TestState_ClosedAccountIsKindAndFlag verifies that a closed account is two
// bytes, its kind and its closed flag, that its ID survives a restore, and
// that an account opened after it keeps the next ID.
//
// Requirement: FR-CHK-046
func TestState_ClosedAccountIsKindAndFlag(t *testing.T) {
	t.Parallel()
	l := tiny(t, 0)
	first, _ := l.Open(KindEscrow, 7, 1)
	second, _ := l.Open(KindAgent, 5, 0)
	if err := l.Close(first); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := l.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction: %v", err)
	}
	l.EndTick()
	b := saved(l)
	// Two accounts; the first is kind 2 (Escrow) and closed: 02 01; the
	// second is kind 0, open, region 0, owner 5, balance 0, no lots.
	want := []byte{2, 2, 1, 0, 0, 0, 5, 0, 0}
	if !bytes.HasPrefix(b, want) {
		t.Errorf("WriteState began % x, want % x", b[:min(len(b), len(want))], want)
	}
	fresh := tiny(t, 0)
	if err := read(fresh, b, 1, false); err != nil {
		t.Fatalf("ReadState: %v", err)
	}
	if v, err := fresh.Account(first); err != nil || !v.Closed || v.Kind != KindEscrow {
		t.Errorf("Account(%d) = %+v, %v, want a closed Escrow account", first, v, err)
	}
	if v, err := fresh.Account(second); err != nil || v.Closed || v.Owner != 5 || v.Region != 0 {
		t.Errorf("Account(%d) = %+v, %v, want the open agent account of owner 5 in region 0", second, v, err)
	}
	if id, err := fresh.Open(KindAgent, 6, 1); err != nil || id != 2 {
		t.Errorf("Open after the restore = %d, %v, want ID 2", id, err)
	}
}

// pendingRig returns a ledger past construction with one funded agent and the
// money boundary of Initial, and the agent's ID, with the totals cleared.
func pendingRig(tb testing.TB) (l *Ledger, mint, agent int) {
	tb.Helper()
	l = tiny(tb, 0)
	var err error
	if mint, err = l.OpenBoundary(Initial, Money); err != nil {
		tb.Fatalf("OpenBoundary: %v", err)
	}
	if agent, err = l.Open(KindAgent, 3, 1); err != nil {
		tb.Fatalf("Open: %v", err)
	}
	if err := l.EndConstruction(); err != nil {
		tb.Fatalf("EndConstruction: %v", err)
	}
	l.EndTick()
	return l, mint, agent
}

// TestState_PendingTotalsAndBaseline verifies that a call between ticks,
// totalled after the last tick's end, is saved as a pending row (region, reason
// code, thing, total), that the restored audit passes with the baseline of the
// boundary taken as the restored balance less the pending total, and that
// ending the tick clears the total and moves the baseline on.
//
// Requirement: FR-CHK-024, FR-CHK-044
func TestState_PendingTotalsAndBaseline(t *testing.T) {
	t.Parallel()
	l, mint, agent := pendingRig(t)
	// An external agent's money: a faucet of 40 in region 1, after the last
	// tick's end, which belongs to the next tick (spec-ledger.md 7.4).
	if err := l.Transfer(mint, agent, 40, Initial); err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	b := saved(l)
	// After the account list (1 account: 01, kind 0, open 00, region 1, owner
	// 3, balance 40, no lots 00 -> 01 00 00 01 03 28 00) and the boundary
	// balance (-40 is 4f in zigzag), the claim block (00 00) and one flow row
	// (01 row: region 1, code 1 (initial), thing 0 (money), total 40).
	wantFlow := []byte{0x4f, 0, 0, 1, 1, 1, 0, 40, 0, 0}
	if !bytes.Contains(b, wantFlow) {
		t.Errorf("WriteState wrote % x, want it to contain the boundary, claims and flow row % x", b, wantFlow)
	}
	fresh := tiny(t, 0)
	if _, err := fresh.OpenBoundary(Initial, Money); err != nil {
		t.Fatalf("OpenBoundary: %v", err)
	}
	if err := read(fresh, b, l.Tick(), false); err != nil {
		t.Fatalf("ReadState: %v", err)
	}
	if err := fresh.Audit(); err != nil {
		t.Errorf("Audit with the pending total: %v", err)
	}
	// The baseline is the balance at the last tick's end: 0, the restored
	// balance of -40 less the -40 that the faucet moved.
	if slot := fresh.off[Initial][thingMoney]; fresh.base[slot] != 0 {
		t.Errorf("baseline of the money boundary = %d, want 0", fresh.base[slot])
	}
	if want, got := normal(l), normal(fresh); !reflect.DeepEqual(want, got) {
		t.Errorf("restored ledger differs from the original")
	}
	for _, x := range []*Ledger{l, fresh} {
		x.EndTick()
		if err := x.Audit(); err != nil {
			t.Errorf("Audit after EndTick: %v", err)
		}
	}
	if slot := fresh.off[Initial][thingMoney]; fresh.base[slot] != -40 || slices.ContainsFunc(fresh.totals, func(v int64) bool { return v != 0 }) {
		t.Errorf("after EndTick baseline = %d and totals %v, want -40 and none", fresh.base[slot], fresh.totals)
	}
}

// TestState_PendingRelocationListAlwaysPresent verifies that the pending
// relocation list is written for every ledger, as a count that is 0 until a
// relocation is recorded, that a non-zero list is accepted only as rows
// sorted by (from, to, thing) between two regions with a total of at least 1,
// that it round-trips, and that the end of the tick clears it.
//
// Requirement: FR-CHK-044
func TestState_PendingRelocationListAlwaysPresent(t *testing.T) {
	t.Parallel()
	empty := tiny(t, 0)
	if err := empty.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction: %v", err)
	}
	// Past construction: 0 accounts, next claim ID 0, 0 claims, 0 flow rows,
	// 0 wear rows, 0 relocation rows: six counts.
	if got, want := saved(empty), make([]byte, 6); !bytes.Equal(got, want) {
		t.Errorf("WriteState of an empty ledger = % x, want % x", got, want)
	}
	build := func(rows [][4]uint64) []byte {
		w := codec.NewWriter(nil)
		for range 5 { // accounts, next claim, claims, flows, wear
			w.U(0)
		}
		w.Count(len(rows))
		for _, r := range rows {
			for _, v := range r {
				w.U(v)
			}
		}
		return w.Out()
	}
	// Row keys: (from, to, thing): money first, then commodity 0 as thing 1.
	valid := [][4]uint64{{0, 1, 0, 5}, {0, 1, 1, 7}, {1, 0, 0, 3}}
	l := tiny(t, 0)
	if err := read(l, build(valid), 0, false); err != nil {
		t.Fatalf("ReadState of a valid list: %v", err)
	}
	if want := []reloc{{0, 1, 0, 5}, {0, 1, 1, 7}, {1, 0, 0, 3}}; !slices.Equal(l.relocs, want) {
		t.Errorf("relocs = %v, want %v", l.relocs, want)
	}
	if got := saved(l); !bytes.Equal(got, build(valid)) {
		t.Errorf("WriteState gave % x, want the bytes read % x", got, build(valid))
	}
	l.EndTick()
	if len(l.relocs) != 0 {
		t.Errorf("relocs after EndTick = %v, want none", l.relocs)
	}
	bad := map[string][][4]uint64{
		"within a region":      {{1, 1, 0, 5}},
		"out of order":         {{1, 0, 0, 5}, {0, 1, 0, 5}},
		"repeated":             {{0, 1, 0, 5}, {0, 1, 0, 5}},
		"thing after thing":    {{0, 1, 1, 5}, {0, 1, 0, 5}},
		"total 0":              {{0, 1, 0, 0}},
		"region out of range":  {{0, 2, 0, 5}},
		"thing out of range":   {{0, 1, 4, 5}},
		"total above int64":    {{0, 1, 0, 1 << 63}},
		"count beyond the end": nil,
	}
	for name, rows := range bad {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			b := build(rows)
			if rows == nil {
				b = append(b[:5], 9) // a relocation count of 9 with no row after it
			}
			if err := read(tiny(t, 0), b, 0, false); !errors.Is(err, codec.ErrMalformed) {
				t.Errorf("ReadState of relocation rows %v = %v, want ErrMalformed", rows, err)
			}
		})
	}
}

// TestState_ConstructionBucketOnlyInConstruction verifies that the
// construction bucket is saved while the ledger is in its construction phase
// and not after, that a restore in construction keeps the bucket and passes the
// audit, that the flag of the section decides whether the bucket is read, and
// that pending totals in construction are refused.
//
// Requirement: FR-CHK-026, FR-CHK-044
func TestState_ConstructionBucketOnlyInConstruction(t *testing.T) {
	t.Parallel()
	g := newRig(t, scriptMaxMoney)
	g.fund(t)
	l := g.l
	inConstruction := saved(l)
	fresh := freshLike(t, g)
	if err := read(fresh, inConstruction, 0, true); err != nil {
		t.Fatalf("ReadState in construction: %v", err)
	}
	if !fresh.InConstruction() || fresh.cleared {
		t.Errorf("restored in construction: InConstruction %v, cleared %v, want true and false", fresh.InConstruction(), fresh.cleared)
	}
	if err := fresh.Audit(); err != nil {
		t.Errorf("Audit of the ledger restored in construction: %v", err)
	}
	if !reflect.DeepEqual(normal(l), normal(fresh)) {
		t.Errorf("ledger restored in construction differs from the original")
	}
	if !slices.ContainsFunc(fresh.bucket, func(v int64) bool { return v != 0 }) {
		t.Errorf("the restored construction bucket is empty, want the funding")
	}
	if err := l.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction: %v", err)
	}
	l.EndTick()
	past := saved(l)
	if len(past) >= len(inConstruction) {
		t.Errorf("saved %d bytes past construction and %d in it, want the bucket gone", len(past), len(inConstruction))
	}
	for name, tt := range map[string]struct {
		b            []byte
		construction bool
	}{
		"bucket bytes read as past construction": {inConstruction, false},
		"no bucket bytes read in construction":   {past, true},
	} {
		if err := read(freshLike(t, g), tt.b, 1, tt.construction); !errors.Is(err, codec.ErrMalformed) {
			t.Errorf("%s: ReadState = %v, want ErrMalformed", name, err)
		}
	}
	// Pending wear units (commodity 0, 5 units) in a ledger that is in
	// construction: refused; the same bytes without the bucket are not.
	w := codec.NewWriter(nil)
	for range 3 { // accounts, next claim, claims
		w.U(0)
	}
	w.U(0) // flow rows
	w.U(1) // wear rows
	w.U(0)
	w.U(5)
	w.U(0) // relocation rows
	noBucket := slices.Clone(w.Out())
	w.U(0) // bucket flow rows
	w.U(0) // bucket wear rows
	if err := read(tiny(t, 0), w.Out(), 0, true); !errors.Is(err, codec.ErrMalformed) {
		t.Errorf("ReadState with pending wear in construction = %v, want ErrMalformed", err)
	}
	if err := read(tiny(t, 0), noBucket, 1, false); err != nil {
		t.Errorf("ReadState of the same pending wear past construction = %v, want nil", err)
	}
}

// TestState_ConstructionRefusesAStepTaken verifies that a section flagged as in
// construction with a tick above 0 is refused: construction ends at the first
// step, so the pair cannot come from a world.
//
// Requirement: FR-CHK-026
func TestState_ConstructionRefusesAStepTaken(t *testing.T) {
	t.Parallel()
	g := newRig(t, scriptMaxMoney)
	g.fund(t)
	if err := read(freshLike(t, g), saved(g.l), 1, true); !errors.Is(err, codec.ErrMalformed) {
		t.Errorf("ReadState(construction, tick 1) = %v, want ErrMalformed", err)
	}
}

// craft writes a section by hand.
func craft(f func(w *codec.Writer)) []byte {
	w := codec.NewWriter(nil)
	f(w)
	return w.Out()
}

// acct writes one open account of the given kind, region, owner and
// balance, followed by lots of (commodity, quantity, born, durability).
func acct(w *codec.Writer, kind, region, owner, balance uint64, lots ...[4]int64) {
	w.U(kind)
	w.B(false)
	w.U(region)
	w.U(owner)
	w.U(balance)
	w.Count(len(lots))
	for _, lt := range lots {
		w.U(uint64(lt[0]))
		w.U(uint64(lt[1]))
		w.I(lt[2])
		w.U(uint64(lt[3]))
	}
}

// tail writes the rest of an empty section past construction: the claim block
// of a ledger that holds no claim, no flow, no wear and no relocation.
func tail(w *codec.Writer) {
	for range 5 { // next claim, claims, flow rows, wear rows, relocation rows
		w.U(0)
	}
}

// TestState_Corrupt verifies that a section that breaks one rule is refused
// with codec.ErrMalformed and never panics, one rule to a case, on a ledger
// of three commodities (one spoils with a shelf life of 3, one wears with a
// durability of 5) at tick 10 with bounds of 1000 money and 100 units. The
// first case is a valid section, so the others fail for their own rule.
//
// Requirement: FR-CHK-012, FR-CHK-013, FR-CHK-014
func TestState_Corrupt(t *testing.T) {
	t.Parallel()
	one := func(f func(w *codec.Writer)) func(w *codec.Writer) {
		return func(w *codec.Writer) { w.Count(1); f(w); tail(w) }
	}
	claim := func(creditor, debtor, origin uint64, ids ...uint64) func(w *codec.Writer) {
		return func(w *codec.Writer) {
			w.Count(2) // two open accounts for the parties
			acct(w, 0, 0, 1, 0)
			acct(w, 0, 1, 2, 0)
			w.U(5) // next claim ID
			w.Count(len(ids))
			for range 7 {
				w.U(0) // totals
			}
			for _, id := range ids {
				w.U(id)
				w.U(creditor)
				w.U(debtor)
				for _, v := range []uint64{10, 1, 10, 1, 500, 3, 9} { // principal, interest, owed, owed, rate, formed, due
					w.U(v)
				}
				w.I(-1)
				w.U(origin)
			}
			w.U(0) // flow rows
			w.U(0) // wear rows
			w.U(0) // relocation rows
		}
	}
	flow := func(rows ...[4]uint64) func(w *codec.Writer) {
		return func(w *codec.Writer) {
			w.Count(0) // accounts
			w.U(0)     // next claim
			w.U(0)     // claims
			w.Count(len(rows))
			for _, r := range rows {
				for _, v := range r {
					w.U(v)
				}
			}
			w.U(0) // wear
			w.U(0) // relocations
		}
	}
	wear := func(rows ...[2]uint64) func(w *codec.Writer) {
		return func(w *codec.Writer) {
			w.Count(0)
			w.U(0)
			w.U(0)
			w.U(0) // flows
			w.Count(len(rows))
			for _, r := range rows {
				w.U(r[0])
				w.U(r[1])
			}
			w.U(0)
		}
	}
	tests := []struct {
		name      string
		build     func(w *codec.Writer)
		maxClaims int
		tick      int64
		valid     bool
	}{
		{"a valid section", one(func(w *codec.Writer) {
			acct(w, 0, 1, 4, 1000, [4]int64{0, 60, 2, 0}, [4]int64{1, 100, 0, 5}, [4]int64{2, 7, 0, 0})
		}), 0, 10, true},
		{"balance above MaxMoney", one(func(w *codec.Writer) { acct(w, 0, 0, 0, 1001) }), 0, 10, false},
		{"region out of range", one(func(w *codec.Writer) { acct(w, 0, 2, 0, 0) }), 0, 10, false},
		{"kind of a boundary", one(func(w *codec.Writer) { acct(w, uint64(KindBoundary), 0, 0, 0) }), 0, 10, false},
		{"kind that is no kind", one(func(w *codec.Writer) { acct(w, 9, 0, 0, 0) }), 0, 10, false},
		{"owner above int64", one(func(w *codec.Writer) { acct(w, 0, 0, 1<<63, 0) }), 0, 10, false},
		{"lot of an unknown commodity", one(func(w *codec.Writer) { acct(w, 0, 0, 0, 0, [4]int64{3, 1, 0, 0}) }), 0, 10, false},
		{"lot with quantity 0", one(func(w *codec.Writer) { acct(w, 0, 0, 0, 0, [4]int64{2, 0, 0, 0}) }), 0, 10, false},
		{"lot above MaxQuantity", one(func(w *codec.Writer) { acct(w, 0, 0, 0, 0, [4]int64{2, 101, 0, 0}) }), 0, 10, false},
		{"commodity above MaxQuantity over two lots", one(func(w *codec.Writer) {
			acct(w, 0, 0, 0, 0, [4]int64{0, 60, 1, 0}, [4]int64{0, 60, 2, 0})
		}), 0, 10, false},
		{"lots out of canonical order", one(func(w *codec.Writer) {
			acct(w, 0, 0, 0, 0, [4]int64{1, 1, 0, 5}, [4]int64{0, 1, 1, 0})
		}), 0, 10, false},
		{"born out of order inside a commodity", one(func(w *codec.Writer) {
			acct(w, 0, 0, 0, 0, [4]int64{0, 1, 2, 0}, [4]int64{0, 1, 1, 0})
		}), 0, 10, false},
		{"repeated lot key", one(func(w *codec.Writer) {
			acct(w, 0, 0, 0, 0, [4]int64{1, 1, 0, 5}, [4]int64{1, 1, 0, 5})
		}), 0, 10, false},
		{"born before the shelf life allows", one(func(w *codec.Writer) { acct(w, 0, 0, 0, 0, [4]int64{0, 1, -3, 0}) }), 0, 10, false},
		{"born in the future", one(func(w *codec.Writer) { acct(w, 0, 0, 0, 0, [4]int64{0, 1, 11, 0}) }), 0, 10, false},
		{"born of a commodity that does not spoil", one(func(w *codec.Writer) { acct(w, 0, 0, 0, 0, [4]int64{2, 1, 1, 0}) }), 0, 10, false},
		{"durability above the commodity's", one(func(w *codec.Writer) { acct(w, 0, 0, 0, 0, [4]int64{1, 1, 0, 6}) }), 0, 10, false},
		{"durability 0 of a wearing commodity", one(func(w *codec.Writer) { acct(w, 0, 0, 0, 0, [4]int64{1, 1, 0, 0}) }), 0, 10, false},
		{"closed account that carries a balance", func(w *codec.Writer) {
			w.Count(1)
			w.U(0)
			w.B(true)
			w.U(5) // a balance a closed account does not have
			tail(w)
		}, 0, 10, false},
		{"claims beyond MaxClaims", func(w *codec.Writer) {
			w.Count(0)
			w.U(3)
			w.Count(1) // one claim, and MaxClaims is 0
			w.U(0)
			w.U(0)
			w.U(0)
		}, 0, 10, false},
		{"valid claims", claim(0, 1, 0, 2, 4), 2, 10, true},
		{"claim IDs repeated", claim(0, 1, 0, 3, 3), 2, 10, false},
		{"claim IDs descending", claim(0, 1, 0, 4, 2), 2, 10, false},
		{"claim ID equal to the next ID", claim(0, 1, 0, 5), 2, 10, false},
		{"claim ID above the next ID", claim(0, 1, 0, 6), 2, 10, false},
		{"claim creditor outside the ledger", claim(2, 1, 0, 2), 2, 10, false},
		{"claim debtor outside the ledger", claim(0, 9, 0, 2), 2, 10, false},
		{"claim origin out of range", claim(0, 1, 2, 2), 2, 10, false},
		{"more claims than MaxClaims", claim(0, 1, 0, 1, 2, 3), 2, 10, false},
		{"valid flow row", flow([4]uint64{1, 1, 0, 40}), 0, 10, true},
		{"flow code no reason has", flow([4]uint64{0, 6, 0, 1}), 0, 10, false},
		{"flow code above the table", flow([4]uint64{0, 42, 0, 1}), 0, 10, false},
		{"flow of a thing the reason does not move", flow([4]uint64{0, 8, 0, 1}), 0, 10, false},
		{"flow rows out of order", flow([4]uint64{0, 1, 0, 1}, [4]uint64{0, 0, 0, 1}), 0, 10, false},
		{"flow row repeated", flow([4]uint64{0, 1, 0, 1}, [4]uint64{0, 1, 0, 1}), 0, 10, false},
		{"flow total 0", flow([4]uint64{0, 1, 0, 0}), 0, 10, false},
		{"flow region out of range", flow([4]uint64{2, 1, 0, 1}), 0, 10, false},
		{"flow thing out of range", flow([4]uint64{0, 1, 4, 1}), 0, 10, false},
		{"flow total above int64", flow([4]uint64{0, 1, 0, 1 << 63}), 0, 10, false},
		{"flow thing a commodity short of the table", flow([4]uint64{0, 0, 4, 1}), 0, 10, false},
		{"valid wear rows", wear([2]uint64{0, 3}, [2]uint64{2, 1}), 0, 10, true},
		{"wear rows out of order", wear([2]uint64{2, 3}, [2]uint64{0, 1}), 0, 10, false},
		{"wear commodity out of range", wear([2]uint64{3, 3}), 0, 10, false},
		{"wear commodity far out of range", wear([2]uint64{4, 3}), 0, 10, false},
		{"wear units 0", wear([2]uint64{0, 0}), 0, 10, false},
		{"lots after a bad lot", one(func(w *codec.Writer) {
			acct(w, 0, 0, 0, 0, [4]int64{2, 0, 0, 0}, [4]int64{2, 1, 0, 0})
		}), 0, 10, false},
		{"claims after a bad claim", claim(2, 1, 0, 2, 4), 2, 10, false},
		{"flow rows after a bad row", flow([4]uint64{2, 1, 0, 1}, [4]uint64{2, 1, 0, 2}), 0, 10, false},
		{"wear rows after a bad row", wear([2]uint64{4, 3}, [2]uint64{5, 1}), 0, 10, false},
		{"relocation rows after a bad row", func(w *codec.Writer) {
			w.Count(0)
			for range 4 { // next claim, claims, flow rows, wear rows
				w.U(0)
			}
			w.Count(2)
			for _, v := range []uint64{1, 1, 0, 5, 0, 1, 0, 5} { // a row within a region, then a valid one
				w.U(v)
			}
		}, 0, 10, false},
		{"accounts after a bad account", func(w *codec.Writer) {
			w.Count(2)
			acct(w, 0, 2, 0, 0)
			acct(w, 0, 0, 0, 0)
			tail(w)
		}, 0, 10, false},
		{"count beyond the bytes", func(w *codec.Writer) { w.U(1 << 40) }, 0, 10, false},
		{"trailing byte", func(w *codec.Writer) { w.Count(0); tail(w); w.U(0) }, 0, 10, false},
		{"truncated", func(w *codec.Writer) { w.Count(0) }, 0, 10, false},
		{"negative tick", func(w *codec.Writer) { w.Count(0); tail(w) }, 0, -1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := read(tiny(t, tt.maxClaims), craft(tt.build), tt.tick, false)
			switch {
			case tt.valid && err != nil:
				t.Errorf("ReadState = %v, want nil", err)
			case !tt.valid && !errors.Is(err, codec.ErrMalformed):
				t.Errorf("ReadState = %v, want ErrMalformed", err)
			}
		})
	}
}

// TestState_EveryPrefixIsMalformed verifies that every proper prefix of a
// real section is refused with ErrMalformed, and never panics, and that the
// section with one more byte is refused as trailing.
//
// Requirement: FR-CHK-013, FR-CHK-016
func TestState_EveryPrefixIsMalformed(t *testing.T) {
	t.Parallel()
	g := newRig(t, scriptMaxMoney)
	g.fund(t)
	s := newScripter(g, &randChooser{r: rand.New(rand.NewPCG(3, 11)), left: 600})
	if err := s.run(); err != nil {
		t.Fatalf("script: %v", err)
	}
	settle(t, s)
	b := saved(g.l)
	tick, construction := g.l.Tick(), g.l.InConstruction()
	for n := range b {
		if err := read(freshLike(t, g), b[:n], tick, construction); !errors.Is(err, codec.ErrMalformed) {
			t.Fatalf("ReadState of the first %d of %d bytes = %v, want ErrMalformed", n, len(b), err)
		}
	}
	if err := read(freshLike(t, g), append(slices.Clone(b), 0), tick, construction); !errors.Is(err, codec.ErrMalformed) {
		t.Errorf("ReadState with a trailing byte = %v, want ErrMalformed", err)
	}
}

// panicsWithInvariant runs f and reports whether it panicked with a message
// that names a violated invariant.
func panicsWithInvariant(f func()) (ok bool) {
	defer func() {
		s, _ := recover().(string)
		ok = strings.Contains(s, "invariant")
	}()
	f()
	return false
}

// TestState_WriteRefusesWhatCannotBeBetweenTicks verifies that a state no
// world has between ticks stops WriteState with a violated invariant instead
// of losing it: a held reservation of money, an earmark of goods, a boundary
// account that follows another kind (the restore could not give it its ID) and
// a negative owner reference.
//
// Guard: reservations are not in the format (spec-checkpoint.md 4.2) and
// account IDs are positions; writing any of these silently would restore a
// ledger that disagrees with the original.
func TestState_WriteRefusesWhatCannotBeBetweenTicks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		build func(t *testing.T) *Ledger
	}{
		{"money reserved", func(t *testing.T) *Ledger {
			l, mint, agent := pendingRig(t)
			if err := l.Transfer(mint, agent, 40, Initial); err != nil {
				t.Fatalf("Transfer: %v", err)
			}
			if err := l.ReserveMoney(agent, 10); err != nil {
				t.Fatalf("ReserveMoney: %v", err)
			}
			return l
		}},
		{"goods earmarked", func(t *testing.T) *Ledger {
			l := tiny(t, 0)
			mint, _ := l.OpenBoundary(Initial, 2)
			agent, _ := l.Open(KindAgent, 3, 0)
			if err := l.Move(mint, agent, 2, 5, Initial); err != nil {
				t.Fatalf("Move: %v", err)
			}
			if err := l.ReserveGoods(agent, 2, 3); err != nil {
				t.Fatalf("ReserveGoods: %v", err)
			}
			return l
		}},
		{"boundary after an agent", func(t *testing.T) *Ledger {
			l := tiny(t, 0)
			if _, err := l.Open(KindAgent, 3, 0); err != nil {
				t.Fatalf("Open: %v", err)
			}
			if _, err := l.OpenBoundary(Initial, Money); err != nil {
				t.Fatalf("OpenBoundary: %v", err)
			}
			return l
		}},
		{"negative owner", func(t *testing.T) *Ledger {
			l := tiny(t, 0)
			if _, err := l.Open(KindAgent, -1, 0); err != nil {
				t.Fatalf("Open: %v", err)
			}
			return l
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			l := tt.build(t)
			if !panicsWithInvariant(func() { saved(l) }) {
				t.Errorf("WriteState did not stop with a violated invariant")
			}
		})
	}
}

// TestRestore_RebuiltHints verifies the claim of FR-LED-030 through the new
// path: a ledger read back by ReadState, with its hints rebuilt from its lots,
// continues exactly as the uninterrupted ledger does through further ticks of
// ageing, the totals and the state equal at every tick.
//
// Requirement: FR-LED-030
func TestRestore_RebuiltHints(t *testing.T) {
	t.Parallel()
	for seed := uint64(1); seed <= 10; seed++ {
		g := newRig(t, scriptMaxMoney)
		g.fund(t)
		s := newScripter(g, &randChooser{r: rand.New(rand.NewPCG(seed, 5)), left: 200})
		if err := s.run(); err != nil {
			t.Fatalf("seed %d: script: %v", seed, err)
		}
		if err := s.endTick(); err != nil {
			t.Fatalf("seed %d: endTick: %v", seed, err)
		}
		a := g.l
		b := freshLike(t, g)
		if err := read(b, saved(a), a.Tick(), false); err != nil {
			t.Fatalf("seed %d: ReadState: %v", seed, err)
		}
		for tick := 0; tick < 12; tick++ {
			for _, l := range []*Ledger{a, b} {
				if err := l.Age(); err != nil {
					t.Fatalf("seed %d: Age: %v", seed, err)
				}
				l.EndTick()
			}
			if !reflect.DeepEqual(normal(a), normal(b)) {
				t.Fatalf("seed %d: restored ledger diverged from the uninterrupted one after %d ticks", seed, tick+1)
			}
		}
	}
}

// pinnedNames is the table of spec-ledger.md 7.2 by position: the code of a
// reason in a checkpoint is the index of its name (spec-checkpoint.md 4.2.7).
var pinnedNames = [...]string{
	"trade", "initial", "recipe_input", "recipe_output", "disposal", "consumption",
	"want_consumption", "spoilage", "wear", "estate_money", "estate_goods", "start_money",
	"relief", "outside_budget", "outside_unspent", "outside_receipts", "outside_offer",
	"outside_unsold", "outside_purchases", "escrow_in", "escrow_out", "escrow_refund",
	"wage_escrow", "wage_pay", "wage_refund", "loan_out", "loan_repay", "loan_interest",
	"rent", "title_sale", "premium", "payout", "transit_out", "transit_in", "haul_use",
	"transit_use", "loss", "split", "merge", "births", "starvation", "deaths",
}

// reasonNames names the reasons this ledger has, in the spec's spelling.
var reasonNames = map[Reason]string{
	Trade: "trade", Initial: "initial", RecipeInput: "recipe_input", RecipeOutput: "recipe_output",
	Disposal: "disposal", Consumption: "consumption", Spoilage: "spoilage", Wear: "wear",
	EstateMoney: "estate_money", EstateGoods: "estate_goods", StartMoney: "start_money",
	Split: "split", Merge: "merge", Births: "births", Starvation: "starvation",
	LoanOut: "loan_out", LoanRepay: "loan_repay", LoanInterest: "loan_interest",
}

// TestReasonCode_Pinned verifies that the code of every reason is the position
// of its name in the table of spec-ledger.md 7.2, that the table in the spec is
// the list this test pins, that the codes are distinct and invertible, and that
// a code no reason has is refused.
//
// Guard: a reason is saved by its code, never by the Go number (N-225). If a
// constant were renumbered, or a row inserted in the spec's table, saved
// checkpoints would restore other reasons without notice.
func TestReasonCode_Pinned(t *testing.T) {
	t.Parallel()
	if len(pinnedNames) != pinnedRows {
		t.Fatalf("the pinned list has %d names, pinnedRows is %d", len(pinnedNames), pinnedRows)
	}
	if len(reasonNames) != int(NumReasons) {
		t.Errorf("reasonNames names %d reasons, the ledger has %d: name the new reason and pin its row", len(reasonNames), NumReasons)
	}
	seen := map[uint64]Reason{}
	for r := Reason(0); r < NumReasons; r++ {
		name := reasonNames[r]
		code := reasonCode(r)
		if code >= pinnedRows || pinnedNames[code] != name {
			t.Errorf("reasonCode(%d, %q) = %d, want the position of %q in the pinned table", r, name, code, name)
		}
		if other, dup := seen[code]; dup {
			t.Errorf("reasons %d and %d have the code %d", other, r, code)
		}
		seen[code] = r
		if back, ok := reasonFromCode(code); !ok || back != r {
			t.Errorf("reasonFromCode(%d) = %d, %v, want %d", code, back, ok, r)
		}
	}
	for code := uint64(0); code <= pinnedRows+1; code++ {
		if _, has := seen[code]; has {
			continue
		}
		if r, ok := reasonFromCode(code); ok {
			t.Errorf("reasonFromCode(%d) = %d, want no reason", code, r)
		}
	}
	if _, ok := reasonFromCode(math.MaxUint64); ok {
		t.Errorf("reasonFromCode(MaxUint64) found a reason")
	}
}

// TestReasonCode_TableOfTheSpec verifies that the pinned list is the table of
// spec-ledger.md 7.2 as it stands: the names of the rows in order, a name that
// two rows share (money and goods) counted once, the rows of configured flows
// (which have no name) left out.
//
// Guard: spec-checkpoint.md 4.2.7 says the code is the position in that table,
// so a row inserted in the middle of the table is a format change that this
// test must catch.
func TestReasonCode_TableOfTheSpec(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("../../doc/specs/spec-ledger.md")
	if err != nil {
		t.Fatalf("read the spec: %v", err)
	}
	text := string(b)
	i := strings.Index(text, "The starting registry:")
	j := strings.Index(text, "A spec that needs a new flow adds a row.")
	if i < 0 || j < i {
		t.Fatalf("the table of 7.2 was not found in spec-ledger.md")
	}
	name := regexp.MustCompile("`([a-z_]+)`")
	var got []string
	for _, line := range strings.Split(text[i:j], "\n") {
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		cell := strings.SplitN(line, " | ", 2)[0]
		for _, m := range name.FindAllStringSubmatch(cell, -1) {
			if len(got) == 0 || got[len(got)-1] != m[1] {
				got = append(got, m[1])
			}
		}
	}
	if !slices.Equal(got, pinnedNames[:]) {
		t.Errorf("the table of spec-ledger.md 7.2 lists %v, want the pinned %v", got, pinnedNames)
	}
}

// TestState_BaselineOverflowIsMalformed verifies that pending totals that
// cannot be taken off a boundary balance without leaving int64, whether the
// totals of two regions add past it or the balance and the total do, are
// refused as malformed instead of wrapping into a baseline.
//
// Guard: the baseline of the audit is arithmetic on input; a wrapped baseline
// would make a corrupt section pass the conservation identity.
func TestState_BaselineOverflowIsMalformed(t *testing.T) {
	t.Parallel()
	stream := func(balance int64, rows ...[4]uint64) []byte {
		return craft(func(w *codec.Writer) {
			w.Count(0) // accounts
			w.I(balance)
			w.U(0) // next claim
			w.U(0) // claims
			w.Count(len(rows))
			for _, r := range rows {
				for _, v := range r {
					w.U(v)
				}
			}
			w.U(0) // wear
			w.U(0) // relocations
		})
	}
	tests := []struct {
		name string
		b    []byte
	}{
		{"totals of two regions add past int64", stream(0, [4]uint64{0, 1, 0, math.MaxInt64}, [4]uint64{1, 1, 0, math.MaxInt64})},
		{"balance and total leave int64", stream(math.MaxInt64, [4]uint64{0, 1, 0, 5})},
		{"the same stream within range", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			l := tiny(t, 0)
			if _, err := l.OpenBoundary(Initial, Money); err != nil {
				t.Fatalf("OpenBoundary: %v", err)
			}
			b, want := tt.b, error(codec.ErrMalformed)
			if b == nil {
				b, want = stream(-5, [4]uint64{0, 1, 0, 5}), nil
			}
			if err := read(l, b, 1, false); !errors.Is(err, want) && err != want {
				t.Errorf("ReadState = %v, want %v", err, want)
			}
		})
	}
}

// TestState_NeedsAFreshLedger verifies that reading into a ledger that is not
// as the root builds it for a restore, one with a non-boundary account, one
// past its tick 0 or one that has left construction, stops with a violated
// invariant: it is a programmer error and not an input.
//
// Guard: ReadState appends accounts after the boundary accounts, so IDs match
// the writer's only if nothing else is open.
func TestState_NeedsAFreshLedger(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		make func(l *Ledger)
	}{
		{"an account is open", func(l *Ledger) { _, _ = l.Open(KindAgent, 0, 0) }},
		{"the tick is past 0", func(l *Ledger) { l.AdvanceTick() }},
		{"construction has ended", func(l *Ledger) { _ = l.EndConstruction() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			l := tiny(t, 0)
			tt.make(l)
			b := craft(func(w *codec.Writer) { w.Count(0); tail(w) })
			if !panicsWithInvariant(func() { _ = read(l, b, 0, false) }) {
				t.Errorf("ReadState did not stop with a violated invariant")
			}
		})
	}
}

// TestState_NoCommodityNoPanic verifies that a lot in a section read into a
// ledger with no commodity at all is refused as malformed and does not index
// the empty table.
//
// Guard: an index that failed its bound reads as 0, which is a valid index
// whenever there is a commodity; with none, the lot reader must stop first.
func TestState_NoCommodityNoPanic(t *testing.T) {
	t.Parallel()
	l, err := New(Config{Regions: 1, MaxMoney: 10, MaxQuantity: 10})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	b := craft(func(w *codec.Writer) {
		w.Count(1)
		acct(w, 0, 0, 0, 0, [4]int64{0, 1, 0, 0})
		tail(w)
	})
	if err := read(l, b, 0, false); !errors.Is(err, codec.ErrMalformed) {
		t.Errorf("ReadState = %v, want ErrMalformed", err)
	}
}

// TestState_ExtremesAreAccepted verifies the largest legal sections of each
// shape: a lot on both bounds of its born range, the most lots an account
// can hold in the ranges of its commodities, many closed accounts and many
// claims at the fewest bytes each, and every legal row of the pending flows,
// the wear units and the relocation totals in key order, which must read back
// to the same bytes. A bound on the bytes an element needs that is too tight
// would refuse them, and a key that does not order the rows would too.
//
// Requirement: FR-CHK-013, FR-CHK-014
func TestState_ExtremesAreAccepted(t *testing.T) {
	t.Parallel()
	t.Run("lots on the bounds", func(t *testing.T) {
		t.Parallel()
		// Commodity 0 spoils with a shelf life of 3, so born runs from -2 to
		// the tick, 10: 13 lots; commodity 1 wears to 5: 5 lots; commodity 2
		// is one lot. 19 lots of 4 bytes each.
		var lots [][4]int64
		for born := int64(-2); born <= 10; born++ {
			lots = append(lots, [4]int64{0, 1, born, 0})
		}
		for d := int64(1); d <= 5; d++ {
			lots = append(lots, [4]int64{1, 1, 0, d})
		}
		lots = append(lots, [4]int64{2, 1, 0, 0})
		b := craft(func(w *codec.Writer) {
			w.Count(1)
			acct(w, 0, 1, 4, 0, lots...)
			tail(w)
		})
		l := tiny(t, 0)
		if err := read(l, b, 10, false); err != nil {
			t.Fatalf("ReadState = %v, want nil", err)
		}
		if got := saved(l); !bytes.Equal(got, b) {
			t.Errorf("WriteState gave % x, want the bytes read % x", got, b)
		}
	})
	t.Run("closed accounts", func(t *testing.T) {
		t.Parallel()
		b := craft(func(w *codec.Writer) {
			w.Count(8)
			for k := range 8 {
				w.U(uint64(k % 6))
				w.B(true)
			}
			tail(w)
		})
		l := tiny(t, 0)
		if err := read(l, b, 1, false); err != nil || l.Len() != 8 {
			t.Errorf("ReadState = %v with %d accounts, want nil and 8", err, l.Len())
		}
	})
	t.Run("claims of the fewest bytes", func(t *testing.T) {
		t.Parallel()
		b := craft(func(w *codec.Writer) {
			w.Count(2)
			acct(w, 0, 0, 1, 0)
			acct(w, 0, 1, 2, 0)
			w.U(20) // next claim ID
			w.Count(11)
			for range 7 {
				w.U(0) // totals
			}
			for id := uint64(1); id <= 11; id++ {
				w.U(id)
				w.U(0) // creditor
				w.U(1) // debtor
				for _, v := range []uint64{1, 0, 1, 0, 0, 0, 1} {
					w.U(v) // principal, interest, owed, owed, rate, formed, due
				}
				w.I(-1)
				w.U(0) // origin
			}
			w.U(0)
			w.U(0)
			w.U(0)
		})
		l := tiny(t, 12)
		if err := read(l, b, 1, false); err != nil || len(l.claims) != 11 {
			t.Errorf("ReadState = %v with %d claims, want nil and 11", err, len(l.claims))
		}
		if got := saved(l); !bytes.Equal(got, b) {
			t.Errorf("WriteState gave % x, want the bytes read % x", got, b)
		}
	})
	t.Run("every row in key order", func(t *testing.T) {
		t.Parallel()
		l := tiny(t, 0)
		var flows, wears, relocs [][]uint64
		total := uint64(0)
		for region := 0; region < l.regions; region++ {
			for code := uint64(0); code < pinnedRows; code++ {
				r, ok := reasonFromCode(code)
				for thing := 0; ok && thing <= len(l.comms); thing++ {
					if _, err := l.slotOf(r, thing-1); err == nil {
						total++
						flows = append(flows, []uint64{uint64(region), code, uint64(thing), total})
					}
				}
			}
		}
		for c := range l.comms {
			wears = append(wears, []uint64{uint64(c), uint64(c) + 1})
		}
		for from := 0; from < l.regions; from++ {
			for to := 0; to < l.regions; to++ {
				for thing := 0; from != to && thing <= len(l.comms); thing++ {
					total++
					relocs = append(relocs, []uint64{uint64(from), uint64(to), uint64(thing), total})
				}
			}
		}
		b := craft(func(w *codec.Writer) {
			w.Count(0) // accounts
			w.U(0)     // next claim
			w.U(0)     // claims
			for _, rows := range [][][]uint64{flows, wears, relocs} {
				w.Count(len(rows))
				for _, row := range rows {
					for _, v := range row {
						w.U(v)
					}
				}
			}
		})
		if err := read(l, b, 1, false); err != nil {
			t.Fatalf("ReadState of %d flow rows, %d wear rows and %d relocation rows = %v, want nil", len(flows), len(wears), len(relocs), err)
		}
		if got := saved(l); !bytes.Equal(got, b) {
			t.Errorf("WriteState gave % x, want the bytes read % x", got, b)
		}
	})
}

// TestState_PendingInConstructionIsRefused verifies that pending totals of any
// two kinds together in a ledger that is in construction are refused, however
// the counts add up: a flow row and a wear row, or a wear row and a relocation
// row. The bytes are the same with the construction flag off, where they are
// accepted.
//
// Requirement: FR-CHK-026, FR-CHK-044
func TestState_PendingInConstructionIsRefused(t *testing.T) {
	t.Parallel()
	pending := func(flows, wear, relocs int) func(w *codec.Writer) {
		return func(w *codec.Writer) {
			w.Count(0)
			w.U(0)
			w.U(0)
			w.Count(flows)
			for range flows {
				w.U(0) // region
				w.U(0) // trade
				w.U(0) // money
				w.U(1) // total
			}
			w.Count(wear)
			for range wear {
				w.U(0)
				w.U(1)
			}
			w.Count(relocs)
			for range relocs {
				w.U(0)
				w.U(1)
				w.U(0)
				w.U(1)
			}
		}
	}
	for name, rows := range map[string][3]int{"flow and wear": {1, 1, 0}, "wear and relocation": {0, 1, 1}} {
		past := craft(pending(rows[0], rows[1], rows[2]))
		inConstruction := append(slices.Clone(past), 0, 0) // an empty bucket
		if err := read(tiny(t, 0), past, 1, false); err != nil {
			t.Errorf("%s: ReadState past construction = %v, want nil", name, err)
		}
		if err := read(tiny(t, 0), inConstruction, 0, true); !errors.Is(err, codec.ErrMalformed) {
			t.Errorf("%s: ReadState in construction = %v, want ErrMalformed", name, err)
		}
	}
}

// FuzzReadState feeds arbitrary bytes to ReadState on a small ledger, and
// asserts that nothing panics and that every section it accepts is written
// back as the same bytes (one spelling, FR-CHK-005), whether the ledger is
// restored in construction or past it.
//
// Requirement: NFR-CHK-003
func FuzzReadState(f *testing.F) {
	valid := craft(func(w *codec.Writer) {
		w.Count(2)
		acct(w, 0, 1, 4, 900, [4]int64{0, 5, 2, 0}, [4]int64{1, 3, 0, 5})
		acct(w, 1, 0, 0, 10)
		w.U(3) // next claim
		w.Count(1)
		for range 7 {
			w.U(0)
		}
		w.U(1)
		w.U(0)
		w.U(1)
		for _, v := range []uint64{10, 1, 10, 1, 500, 3, 9} {
			w.U(v)
		}
		w.I(-1)
		w.U(0)
		w.Count(1) // flows
		for _, v := range []uint64{1, 1, 0, 40} {
			w.U(v)
		}
		w.Count(1) // wear
		w.U(0)
		w.U(5)
		w.Count(1) // relocations
		for _, v := range []uint64{0, 1, 0, 7} {
			w.U(v)
		}
	})
	f.Add(valid, false)
	f.Add(append(slices.Clone(valid), 0, 0), true)
	f.Add([]byte{0, 0, 0, 0, 0, 0}, false)
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}, false)
	f.Fuzz(func(t *testing.T, data []byte, construction bool) {
		l := tiny(t, 3)
		if err := read(l, data, 10, construction); err != nil {
			return
		}
		if again := saved(l); !bytes.Equal(again, data) {
			t.Fatalf("accepted % x and wrote % x", data, again)
		}
	})
}
