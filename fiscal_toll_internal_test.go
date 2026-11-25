package economysim

import (
	"slices"
	"testing"
)

// The tests of the tolls of the fiscal module (spec-fiscal.md 7, plan 3-fis Phase 7):
// the crossings of a path, the check of row 22a, the charge of step 3a, the aggregate
// of P3.6c and the toll a route row shows. They run on the world of shipConfig with the
// edges b_far to c_isle added and a polity over the three regions.

// tollJurisdictions are the jurisdictions of the corridor that hold a region: the realm,
// a grouping, and below it hills (seat a_home), inland (seat b_far) and coast (seat
// c_isle) as members names them, each holding 1 000.
func tollJurisdictions(members map[string]string) []Jurisdiction {
	seats := map[string]string{"hills": "a_home", "inland": "b_far", "coast": "c_isle"}
	out := []Jurisdiction{{ID: "realm"}}
	for _, id := range []string{"coast", "hills", "inland"} {
		if slices.Contains([]string{members["a_home"], members["b_far"], members["c_isle"]}, id) {
			out = append(out, Jurisdiction{ID: id, Parent: "realm", Seat: seats[id], StartMoney: 1_000})
		}
	}
	return out
}

// tollWorld builds the corridor a_home, b_far, c_isle (edges of 40 both ways) with the
// regions under the jurisdictions in members and the tolls, every household holding
// 100. The agents: estates 0 to 2, the treasuries of the governed jurisdictions in
// ascending ID after them, then the households, two in a_home and two in b_far.
func tollWorld(t *testing.T, members map[string]string, tolls ...Toll) (*World, *evLog) {
	t.Helper()
	w, rec := newShipWorld(t, func(c *Config) {
		c.Edges = append(c.Edges, Edge{From: "b_far", To: "c_isle", Distance: 40}, Edge{From: "c_isle", To: "b_far", Distance: 40})
		c.Modules.Polity = &PolityConfig{Jurisdictions: tollJurisdictions(members)}
		for _, r := range []string{"a_home", "b_far", "c_isle"} {
			c.Modules.Polity.Members = append(c.Modules.Polity.Members, Member{Region: r, Jurisdiction: members[r]})
		}
		c.Modules.Fiscal = &Fiscal{Tolls: tolls}
	}, nil)
	money := map[AgentID]int64{}
	for i := w.firstHousehold; i < len(w.agents); i++ {
		money[AgentID(i)] = 100
	}
	setBalances(t, w, money)
	rec.events = nil
	return w, rec
}

// inlandBorder is the corridor of spec-fiscal.md 18.2: a_home and b_far are inland
// (the same province, no border between them) and c_isle is the coast.
var inlandBorder = map[string]string{"a_home": "inland", "b_far": "inland", "c_isle": "coast"}

// threeProvinces puts each region in a jurisdiction of its own.
var threeProvinces = map[string]string{"a_home": "hills", "b_far": "inland", "c_isle": "coast"}

// ferryAndGate are the tolls of the example: inland charges 4 a carrying unit on leaving
// it and 2 on entering it, and the coast 7 on entering it, on foot only.
func ferryAndGate() []Toll {
	return []Toll{
		{ID: "ferry", Authority: "inland", On: TollExit, Amount: 4},
		{ID: "gate", Authority: "inland", On: TollEnter, Amount: 2},
		{ID: "pier", Authority: "coast", On: TollEnter, Mode: "foot", Amount: 7},
	}
}

// TestFiscal_TollCrossings verifies which edges a toll acts on (spec-fiscal.md 7.2,
// 18.2): the crossing of each edge of the path and not of the whole route, a pass
// through a jurisdiction that is not left is not charged, a border is charged once per
// crossing, and the mode of a toll limits it.
//
// Arithmetic, per carrying unit at the full scale, on the corridor of ferryAndGate:
// a_home to b_far is inside inland: 0. b_far to c_isle leaves inland (ferry 4) and
// enters coast (pier 7 on foot only): 4 by coach, 11 on foot. a_home to c_isle is the
// same two edges, the first inside inland: the same 4 and 11. c_isle to a_home enters
// inland (gate 2) and then moves inside it: 2 by either mode.
//
// Requirement: FR-FIS-034
func TestFiscal_TollCrossings(t *testing.T) {
	t.Parallel()
	w, _ := tollWorld(t, inlandBorder, ferryAndGate()...)
	tests := []struct {
		name     string
		from, to RegionID
		coach    int64
		foot     int64
	}{
		{"inside one province", regHome, regFar, 0, 0},
		{"leaving inland to the coast", regFar, regIsle, 4, 11},
		{"passing through inland and leaving it", regHome, regIsle, 4, 11},
		{"entering inland from the coast", regIsle, regHome, 2, 2},
		{"entering inland from the coast, one edge", regIsle, regFar, 2, 2},
	}
	for _, tt := range tests {
		if got := w.viewToll(tt.from, tt.to, modeCoach, true); got != tt.coach {
			t.Errorf("viewToll(%s, coach) = %d, want %d", tt.name, got, tt.coach)
		}
		if got := w.viewToll(tt.from, tt.to, modeFoot, true); got != tt.foot {
			t.Errorf("viewToll(%s, foot) = %d, want %d", tt.name, got, tt.foot)
		}
	}
	if got := w.viewToll(regFar, regIsle, modeCoach, false); got != 0 {
		t.Errorf("viewToll of a route that does not exist = %d, want 0", got)
	}
}

// TestFiscal_TollPathOrder verifies that the tolls of a shipment come in path order and,
// on one edge, in ascending toll ID, and that a border is charged each time it is
// crossed, each crossing by itself.
//
// Arithmetic: with three provinces a_home to c_isle has two edges. The first leaves hills
// (a_exit 5) and enters inland (z_gate 2): a_exit before z_gate by ID. The second leaves
// inland (m_ferry 4) and enters coast (no toll). The charges are 5, 2, 4: 11 in all.
//
// Requirement: FR-FIS-034, FR-FIS-036
func TestFiscal_TollPathOrder(t *testing.T) {
	t.Parallel()
	w, _ := tollWorld(t, threeProvinces,
		Toll{ID: "z_gate", Authority: "inland", On: TollEnter, Amount: 2},
		Toll{ID: "a_exit", Authority: "hills", On: TollExit, Amount: 5},
		Toll{ID: "m_ferry", Authority: "inland", On: TollExit, Amount: 4})
	got, total, ok := w.fiscal.charges(w, nil, regHome, regIsle, modeCoach, 1)
	if !ok || total != 11 {
		t.Fatalf("charges() = %+v, total %d, ok %v, want a total of 11", got, total, ok)
	}
	var ids []string
	var dues []int64
	for _, c := range got {
		ids, dues = append(ids, w.fiscal.flows[c.flow].id), append(dues, c.due)
	}
	if !slices.Equal(ids, []string{"a_exit", "z_gate", "m_ferry"}) || !slices.Equal(dues, []int64{5, 2, 4}) {
		t.Errorf("tolls = %v with dues %v, want [a_exit z_gate m_ferry] with [5 2 4]", ids, dues)
	}
}

// TestFiscal_TollTruncatesOnce verifies that a toll truncates once per crossing, over
// the whole of the units: the example of spec-fiscal.md 18.2.
//
// Arithmetic: Amount 3 at a scale of 5 000 on 3 carrying units is
// floor(3 * 5 000 * 3 / 10^4) = floor(4.5) = 4, not 3 * floor(1.5) = 3. A shipment of
// no units counts as one: floor(3 * 5 000 / 10^4) = 1.
//
// Requirement: FR-FIS-034
func TestFiscal_TollTruncatesOnce(t *testing.T) {
	t.Parallel()
	w, _ := tollWorld(t, threeProvinces, Toll{ID: "exit", Authority: "hills", On: TollExit, Amount: 3})
	if err := w.setFiscalScale(0, 5_000, CauseConsumer); err != nil {
		t.Fatalf("setFiscalScale() error = %v, want nil", err)
	}
	for _, tt := range []struct{ n, want int64 }{{3, 4}, {0, 1}, {1, 1}, {2, 3}} {
		if _, got, _ := w.fiscal.charges(w, nil, regHome, regFar, modeCoach, tt.n); got != tt.want {
			t.Errorf("toll of %d units at scale 5000 = %d, want %d", tt.n, got, tt.want)
		}
	}
}

// TestFiscal_TollScaleAndWindow verifies that a toll at scale 0 or outside its window
// charges nothing and that a route row shows the toll at the scale in force.
//
// Arithmetic: ferry is 4 a unit; at scale 5 000 it is 2, at 0 it is 0; with a window
// that starts at tick 5 it is 0 at tick 0.
//
// Requirement: FR-FIS-034, FR-FIS-044
func TestFiscal_TollScaleAndWindow(t *testing.T) {
	t.Parallel()
	late := Toll{ID: "ferry", Authority: "inland", On: TollExit, Amount: 4}
	w, _ := tollWorld(t, inlandBorder, late)
	for _, tt := range []struct{ scale, want int64 }{{10_000, 4}, {5_000, 2}, {0, 0}} {
		if err := w.setFiscalScale(0, tt.scale, CauseConsumer); err != nil {
			t.Fatalf("setFiscalScale(%d) error = %v, want nil", tt.scale, err)
		}
		if got := w.viewToll(regFar, regIsle, modeCoach, true); got != tt.want {
			t.Errorf("toll at scale %d = %d, want %d", tt.scale, got, tt.want)
		}
	}
	late.Window = Window{Start: 5}
	w, _ = tollWorld(t, inlandBorder, late)
	if got := w.viewToll(regFar, regIsle, modeCoach, true); got != 0 {
		t.Errorf("toll before its window = %d, want 0", got)
	}
}

// TestFiscal_RouteToll verifies that the route rows of the view carry the toll of a
// carrying unit: a pure read of the configuration, the scales and the graph, rebuilt
// with the tables of the tick (so a scale that changed shows at the next P3.1), and 0
// with the module off.
//
// Arithmetic: from b_far, on the corridor of ferryAndGate, c_isle costs 4 by coach and
// 4 + 7 = 11 on foot, and a_home 0 (inside inland); at scale 0 the toll is 0.
//
// Requirement: FR-FIS-034
func TestFiscal_RouteToll(t *testing.T) {
	t.Parallel()
	w, _ := tollWorld(t, inlandBorder, ferryAndGate()...)
	rows := func(w *World) map[[2]int]int64 {
		w.p31Fix()
		out := map[[2]int]int64{}
		for _, r := range w.transport.tables.routes[regFar] {
			out[[2]int{int(r.To), int(r.Mode)}] = r.Toll
		}
		return out
	}
	want := map[[2]int]int64{{int(regIsle), int(modeCoach)}: 4, {int(regIsle), int(modeFoot)}: 11, {int(regHome), int(modeCoach)}: 0}
	got := rows(w)
	for k, v := range want {
		if got[k] != v {
			t.Errorf("toll of route (to %d, mode %d) = %d, want %d", k[0], k[1], got[k], v)
		}
	}
	if len(got) == 0 {
		t.Fatal("route rows of b_far = none, want rows")
	}
	for i := range w.fiscal.flows {
		if err := w.setFiscalScale(i, 0, CauseConsumer); err != nil {
			t.Fatalf("setFiscalScale(%d) error = %v, want nil", i, err)
		}
	}
	for k, v := range rows(w) {
		if v != 0 {
			t.Errorf("toll of route (to %d, mode %d) = %d at scale 0, want 0", k[0], k[1], v)
		}
	}
	off, _ := newShipWorld(t, nil, nil)
	for k, v := range rows(off) {
		if v != 0 {
			t.Errorf("toll of route (to %d, mode %d) with the module off = %d, want 0", k[0], k[1], v)
		}
	}
}
