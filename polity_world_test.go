package economysim_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// Region and jurisdiction IDs of example A (spec-polity.md 12.1) in the world of
// polityWorldConfig: regions by ID order capital 0, farm 1, harbour 2, hills 3,
// mill 4, north 5; jurisdictions coast 0, crown 1, inland 2, uplands 3.
const (
	rCapital, rFarm, rHarbour, rHills, rMill, rNorth = 0, 1, 2, 3, 4, 5
	jCoast, jCrown, jInland, jUplands                = 0, 1, 2, 3
)

// polityWorldConfig is polityConfig with the ten edges of example A (12.1), so the
// graph has the edge IDs the spec lists: 0 capital>harbour, 1 capital>mill, 2
// farm>mill, 3 harbour>capital, 4 harbour>mill, 5 hills>mill, 6 mill>capital, 7
// mill>farm, 8 mill>harbour, 9 mill>hills.
func polityWorldConfig() economysim.Config {
	c := polityConfig()
	for _, e := range [][2]string{
		{"capital", "harbour"}, {"capital", "mill"}, {"farm", "mill"}, {"harbour", "capital"}, {"harbour", "mill"},
		{"hills", "mill"}, {"mill", "capital"}, {"mill", "farm"}, {"mill", "harbour"}, {"mill", "hills"},
	} {
		c.Edges = append(c.Edges, economysim.Edge{From: e[0], To: e[1], Distance: 10})
	}
	return c
}

// polityWorld builds a world over polityWorldConfig.
func polityWorld(t testing.TB, opts ...economysim.Option) *economysim.World {
	t.Helper()
	w, err := economysim.NewWorld(polityWorldConfig(), opts...)
	if err != nil {
		t.Fatalf("NewWorld(polityWorldConfig()) error = %v, want nil", err)
	}
	return w
}

func jids(ids ...int) []economysim.JurisdictionID {
	out := make([]economysim.JurisdictionID, len(ids))
	for i, id := range ids {
		out[i] = economysim.JurisdictionID(id)
	}
	return out
}

// TestPolity_Queries verifies every query of spec-polity.md 4.2.1 through a world
// built over example A: the chains, covers, common jurisdiction, covered regions,
// crossings of the table of 12.1, and the parent, depth, children, seat and
// governed lists.
//
// Requirement: FR-POL-006, FR-POL-007, FR-POL-008, FR-POL-009, FR-POL-010, FR-POL-014, FR-POL-002
func TestPolity_Queries(t *testing.T) {
	t.Parallel()
	p := polityWorld(t).Polity()
	if p.Len() != 4 {
		t.Fatalf("Len() = %d, want 4", p.Len())
	}
	for id, want := range []string{"coast", "crown", "inland", "uplands"} {
		if got := p.Name(economysim.JurisdictionID(id)); got != want {
			t.Errorf("Name(%d) = %q, want %q (ascending by ID string)", id, got, want)
		}
		if j, ok := p.Lookup(want); !ok || int(j) != id {
			t.Errorf("Lookup(%q) = %d, %v, want %d, true", want, j, ok, id)
		}
	}
	if j, ok := p.Lookup("nowhere"); ok || j != economysim.NoJurisdiction {
		t.Errorf("Lookup(nowhere) = %d, %v, want NoJurisdiction, false", j, ok)
	}
	chains := map[economysim.RegionID][]economysim.JurisdictionID{
		rCapital: jids(jCrown), rFarm: jids(jInland, jCrown), rHarbour: jids(jCoast, jCrown),
		rHills: jids(jUplands, jInland, jCrown), rMill: jids(jInland, jCrown), rNorth: jids(jCrown),
	}
	for r, want := range chains {
		if got := p.Chain(r); !slices.Equal(got, want) {
			t.Errorf("Chain(%d) = %v, want %v", r, got, want)
		}
		if got, _ := p.Of(r); got != want[0] {
			t.Errorf("Of(%d) = %d, want %d", r, got, want[0])
		}
	}
	for _, tt := range []struct {
		from, to    economysim.RegionID
		exit, enter []economysim.JurisdictionID
		common      economysim.JurisdictionID
		depth       int
	}{
		{rHills, rHarbour, jids(jUplands, jInland), jids(jCoast), jCrown, 0},
		{rHarbour, rHills, jids(jCoast), jids(jInland, jUplands), jCrown, 0},
		{rFarm, rMill, nil, nil, jInland, 1},
		{rFarm, rCapital, jids(jInland), nil, jCrown, 0},
		{rHills, rFarm, jids(jUplands), nil, jInland, 1},
		{rCapital, rMill, nil, jids(jInland), jCrown, 0},
	} {
		got := p.Crossing(tt.from, tt.to)
		if !slices.Equal(got.Exit, tt.exit) || !slices.Equal(got.Enter, tt.enter) {
			t.Errorf("Crossing(%d, %d) = %v, want exit %v enter %v", tt.from, tt.to, got, tt.exit, tt.enter)
		}
		if c := p.Common(tt.from, tt.to); c != tt.common {
			t.Errorf("Common(%d, %d) = %d, want %d", tt.from, tt.to, c, tt.common)
		}
		if d, ok := p.Depth(p.Common(tt.from, tt.to)); !ok || d != tt.depth {
			t.Errorf("border depth of (%d, %d) = %d, %v, want %d", tt.from, tt.to, d, ok, tt.depth)
		}
	}
	if got := p.Regions(jInland); !slices.Equal(got, []economysim.RegionID{rFarm, rHills, rMill}) {
		t.Errorf("Regions(inland) = %v, want [farm hills mill]", got)
	}
	if !p.Covers(jCrown, rHills) || p.Covers(jCoast, rHills) {
		t.Error("Covers(crown, hills) must hold and Covers(coast, hills) must not")
	}
	if par, ok := p.Parent(jUplands); !ok || par != jInland {
		t.Errorf("Parent(uplands) = %d, %v, want inland, true", par, ok)
	}
	if _, ok := p.Parent(jCrown); ok {
		t.Error("Parent(crown) ok, want false for a root")
	}
	if d, ok := p.Depth(jUplands); !ok || d != 2 {
		t.Errorf("Depth(uplands) = %d, %v, want 2, true", d, ok)
	}
	if got := p.Children(jCrown); !slices.Equal(got, jids(jCoast, jInland)) {
		t.Errorf("Children(crown) = %v, want [coast inland]", got)
	}
	if got := p.Governed(); !slices.Equal(got, jids(jCrown, jInland)) {
		t.Errorf("Governed() = %v, want [crown inland]", got)
	}
	if r, ok := p.Seat(jInland); !ok || r != rMill {
		t.Errorf("Seat(inland) = %d, %v, want mill, true", r, ok)
	}
	if _, ok := p.Seat(jCoast); ok {
		t.Error("Seat(coast) ok, want false for a grouping")
	}
	// Out of range: the zero result and no panic (FR-POL-012).
	for _, id := range []economysim.JurisdictionID{-5, 4, 1 << 30} {
		if p.Name(id) != "" || p.Children(id) != nil || p.Regions(id) != nil || p.Covers(id, 0) {
			t.Errorf("a query with the jurisdiction %d did not answer none", id)
		}
		if _, ok := p.Seat(id); ok {
			t.Errorf("Seat(%d) ok, want false", id)
		}
	}
	if c := p.Common(-1, 99); c != economysim.NoJurisdiction {
		t.Errorf("Common(-1, 99) = %d, want NoJurisdiction", c)
	}
}

// TestPolity_Between verifies Between on example A: the edges harbour>mill [4] and
// mill>harbour [8] between the coast and the inland province, none between nested
// or equal jurisdictions, a closed edge still listed, and the chains and crossings
// unchanged by closing edges 4 and 8 (an embargo of the coast, spec-polity.md 12.1).
//
// Requirement: FR-POL-011, FR-POL-033
func TestPolity_Between(t *testing.T) {
	t.Parallel()
	w := polityWorld(t)
	p, g := w.Polity(), w.Graph()
	for _, tt := range []struct {
		name string
		a, b economysim.JurisdictionID
		want []economysim.EdgeID
	}{
		{"coast into inland", jCoast, jInland, []economysim.EdgeID{4}},
		{"inland into coast", jInland, jCoast, []economysim.EdgeID{8}},
		{"nested, outward", jCoast, jCrown, nil},
		{"nested, inward", jCrown, jCoast, nil},
		{"equal", jInland, jInland, nil},
		{"unknown", -1, jInland, nil},
	} {
		if got := p.Between(g, tt.a, tt.b); !slices.Equal(got, tt.want) {
			t.Errorf("Between(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
	before := p.Crossing(rHarbour, rMill)
	if err := w.SetEdgeStatuses([]economysim.EdgeStatus{
		{From: rHarbour, To: rMill, Closed: true, FactorBP: 10_000}, {From: rMill, To: rHarbour, Closed: true, FactorBP: 10_000},
	}); err != nil {
		t.Fatalf("SetEdgeStatuses error = %v, want nil", err)
	}
	if got := p.Between(w.Graph(), jCoast, jInland); !slices.Equal(got, []economysim.EdgeID{4}) {
		t.Errorf("Between(coast, inland) with the edge closed = %v, want [4]: a closed edge is listed", got)
	}
	after := w.Polity()
	if !reflect.DeepEqual(after, p) || !reflect.DeepEqual(after.Crossing(rHarbour, rMill), before) {
		t.Error("closing edges changed the polity or a crossing, want neither")
	}
}

// TestPolity_DeepCopy verifies that mutating a returned list changes nothing, that
// two calls give equal values and that the value is readable on a failed world.
//
// Requirement: FR-POL-013
func TestPolity_DeepCopy(t *testing.T) {
	t.Parallel()
	w := polityWorld(t)
	p := w.Polity()
	c := p.Chain(rHills)
	c[0] = 99
	p.Governed()[0] = 99
	p.Regions(jCrown)[0] = 99
	p.Children(jCrown)[0] = 99
	if got := p.Chain(rHills); got[0] != jUplands {
		t.Errorf("Chain(hills) = %v after mutating a copy, want it unchanged", got)
	}
	if !reflect.DeepEqual(w.Polity(), w.Polity()) || !reflect.DeepEqual(w.Polity(), polityWorld(t).Polity()) {
		t.Error("World.Polity() is not equal across calls and worlds")
	}
	a, b := w.Polity(), w.Polity()
	a.Chain(rHills)[0] = 7
	if b.Chain(rHills)[0] != jUplands {
		t.Error("two values from World.Polity() share storage")
	}
}

// TestPolity_OffIsEmpty verifies that a world with the module off returns the empty
// Polity and every query answers none.
//
// Requirement: FR-POL-001
func TestPolity_OffIsEmpty(t *testing.T) {
	t.Parallel()
	c := referenceConfig()
	w, err := economysim.NewWorld(c)
	if err != nil {
		t.Fatalf("NewWorld error = %v, want nil", err)
	}
	p := w.Polity()
	if p.Len() != 0 || len(p.Chain(0)) != 0 || p.Common(0, 0) != economysim.NoJurisdiction || len(p.Governed()) != 0 ||
		len(p.Crossing(0, 0).Exit) != 0 || p.Regions(0) != nil || p.Name(0) != "" {
		t.Errorf("Polity() of a world with the module off = %+v, want none", p)
	}
	if _, ok := p.Of(0); ok {
		t.Error("Of(0) ok, want false")
	}
	if !reflect.DeepEqual(p, economysim.Polity{}) {
		t.Error("the polity of a world with the module off must be the zero value")
	}
	snap, err := w.Snapshot()
	if err != nil || snap.Names.Jurisdictions != nil || slices.Contains(snap.Modules, "polity") {
		t.Errorf("Snapshot() = %v, %v, want Names.Jurisdictions nil and no polity module", snap.Modules, err)
	}
}

// TestPolity_AllocsPerRun verifies that the scalar queries allocate nothing.
//
// Requirement: NFR-POL-001
func TestPolity_AllocsPerRun(t *testing.T) {
	p := polityWorld(t).Polity()
	var sink int
	allocs := testing.AllocsPerRun(100, func() {
		_ = p.Name(jCrown)
		j, _ := p.Lookup("inland")
		par, _ := p.Parent(j)
		d, _ := p.Depth(par)
		o, _ := p.Of(rHills)
		s, _ := p.Seat(j)
		if p.Covers(o, rMill) {
			sink++
		}
		sink += int(p.Common(rHills, rHarbour)) + p.Len() + d + int(s)
	})
	if allocs != 0 {
		t.Errorf("the scalar queries allocated %v times per run, want 0 (sink %d)", allocs, sink)
	}
}

// TestPolity_NewPolity verifies that NewPolity runs the rules of spec-polity.md 9
// and builds the value the world holds, and refuses a mistake with its sentinel.
//
// Requirement: FR-POL-002, FR-POL-003
func TestPolity_NewPolity(t *testing.T) {
	t.Parallel()
	got, err := economysim.NewPolity(polityWorldConfig())
	if err != nil {
		t.Fatalf("NewPolity error = %v, want nil", err)
	}
	if want := polityWorld(t).Polity(); !reflect.DeepEqual(got, want) {
		t.Errorf("NewPolity() = %+v, want the value of the world %+v", got, want)
	}
	bad := polityWorldConfig()
	bad.Modules.Polity.Jurisdictions[1].Parent = "coast"
	if _, err := economysim.NewPolity(bad); err == nil {
		t.Error("NewPolity(cycle) error = nil, want ErrJurisdictionCycle")
	}
	if p, err := economysim.NewPolity(referenceConfig()); err != nil || p.Len() != 0 {
		t.Errorf("NewPolity(module off) = %+v, %v, want the empty polity", p, err)
	}
}

// TestPolity_NoRegionFieldRead verifies by a text check that no field of a region
// is read from a jurisdiction (spec-polity.md 6.1.4): no root Go file other than
// polity.go and its tests mentions the polity types or the jurisdiction queries in
// a rule path, and polity.go reads none of the region tunables of
// spec-regions.md 5.1.3.
//
// Requirement: FR-POL-024
//
// Guard: the spec forbids inheritance of a region field down the tree; a reader
// that took a price or a yield from a jurisdiction would change results without
// changing the hash of any region.
func TestPolity_NoRegionFieldRead(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{"polity.go": true, "config.go": true, "construct.go": true, "event.go": true, "world.go": true, "validate.go": true,
		"hash.go": true, "errors.go": true, "bounds.go": true, "defaults.go": true, "doc.go": true}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || allowed[f] {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "Jurisdiction") || strings.Contains(string(b), "w.polity") {
			t.Errorf("%s mentions a jurisdiction: only polity.go and the catalogue files may (architecture.md 2.10.2)", f)
		}
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "polity.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	region := map[string]bool{"ReferencePrice": true, "ExtraUpkeep": true, "ReferenceBasket": true, "Yields": true, "StartMoney": false}
	ast.Inspect(file, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok && region[sel.Sel.Name] {
			t.Errorf("polity.go:%d reads %s: a jurisdiction inherits no region field", fset.Position(sel.Pos()).Line, sel.Sel.Name)
		}
		return true
	})
}

// TestPolity_NoDraws verifies by a text check that the module draws nothing: neither
// polity.go nor internal/polity imports a random source, and neither calls the
// world's. The equal results of the neutrality proof (S112) are the behavioural half.
//
// Requirement: FR-POL-020
func TestPolity_NoDraws(t *testing.T) {
	t.Parallel()
	for _, f := range []string{"polity.go", "internal/polity/polity.go"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, banned := range []string{"math/rand", "crypto/rand", "w.rand", "keyed(", "time.Now"} {
			if strings.Contains(string(b), banned) {
				t.Errorf("%s contains %q: the module takes no draw and reads no clock", f, banned)
			}
		}
	}
}

// TestPolity_Observation verifies that a logger and an observer change no result
// (NFR-POL-003) and that the construction line carries the module name and the
// counts: polity in modules, jurisdictions 4 and governed 2 (FR-POL-025).
//
// Requirement: NFR-POL-003, FR-POL-025
func TestPolity_Observation(t *testing.T) {
	t.Parallel()
	h := newCaptureHandler()
	plain := polityWorld(t)
	watched := polityWorld(t, economysim.WithLogger(slog.New(h)), economysim.WithObserver(economysim.ObserverFunc(func(economysim.Event) {})))
	for range 3 {
		if err := plain.Step(); err != nil {
			t.Fatal(err)
		}
		if err := watched.Step(); err != nil {
			t.Fatal(err)
		}
	}
	a, _ := plain.Snapshot()
	b, _ := watched.Snapshot()
	if !reflect.DeepEqual(a, b) {
		t.Error("a logger and an observer changed the snapshot")
	}
	var attrs map[string]string
	for _, l := range h.sink.lines {
		if l.msg == "world created" {
			attrs = map[string]string{}
			for _, at := range l.attrs {
				attrs[at.Key] = at.Value.String()
			}
		}
	}
	if attrs == nil {
		t.Fatalf("no world created line in %v", h.sink.lines)
	}
	if attrs["jurisdictions"] != "4" || attrs["governed"] != "2" || !strings.Contains(attrs["modules"], "polity") {
		t.Errorf("world created attrs = %v, want jurisdictions 4, governed 2 and polity in modules", attrs)
	}
}

// TestPolity_Deterministic verifies that two worlds from one configuration and seed
// give the same polity and the same snapshot after some steps, with no map in a list.
//
// Requirement: NFR-POL-004
func TestPolity_Deterministic(t *testing.T) {
	t.Parallel()
	one, two := polityWorld(t, economysim.WithSeed(5)), polityWorld(t, economysim.WithSeed(5))
	for range 5 {
		if err := one.Step(); err != nil {
			t.Fatal(err)
		}
		if err := two.Step(); err != nil {
			t.Fatal(err)
		}
	}
	a, _ := one.Snapshot()
	b, _ := two.Snapshot()
	if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(one.Polity(), two.Polity()) {
		t.Error("two runs of one configuration and seed differ")
	}
}

// TestPolity_BuildCost verifies the structure of the construction cost of
// spec-polity.md 4.4.2 at the three sizes: the number of entries the tables hold is
// the sum of the chain lengths (n * depth) and not more, counted from the public
// queries, with no timing assertion.
//
// Requirement: NFR-POL-002
func TestPolity_BuildCost(t *testing.T) {
	t.Parallel()
	for _, size := range []struct{ j, n, levels int }{{4, 5, 2}, {40, 100, 4}, {1000, 1000, 8}} {
		c := referenceConfig()
		c.Regions = c.Regions[:0]
		pc := &economysim.PolityConfig{}
		for i := range size.j {
			j := economysim.Jurisdiction{ID: "j" + itoa4(i)}
			if i%size.levels != 0 {
				j.Parent = "j" + itoa4(i-1) // a forest of chains of size.levels levels
			}
			pc.Jurisdictions = append(pc.Jurisdictions, j)
		}
		for i := range size.n {
			r := economysim.DefaultRegion()
			r.ID = "r" + itoa4(i)
			c.Regions = append(c.Regions, r)
			pc.Members = append(pc.Members, economysim.Member{Region: r.ID, Jurisdiction: "j" + itoa4(i%size.j)})
		}
		c.Groups = nil
		c.Modules.Polity = pc
		p, err := economysim.NewPolity(c)
		if err != nil {
			t.Fatalf("NewPolity(%d jurisdictions, %d regions) error = %v, want nil", size.j, size.n, err)
		}
		entries := 0
		for r := range size.n {
			entries += len(p.Chain(economysim.RegionID(r)))
		}
		if bound := size.n * size.levels; entries > bound {
			t.Errorf("chains hold %d entries at (%d, %d, %d), want at most n * levels = %d", entries, size.j, size.n, size.levels, bound)
		}
	}
}

// itoa4 writes i as four decimal digits, so IDs sort as numbers.
func itoa4(i int) string {
	return string([]byte{'0' + byte(i/1000%10), '0' + byte(i/100%10), '0' + byte(i/10%10), '0' + byte(i%10)})
}

// Example_polityCrossing shows what a good leaves and enters going from the hills to
// the harbour in example A of spec-polity.md 12.1.
func Example_polityCrossing() {
	p, err := economysim.NewPolity(polityWorldConfig())
	if err != nil {
		panic(err)
	}
	names := func(ids []economysim.JurisdictionID) []string {
		out := []string{}
		for _, id := range ids {
			out = append(out, p.Name(id))
		}
		return out
	}
	x := p.Crossing(rHills, rHarbour)
	fmt.Println("exit:", names(x.Exit), "enter:", names(x.Enter))
	// Output: exit: [uplands inland] enter: [coast]
}
