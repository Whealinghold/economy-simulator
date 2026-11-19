package economysim

import (
	"bytes"
	"encoding/binary"
	"slices"
	"testing"
)

// TestFiscalHashEncodingPinned verifies the bytes of the fiscal module under the ids of
// spec-config.md 8.2.2: the module is id 20 of Modules, written even when empty (`0014
// 0000`) and between Polity 19 and Institutions 21; a Fiscal has Taxes 1 to Releases 6;
// a window is Start 1, Until 2 and Seasons 3 and is written only when it has a field; a
// flow has ID 1 and Authority 2 and then its fields in declaration order, with Label
// having no id; a target has Jurisdiction as id 5 and writes nothing for it when empty.
//
// Requirement: FR-FIS-005, FR-CFG-093
//
// Guard: the ids are part of the hash of every configuration with the module on; an id
// moved, reused or written for a field at its off value would change the hash of a
// stored configuration without a test noticing.
func TestFiscalHashEncodingPinned(t *testing.T) {
	t.Parallel()
	id := func(n uint16) []byte { return binary.BigEndian.AppendUint16(nil, n) }
	u64 := func(n uint64) []byte { return binary.BigEndian.AppendUint64(nil, n) }
	str := func(s string) []byte { return append(u64(uint64(len(s))), s...) }
	end := []byte{0, 0}

	e := &encoder{}
	pairsModules(e, Modules{Polity: &PolityConfig{}, Fiscal: &Fiscal{}, Institutions: &Institutions{}})
	pair := append(id(20), 0, 0)
	at := bytes.Index(e.buf, pair)
	if at < 0 || bytes.Index(e.buf, append(id(19), 0, 0)) > at || at > bytes.Index(e.buf, append(id(21), 0, 0)) {
		t.Errorf("pairsModules(Polity, Fiscal, Institutions) = % x, want the empty pair % x after id 19 and before id 21", e.buf, pair)
	}

	// A window is a nested struct: its pairs and a terminator, which the flow's own
	// terminator follows.
	window := Window{Start: 3, Until: 9, Seasons: []string{"winter"}}
	wantWindow := bytes.Join([][]byte{id(1), u64(3), id(2), u64(9), id(3), u64(1), str("winter"), end}, nil)
	e = &encoder{}
	pairsFiscal(e, Fiscal{
		Taxes:     []Tax{{ID: "t", Authority: "a", Target: FlowTarget{Kind: "treasury", Jurisdiction: "j"}, Amount: 1, RateBP: 2, Exempt: 3, Per: "tick", Window: window, Label: "ignored"}},
		Duties:    []Duty{{ID: "d", Authority: "a", Region: "r", Commodity: "c", Payer: "seller", Scope: "any", RateBP: 4, Window: window}},
		Tolls:     []Toll{{ID: "l", Authority: "a", On: "exit", Mode: "m", Amount: 5}},
		Payouts:   []Payout{{ID: "p", Authority: "a", Target: FlowTarget{Kind: "pool"}, Commodity: "c", Amount: 6, Level: 7, Per: "year", NotIncome: true}},
		Purchases: []Purchase{{ID: "b", Authority: "a", Commodity: "c", Quantity: 8, LimitBP: 9, Hold: 10}},
		Releases:  []Release{{ID: "s", Authority: "a", Commodity: "c", Quantity: 11, LimitBP: 12, Reserve: 13}},
	})
	want := bytes.Join([][]byte{
		id(1), u64(1), // Taxes
		id(1), str("t"), id(2), str("a"),
		id(3), id(1), str("treasury"), id(5), str("j"), end, // Target: Kind 1, Jurisdiction 5
		id(4), u64(1), id(5), u64(2), id(6), u64(3), id(7), str("tick"),
		id(8), wantWindow, end,
		id(2), u64(1), // Duties
		id(1), str("d"), id(2), str("a"), id(3), str("r"), id(4), str("c"), id(5), str("seller"), id(6), str("any"), id(7), u64(4),
		id(8), wantWindow, end,
		id(3), u64(1), // Tolls
		id(1), str("l"), id(2), str("a"), id(3), str("exit"), id(4), str("m"), id(5), u64(5), end,
		id(4), u64(1), // Payouts
		id(1), str("p"), id(2), str("a"), id(3), id(1), str("pool"), end, id(4), str("c"), id(5), u64(6), id(6), u64(7),
		id(7), str("year"), id(8), {1}, end,
		id(5), u64(1), // Purchases
		id(1), str("b"), id(2), str("a"), id(3), str("c"), id(4), u64(8), id(5), u64(9), id(6), u64(10), end,
		id(6), u64(1), // Releases
		id(1), str("s"), id(2), str("a"), id(3), str("c"), id(4), u64(11), id(5), u64(12), id(6), u64(13), end,
	}, nil)
	if !bytes.Equal(e.buf, want) {
		t.Errorf("pairsFiscal = % x\nwant           % x", e.buf, want)
	}
}

// TestFiscalHashCanonicalSpelling verifies that the canonical copy spells the defaults
// and the order the hash reads: an empty Per is tick, an empty Scope is any, the lists
// are in ID order and the seasons of a window in the calendar's order.
//
// Requirement: FR-FIS-005
func TestFiscalHashCanonicalSpelling(t *testing.T) {
	t.Parallel()
	cal := DefaultCalendar()
	c := Config{Calendar: cal, Modules: Modules{Fiscal: &Fiscal{
		Taxes:     []Tax{{ID: "b", Window: Window{Seasons: []string{"winter", "spring"}}}, {ID: "a"}},
		Duties:    []Duty{{ID: "d"}},
		Payouts:   []Payout{{ID: "p"}},
		Purchases: []Purchase{{ID: "q", Window: Window{Seasons: []string{"summer", "autumn"}}}},
	}}}
	got := c.canonical().Modules.Fiscal
	if got.Taxes[0].ID != "a" || got.Taxes[1].Per != PerTick || got.Duties[0].Scope != ScopeAny || got.Payouts[0].Per != PerTick {
		t.Errorf("canonical Fiscal = %+v, want taxes by ID and Per tick, Scope any", got)
	}
	if s := got.Taxes[1].Window.Seasons; !slices.Equal(s, []string{"spring", "winter"}) {
		t.Errorf("canonical tax seasons = %v, want [spring winter], the calendar's order", s)
	}
	if s := got.Purchases[0].Window.Seasons; !slices.Equal(s, []string{"summer", "autumn"}) {
		t.Errorf("canonical purchase seasons = %v, want [summer autumn], the calendar's order", s)
	}
}

// TestFiscalCanonical_NilAndOwnership verifies that the canonical copy of a nil module is
// nil and that it sorts a copy of the seasons, never the consumer's own list.
//
// Requirement: FR-FIS-005
func TestFiscalCanonical_NilAndOwnership(t *testing.T) {
	t.Parallel()
	cal := DefaultCalendar()
	if (*Fiscal)(nil).canonical(cal.Seasons) != nil {
		t.Error("canonical of a nil Fiscal is not nil")
	}
	in := &Fiscal{Taxes: []Tax{{ID: "x", Window: Window{Seasons: []string{"winter", "spring"}}}}}
	in.canonical(cal.Seasons)
	if in.Taxes[0].Window.Seasons[0] != "winter" {
		t.Error("canonical sorted the seasons of the consumer's own list")
	}
}
