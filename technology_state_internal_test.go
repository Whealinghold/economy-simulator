package economysim

import (
	"log/slog"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/production"
	tec "github.com/Whealinghold/economy-sim/internal/technology"
)

// The state of the technology module (plan 3-tec, Phase 4): the cells, the key and the
// gain reader, on small Go-built worlds. The worlds of the overlays are the tests of
// technology_world_test.go.

// techField is a field on the recipe mill with learning on: Base 100, 300 basis points
// a doubling, a cap of 5 000.
func techField() TechField {
	return TechField{ID: "milling", Recipes: []string{"mill"}, CapBP: 5_000, Base: 100, DoublingBP: 300}
}

// techMill is millConfig with the technology module on and the given fields.
func techMill(duration, households int64, fields ...TechField) Config {
	c := millConfig(duration, households)
	c.Modules.Technology = &Technology{Fields: fields}
	return c
}

// techTwoByTwo is two regions (home, away) and two fields (milling on mill, grinding on
// grind), so a cell index shows both orders.
func techTwoByTwo() Config {
	c := techMill(2, 2, techField())
	grind := c.Recipes[0]
	grind.ID = "grind"
	c.Recipes = append(c.Recipes, grind)
	c.Modules.Technology.Fields = append(c.Modules.Technology.Fields, TechField{ID: "grinding", Recipes: []string{"grind"}, CapBP: 1_000, TrendBP: 40})
	away := DefaultRegion()
	away.ID = "away"
	c.Regions = append(c.Regions, away)
	return c
}

// TestNewTechnology_CellsAscendingZero verifies that a world with two regions and two
// fields holds four cells, all zero, in ascending (region, field) order, and that the
// module is nil when it is off.
//
// Requirement: FR-TEC-004
func TestNewTechnology_CellsAscendingZero(t *testing.T) {
	t.Parallel()
	w := newMill(t, techTwoByTwo())
	if got := len(w.tec.cells); got != 4 {
		t.Fatalf("len(cells) = %d, want 4 (2 regions x 2 fields)", got)
	}
	for i, c := range w.tec.cells {
		if c != (tec.Cell{}) || w.tec.gain[i] != 0 || w.tec.counter[i] != 0 {
			t.Errorf("cell %d = %+v, gain %d, counter %d, want all zero", i, c, w.tec.gain[i], w.tec.counter[i])
		}
	}
	for cell, want := range [][2]RegionID{{0, 0}, {0, 1}, {1, 0}, {1, 1}} {
		region, ok := w.tec.cellOf(want[0], int(want[1]))
		if !ok || region != cell {
			t.Errorf("cellOf(region %d, field %d) = %d, %v, want %d", want[0], want[1], region, ok, cell)
		}
	}
	if off := newMill(t, millConfig(2, 2)); off.tec != nil || off.prod.tec != nil {
		t.Errorf("a world with the module off has tec %v and prod.tec %v, want nil", off.tec, off.prod.tec)
	}
}

// TestNewTechnology_OutsideNodeHasNoCell verifies that an outside node has no cell:
// the world of the external module with a coast node and one ordinary region holds one
// cell, the gain read of the node is neutral and a count in the node adds nothing.
//
// Requirement: FR-TEC-004, FR-TEC-036
func TestNewTechnology_OutsideNodeHasNoCell(t *testing.T) {
	t.Parallel()
	c := extConfig()
	c.Modules.Technology = &Technology{Fields: []TechField{{ID: "making", Recipes: []string{"make"}, CapBP: 1_000, TrendBP: 40}}}
	w, _, _ := extWorld(t, c)
	if len(w.tec.cells) != 1 {
		t.Fatalf("len(cells) = %d, want 1 (the node coast has none)", len(w.tec.cells))
	}
	coast, lowland := RegionID(0), RegionID(1)
	if !w.isOutsideNode(coast) {
		t.Fatalf("region 0 is not the outside node: the fixture changed")
	}
	if _, ok := w.tec.cellOf(coast, 0); ok {
		t.Errorf("cellOf(coast) ok = true, want false")
	}
	w.tec.cells[0].Advance, w.tec.gain[0] = 700, 700
	if got := w.tec.factor(coast, 0); got != production.BP {
		t.Errorf("factor(coast) = %d, want 10 000", got)
	}
	if got := w.tec.factor(lowland, 0); got != production.BP+700 {
		t.Errorf("factor(lowland) = %d, want 10 700", got)
	}
	w.tec.count(coast, 0, 5)
	if w.tec.counter[0] != 0 {
		t.Errorf("a count in the node moved the counter to %d, want 0", w.tec.counter[0])
	}
}

// TestKey_DerivedNoWord verifies that the key is the derived value of the effective
// seed (technology.Key of siteTechnology) and that a world with a discovery field has
// the same construction as the world with the module off: no word was taken, so the
// agents' creation draws and every later draw are the same (FR-TEC-011).
//
// Requirement: FR-TEC-011, FR-TEC-012
func TestKey_DerivedNoWord(t *testing.T) {
	t.Parallel()
	field := techField()
	field.DiscoveryBP, field.StepBP = 5_000, 100
	on := newMill(t, techMill(2, 6, field), WithSeed(7))
	off := newMill(t, millConfig(2, 6), WithSeed(7))
	if want := tec.Key(7, siteTechnology); on.tec.key != want {
		t.Errorf("key = %#x, want tec.Key(7, siteTechnology) = %#x", on.tec.key, want)
	}
	if other := newMill(t, techMill(2, 6, field), WithSeed(8)); other.tec.key == on.tec.key {
		t.Errorf("seeds 7 and 8 give the same key %#x, want a function of the seed", on.tec.key)
	}
	if !reflect.DeepEqual(on.agents, off.agents) {
		t.Errorf("the agents of the world with a discovery field differ from the world with the module off: a word was taken")
	}
}

// TestNames_TechFields verifies the names table: one string per field in ID order, nil
// with the module off, and that the construction summary carries the count of fields
// and the module name.
//
// Requirement: FR-TEC-025
func TestNames_TechFields(t *testing.T) {
	t.Parallel()
	c := techTwoByTwo()
	if got, want := namesOf(&newMill(t, c).cfg).TechFields, []string{"grinding", "milling"}; !reflect.DeepEqual(got, want) {
		t.Errorf("TechFields = %v, want %v (ID order, not declaration order)", got, want)
	}
	off := millConfig(2, 2)
	if got := namesOf(&off).TechFields; got != nil {
		t.Errorf("TechFields with the module off = %v, want nil", got)
	}
	lines := newCapture()
	newMill(t, c, WithLogger(slog.New(lines)))
	created := lines.take("world created")
	if len(created) != 1 || created[0].attrs["fields"] != 2 {
		t.Fatalf("the construction summary = %+v, want one line with fields 2", created)
	}
	if got := modulesOn(&c); got[len(got)-1] != "technology" {
		t.Errorf("modulesOn = %v, want technology listed", got)
	}
	lines = newCapture()
	newMill(t, off, WithLogger(slog.New(lines)))
	if created := lines.take("world created"); len(created) != 1 {
		t.Fatalf("the construction summary with the module off = %+v, want one line", created)
	} else if _, ok := created[0].attrs["fields"]; ok {
		t.Errorf("the summary with the module off has fields, want none")
	}
}

// TestTechFactor_NeutralWhenOff verifies the reader: 10 000 with the module off, for a
// recipe in no field and for an outside node; 10 000 plus the gain otherwise.
//
// Requirement: FR-TEC-019, FR-TEC-034
func TestTechFactor_NeutralWhenOff(t *testing.T) {
	t.Parallel()
	off := newMill(t, millConfig(2, 2))
	if got := off.prod.techFactor(0, 0); got != production.BP {
		t.Errorf("techFactor with the module off = %d, want 10 000", got)
	}
	c := techTwoByTwo()
	c.Modules.Technology.Fields = c.Modules.Technology.Fields[:1] // milling only: grind is in no field
	w := newMill(t, c)
	w.tec.gain[w.tec.nf*1+0] = 900 // the cell of (away, milling)
	for _, tt := range []struct {
		region RegionID
		recipe RecipeID
		want   int64
	}{{0, 1, 10_000}, {1, 1, 10_900}, {1, 0, 10_000}} {
		if got := w.prod.techFactor(tt.region, tt.recipe); got != tt.want {
			t.Errorf("techFactor(region %d, recipe %d) = %d, want %d", tt.region, tt.recipe, got, tt.want)
		}
	}
}

// TestGain_PositionsReadOneValue verifies that the three readers of the effective factor
// in a tick read the one gain: with an advance of 1 000 planted by a call, the cost
// anchor of P1.2, the view that Work receives at P2.5 and the job that P2.7 starts all
// carry 10 000 * 11 000 / 10 000 = 11 000 (the site factor 10 000 times the technology
// term), and a job keeps the yield it started with when the gain changes later.
//
// Requirement: FR-TEC-010, FR-TEC-018
func TestGain_PositionsReadOneValue(t *testing.T) {
	t.Parallel()
	var seen []int64
	s := &tapStrategy{onWork: func(v *AgentView) (JobRequest, bool) {
		seen = append(seen, v.Recipes[0].Effective)
		return JobRequest{}, false
	}}
	c := techMill(3, 2, techField())
	c.Roles[0].Strategy = "tap"
	w := newMill(t, c, WithStrategy("tap", s))
	if err := w.AdjustTechnology([]TechnologyAdjustment{{Region: 0, Field: 0, DeltaBP: 1_000}}); err != nil {
		t.Fatalf("AdjustTechnology() error = %v, want nil", err)
	}
	step(t, w, 1)
	if len(seen) == 0 || seen[0] != 11_000 {
		t.Errorf("Work saw the effective factors %v, want 11 000 first", seen)
	}
	if got := w.prod.anchor[0].YieldBP; got != 11_000 {
		t.Errorf("anchor yield = %d, want 11 000", got)
	}
	var started int
	for i := range w.agents {
		if a := &w.agents[i]; a.hasJob() {
			started++
			if a.job.yield != 11_000 {
				t.Errorf("job of agent %d has yield %d, want 11 000", a.id, a.job.yield)
			}
		}
	}
	if started == 0 {
		t.Fatalf("no job started, want the households to start the recipe")
	}
	if err := w.AdjustTechnology([]TechnologyAdjustment{{Region: 0, Field: 0, DeltaBP: 1_000}}); err != nil {
		t.Fatalf("AdjustTechnology() error = %v, want nil", err)
	}
	for i := range w.agents {
		if a := &w.agents[i]; a.hasJob() && a.job.yield != 11_000 {
			t.Errorf("a gain set during a job moved its yield to %d, want 11 000 kept", a.job.yield)
		}
	}
}
