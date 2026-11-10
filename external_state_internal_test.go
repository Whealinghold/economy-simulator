package economysim

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// workedTick runs the one tick of example A of spec-external-sector.md 18.1 (the same
// orders as TestExternal_WorkedTick) and returns the world, the script and the log.
func workedTick(t *testing.T, opts ...Option) (*World, *extScript, *evLog) {
	t.Helper()
	w, s, rec := extWorld(t, extConfig(), opts...)
	grain, salt := commodityOf(t, w, "grain"), commodityOf(t, w, "salt")
	for i, o := range []struct {
		limit Price
		units Quantity
	}{{15, 20}, {17, 15}, {19, 10}, {21, 30}} {
		id := AgentID(w.firstHousehold + i)
		give(t, w, id, grain, 100)
		s.plan[id] = []Intent{{Commodity: grain, Side: Ask, Quantity: o.units, Limit: o.limit}}
	}
	for i, o := range []struct {
		limit Price
		units Quantity
	}{{50, 10}, {45, 10}, {41, 10}} {
		id := AgentID(w.firstHousehold + 4 + i)
		fund(t, w, id, 10_000)
		s.plan[id] = []Intent{{Commodity: salt, Side: Bid, Quantity: o.units, Limit: o.limit}}
	}
	step(t, w, 1)
	return w, s, rec
}

// TestExternal_Events verifies the three events of spec-external-sector.md 15.1: the
// fields of example A, the order (OutsideTraded by commodity with the buy side first,
// OutsideEnded after the OutsideTraded of its tick), none when nothing was wanted or
// cut (the ended sides of example B place nothing afterwards), none with the module
// off, and no map, pointer or slice in a field.
//
// Requirement: FR-EXT-036, FR-EXT-037
func TestExternal_Events(t *testing.T) {
	t.Parallel()
	_, _, rec := workedTick(t)
	got := of[OutsideTraded](rec)
	if len(got) != 2 || got[0].Side != Bid || got[1].Side != Ask || got[0].Commodity >= got[1].Commodity {
		t.Errorf("OutsideTraded of the worked tick = %+v, want the grain buy side then the salt sell side", got)
	}
	for _, e := range []Event{OutsideTraded{}, OutsideEnded{}, OutsideShocked{}} {
		rt := reflect.TypeOf(e)
		for i := range rt.NumField() {
			if k := rt.Field(i).Type.Kind(); k == reflect.Map || k == reflect.Pointer || k == reflect.Slice {
				t.Errorf("%s.%s is a %v, want plain values", rt.Name(), rt.Field(i).Name, k)
			}
		}
	}

	w, s, capRec := extWorld(t, capConfig())
	ore := commodityOf(t, w, "ore")
	miner := AgentID(w.firstHousehold)
	s.plan[miner] = []Intent{{Commodity: ore, Side: Ask, Quantity: 200, Limit: 90}}
	for range 10 {
		give(t, w, miner, ore, 200)
		step(t, w, 1)
	}
	last := int64(-1)
	for _, e := range capRec.events {
		switch v := e.(type) {
		case OutsideTraded:
			last = v.Tick
		case OutsideEnded:
			if v.Tick != last {
				t.Errorf("OutsideEnded at tick %d after the last OutsideTraded of tick %d, want the same tick", v.Tick, last)
			}
		}
	}
	for _, e := range of[OutsideTraded](capRec) {
		if e.Commodity == ore && e.Tick > 7 {
			t.Errorf("OutsideTraded %+v after the side ended in tick 7, want none: nothing was wanted or cut", e)
		}
	}

	c := extConfig()
	c.Modules.External = nil
	_, _, off := extWorld(t, c)
	if n := len(of[OutsideTraded](off)) + len(of[OutsideEnded](off)) + len(of[OutsideShocked](off)); n != 0 {
		t.Errorf("outside events with the module off = %d, want 0", n)
	}
}

// TestExternal_Logs verifies FR-EXT-038 with a capturing handler over example B with a
// walk: the tick logs nothing at Info or Error, Warn only "outside side ended" with the
// keys node, region, commodity, side and cause, and Debug only "outside side" and
// "outside drift", each with its keys; with a handler that refuses Debug the tick
// writes only the Warn lines.
//
// Requirement: FR-EXT-038
func TestExternal_Logs(t *testing.T) {
	t.Parallel()
	c := capConfig()
	c.Modules.External.Nodes[0].Lines[0].WalkStepBP = 100
	c.Modules.External.Nodes[0].Lines[0].WalkBoundBP = 500
	lines := newCapture()
	w, s, _ := extWorld(t, c, WithLogger(slog.New(lines)))
	ore := commodityOf(t, w, "ore")
	miner := AgentID(w.firstHousehold)
	s.plan[miner] = []Intent{{Commodity: ore, Side: Ask, Quantity: 200, Limit: 90}}
	mark := len(*lines.lines)
	for range 8 {
		give(t, w, miner, ore, 200)
		step(t, w, 1)
	}
	want := map[string][]string{
		"outside side ended": {"tick", "node", "region", "commodity", "side", "cause"},
		"outside side":       {"tick", "node", "region", "commodity", "side", "price", "quantity", "value"},
		"outside drift":      {"tick", "node", "commodity", "drift"},
	}
	seen := map[string]int{}
	for _, l := range (*lines.lines)[mark:] {
		keys, ok := want[l.msg]
		if !ok {
			continue
		}
		seen[l.msg]++
		for _, k := range keys {
			if _, num := l.attrs[k]; !num && l.strs[k] == "" {
				t.Errorf("log %q has no key %q (attrs %v, strings %v)", l.msg, k, l.attrs, l.strs)
			}
		}
		if (l.msg == "outside side ended") != (l.level == slog.LevelWarn) || (l.msg != "outside side ended" && l.level != slog.LevelDebug) {
			t.Errorf("log %q at level %v, want Warn for the end and Debug for the others", l.msg, l.level)
		}
	}
	for msg := range want {
		if seen[msg] == 0 {
			t.Errorf("no %q line in 8 ticks, want at least one", msg)
		}
	}
	if seen["outside side ended"] != 1 {
		t.Errorf("outside side ended lines = %d, want 1 (ore in tick 7; no buyer takes the reagent)", seen["outside side ended"])
	}
	for _, l := range (*lines.lines)[mark:] {
		if l.level == slog.LevelInfo || l.level == slog.LevelError {
			t.Errorf("tick logged %q at %v, want no Info or Error in a tick", l.msg, l.level)
		}
	}

	var quiet bytes.Buffer
	w2, s2, _ := extWorld(t, c, WithLogger(slog.New(slog.NewTextHandler(&quiet, &slog.HandlerOptions{Level: slog.LevelWarn}))))
	s2.plan[miner] = s.plan[miner]
	quiet.Reset()
	for range 8 {
		give(t, w2, miner, ore, 200)
		step(t, w2, 1)
	}
	if n := strings.Count(quiet.String(), "msg=\"outside"); n != 1 || strings.Contains(quiet.String(), "level=DEBUG") {
		t.Errorf("a Warn handler got %d outside lines in %q, want the one ended side only and no Debug", n, quiet.String())
	}
}

// TestExternal_Shock verifies the internal setter of spec-external-sector.md 6.5: the
// slot is 10 000 until set, a changed value gives one OutsideShocked with From and To
// and an unchanged one nothing, the new mid applies from the next step (the quote of
// the salt side is ceil(48 * 1.05) = 51 against 42), and the slot is saved in a
// checkpoint and read back (it is not part of the configuration, so the restore under
// the same configuration accepts it).
//
// Requirement: FR-EXT-027, FR-EXT-037
func TestExternal_Shock(t *testing.T) {
	t.Parallel()
	w, _, rec := extWorld(t, extConfig())
	x := w.external
	for i := range x.lines {
		if x.lines[i].shock != wholeBP {
			t.Fatalf("line %d shock = %d before any call, want 10000", i, x.lines[i].shock)
		}
	}
	salt, grain := commodityOf(t, w, "salt"), commodityOf(t, w, "grain")
	ev := x.applyShocks(w.Tick(), []outsideShock{{0, salt, 12_000}, {0, grain, 10_000}})
	if want := []OutsideShocked{{Tick: 0, Node: 0, Commodity: salt, From: 10_000, To: 12_000}}; !slices.Equal(ev, want) {
		t.Errorf("applyShocks events = %+v, want %+v: one for the changed key, none for the unchanged", ev, want)
	}
	if again := x.applyShocks(w.Tick(), []outsideShock{{0, salt, 12_000}}); len(again) != 0 {
		t.Errorf("applyShocks of an unchanged value = %+v, want no event", again)
	}
	step(t, w, 1)
	if got := of[OutsideTraded](rec); len(got) != 2 || got[1].Commodity != salt || got[1].Mid != 48 || got[1].Price != 51 {
		t.Errorf("OutsideTraded after the shock = %+v, want the salt side at mid 48 and limit 51", got)
	}
	cp, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() = %v, want nil", err)
	}
	r, err := RestoreWorld(extConfig(), cp, WithStrategy("script", &extScript{plan: map[AgentID][]Intent{}}))
	if err != nil {
		t.Fatalf("RestoreWorld() = %v, want nil", err)
	}
	if got := r.external.lines[1].shock; got != 12_000 {
		t.Errorf("restored shock of the salt line = %d, want 12000", got)
	}
}

// TestExternal_Snapshot verifies FR-EXT-046 after the worked tick of 18.1: the lines
// (mid 20 and 40, no drift, shock 10 000), the sides (bid 18 for grain with 35 units
// and list value 630; ask 42 for salt with 20 units and 840), Trade per commodity as
// the sums of the sides' units, the six boundary balances of the tick, the two
// counterparties with their account IDs and gate, and a deep copy (changing a returned
// slice changes nothing of the world's next snapshot).
//
// Requirement: FR-EXT-046
func TestExternal_Snapshot(t *testing.T) {
	t.Parallel()
	w, _, _ := workedTick(t)
	snap, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	grain, salt := commodityOf(t, w, "grain"), commodityOf(t, w, "salt")
	o := snap.Outside
	wantLines := []OutsideLineRow{{Node: 0, Commodity: grain, Mid: 20, ShockBP: 10_000}, {Node: 0, Commodity: salt, Mid: 40, ShockBP: 10_000}}
	wantSides := []OutsideSideRow{
		{Node: 0, Gate: 1, Commodity: grain, Side: Bid, Quote: 18, Units: 35, ListValue: 630},
		{Node: 0, Gate: 1, Commodity: salt, Side: Ask, Quote: 42, Units: 20, ListValue: 840},
	}
	wantTrade := []OutsideTradeRow{{Commodity: grain, Exports: 35}, {Commodity: salt, Imports: 20}}
	if !slices.Equal(o.Lines, wantLines) || !slices.Equal(o.Sides, wantSides) || !slices.Equal(o.Trade, wantTrade) {
		t.Errorf("Outside = %+v, want lines %+v, sides %+v, trade %+v", o, wantLines, wantSides, wantTrade)
	}
	cps := w.external.cps
	wantCps := []CounterpartyRow{
		{Agent: cps[0].agent, Node: 0, Gate: 1, Side: Bid, Account: int64(cps[0].account)},
		{Agent: cps[1].agent, Node: 0, Gate: 1, Side: Ask, Account: int64(cps[1].account)},
	}
	if !slices.Equal(o.Counterparties, wantCps) || cps[0].account == 0 {
		t.Errorf("Counterparties = %+v, want %+v", o.Counterparties, wantCps)
	}
	boundary := map[Reason]int64{}
	for _, r := range snap.Boundary {
		boundary[r.Reason] += r.Balance
	}
	for r, want := range map[Reason]int64{"outside_budget": -900, "outside_unspent": 310, "outside_receipts": 900,
		"outside_offer": -30, "outside_unsold": 10, "outside_purchases": 35} {
		if boundary[r] != want {
			t.Errorf("Boundary balance of %s = %d, want %d", r, boundary[r], want)
		}
	}
	o.Lines[0].Mid, o.Sides[0].Units, o.Trade[0].Exports, o.Counterparties[0].Account = 999, 999, 999, 999
	again, err := w.Snapshot()
	if err != nil || !slices.Equal(again.Outside.Lines, wantLines) || !slices.Equal(again.Outside.Sides, wantSides) || !slices.Equal(again.Outside.Trade, wantTrade) {
		t.Errorf("Snapshot() after changing a returned slice = %+v (%v), want the rows unchanged: a deep copy", again.Outside, err)
	}
	c := extConfig()
	c.Modules.External = nil
	off, _, _ := extWorld(t, c)
	if s, _ := off.Snapshot(); !reflect.DeepEqual(s.Outside, OutsideState{}) {
		t.Errorf("Outside with the module off = %+v, want the zero value", s.Outside)
	}
}

// TestExternal_Lines verifies the pure read of FR-EXT-047: no world is built, the lines
// come in the order of the configuration with their gates, the quotes at tick 0 (the
// bid 18 of the grain buy side, the ask 42 of the salt sell side) and the mean
// capacity (the capacity times the window averaged over the seasons by their days,
// truncated, here with the first season of the calendar closed).
//
// Requirement: FR-EXT-047
func TestExternal_Lines(t *testing.T) {
	t.Parallel()
	c := extConfig()
	got := externalLines(&c)
	want := []lineRead{
		{Node: "coast", Commodity: "grain", Buy: sideRead{Present: true, MeanCapacity: 50, Gates: []gateRead{{Gate: "lowland", Quote: 18}}}},
		{Node: "coast", Commodity: "salt", Sell: sideRead{Present: true, MeanCapacity: 30, Gates: []gateRead{{Gate: "lowland", Quote: 42}}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("externalLines = %+v, want %+v", got, want)
	}
	seasons := c.Calendar.Seasons
	var days int64
	for _, se := range seasons {
		days += se.Days
	}
	c.Modules.External.Nodes[0].Lines[1].Buy.Window = []SeasonFactor{{Season: seasons[0].ID, BP: 0}} // the grain line is the second as written
	if got, want := externalLines(&c)[0].Buy.MeanCapacity, 50*(days-seasons[0].Days)/days; got != want {
		t.Errorf("mean capacity with the first season closed = %d, want 50 * (%d - %d) / %d = %d", got, days, seasons[0].Days, days, want)
	}
	c.Modules.External = nil
	if got := externalLines(&c); got != nil {
		t.Errorf("externalLines with the module off = %+v, want nil", got)
	}
}

// TestFork_ExternalRestore verifies FR-EXT-016 on the world of example B with a walk and
// a reagent window of 75 % (a carry): checkpoints taken after one step, before and after the ore side ended
// (a cap of 600 units, tick 2) find every counterparty account empty,
// and a world restored from each one, run on for 4 ticks with the same re-supply, has
// the same snapshot and the same checkpoint bytes as the uninterrupted one.
//
// Requirement: FR-EXT-016
func TestFork_ExternalRestore(t *testing.T) {
	t.Parallel()
	cfg := func() Config {
		c := capConfig()
		c.Modules.External.Nodes[0].Lines[0].WalkStepBP = 100
		c.Modules.External.Nodes[0].Lines[0].WalkBoundBP = 500
		c.Modules.External.Nodes[0].Lines[0].Buy.TotalUnits = 600 // the ore side ends in tick 2
		for _, s := range c.Calendar.Seasons {
			c.Modules.External.Nodes[0].Lines[1].Sell.Window = append(c.Modules.External.Nodes[0].Lines[1].Sell.Window, SeasonFactor{Season: s.ID, BP: 7_500})
		}
		return c
	}
	run := func(t *testing.T, w *World, s *extScript, n int) {
		t.Helper()
		ore := commodityOf(t, w, "ore")
		miner := AgentID(w.firstHousehold)
		s.plan[miner] = []Intent{{Commodity: ore, Side: Ask, Quantity: 200, Limit: 90}}
		for range n {
			give(t, w, miner, ore, 200)
			step(t, w, 1)
		}
	}
	for _, n := range []int{1, 3, 6} {
		t.Run(fmt.Sprintf("N %d", n), func(t *testing.T) {
			w, s, _ := extWorld(t, cfg())
			run(t, w, s, n)
			cp, err := w.Checkpoint()
			if err != nil {
				t.Fatalf("Checkpoint() at tick %d = %v, want nil", n, err)
			}
			rs := &extScript{plan: map[AgentID][]Intent{}}
			r, err := RestoreWorld(cfg(), cp, WithStrategy("script", rs), WithAudit())
			if err != nil {
				t.Fatalf("RestoreWorld() at tick %d = %v, want nil", n, err)
			}
			for _, cpt := range r.external.cps {
				if v, _ := r.ledger.Account(cpt.account); v.Balance != 0 {
					t.Errorf("counterparty %d holds %d in the checkpoint of tick %d, want an empty account", cpt.agent, v.Balance, n)
				}
			}
			run(t, w, s, 4)
			run(t, r, rs, 4)
			a, errA := w.Checkpoint()
			b, errB := r.Checkpoint()
			sa, _ := w.Snapshot()
			sb, _ := r.Snapshot()
			if errA != nil || errB != nil || !reflect.DeepEqual(a, b) || !reflect.DeepEqual(sa.Outside, sb.Outside) {
				t.Errorf("after the restore at tick %d and 4 more ticks: checkpoints equal %v (%v, %v), outside state equal %v, want both equal", n, reflect.DeepEqual(a, b), errA, errB, reflect.DeepEqual(sa.Outside, sb.Outside))
			}
		})
	}
}

// TestExternal_CheckpointRefusesBadState verifies the bounds of tag 18 on read: a
// drift outside the walk bound, a shock of 0, a carry of 10 000 and units above the cap are refused with
// ErrCheckpointCorrupt, as a checkpoint that no run wrote.
//
// Requirement: FR-EXT-016
func TestExternal_CheckpointRefusesBadState(t *testing.T) {
	t.Parallel()
	for name, bad := range map[string]func(x *external){
		"drift":  func(x *external) { x.lines[0].drift = x.lines[0].cfg.WalkBoundBP + 1 },
		"shock":  func(x *external) { x.lines[0].shock = 0 },
		"carry":  func(x *external) { x.rows[0].carry = wholeBP },
		"capped": func(x *external) { x.rows[0].units = MaxLineTotal + 1 },
	} {
		t.Run(name, func(t *testing.T) {
			c := extConfig()
			c.Modules.External.Nodes[0].Lines[0].WalkStepBP = 50
			c.Modules.External.Nodes[0].Lines[0].WalkBoundBP = 100
			w, _, _ := extWorld(t, c)
			bad(w.external)
			cp, err := w.Checkpoint()
			if err != nil {
				t.Fatalf("Checkpoint() = %v, want nil", err)
			}
			if _, err := RestoreWorld(c, cp, WithStrategy("script", &extScript{plan: map[AgentID][]Intent{}})); !errors.Is(err, ErrCheckpointCorrupt) {
				t.Errorf("RestoreWorld of a checkpoint with a bad %s = %v, want ErrCheckpointCorrupt", name, err)
			}
		})
	}
}

// TestExternal_StepAllocatesNothingOfItsOwn verifies NFR-EXT-001 at the call sites: with
// the default logger and observer the walk and the mid of the tick allocate nothing
// once the world is built.
//
// Requirement: NFR-EXT-001
func TestExternal_StepAllocatesNothingOfItsOwn(t *testing.T) { //nolint:paralleltest // AllocsPerRun panics in a parallel test
	c := extConfig()
	c.Modules.External.Nodes[0].Lines[0].WalkStepBP = 100
	c.Modules.External.Nodes[0].Lines[0].WalkBoundBP = 500
	w := newMill(t, c, WithStrategy("script", &extScript{plan: map[AgentID][]Intent{}}))
	if got := testing.AllocsPerRun(50, func() { w.p18Walk(); w.external.setMids(w) }); got != 0 {
		t.Errorf("allocations of p18Walk and setMids = %v, want 0", got)
	}
}

// extNodes returns the world of example A with n nodes on the one gate: coast and
// n - 1 more regions, each with the two lines of the first.
func extNodes(n int) Config {
	c := extConfig()
	first := c.Modules.External.Nodes[0]
	for i := 1; i < n; i++ {
		r := DefaultRegion()
		r.ID = fmt.Sprintf("coast_%02d", i)
		c.Regions = append(c.Regions, r)
		c.Edges = append(c.Edges, Edge{From: r.ID, To: "lowland", Distance: 30}, Edge{From: "lowland", To: r.ID, Distance: 30})
		node := first
		node.Region = r.ID
		c.Modules.External.Nodes = append(c.Modules.External.Nodes, node)
	}
	return c
}

// BenchmarkExternal_Tick reports the cost of a tick of the world of example A with 1,
// 4 and 16 nodes on one gate: proportional to nodes times gates times lines.
//
// Requirement: NFR-EXT-002
func BenchmarkExternal_Tick(b *testing.B) {
	for _, n := range []int{1, 4, 16} {
		b.Run(fmt.Sprintf("nodes %d", n), func(b *testing.B) {
			w, err := NewWorld(extNodes(n), WithStrategy("script", &extScript{plan: map[AgentID][]Intent{}}))
			if err != nil {
				b.Fatalf("NewWorld() = %v, want nil", err)
			}
			b.ReportAllocs()
			for b.Loop() {
				if err := w.Step(); err != nil {
					b.Fatalf("Step() = %v, want nil", err)
				}
			}
		})
	}
}

// TestExternal_ManyNodesRun verifies that the benchmark's world is valid and runs with
// the audit on: 16 nodes on one gate give 32 counterparties in agent ID order.
//
// Requirement: NFR-EXT-002
func TestExternal_ManyNodesRun(t *testing.T) {
	t.Parallel()
	w, _, _ := extWorld(t, extNodes(16))
	step(t, w, 2)
	if n := len(w.external.cps); n != 32 {
		t.Errorf("counterparties = %d, want 32 (a buyer and a seller for each of 16 nodes)", n)
	}
}
