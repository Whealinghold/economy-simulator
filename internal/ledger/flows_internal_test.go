package ledger

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
)

// flowLedger returns a one-region, one-commodity ledger with the given flows
// and declared reasons, and the IDs of the flow boundaries opened in flow order.
func flowLedger(tb testing.TB, flows []Class, declared ...Reason) (l *Ledger, boundaries []int) {
	tb.Helper()
	l, err := New(Config{
		Commodities: []Commodity{{}}, Regions: 1, MaxMoney: 1000, MaxQuantity: 100,
		Declared: declared, Flows: flows, MaxClaims: 2,
	})
	if err != nil {
		tb.Fatalf("New: %v", err)
	}
	for i := range flows {
		id, err := l.OpenBoundary(FlowReason(i), Money)
		if err != nil {
			tb.Fatalf("OpenBoundary(flow %d): %v", i, err)
		}
		boundaries = append(boundaries, id)
	}
	return l, boundaries
}

// TestNew_FlowReasonsOpenBoundariesInOrder verifies that each flow of the
// configuration has the reason FlowReason(i), a money boundary opened once
// whose ID follows the order it is opened in, and no goods row; and that a
// configuration that is neither a faucet nor a sink, or has too many flows, is
// refused.
//
// Requirement: FR-MON-003, FR-MON-029
func TestNew_FlowReasonsOpenBoundariesInOrder(t *testing.T) {
	t.Parallel()
	l, ids := flowLedger(t, []Class{Faucet, Sink, Faucet})
	for i, id := range ids {
		if id != i {
			t.Errorf("OpenBoundary(FlowReason(%d)) = account %d, want %d (opened in flow order)", i, id, i)
		}
		if got, ok := l.Boundary(FlowReason(i), Money); !ok || got != id {
			t.Errorf("Boundary(FlowReason(%d)) = %d, %v, want %d, true", i, got, ok, id)
		}
	}
	if _, err := l.OpenBoundary(FlowReason(1), Money); !errors.Is(err, ErrDuplicateBoundary) {
		t.Errorf("second OpenBoundary(flow 1) = %v, want ErrDuplicateBoundary", err)
	}
	if _, err := l.OpenBoundary(FlowReason(0), 0); !errors.Is(err, ErrReasonMismatch) {
		t.Errorf("OpenBoundary(flow 0, goods) = %v, want ErrReasonMismatch: a flow has no goods row", err)
	}
	if _, err := l.OpenBoundary(FlowReason(3), Money); !errors.Is(err, ErrReasonMismatch) {
		t.Errorf("OpenBoundary(a flow past the configured ones) = %v, want ErrReasonMismatch", err)
	}
	for name, cfg := range map[string][]Class{
		"internal class": {Faucet, Internal},
		"unknown class":  {Class(9)},
		"too many flows": make([]Class, 256-int(NumReasons)),
	} {
		cfg := cfg
		if name == "too many flows" {
			for i := range cfg {
				cfg[i] = Faucet
			}
		}
		_, err := New(Config{Commodities: []Commodity{{}}, Regions: 1, MaxMoney: 1, MaxQuantity: 1, Flows: cfg})
		if !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("New with %s = %v, want ErrInvalidConfig", name, err)
		}
	}
}

// TestTransfer_FlowReasonFaucetAndSinkKinds verifies the rows of a flow: a
// faucet runs from its boundary to an agent or an estate, a sink the other
// way, and every other pairing, between two non-boundary accounts included, is
// ErrReasonMismatch with the ledger unchanged.
//
// Requirement: FR-MON-004, FR-MON-043
func TestTransfer_FlowReasonFaucetAndSinkKinds(t *testing.T) {
	t.Parallel()
	l, b := flowLedger(t, []Class{Faucet, Sink})
	agent, _ := l.Open(KindAgent, 1, 0)
	other, _ := l.Open(KindAgent, 2, 0)
	estate, _ := l.Open(KindEstate, 3, 0)
	if err := l.Transfer(b[0], agent, 50, FlowReason(0)); err != nil {
		t.Fatalf("faucet to an agent: %v", err)
	}
	if err := l.Transfer(b[0], estate, 5, FlowReason(0)); err != nil {
		t.Fatalf("faucet to an estate: %v", err)
	}
	if err := l.Transfer(agent, b[1], 20, FlowReason(1)); err != nil {
		t.Fatalf("sink from an agent: %v", err)
	}
	if err := l.Transfer(estate, b[1], 5, FlowReason(1)); err != nil {
		t.Fatalf("sink from an estate: %v", err)
	}
	before := fingerprint(l)
	refused := []struct {
		name     string
		from, to int
		r        Reason
	}{
		{"faucet reason back into its boundary", agent, b[0], FlowReason(0)},
		{"sink reason out of its boundary", b[1], agent, FlowReason(1)},
		{"faucet reason between agents", agent, other, FlowReason(0)},
		{"sink reason between agents", agent, other, FlowReason(1)},
		{"agent to estate under a flow", agent, estate, FlowReason(0)},
		{"another flow's boundary", b[1], agent, FlowReason(0)},
		{"a reason past the flows", b[0], agent, FlowReason(2)},
	}
	for _, tt := range refused {
		if err := l.Transfer(tt.from, tt.to, 1, tt.r); !errors.Is(err, ErrReasonMismatch) {
			t.Errorf("Transfer(%s) = %v, want ErrReasonMismatch", tt.name, err)
		}
	}
	if after := fingerprint(l); !reflect.DeepEqual(before, after) {
		t.Errorf("a refused transfer under a flow reason changed the ledger")
	}
	if v, _ := l.Account(agent); v.Balance != 30 {
		t.Errorf("agent balance = %d, want 50 granted less 20 levied = 30", v.Balance)
	}
}

// TestAppendFlows_FlowBlockBeforeDeclared verifies the order of the totals of
// a region: the built-in reasons, then the flows in flow order, then the
// declared reasons, so the flow block precedes the declared module rows
// (plan 2-6 D323(b)).
//
// Requirement: FR-MON-029
func TestAppendFlows_FlowBlockBeforeDeclared(t *testing.T) {
	t.Parallel()
	l, b := flowLedger(t, []Class{Faucet, Sink}, LoanOut)
	mint, err := l.OpenBoundary(Initial, Money)
	if err != nil {
		t.Fatalf("OpenBoundary(Initial): %v", err)
	}
	a, _ := l.Open(KindAgent, 1, 0)
	c, _ := l.Open(KindAgent, 2, 0)
	for _, m := range []struct {
		from, to int
		v        int64
		r        Reason
	}{{mint, a, 100, Initial}, {a, c, 10, LoanOut}, {b[0], a, 7, FlowReason(0)}, {a, b[1], 3, FlowReason(1)}} {
		if err := l.Transfer(m.from, m.to, m.v, m.r); err != nil {
			t.Fatalf("Transfer under reason %d: %v", m.r, err)
		}
	}
	var got []Reason
	for _, f := range l.AppendFlows(nil, true) {
		got = append(got, f.Reason)
	}
	want := []Reason{Initial, FlowReason(0), FlowReason(1), LoanOut}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("AppendFlows reasons = %v, want %v (built-in, flows, declared)", got, want)
	}
}

// TestNew_FlowAccountsBeforeWantConsumption verifies that flow boundaries
// opened first take the first account IDs, ahead of a declared reason's
// boundary opened after them: the creation order of the root's newLedger
// (spec-ledger.md 3.4.2, N-226).
//
// Requirement: FR-MON-003
func TestNew_FlowAccountsBeforeWantConsumption(t *testing.T) {
	t.Parallel()
	l, ids := flowLedger(t, []Class{Sink, Faucet}, WantConsumption)
	want, err := l.OpenBoundary(WantConsumption, 0)
	if err != nil {
		t.Fatalf("OpenBoundary(WantConsumption): %v", err)
	}
	for _, id := range ids {
		if id >= want {
			t.Errorf("flow boundary %d is not before the want_consumption boundary %d", id, want)
		}
	}
	if l.accounts[ids[0]].reason != FlowReason(0) || l.accounts[ids[1]].reason != FlowReason(1) {
		t.Errorf("flow boundary reasons = %d, %d, want %d, %d", l.accounts[ids[0]].reason, l.accounts[ids[1]].reason, FlowReason(0), FlowReason(1))
	}
}

// TestAudit_CoversFlowReasons verifies that the audit checks the conservation
// of a flow reason: a boundary whose balance moved differently from its flow
// total is IdentityConservation, for a faucet and for a sink.
//
// Requirement: FR-MON-002, FR-MON-026, FR-MON-027
func TestAudit_CoversFlowReasons(t *testing.T) {
	t.Parallel()
	for i, class := range []Class{Faucet, Sink} {
		l, b := flowLedger(t, []Class{Faucet, Sink})
		agent, _ := l.Open(KindAgent, 1, 0)
		if err := l.Transfer(b[0], agent, 40, FlowReason(0)); err != nil {
			t.Fatalf("grant: %v", err)
		}
		if err := l.Transfer(agent, b[1], 15, FlowReason(1)); err != nil {
			t.Fatalf("levy: %v", err)
		}
		if err := l.Audit(); err != nil {
			t.Fatalf("Audit of a ledger with flows: %v", err)
		}
		l.accounts[b[i]].balance++
		var ae *AuditError
		if err := l.Audit(); !errors.As(err, &ae) || ae.Identity != IdentityConservation && ae.Identity != IdentityZeroTotal {
			t.Errorf("Audit with a corrupted %v boundary = %v, want a conservation or zero-total failure", class, err)
		}
	}
}

// TestState_RoundTripWithFlows verifies that a ledger with flows restores its
// boundary balances and pending totals under flow reasons, that the bytes
// carry the code pinnedRows + i and not the Go number, and that a total whose
// code is at or above pinnedRows plus the number of flows is ErrMalformed.
//
// Requirement: FR-MON-040
func TestState_RoundTripWithFlows(t *testing.T) {
	t.Parallel()
	build := func() (*Ledger, []int) { return flowLedger(t, []Class{Faucet, Sink}) }
	l, b := build()
	agent, _ := l.Open(KindAgent, 3, 0)
	if err := l.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction: %v", err)
	}
	l.EndTick()
	if err := l.Transfer(b[0], agent, 40, FlowReason(0)); err != nil {
		t.Fatalf("grant: %v", err)
	}
	if err := l.Transfer(agent, b[1], 12, FlowReason(1)); err != nil {
		t.Fatalf("levy: %v", err)
	}
	bytesOut := saved(l)
	// The pending rows are (region 0, code, thing 0 money, total): the grant is
	// code pinnedRows (42), the levy pinnedRows + 1 (43).
	for _, row := range [][]byte{{0, pinnedRows, 0, 40}, {0, pinnedRows + 1, 0, 12}} {
		if !bytes.Contains(bytesOut, row) {
			t.Errorf("WriteState wrote % x, want it to contain the flow row % x", bytesOut, row)
		}
	}
	fresh, _ := build()
	if err := read(fresh, bytesOut, l.Tick(), false); err != nil {
		t.Fatalf("ReadState: %v", err)
	}
	if err := fresh.Audit(); err != nil {
		t.Errorf("Audit after the restore: %v", err)
	}
	for i := range b {
		if want, got := l.accounts[b[i]].balance, fresh.accounts[b[i]].balance; want != got {
			t.Errorf("flow %d boundary balance after the restore = %d, want %d", i, got, want)
		}
	}
	if want, got := normal(l), normal(fresh); !reflect.DeepEqual(want, got) {
		t.Errorf("restored ledger with flows differs from the original")
	}
	bad := bytes.Replace(bytesOut, []byte{0, pinnedRows + 1, 0, 12}, []byte{0, pinnedRows + 2, 0, 12}, 1)
	again, _ := build()
	if err := read(again, bad, l.Tick(), false); !errors.Is(err, codec.ErrMalformed) {
		t.Errorf("ReadState with a flow code past the flows = %v, want codec.ErrMalformed", err)
	}
}

// TestNew_NoFlowsLeavesLayoutUnchanged verifies that a ledger with no flow, nil
// or an empty list, has the layout and the fields it had before the flows
// existed, and that flows only append slots after every existing one.
//
// Guard: the per-position state hash walks the World by reflection, so a field
// that is not nil at zero flows, or a moved offset, would move the position
// files of testdata/golden/off (architecture.md 2.10.5). Without this test the
// failure shows only in the root's golden test.
func TestNew_NoFlowsLeavesLayoutUnchanged(t *testing.T) {
	t.Parallel()
	cfg := Config{Commodities: []Commodity{{}, {}}, Regions: 2, MaxMoney: 10, MaxQuantity: 10, Declared: []Reason{LoanOut}}
	plain, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cfg.Flows = []Class{}
	empty, err := New(cfg)
	if err != nil {
		t.Fatalf("New with an empty Flows: %v", err)
	}
	if !reflect.DeepEqual(plain, empty) {
		t.Errorf("a ledger with an empty Flows differs from one with no Flows")
	}
	if plain.flowClass != nil || plain.flowOff != nil {
		t.Errorf("flow fields = %v, %v at zero flows, want nil", plain.flowClass, plain.flowOff)
	}
	cfg.Flows = []Class{Faucet, Sink}
	with, err := New(cfg)
	if err != nil {
		t.Fatalf("New with flows: %v", err)
	}
	if with.off != plain.off || !reflect.DeepEqual(with.later, plain.later) {
		t.Errorf("flows moved the offsets of the existing reasons: off %v later %v, want %v %v", with.off, with.later, plain.off, plain.later)
	}
	if with.stride != plain.stride+2 || with.flowOff[0] != plain.stride || with.flowOff[1] != plain.stride+1 {
		t.Errorf("stride %d, flow offsets %v, want stride %d and the flows at %d, %d", with.stride, with.flowOff, plain.stride+2, plain.stride, plain.stride+1)
	}
}

// threeFlowLedger returns a two-region, two-commodity ledger with three flows
// (faucet, sink, faucet) and their boundaries; with agents it also has one
// agent per region and is past construction, without it it is in construction
// with only its boundaries, which is what a restore needs.
func threeFlowLedger(t *testing.T, agents bool) (l *Ledger, boundary, agent []int) {
	t.Helper()
	l, err := New(Config{
		Commodities: []Commodity{{}, {}}, Regions: 2, MaxMoney: 1000, MaxQuantity: 100,
		Flows: []Class{Faucet, Sink, Faucet},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for i := range 3 {
		id, err := l.OpenBoundary(FlowReason(i), Money)
		if err != nil {
			t.Fatalf("OpenBoundary(flow %d): %v", i, err)
		}
		boundary = append(boundary, id)
	}
	if !agents {
		return l, boundary, nil
	}
	for region := range 2 {
		id, err := l.Open(KindAgent, region, region)
		if err != nil {
			t.Fatalf("Open(region %d): %v", region, err)
		}
		agent = append(agent, id)
	}
	if err := l.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction: %v", err)
	}
	l.EndTick()
	return l, boundary, agent
}

// TestState_RoundTripThreeFlowsTwoRegions verifies that the totals of every
// flow in every region survive a save and a restore, that they are written in
// the order (region, code, thing) with the codes pinnedRows, +1 and +2 for the
// three flows, and that the last flow's code is the largest one accepted.
//
// Guard: the code arithmetic (codeOf, reasonOfCode, the loop bound of eachFlow
// and the row key of readFlows) is invisible with one or two flows in one
// region; a mutation of any of them passed the two-flow round trip. Without
// this test a third flow, or a flow total in a second region, could be lost or
// read out of order on a restore.
func TestState_RoundTripThreeFlowsTwoRegions(t *testing.T) {
	t.Parallel()
	l, b, a := threeFlowLedger(t, true)
	// Region r gets a grant of flow 0 (10+r, then 20 more to fund the levy), a
	// levy of flow 1 (5+r) and a grant of flow 2 (3+r): totals 30+r, 5+r, 3+r.
	for r := range 2 {
		for i, amount := range []int64{int64(10 + r), int64(5 + r), int64(3 + r)} {
			var err error
			if i == 1 {
				if err = l.Transfer(b[0], a[r], 20, FlowReason(0)); err == nil {
					err = l.Transfer(a[r], b[1], amount, FlowReason(1))
				}
			} else {
				err = l.Transfer(b[i], a[r], amount, FlowReason(i))
			}
			if err != nil {
				t.Fatalf("region %d flow %d: %v", r, i, err)
			}
		}
	}
	out := saved(l)
	var flows []Flow
	flows = l.AppendFlows(flows, false)
	wantRows := map[[2]int]int64{}
	for _, f := range flows {
		wantRows[[2]int{f.Region, int(f.Reason)}] = f.Total
	}
	for r := range 2 {
		for i := range 3 {
			if wantRows[[2]int{r, int(FlowReason(i))}] == 0 {
				t.Fatalf("setup: no total for flow %d in region %d", i, r)
			}
		}
	}
	// The last flow's rows carry the code pinnedRows+2.
	for r, want := range []byte{3, 4} {
		if row := []byte{byte(r), pinnedRows + 2, 0, want}; !bytes.Contains(out, row) {
			t.Errorf("WriteState wrote % x, want it to contain the flow row % x (region %d, code %d, total %d)", out, row, r, pinnedRows+2, want)
		}
	}
	build := func() *Ledger { f, _, _ := threeFlowLedger(t, false); return f }
	fresh := build()
	if err := read(fresh, out, l.Tick(), false); err != nil {
		t.Fatalf("ReadState: %v", err)
	}
	if want, got := normal(l), normal(fresh); !reflect.DeepEqual(want, got) {
		t.Errorf("restored three-flow ledger differs from the original")
	}
	// A code one past the last flow is refused, the last flow's own is not.
	last := []byte{0, pinnedRows + 2, 0, 3}
	if !bytes.Contains(out, last) {
		t.Fatalf("setup: no row % x in % x", last, out)
	}
	bad := bytes.Replace(out, last, []byte{0, pinnedRows + 3, 0, 3}, 1)
	if err := read(build(), bad, l.Tick(), false); !errors.Is(err, codec.ErrMalformed) {
		t.Errorf("ReadState with the code pinnedRows+3 of three flows = %v, want codec.ErrMalformed", err)
	}
}

// TestState_FlowRowOrderAcrossRegions verifies the strict ascent of flow rows
// when flow codes and regions interleave: region 1 with the lowest flow code
// follows region 0 with the highest, and the reverse, a repeat or a
// descending code inside a region is refused.
//
// Guard: the row key mixes region, code and thing; a wrong stride in it
// accepts a descending pair or refuses a valid one.
func TestState_FlowRowOrderAcrossRegions(t *testing.T) {
	t.Parallel()
	row := func(region, code, thing, total uint64) [4]uint64 { return [4]uint64{region, code, thing, total} }
	hi, lo := uint64(pinnedRows+2), uint64(pinnedRows)
	tests := []struct {
		name string
		rows [][4]uint64
		ok   bool
	}{
		{"region 1 low code after region 0 high code", [][4]uint64{row(0, hi, 0, 1), row(1, lo, 0, 1)}, true},
		{"region 0 high code after region 1 low code", [][4]uint64{row(1, lo, 0, 1), row(0, hi, 0, 1)}, false},
		{"flow codes ascending in a region", [][4]uint64{row(1, lo, 0, 1), row(1, lo+1, 0, 1), row(1, hi, 0, 1)}, true},
		{"flow codes descending in a region", [][4]uint64{row(1, hi, 0, 1), row(1, lo+1, 0, 1)}, false},
		{"flow row repeated", [][4]uint64{row(1, hi, 0, 1), row(1, hi, 0, 1)}, false},
		{"a table reason after the last flow of the region before", [][4]uint64{row(0, hi, 0, 1), row(1, 1, 0, 1)}, true},
		{"a table reason before a flow of the same region", [][4]uint64{row(0, 1, 0, 1), row(0, lo, 0, 1)}, true},
		{"a flow before a table reason of the same region", [][4]uint64{row(0, lo, 0, 1), row(0, 1, 0, 1)}, false},
		{"flow code one past the last flow", [][4]uint64{row(0, hi+1, 0, 1)}, false},
		{"flow code of the last flow", [][4]uint64{row(0, hi, 0, 1)}, true},
		{"flow row of a commodity", [][4]uint64{row(0, lo, 1, 1)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// No boundary is opened, so the section lists no account.
			l, err := New(Config{
				Commodities: []Commodity{{}, {}}, Regions: 2, MaxMoney: 1000, MaxQuantity: 100,
				Flows: []Class{Faucet, Sink, Faucet},
			})
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			w := codec.NewWriter(nil)
			w.Count(0) // accounts
			w.U(0)     // next claim
			w.U(0)     // claims
			w.Count(len(tt.rows))
			for _, r := range tt.rows {
				for _, v := range r {
					w.U(v)
				}
			}
			w.U(0) // wear
			w.U(0) // relocations
			// Only the byte format is under test, not the audit of balances.
			err = read(l, w.Out(), 1, false)
			if tt.ok && errors.Is(err, codec.ErrMalformed) {
				t.Errorf("ReadState(%v) = %v, want the rows accepted", tt.rows, err)
			}
			if !tt.ok && !errors.Is(err, codec.ErrMalformed) {
				t.Errorf("ReadState(%v) = %v, want codec.ErrMalformed", tt.rows, err)
			}
		})
	}
}
