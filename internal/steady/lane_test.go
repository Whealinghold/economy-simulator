package steady_test

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// carrier is the hand model behind the lane tests. One commodity. Region 0 has 10
// households and no upkeep and a farm: duration 1, J 365 (so no capacity row),
// factor 10 000, one unit out per execution. Region 1 has 10 households whose
// upkeep is 10 units a tick and no activity. The lane takes the unit from region 0
// to region 1.
func carrier(ln steady.Lane) steady.Model {
	return steady.Model{
		Commodities: 1,
		Regions:     []steady.Region{{Households: 10, Upkeep: []int64{0}}, {Households: 10, Upkeep: []int64{10}}},
		Year:        365,
		Phi:         steady.Ratio{Num: 1, Den: 1},
		Activities: []steady.Activity{
			{Region: 0, Duration: 1, Starts: 365, FactorSum: 365 * 10000, Outputs: []steady.Item{{Commodity: 0, Qty: 1}}},
		},
		Lanes: []steady.Lane{ln},
	}
}

// entryOf returns the exact value of the entry of col in row, and whether there
// is one.
func entryOf(col steady.Column, row int) (*big.Rat, bool) {
	for _, e := range col.Entries {
		if e.Row != row {
			continue
		}
		if e.Wide != nil {
			return e.Wide, true
		}
		return big.NewRat(e.Val.Num, e.Val.Den), true
	}
	return nil, false
}

// TestBuild_LaneColumn verifies the column of one lane: +1 in the goods row of
// its origin, minus (10 000 - LossBP) / 10 000 in that of its destination, and
// Labour / 10 000 in the labour row of its origin, with cost 0, no capacity row,
// and lambda after it.
//
// Two regions and one commodity: goods rows 0 (region 0) and 1 (region 1), labour
// rows 2 and 3, no capacity row (J d = 365 is not below 365). A loss of 5 000
// leaves 5 000 / 10 000 = 1/2 at the destination; labour 2 500 charges 1/4 of a
// household-tick a unit.
//
// Requirement: FR-VIA-043
func TestBuild_LaneColumn(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		lane   steady.Lane
		to     string
		labour string
	}{
		{"no loss no labour", steady.Lane{From: 0, To: 1}, "-1", ""},
		{"half lost", steady.Lane{From: 0, To: 1, LossBP: 5000, Labour: 2500}, "-1/2", "1/4"},
		{"reverse direction", steady.Lane{From: 1, To: 0, LossBP: 9999, Labour: 10000}, "-1/10000", "1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, l, err := steady.Build(carrier(tt.lane))
			if err != nil {
				t.Fatalf("Build(%+v) error = %v, want nil", tt.lane, err)
			}
			if len(p.Cols) != 3 || l.Lambda != 2 || l.Lanes != 1 || l.GoodsRows != 2 {
				t.Fatalf("Build(%+v): %d columns, lambda at %d, %d lanes, %d goods rows, want 3, 2, 1, 2", tt.lane, len(p.Cols), l.Lambda, l.Lanes, l.GoodsRows)
			}
			col := p.Cols[1]
			if col.Cost.Num != 0 {
				t.Errorf("Build(%+v): lane cost = %v, want 0", tt.lane, col.Cost)
			}
			if v, ok := entryOf(col, tt.lane.From); !ok || v.Cmp(big.NewRat(1, 1)) != 0 {
				t.Errorf("Build(%+v): origin entry = %v, %v, want 1", tt.lane, v, ok)
			}
			if v, ok := entryOf(col, tt.lane.To); !ok || v.RatString() != tt.to {
				t.Errorf("Build(%+v): destination entry = %v, %v, want %s", tt.lane, v, ok, tt.to)
			}
			v, ok := entryOf(col, l.LabourRow(tt.lane.From))
			switch {
			case tt.labour == "" && ok:
				t.Errorf("Build(%+v): labour entry = %v, want none", tt.lane, v)
			case tt.labour != "" && (!ok || v.RatString() != tt.labour):
				t.Errorf("Build(%+v): labour entry = %v, %v, want %s", tt.lane, v, ok, tt.labour)
			}
			if len(col.Entries) > 3 {
				t.Errorf("Build(%+v): lane has %d entries, want at most 3 (no capacity row)", tt.lane, len(col.Entries))
			}
		})
	}
}

// TestSolve_LaneHandLP verifies the optimum of a two-region programme solved by
// hand, certificate included, and that the price difference between the regions
// is the cost of the lane in labour.
//
// Region 1 needs 10 units a tick; the lane moves x units and costs half a
// household-tick a unit (labour 5 000), the farm makes one unit a household-tick.
// With lambda, lane = 10 lambda, farm = lane, labour of region 0: farm + lane / 2
// = 15 lambda <= 10, so lambda = 2/3 and SurplusBP = floor(6 666.7) = 6 666.
// Duals: sum p D = 1 gives p1 = 1/10; the farm is active so p0 = w0; the lane is
// active so p1 = p0 + w0 / 2 = 3 w0 / 2, so w0 = p0 = 1/15; check 10 w0 = 2/3.
// p1 - p0 = 1/30 = w0 * labour / 10 000. Flows at lambda 1: farm 10, lane 10.
//
// Requirement: FR-VIA-043
func TestSolve_LaneHandLP(t *testing.T) {
	t.Parallel()
	o, err := steady.SolveModel(context.Background(), carrier(steady.Lane{From: 0, To: 1, Labour: 5000}), 1000)
	if err != nil {
		t.Fatalf("SolveModel error = %v, want nil", err)
	}
	got := []string{str(o.Lambda), fmt.Sprint(o.SurplusBP), str(o.Price[0]), str(o.Price[1]), str(o.Wage[0]), str(o.Wage[1]), str(o.Flow[0]), str(o.Flow[1])}
	want := []string{"2/3", "6666", "1/15", "1/10", "1/15", "0", "10", "10"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SolveModel(carrier) lambda, bp, p0, p1, w0, w1, farm, lane = %v, want %v", got, want)
	}
	if len(o.Flow) != 2 || len(o.Active) != 1 {
		t.Errorf("SolveModel(carrier): %d flows, %d active, want 2 flows (activity then lane) and 1 active", len(o.Flow), len(o.Active))
	}
}

// TestLane_LabourRowCharged verifies that a lane of labour 10 000 per 10 000 units
// uses one household-tick a unit: with the farm making one unit a household-tick
// and region 0 having 10 households, the lane moves lambda * 10 units and the
// farm and the lane share the 10 household-ticks, so 10 lambda + 10 lambda <= 10
// gives lambda = 1/2.
//
// Requirement: FR-VIA-043
func TestLane_LabourRowCharged(t *testing.T) {
	t.Parallel()
	o, err := steady.SolveModel(context.Background(), carrier(steady.Lane{From: 0, To: 1, Labour: 10000}), 1000)
	if err != nil {
		t.Fatalf("SolveModel error = %v, want nil", err)
	}
	if got := str(o.Lambda); got != "1/2" {
		t.Errorf("SolveModel(carrier, labour 10 000) lambda = %s, want 1/2", got)
	}
}

// TestSolve_NoLaneIsAutarky verifies that a model with no lane builds the
// programme it built before the lane form existed: a nil and an empty lane list
// give the same programme and layout, with one goods row for each commodity and
// lambda straight after the activities.
//
// Guard: the autarky and collapsed free-trade forms are untouched by the lane
// form; without this test a change to the row index would move every table.
//
// Requirement: FR-VIA-043
func TestSolve_NoLaneIsAutarky(t *testing.T) {
	t.Parallel()
	m := oneRegion()
	e := oneRegion()
	e.Lanes = []steady.Lane{}
	p1, l1, err1 := steady.Build(m)
	p2, l2, err2 := steady.Build(e)
	if err1 != nil || err2 != nil {
		t.Fatalf("Build errors = %v, %v, want nil", err1, err2)
	}
	if !reflect.DeepEqual(p1, p2) || !reflect.DeepEqual(l1, l2) {
		t.Errorf("Build(nil lanes) = %v, %v, Build(empty lanes) = %v, %v, want equal", p1, l1, p2, l2)
	}
	if l1.GoodsRows != m.Commodities || l1.Lanes != 0 || l1.Lambda != len(m.Activities) {
		t.Errorf("Build(oneRegion) layout = %+v, want %d goods rows, no lane, lambda at %d", l1, m.Commodities, len(m.Activities))
	}
}

// TestBuild_LaneErrors verifies the refusals of a lane list: a self-lane, a
// negative labour and a loss outside 0 to 9 999 are ErrBadProgramme, a fixed term
// longer than the goods rows is too, and a lane out of range panics as a
// violated programmer invariant.
//
// Requirement: FR-VIA-063
func TestBuild_LaneErrors(t *testing.T) {
	t.Parallel()
	bad := []struct {
		name string
		lane steady.Lane
	}{
		{"self lane", steady.Lane{From: 1, To: 1}},
		{"negative labour", steady.Lane{From: 0, To: 1, Labour: -1}},
		{"negative loss", steady.Lane{From: 0, To: 1, LossBP: -1}},
		{"loss of everything", steady.Lane{From: 0, To: 1, LossBP: 10000}},
	}
	for _, tt := range bad {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, _, err := steady.Build(carrier(tt.lane)); !errors.Is(err, steady.ErrBadProgramme) {
				t.Errorf("Build(%+v) error = %v, want ErrBadProgramme", tt.lane, err)
			}
		})
	}
	t.Run("fixed term longer than the rows", func(t *testing.T) {
		t.Parallel()
		m := carrier(steady.Lane{From: 0, To: 1})
		m.Fixed = make([]steady.Ratio, 3) // 2 regions * 1 commodity = 2 rows
		for i := range m.Fixed {
			m.Fixed[i] = steady.Ratio{Num: 0, Den: 1}
		}
		if _, _, err := steady.Build(m); !errors.Is(err, steady.ErrBadProgramme) {
			t.Errorf("Build(3 fixed terms, 2 rows) error = %v, want ErrBadProgramme", err)
		}
	})
	for _, ln := range []steady.Lane{{From: 0, To: 2}, {From: -1, To: 1}, {From: 0, To: 1, Commodity: 1}} {
		t.Run(fmt.Sprintf("out of range %+v", ln), func(t *testing.T) {
			t.Parallel()
			defer func() {
				if recover() == nil {
					t.Errorf("Build(%+v) did not panic, want a panic for a lane out of range", ln)
				}
			}()
			_, _, _ = steady.Build(carrier(ln))
		})
	}
}

// laneRNG is a small deterministic generator for the oracle models.
type laneRNG uint64

func (r *laneRNG) next(n int64) int64 {
	*r = *r*6364136223846793005 + 1442695040888963407
	return int64(uint64(*r)>>33) % n
}

// randomChain builds a model of 2 or 3 regions and 2 or 3 commodities with
// random activities, in region order, and the same model with the complete set
// of zero-cost lanes in the lane form.
func randomChain(seed uint64) (collapsed, lanes steady.Model) {
	r := laneRNG(seed)
	nr, nc := 2+int(r.next(2)), 2+int(r.next(2))
	m := steady.Model{Commodities: nc, Year: 365, Phi: steady.Ratio{Num: 1, Den: 1}}
	for i := 0; i < nr; i++ {
		up := make([]int64, nc)
		for c := range up {
			up[c] = r.next(4)
		}
		m.Regions = append(m.Regions, steady.Region{Households: 5 + r.next(20), Upkeep: up})
	}
	for i := 0; i < nr; i++ {
		for k := int(r.next(3)) + 1; k > 0; k-- {
			a := steady.Activity{Region: i, Duration: 1 + r.next(5)}
			a.Starts = 365 / a.Duration
			a.FactorSum = a.Starts * (5000 + r.next(10001))
			a.Outputs = []steady.Item{{Commodity: int(r.next(int64(nc))), Qty: 1 + r.next(6)}}
			if r.next(2) == 0 {
				in := int(r.next(int64(nc)))
				if in != a.Outputs[0].Commodity {
					a.Inputs = []steady.Item{{Commodity: in, Qty: 1}}
				}
			}
			m.Activities = append(m.Activities, a)
		}
	}
	collapsed = m
	lanes = m
	for from := 0; from < nr; from++ {
		for to := 0; to < nr; to++ {
			for c := 0; c < nc && from != to; c++ {
				lanes.Lanes = append(lanes.Lanes, steady.Lane{From: from, To: to, Commodity: c})
			}
		}
	}
	return collapsed, lanes
}

// TestSolve_LanesOracleModel verifies that the complete set of zero-cost lanes in
// the lane form and the collapsed free-trade model agree on the surplus factor,
// its basis points, and on whether the programme has an optimum, over 200
// generated models. The seed is in the message.
//
// Requirement: FR-VIA-043, FR-VIA-044
func TestSolve_LanesOracleModel(t *testing.T) {
	t.Parallel()
	solved := 0
	for seed := uint64(1); seed <= 200; seed++ {
		collapsed, lanes := randomChain(seed)
		a, errA := steady.SolveModel(context.Background(), collapsed, 100_000)
		b, errB := steady.SolveModel(context.Background(), lanes, 100_000)
		if (errA == nil) != (errB == nil) || (errA != nil && !errors.Is(errB, errA) && errA.Error() != errB.Error()) {
			t.Fatalf("seed %d: collapsed error = %v, lane form error = %v, want the same", seed, errA, errB)
		}
		if errA != nil {
			continue
		}
		solved++
		if str(a.Lambda) != str(b.Lambda) || a.SurplusBP != b.SurplusBP {
			t.Errorf("seed %d: collapsed lambda %s (%d bp), lane form lambda %s (%d bp), want equal", seed, str(a.Lambda), a.SurplusBP, str(b.Lambda), b.SurplusBP)
		}
	}
	if solved < 50 {
		t.Errorf("only %d of 200 generated models solved, want at least 50 so the oracle is exercised", solved)
	}
}
