package economysim

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/rumour"
)

// This file tests the record, the filler and the function of the information
// module (information.go, plan 2-13 Phase 4) over the fixture of
// information_helpers_internal_test.go.

// infoLag is the lag of the fixture's news from a source region to a reader
// region, [source][reader], at an information speed of 40: the distances are
// a-b 40 both ways, b to c 80, c to b 120, so a to c is 120 and c to a 160, and
// Lag = 1 + ceil(distance / 40): 2 for 40, 3 for 80, 4 for 120, 5 for 160.
var infoLag = [3][3]int64{{0, 2, 4}, {2, 0, 3}, {5, 4, 0}}

// agentIn returns the first live household of a region.
func agentIn(t *testing.T, w *World, region RegionID) *agent {
	t.Helper()
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		if a := &w.agents[i]; a.weight >= 1 && w.regionOf(a) == region {
			return a
		}
	}
	t.Fatalf("no live household in region %d", region)
	return nil
}

// roleOf returns the ID of a role of the world by its string.
func roleOf(t *testing.T, w *World, id string) RoleID {
	t.Helper()
	r, ok := w.cfg.roleID(id)
	if !ok {
		t.Fatalf("role %q is not in the configuration", id)
	}
	return r
}

// viewsAt returns the Trade views the script recorded at a tick.
func viewsAt(s *infoScript, tick int64) []seenView {
	var out []seenView
	for _, v := range s.trades {
		if v.tick == tick {
			out = append(out, v)
		}
	}
	return out
}

// pcgState returns the bytes of the world's PCG, to compare two points of a
// stream.
func pcgState(t *testing.T, w *World) []byte {
	t.Helper()
	b, err := w.rnd.pcg.MarshalBinary()
	if err != nil {
		t.Fatalf("PCG.MarshalBinary() = %v, want nil", err)
	}
	return b
}

// halfEvenScaled is p * (10 000 + e) / 10 000 rounded half to even, written
// here independently of internal/rumour.
func halfEvenScaled(p, e int64) int64 {
	n := p * (10_000 + e)
	q, r := n/10_000, n%10_000
	if r > 5_000 || (r == 5_000 && q%2 == 1) {
		q++
	}
	return q
}

// TestSources_ActiveMarkets verifies that the sources are the pairs (region,
// commodity) whose commodity is in the tradable set of a role with a group in the
// region, in ascending order, and that a region with no group contributes none.
//
// The fixture: miller (grain and flour) in a and b, carrier (grain alone) in c,
// so the sources are (a, flour), (a, grain), (b, flour), (b, grain), (c, grain);
// flour is 0 and grain 1. A region d with no group adds nothing.
//
// Requirement: FR-INF-002
func TestSources_ActiveMarkets(t *testing.T) {
	t.Parallel()
	want := [][2]int{{0, 0}, {0, 1}, {1, 0}, {1, 1}, {2, 1}}
	w, _ := newInfoWorld(t, func(c *Config) {
		d := DefaultRegion()
		d.ID = "d"
		c.Regions = append(c.Regions, d)
		c.Edges = append(c.Edges, Edge{From: "c", To: "d", Distance: 40})
	})
	var got [][2]int
	for _, s := range w.info.sources {
		got = append(got, [2]int{int(s.region), int(s.commodity)})
	}
	if !slices.Equal(got, want) {
		t.Errorf("sources = %v, want %v (ascending, the region with no group left out)", got, want)
	}
	off, _ := newInfoWorld(t, func(c *Config) { c.Modules.Information = nil; c.Roles[0].Informed, c.Roles[1].Informed = false, false })
	if off.info != nil {
		t.Errorf("World.info with the module off = %v, want nil", off.info)
	}
}

// TestRecord_AppendEveryTick verifies that after every step the record holds the
// observation of that tick for every source: the market's published price, the
// window mean after the push and the published age, a tick in which nothing
// traded included.
//
// Arithmetic: grain in a clears at 1 000 in tick 0 and 1 007 in tick 1, so the
// mean after tick 1 is (1 000 + 1 007) / 2 = 1 003.5, half to even 1 004. Flour
// never trades: its price stays 3 000, its window is empty (mean 0) and its age
// is the ticks run, t + 1.
//
// Requirement: FR-INF-004, FR-INF-005
func TestRecord_AppendEveryTick(t *testing.T) {
	t.Parallel()
	w, _ := newInfoWorld(t, nil)
	for tick := range int64(8) {
		step(t, w, 1)
		if h := w.info.rec.Held(); int64(h) != tick+1 {
			t.Fatalf("tick %d: record holds %d ticks, want %d", tick, h, tick+1)
		}
		for i, s := range w.info.sources {
			got, ok := w.info.rec.At(i, tick)
			rep := &w.markets.reports[s.market]
			mean, _ := w.markets.windows[s.market].Mean()
			if want := (rumour.Obs{Price: int64(rep.Price), Mean: mean, Age: rep.Age}); !ok || got != want {
				t.Errorf("tick %d source %d: observation = %+v (%v), want the market's %+v", tick, i, got, ok, want)
			}
			if s.commodity == flour && got != (rumour.Obs{Price: 3_000, Mean: 0, Age: tick + 1}) {
				t.Errorf("tick %d: flour observation = %+v, want price 3000, no mean, age %d", tick, got, tick+1)
			}
		}
	}
	if got, _ := w.info.rec.At(1, 1); got != (rumour.Obs{Price: 1_007, Mean: 1_004, Age: 0}) {
		t.Errorf("grain in a at tick 1 = %+v, want {1007 1004 0}", got)
	}
}

// TestRecord_EntryIsMarketPublication verifies, over a world in which shipments
// are dispatched and in flight, that the record's entry equals the market's
// publication every tick and that a trip changes no entry: the record is a
// function of the markets alone.
//
// Requirement: FR-INF-028
func TestRecord_EntryIsMarketPublication(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, func(c *Config) {
		c.Roles[0].Informed = true
		c.Modules.Information = &Information{InfoSpeed: 20, RumourMaxAge: 10}
	}, twoTrips)
	for tick := range int64(6) {
		step(t, w, 1)
		for i, s := range w.info.sources {
			got, ok := w.info.rec.At(i, tick)
			rep := &w.markets.reports[s.market]
			mean, _ := w.markets.windows[s.market].Mean()
			if want := (rumour.Obs{Price: int64(rep.Price), Mean: mean, Age: rep.Age}); !ok || got != want {
				t.Errorf("tick %d source %d: observation = %+v (%v), want %+v", tick, i, got, ok, want)
			}
		}
	}
	if len(w.transport.ships) == 0 && w.transport.next == 0 {
		t.Error("no shipment was dispatched, want the trips of twoTrips in the run")
	}
}

// TestRecord_OnlyP52Writes verifies by parsing the root package that the only
// writer of the price record is p52Record (and, for a restore, the section 16
// reader), and that no exported identifier has the record's type.
//
// Guard: the record is a record of what the markets published; a second writer,
// or an exported way to write it, would let a strategy or another module forge
// the news (FR-INF-006).
//
// Requirement: FR-INF-006
func TestRecord_OnlyP52Writes(t *testing.T) {
	t.Parallel()
	fset := token.NewFileSet()
	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("Glob() = %v, want nil", err)
	}
	files := map[string]*ast.File{}
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("ParseFile(%s) = %v, want nil", name, err)
		}
		files[name] = f
	}
	allowed := map[string]bool{"p52Record": true, "loadInformation": true}
	for name, file := range files {
		for _, d := range file.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			ast.Inspect(fn, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || (sel.Sel.Name != "Put" && sel.Sel.Name != "Seal" && sel.Sel.Name != "Load") {
					return true
				}
				if x, ok := sel.X.(*ast.SelectorExpr); ok && x.Sel.Name == "rec" && !allowed[fn.Name.Name] {
					t.Errorf("%s: %s calls rec.%s, want only p52Record to write the record", name, fn.Name.Name, sel.Sel.Name)
				}
				return true
			})
			if fn.Name.IsExported() && fn.Type.Results != nil {
				for _, r := range fn.Type.Results.List {
					if strings.Contains(types(r.Type), "rumour.Record") {
						t.Errorf("%s: exported %s returns the price record", name, fn.Name.Name)
					}
				}
			}
		}
	}
	if f, _ := reflect.TypeOf(World{}).FieldByName("info"); f.IsExported() {
		t.Error("World.info is exported, want it unexported")
	}
}

// types renders a type expression of the parsed source as text.
func types(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.StarExpr:
		return "*" + types(x.X)
	case *ast.SelectorExpr:
		return types(x.X) + "." + x.Sel.Name
	case *ast.Ident:
		return x.Name
	}
	return ""
}

// TestRecord_DepthAndNoSeed verifies that the record is empty at construction
// although every market has a seed price, holds at most RumourMaxAge ticks, and
// refuses a vintage that was overwritten.
//
// Arithmetic: with a horizon of 4 and ticks 0 to 6 written, the record holds
// ticks 3 to 6; tick 2 was overwritten and tick 7 is not written yet.
//
// Requirement: FR-INF-007
func TestRecord_DepthAndNoSeed(t *testing.T) {
	t.Parallel()
	w, _ := newInfoWorld(t, func(c *Config) { c.Modules.Information.RumourMaxAge = 4 })
	if h := w.info.rec.Held(); h != 0 {
		t.Errorf("record holds %d ticks at construction, want 0 (it is not filled from the seed prices)", h)
	}
	if _, ok := w.info.rec.At(0, 0); ok {
		t.Error("At(0, 0) before any step is ok, want no observation")
	}
	step(t, w, 7)
	if h := w.info.rec.Held(); h != 4 {
		t.Errorf("record holds %d ticks after 7, want 4", h)
	}
	for vintage, want := range map[int64]bool{2: false, 3: true, 6: true, 7: false} {
		if _, ok := w.info.rec.At(0, vintage); ok != want {
			t.Errorf("At(0, %d) ok = %v, want %v", vintage, ok, want)
		}
	}
}

// TestLag_FromDistance verifies the lag of every ordered pair of regions of the
// fixture against the table derived from the distances, and TestLag_Asymmetric
// that the lag follows the direction of the edges.
//
// Requirement: FR-INF-008
func TestLag_FromDistance(t *testing.T) {
	t.Parallel()
	w, s := newInfoWorld(t, nil)
	step(t, w, 6)
	n := 0
	for _, v := range viewsAt(s, 5) {
		for _, r := range v.rumours {
			n++
			if want := infoLag[r.Region][v.region]; r.Lag != want {
				t.Errorf("agent %d (region %d): lag of the news from region %d = %d, want %d", v.agent, v.region, r.Region, r.Lag, want)
			}
		}
	}
	if n == 0 {
		t.Error("no rumour was recorded at tick 5, want the fixture's candidates")
	}
}

// TestLag_Asymmetric verifies that the news from a to c takes 4 ticks and the news
// from c to a takes 5, because the distances over the open edges are 120 and 160.
//
// Requirement: FR-INF-009
func TestLag_Asymmetric(t *testing.T) {
	t.Parallel()
	w, s := newInfoWorld(t, nil)
	step(t, w, 6)
	lagOf := func(reader RegionID, from RegionID) int64 {
		for _, v := range viewsAt(s, 5) {
			if v.region == reader {
				for _, r := range v.rumours {
					if r.Region == from && r.Commodity == grain {
						return r.Lag
					}
				}
			}
		}
		return -1
	}
	if ac, ca := lagOf(infoC, infoA), lagOf(infoA, infoC); ac != 4 || ca != 5 {
		t.Errorf("lag a to c = %d and c to a = %d, want 4 and 5", ac, ca)
	}
}

// TestRumours_Cases verifies the four results of a candidate through the tally
// of rumoursFor, one case each and the first that applies winning: a cut graph is
// NoPath (and wins over a lag above the horizon), a short horizon TooOld (and wins
// over a lag above the tick), a young run NotYet.
//
// Arithmetic, reader a with the miller's candidates b flour, b grain, c grain
// (3 in all): the lags are 2 (b) and 5 (c). With c cut off (the edge c to b
// removed) c has no path; with a horizon of 3 the lag 5 is too old; at tick 1 both
// lags are above the tick.
//
// Requirement: FR-INF-010, FR-INF-011, FR-INF-012, FR-INF-013
func TestRumours_Cases(t *testing.T) {
	t.Parallel()
	cut := func(c *Config) { c.Edges = c.Edges[:3] } // the edge c to b removed
	for _, tt := range []struct {
		name  string
		mut   func(c *Config)
		steps int
		want  rumourTally
	}{
		{"all offered", nil, 6, rumourTally{Offered: 3, MaxLag: 5}},
		{"cut graph", cut, 6, rumourTally{Offered: 2, NoPath: 1, MaxLag: 2}},
		{"short horizon", func(c *Config) { c.Modules.Information.RumourMaxAge = 3 }, 6, rumourTally{Offered: 2, TooOld: 1, MaxLag: 2}},
		{"young run", nil, 1, rumourTally{NotYet: 3}},
		{"no path wins over too old", func(c *Config) { cut(c); c.Modules.Information.RumourMaxAge = 3 }, 6, rumourTally{Offered: 2, NoPath: 1, MaxLag: 2}},
		{"too old wins over not yet", func(c *Config) { c.Modules.Information.RumourMaxAge = 3 }, 1, rumourTally{NotYet: 2, TooOld: 1}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _ := newInfoWorld(t, tt.mut)
			step(t, w, tt.steps)
			var got rumourTally
			q := rumourQuery{reader: infoA, commodities: w.roles.rows[roleOf(t, w, "miller")].tradable, asOf: w.Tick()}
			rs := w.rumoursFor(nil, q, &got)
			if got != tt.want || int64(len(rs)) != got.Offered {
				t.Errorf("rumoursFor(reader a, tick %d) tally = %+v with %d rumours, want %+v", w.Tick(), got, len(rs), tt.want)
			}
		})
	}
}

// TestRumours_DrawOrder verifies that the error draws are taken from the stream
// keyed (siteInform, tick, agent), one per candidate, offered or not, in
// ascending source region and then commodity, and that a rumour's price is the
// record's price at the vintage scaled by the draw of its candidate.
//
// Arithmetic, agent 3 (a miller in region a) at tick 5: the candidates are
// (b, flour), (b, grain) and (c, grain), so three draws of drawRange(-500, 500)
// from the keyed stream; the prices are the record's at ticks 3, 3 and 0, each
// times (10 000 + e) / 10 000 half to even. At tick 1 nothing is offered (a young
// run) yet the same three draws are taken.
//
// Requirement: FR-INF-014, FR-INF-015
func TestRumours_DrawOrder(t *testing.T) {
	t.Parallel()
	for _, steps := range []int{1, 5} {
		w, _ := newInfoWorld(t, func(c *Config) { c.Modules.Information.RumourErrorBP = 500 })
		step(t, w, steps)
		tick, a := w.Tick(), &w.agents[3]
		v := AgentView{Region: infoA}
		w.fillRumours(&v, a)

		ref := newStream(w.rnd.seed)
		ref.keyed(siteInform, tick, int64(a.id))
		var want []Rumour
		for _, c := range []struct {
			region RegionID
			comm   CommodityID
			source int
		}{{infoB, flour, 2}, {infoB, grain, 3}, {infoC, grain, 4}} {
			e := ref.drawRange(-500, 500)
			lag := infoLag[c.region][infoA]
			if lag > tick {
				continue
			}
			o, _ := w.info.rec.At(c.source, tick-lag)
			r := Rumour{Region: c.region, Commodity: c.comm, Price: Price(halfEvenScaled(o.Price, e)), Lag: lag, SourceAge: o.Age}
			if o.Mean > 0 {
				r.Mean = Price(halfEvenScaled(o.Mean, e))
			}
			want = append(want, r)
		}
		if !slices.Equal(v.Rumours, want) {
			t.Errorf("tick %d: rumours = %+v, want %+v", tick, v.Rumours, want)
		}
		if got, wantState := pcgState(t, w), func() []byte { b, _ := ref.pcg.MarshalBinary(); return b }(); !bytes.Equal(got, wantState) {
			t.Errorf("tick %d: the stream after the view is not the keyed stream after 3 draws, want one draw per candidate offered or not", tick)
		}
	}
}

// TestRumoursFor_NoViewNoDraw verifies that an agent of a role that is not
// informed gets no rumours and takes no draw, that with an error of 0 an informed
// agent takes none either, and that with an error above 0 it does.
//
// Requirement: FR-INF-014, FR-INF-047, FR-INF-051
func TestRumoursFor_NoViewNoDraw(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		errorBP  int64
		informed bool
		draws    bool
	}{
		{"not informed with an error", 500, false, false},
		{"informed with no error", 0, true, false},
		{"informed with an error", 500, true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _ := newInfoWorld(t, func(c *Config) {
				c.Modules.Information.RumourErrorBP = tt.errorBP
				c.Roles[0].Informed = tt.informed
			})
			step(t, w, 5)
			a := &w.agents[3]
			before := pcgState(t, w)
			v := AgentView{Region: infoA}
			w.fillRumours(&v, a)
			if drew := !bytes.Equal(before, pcgState(t, w)); drew != tt.draws {
				t.Errorf("fillRumours drew = %v, want %v", drew, tt.draws)
			}
			if has := len(v.Rumours) > 0; has != tt.informed {
				t.Errorf("fillRumours gave %d rumours, want some = %v", len(v.Rumours), tt.informed)
			}
		})
	}
}

// TestRumour_LagAgeVintage verifies that a rumour is made from the observation
// of the vintage tick, that Lag and SourceAge are the record's and are not changed
// by the error, and that two agents of a region read the same vintage and differ
// only in price.
//
// Arithmetic, error 0, tick 5: grain in b with lag 2 is the record's price at
// tick 3, infoPrice(b, 3) = 1 100 + 21 = 1 121; flour in b has never traded, so
// its age at tick 3 is 4.
//
// Requirement: FR-INF-016, FR-INF-017, FR-INF-050
func TestRumour_LagAgeVintage(t *testing.T) {
	t.Parallel()
	w, s := newInfoWorld(t, nil)
	step(t, w, 6)
	for _, v := range viewsAt(s, 5) {
		for _, r := range v.rumours {
			vintage := 5 - r.Lag
			switch r.Commodity {
			case grain:
				if want := infoPrice(r.Region, vintage); r.Price != want || r.SourceAge != 0 {
					t.Errorf("agent %d: grain from %d price %d age %d, want the vintage-%d price %d and age 0", v.agent, r.Region, r.Price, r.SourceAge, vintage, want)
				}
			case flour:
				if r.Price != 3_000 || r.Mean != 0 || r.SourceAge != vintage+1 {
					t.Errorf("agent %d: flour from %d = %+v, want price 3000, no mean, age %d", v.agent, r.Region, r, vintage+1)
				}
			}
		}
	}

	e, se := newInfoWorld(t, func(c *Config) { c.Modules.Information.RumourErrorBP = 500 })
	step(t, e, 6)
	byAgent := map[AgentID][]Rumour{}
	for _, v := range viewsAt(se, 5) {
		byAgent[v.agent] = v.rumours
	}
	r3, r4 := byAgent[3], byAgent[4]
	if len(r3) != len(r4) || len(r3) == 0 {
		t.Fatalf("agents 3 and 4 hold %d and %d rumours, want the same non-zero number", len(r3), len(r4))
	}
	differ := false
	for i := range r3 {
		a, b := r3[i], r4[i]
		if a.Region != b.Region || a.Commodity != b.Commodity || a.Lag != b.Lag || a.SourceAge != b.SourceAge {
			t.Errorf("rumour %d differs beyond the price: %+v and %+v", i, a, b)
		}
		differ = differ || a.Price != b.Price
	}
	if !differ {
		t.Error("two agents with an error of 500 read the same prices, want independent errors")
	}
}

// TestRumour_FloorAtMaxSpeed verifies that at the greatest information speed every
// lag is 2 and the vintage is the tick before the previous one, never later.
//
// Requirement: FR-INF-018
func TestRumour_FloorAtMaxSpeed(t *testing.T) {
	t.Parallel()
	w, s := newInfoWorld(t, func(c *Config) { c.Modules.Information.InfoSpeed = MaxInfoSpeed })
	step(t, w, 6)
	for _, v := range viewsAt(s, 5) {
		for _, r := range v.rumours {
			if r.Lag != 2 {
				t.Errorf("agent %d: lag from %d = %d, want 2", v.agent, r.Region, r.Lag)
			}
			if r.Commodity == grain && r.Price != infoPrice(r.Region, 3) {
				t.Errorf("agent %d: grain from %d = %d, want the price of tick 3, %d", v.agent, r.Region, r.Price, infoPrice(r.Region, 3))
			}
		}
	}
}

// TestRumoursFor_AsOfNextTick verifies that the reading of phase 7, as of the
// next tick, finds a rumour a view of the same tick does not, and that its vintage
// is not later than tick t - 1.
//
// Arithmetic: after one step the tick is 1. A view has asOf 1 and the lag 2 is
// above it (NotYet); the migration pass reads as of 2, so the lag 2 is offered
// with the vintage 2 - 2 = 0.
//
// Requirement: FR-INF-018, FR-INF-051
func TestRumoursFor_AsOfNextTick(t *testing.T) {
	t.Parallel()
	w, _ := newInfoWorld(t, nil)
	step(t, w, 1)
	q := rumourQuery{reader: infoA, regions: []RegionID{infoB}, commodities: []CommodityID{grain}, asOf: w.Tick()}
	var tally rumourTally
	if rs := w.rumoursFor(nil, q, &tally); len(rs) != 0 || tally.NotYet != 1 {
		t.Errorf("rumoursFor(asOf 1) = %+v tally %+v, want none and one NotYet", rs, tally)
	}
	q.asOf = w.Tick() + 1
	rs := w.rumoursFor(nil, q, nil)
	o, _ := w.info.rec.At(3, 0)
	if want := []Rumour{{Region: infoB, Commodity: grain, Price: Price(o.Price), Mean: Price(o.Mean), Lag: 2, SourceAge: o.Age}}; !slices.Equal(rs, want) {
		t.Errorf("rumoursFor(asOf 2) = %+v, want %+v", rs, want)
	}
}

// TestRumour_Shape verifies that a rumour has exactly the six fields of the spec,
// with no stock, volume, quote or hop count.
//
// Requirement: FR-INF-019
func TestRumour_Shape(t *testing.T) {
	t.Parallel()
	typ := reflect.TypeOf(Rumour{})
	var got []string
	for i := range typ.NumField() {
		got = append(got, typ.Field(i).Name)
	}
	if want := []string{"Region", "Commodity", "Price", "Mean", "Lag", "SourceAge"}; !slices.Equal(got, want) {
		t.Errorf("Rumour fields = %v, want %v", got, want)
	}
}

// TestView_OrderAndOwnRegion verifies that a view carries at most one rumour per
// (region, commodity), sorted by region then commodity, never about the agent's
// own region and only about commodities of its tradable set.
//
// The carrier trades grain alone, so it is shown no flour although a and b are
// flour sources.
//
// Requirement: FR-INF-020
func TestView_OrderAndOwnRegion(t *testing.T) {
	t.Parallel()
	w, s := newInfoWorld(t, nil)
	step(t, w, 8)
	carrier := roleOf(t, w, "carrier")
	seen := 0
	for _, v := range s.trades {
		seen += len(v.rumours)
		for i, r := range v.rumours {
			if r.Region == v.region {
				t.Errorf("tick %d agent %d: a rumour about its own region %d", v.tick, v.agent, r.Region)
			}
			if i > 0 {
				p := v.rumours[i-1]
				if p.Region > r.Region || (p.Region == r.Region && p.Commodity >= r.Commodity) {
					t.Errorf("tick %d agent %d: rumours %+v then %+v, want strictly ascending (region, commodity)", v.tick, v.agent, p, r)
				}
			}
			if w.agents[v.agent].role == carrier && r.Commodity != grain {
				t.Errorf("tick %d: the carrier was told of commodity %d, want grain alone", v.tick, r.Commodity)
			}
		}
	}
	if seen == 0 {
		t.Error("no rumour in any view, want some")
	}
}

// TestView_TradeViewOfInformedOnly verifies that the Work view of an informed
// agent and every view of an agent of a role that is not informed carry no
// rumours.
//
// Requirement: FR-INF-021
func TestView_TradeViewOfInformedOnly(t *testing.T) {
	t.Parallel()
	w, s := newInfoWorld(t, func(c *Config) { c.Roles[1].Informed = false })
	step(t, w, 6)
	carrier := roleOf(t, w, "carrier")
	informed := 0
	for _, v := range s.works {
		if len(v.rumours) != 0 {
			t.Errorf("tick %d agent %d: the Work view carries %d rumours, want none", v.tick, v.agent, len(v.rumours))
		}
	}
	for _, v := range s.trades {
		if w.agents[v.agent].role == carrier && len(v.rumours) != 0 {
			t.Errorf("tick %d agent %d: a role that is not informed was told %d rumours", v.tick, v.agent, len(v.rumours))
		}
		if w.agents[v.agent].role != carrier && len(v.rumours) > 0 {
			informed++
		}
	}
	if informed == 0 || len(s.works) == 0 {
		t.Errorf("informed views with rumours = %d, Work views = %d, want both above 0", informed, len(s.works))
	}
}

// TestView_NoViewNoRumour verifies that an estate pseudo-agent is never given a
// view and that a removed agent is given none from its removal on, so neither
// gets a rumour or a draw.
//
// Requirement: FR-INF-022
func TestView_NoViewNoRumour(t *testing.T) {
	t.Parallel()
	w, s := newInfoWorld(t, nil)
	step(t, w, 2)
	if err := w.removeAgent(&w.agents[4], CauseConsumer); err != nil {
		t.Fatalf("removeAgent(4) = %v, want nil", err)
	}
	removedAt := w.Tick()
	step(t, w, 3)
	for _, v := range s.trades {
		if int(v.agent) < len(w.cfg.Regions) {
			t.Errorf("tick %d: the estate pseudo-agent %d was given a view", v.tick, v.agent)
		}
		if v.agent == 4 && v.tick >= removedAt {
			t.Errorf("tick %d: the removed agent 4 was given a view", v.tick)
		}
	}
}

// TestRumour_CohortNotMultiplied verifies that an agent of weight above 1 gets
// one view and one set of draws, and that its rumoured prices are the markets'
// prices, not multiplied by its weight.
//
// Arithmetic: the group of region a is 4 households in one agent of weight 4;
// at tick 5 the news of grain in b with lag 2 is the price of tick 3, 1 121,
// whatever the reader's weight. With an error of 500 the agent takes 3 draws.
//
// Requirement: FR-INF-023
func TestRumour_CohortNotMultiplied(t *testing.T) {
	t.Parallel()
	heavy := func(c *Config) {
		c.Groups[0] = Group{Region: "a", Role: "miller", Households: 4, Scale: 4}
	}
	w, s := newInfoWorld(t, heavy)
	step(t, w, 6)
	var got *seenView
	for _, v := range viewsAt(s, 5) {
		if v.region == infoA {
			got = &v
		}
	}
	if got == nil || got.weight != 4 {
		t.Fatalf("view of region a = %+v, want one view of an agent of weight 4", got)
	}
	if r, ok := (&AgentView{Rumours: got.rumours}).Rumour(infoB, grain); !ok || r.Price != infoPrice(infoB, 3) {
		t.Errorf("rumour of grain in b = %+v (%v), want the price %d, not multiplied by the weight", r, ok, infoPrice(infoB, 3))
	}

	e, _ := newInfoWorld(t, func(c *Config) { heavy(c); c.Modules.Information.RumourErrorBP = 500 })
	step(t, e, 6)
	a := agentIn(t, e, infoA)
	v := AgentView{Region: infoA}
	e.fillRumours(&v, a)
	ref := newStream(e.rnd.seed)
	ref.keyed(siteInform, e.Tick(), int64(a.id))
	for range 3 {
		ref.drawRange(-500, 500)
	}
	if want, _ := ref.pcg.MarshalBinary(); !bytes.Equal(pcgState(t, e), want) {
		t.Error("the cohort's view took a number of draws other than its 3 candidates")
	}
}

// TestAgentView_Rumour verifies the lookup of a rumour in a view: found, and not
// found for a region or commodity the view lacks, an empty view and keys beyond
// both ends, with no allocation.
//
// Requirement: FR-INF-024
func TestAgentView_Rumour(t *testing.T) {
	t.Parallel()
	v := AgentView{Rumours: []Rumour{
		{Region: 1, Commodity: 0, Price: 10}, {Region: 1, Commodity: 2, Price: 12},
		{Region: 3, Commodity: 1, Price: 31}, {Region: 5, Commodity: 0, Price: 50},
	}}
	for _, tt := range []struct {
		name   string
		region RegionID
		comm   CommodityID
		want   Price
		ok     bool
	}{
		{"first", 1, 0, 10, true}, {"same region later commodity", 1, 2, 12, true},
		{"middle", 3, 1, 31, true}, {"last", 5, 0, 50, true},
		{"commodity missing in a region", 1, 1, 0, false}, {"region missing", 2, 0, 0, false},
		{"before the first", 0, 0, 0, false}, {"after the last", 6, 0, 0, false},
		{"commodity after the last of a region", 5, 1, 0, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, ok := v.Rumour(tt.region, tt.comm)
			if ok != tt.ok || r.Price != tt.want {
				t.Errorf("Rumour(%d, %d) = %+v, %v, want price %d, %v", tt.region, tt.comm, r, ok, tt.want, tt.ok)
			}
		})
	}
	if _, ok := (&AgentView{}).Rumour(0, 0); ok {
		t.Error("Rumour on an empty view found one, want false")
	}
}

// TestRumour_Allocations verifies, with the default logger and observer, that
// building the rumours of a view and looking one up allocate nothing and that the
// append of the record allocates nothing.
//
// Requirement: NFR-INF-001
func TestRumour_Allocations(t *testing.T) {
	w, _ := newInfoWorld(t, func(c *Config) { c.Modules.Information.RumourErrorBP = 500 })
	step(t, w, 6)
	a, v := &w.agents[3], AgentView{Region: infoA}
	if n := testing.AllocsPerRun(50, func() { v.Rumours = nil; w.fillRumours(&v, a) }); n != 0 {
		t.Errorf("allocations per fillRumours = %v, want 0", n)
	}
	if len(v.Rumours) == 0 {
		t.Fatal("fillRumours gave no rumour, want the measurement to cover a built view")
	}
	if n := testing.AllocsPerRun(50, func() { v.Rumour(infoB, grain) }); n != 0 {
		t.Errorf("allocations per Rumour = %v, want 0", n)
	}
	before := mallocs()
	w.p52Record()
	if n := mallocs() - before; n != 0 {
		t.Errorf("allocations of the append of the record = %d, want 0", n)
	}
}

// TestRumours_StatusChangeStepsVintage verifies, as a unit, the closure of the
// scenario S48: closing an edge raises the lag and steps the vintage back at the
// next view, closing the last way gives NoPath, reopening restores the lags, and a
// rumour already built does not change.
//
// Arithmetic: with an added edge a to c of 40, the news from a to the reader c
// has lag 1 + ceil(40/40) = 2. Closed, it goes a to b to c, 40 + 80 = 120, lag
// 1 + 3 = 4; at tick 6 the vintage steps from 4 to 2. With b to c closed too there
// is no walk from a to c.
//
// Requirement: FR-INF-026, FR-INF-010
func TestRumours_StatusChangeStepsVintage(t *testing.T) {
	t.Parallel()
	w, _ := newInfoWorld(t, func(c *Config) { c.Edges = append(c.Edges, Edge{From: "a", To: "c", Distance: 40}) })
	step(t, w, 6)
	read := func() ([]Rumour, rumourTally) {
		var tally rumourTally
		q := rumourQuery{reader: infoC, regions: []RegionID{infoA}, commodities: []CommodityID{grain}, asOf: w.Tick()}
		return w.rumoursFor(nil, q, &tally), tally
	}
	set := func(from, to RegionID, closed bool) {
		t.Helper()
		if err := w.setEdgeStatuses([]EdgeStatus{{From: from, To: to, Closed: closed, FactorBP: 10_000}}, CauseConsumer); err != nil {
			t.Fatalf("setEdgeStatuses() = %v, want nil", err)
		}
	}
	built, _ := read()
	if len(built) != 1 || built[0].Lag != 2 {
		t.Fatalf("rumours before the closure = %+v, want one with lag 2", built)
	}
	before := slices.Clone(built)
	set(infoA, infoC, true)
	if !slices.Equal(built, before) {
		t.Error("a rumour already built changed with the closure, want it unchanged")
	}
	if after, _ := read(); len(after) != 1 || after[0].Lag != 4 || after[0].Price != Price(mustAt(t, w, 1, 2).Price) {
		t.Errorf("rumours after the closure = %+v, want lag 4 and the price of tick 2", after)
	}
	set(infoB, infoC, true)
	if after, tally := read(); len(after) != 0 || tally.NoPath != 1 {
		t.Errorf("rumours with both ways closed = %+v tally %+v, want none and one NoPath", after, tally)
	}
	set(infoA, infoC, false)
	set(infoB, infoC, false)
	if again, _ := read(); !slices.Equal(again, before) {
		t.Errorf("rumours after the reopening = %+v, want the old %+v", again, before)
	}
}

// mustAt returns the observation of source i at a tick, failing the test when the
// record does not hold it.
func mustAt(t *testing.T, w *World, i int, tick int64) rumour.Obs {
	t.Helper()
	o, ok := w.info.rec.At(i, tick)
	if !ok {
		t.Fatalf("the record holds no observation of source %d at tick %d", i, tick)
	}
	return o
}

// TestRumours_RelocationChangesReader verifies that moving an agent changes only
// the region its rumours are about from its next view, and moves no state of the
// module.
//
// Requirement: FR-INF-027
func TestRumours_RelocationChangesReader(t *testing.T) {
	t.Parallel()
	w, _ := newInfoWorld(t, nil)
	step(t, w, 6)
	a := &w.agents[3]
	rec := w.info.rec.Entries(nil, 0)
	if err := w.relocate(a.id, infoC, CauseConsumer); err != nil {
		t.Fatalf("relocate() = %v, want nil", err)
	}
	if got := w.info.rec.Entries(nil, 0); !slices.Equal(got, rec) {
		t.Error("the record changed with a relocation, want none of the module's state moved")
	}
	v := AgentView{Region: infoC}
	w.fillRumours(&v, a)
	if len(v.Rumours) == 0 {
		t.Fatal("no rumour for the relocated agent, want its new reader's")
	}
	for _, r := range v.Rumours {
		if r.Region == infoC || r.Lag != infoLag[r.Region][infoC] {
			t.Errorf("rumour %+v for a reader in c, want none about c and the lag %d", r, infoLag[r.Region][infoC])
		}
	}
}

// runOnOff runs two worlds of the default strategy for n ticks, the first with the
// module on and its role informed, the second with it off, and returns their
// events (without WorldCreated, which carries the hash and the module list, and
// RumoursServed, which only the module on emits) and snapshots (without the hash and the module list).
func runOnOff(t *testing.T, errorBP int64, n int) (on, off []Event, sOn, sOff Snapshot) {
	t.Helper()
	build := func(on bool) (*World, *evLog) {
		c := infoConfig()
		c.Roles[0].Strategy, c.Roles[1].Strategy = "default", "default"
		if !on {
			c.Modules.Information = nil
			c.Roles[0].Informed, c.Roles[1].Informed = false, false
		} else {
			c.Modules.Information.RumourErrorBP = errorBP
		}
		rec := &evLog{}
		return newMill(t, c, WithSeed(7), WithObserver(rec), WithAudit()), rec
	}
	a, ra := build(true)
	b, rb := build(false)
	step(t, a, n)
	step(t, b, n)
	strip := func(es []Event) []Event {
		return slices.DeleteFunc(slices.Clone(es), func(e Event) bool {
			switch e.(type) {
			case WorldCreated, RumoursServed:
				return true
			}
			return false
		})
	}
	var err error
	if sOn, err = a.Snapshot(); err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	if sOff, err = b.Snapshot(); err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	// The hash and the list of the modules that are on name the module itself.
	sOn.ConfigHash, sOff.ConfigHash, sOn.Modules, sOff.Modules = [32]byte{}, [32]byte{}, nil, nil
	return strip(ra.events), strip(rb.events), sOn, sOff
}

// TestInformation_IgnoringStrategyChangesNothing verifies that a world with the
// module on and its roles informed, whose strategy ignores rumours, gives the same
// events and snapshot as one with it off, apart from the configuration hash, with
// and without an error (the error draws are keyed apart from every other site).
//
// Requirement: FR-INF-025, FR-INF-047
func TestInformation_IgnoringStrategyChangesNothing(t *testing.T) {
	t.Parallel()
	for _, errorBP := range []int64{0, 500} {
		on, off, sOn, sOff := runOnOff(t, errorBP, 30)
		if !reflect.DeepEqual(on, off) {
			t.Errorf("error %d: the events of the world with the module on differ from the world with it off (%d and %d events)", errorBP, len(on), len(off))
		}
		if !reflect.DeepEqual(sOn, sOff) {
			a, b := reflect.ValueOf(sOn), reflect.ValueOf(sOff)
			for i := range a.NumField() {
				if !reflect.DeepEqual(a.Field(i).Interface(), b.Field(i).Interface()) {
					t.Errorf("error %d: Snapshot.%s with the module on differs from the one with it off", errorBP, a.Type().Field(i).Name)
				}
			}
		}
	}
}

// TestInformation_Conservation verifies that the module creates and destroys no
// money or goods: the boundary flows of a world with the module on equal those of
// one with it off, and the audit of both worlds (on in runOnOff) holds.
//
// Requirement: FR-INF-049
func TestInformation_Conservation(t *testing.T) {
	t.Parallel()
	on, off, _, _ := runOnOff(t, 500, 30)
	flows := func(es []Event) (out []BoundaryFlow) {
		for _, e := range es {
			if f, ok := e.(BoundaryFlow); ok {
				out = append(out, f)
			}
		}
		return out
	}
	if a, b := flows(on), flows(off); len(a) == 0 || !slices.Equal(a, b) {
		t.Errorf("boundary flows with the module on = %d rows, with it off = %d, want equal and non-empty", len(a), len(b))
	}
}
