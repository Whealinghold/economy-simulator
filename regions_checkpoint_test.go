package economysim_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// regionsCfg is the reference configuration with the regions east and south, a
// group of cooks in each, and five edges: IDs sort east (0), north (1), south (2),
// and the EdgeIDs are 0 east>north 30, 1 east>south 90, 2 north>east 30, 3
// north>south 30, 4 south>north 30.
func regionsCfg() economysim.Config {
	c := withRegions(referenceConfig(), "east", "south")
	for _, id := range []string{"east", "south"} {
		c.Groups = append(c.Groups, economysim.Group{Region: id, Role: "cook", Households: 10})
	}
	c.Edges = []economysim.Edge{edgeOf("east", "north", 30), edgeOf("east", "south", 90), edgeOf("north", "east", 30), edgeOf("north", "south", 30), edgeOf("south", "north", 30)}
	return c
}

// regionsStatuses are three non-default statuses by EdgeID 0, 1 and 3: closed, closed
// with a factor that is kept, and a factor of 15 000.
var regionsStatuses = []economysim.EdgeStatus{
	{From: 0, To: 1, Closed: true, FactorBP: 10_000},
	{From: 0, To: 2, Closed: true, FactorBP: 20_000},
	{From: 1, To: 2, FactorBP: 15_000},
}

// regionsBody is the body of tag 11 for rows of (From, To, Closed, FactorBP): the
// count, then each row as four varints (spec-checkpoint.md 4.2, tag 11).
func regionsBody(rows ...[4]uint64) []byte {
	b := binary.AppendUvarint(nil, uint64(len(rows)))
	for _, r := range rows {
		for _, v := range r {
			b = binary.AppendUvarint(b, v)
		}
	}
	return b
}

// regionsWorld returns a world on regionsCfg after 3 ticks with the statuses set.
func regionsWorld(t *testing.T, statuses []economysim.EdgeStatus) *economysim.World {
	t.Helper()
	w, err := economysim.NewWorld(regionsCfg(), economysim.WithSeed(1))
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	for range 3 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}
	if statuses != nil {
		if err := economysim.SetEdgeStatusesForTest(w, statuses, economysim.CauseConsumer); err != nil {
			t.Fatalf("SetEdgeStatusesForTest() error = %v, want nil", err)
		}
	}
	return w
}

// graphAnswers is every answer of the queries of spec-regions.md 4.3.2 for every
// region and pair, as text, so that two graphs are compared by what they say.
func graphAnswers(g economysim.Graph) []string {
	out := []string{fmt.Sprint("len ", g.Len(), " edges ", g.Edges(), " components ", g.Components())}
	for i := range g.Edges() {
		e, ok := g.EdgeAt(economysim.EdgeID(i))
		out = append(out, fmt.Sprint("edge ", e, ok))
	}
	for a := range economysim.RegionID(g.Len()) {
		c, ok := g.Component(a)
		out = append(out, fmt.Sprint("comp ", a, c, ok, g.Out(a), g.In(a), g.Within(a, 100)))
		for b := range economysim.RegionID(g.Len()) {
			d, dok := g.Distance(a, b)
			n, nok := g.NextHop(a, b)
			p, pok := g.Path(a, b)
			e, eok := g.Edge(a, b)
			out = append(out, fmt.Sprint(a, b, d, dok, n, nok, p, pok, e, eok))
		}
	}
	return out
}

// TestCheckpoint_RegionsSection verifies tag 11: a world with three non-default
// statuses writes them as (From, To, Closed, FactorBP) in ascending EdgeID, a world
// with none writes the count 0, and a restore gives a world whose snapshot, queries
// and next checkpoint equal the original's, including a closed edge that keeps a
// factor other than 10 000.
//
// Arithmetic: the rows are the EdgeIDs 0 (east>north, closed), 1 (east>south, closed,
// factor 20 000) and 3 (north>south, factor 15 000), so the body is the count 3 and
// (0,1,1,10000), (0,2,1,20000), (1,2,0,15000).
//
// Requirement: FR-REG-052, FR-CHK-009
func TestCheckpoint_RegionsSection(t *testing.T) {
	t.Parallel()
	w := regionsWorld(t, regionsStatuses)
	raw := mustMarshal(t, mustCheckpoint(t, w))
	want := regionsBody([4]uint64{0, 1, 1, 10_000}, [4]uint64{0, 2, 1, 20_000}, [4]uint64{1, 2, 0, 15_000})
	if got := decodeFrame(t, raw).sectionOf(t, 11); !bytes.Equal(got, want) {
		t.Errorf("tag 11 = % x, want % x", got, want)
	}
	var ck economysim.Checkpoint
	if err := ck.UnmarshalBinary(raw); err != nil {
		t.Fatalf("UnmarshalBinary() error = %v, want nil", err)
	}
	r, err := economysim.RestoreWorld(regionsCfg(), ck)
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	if got, want := graphAnswers(r.Graph()), graphAnswers(w.Graph()); !slices.Equal(got, want) {
		t.Errorf("the restored world's graph answers differ from the original's:\n got %q\nwant %q", got, want)
	}
	if a, b := mustSnapshot(t, w), mustSnapshot(t, r); !reflect.DeepEqual(a.Edges, b.Edges) || a.Components != b.Components || len(b.Edges) != 3 {
		t.Errorf("restored snapshot edges %+v and %d components, want those of the original %+v and %d", b.Edges, b.Components, a.Edges, a.Components)
	}
	if again := mustMarshal(t, mustCheckpoint(t, r)); !bytes.Equal(raw, again) {
		t.Errorf("the restored world's checkpoint differs from the one it was restored from")
	}

	none := mustMarshal(t, mustCheckpoint(t, regionsWorld(t, nil)))
	if got := decodeFrame(t, none).sectionOf(t, 11); !bytes.Equal(got, []byte{0}) {
		t.Errorf("tag 11 of a world with no status = % x, want the count 0", got)
	}
}

// TestRestore_RegionsSectionCorrupt verifies that the reader of tag 11 refuses, with
// ErrCheckpointCorrupt naming section 11, a row for a pair that is not an edge, a
// region out of range, rows out of order or repeated, a row at the default status, a
// factor of 0 or above the bound, a boolean that is neither, a count above the bytes
// left, bytes after the last row and a section that ends early.
//
// Arithmetic: the good rows are (0,1,1,10000), (0,2,1,20000), (1,2,0,15000); 0 to 0
// is no edge of regionsCfg; the bound is MaxEdgeFactorBP = 100 000, so 100 001 is
// one above it.
//
// Requirement: FR-REG-052, FR-CHK-012, FR-CHK-013
func TestRestore_RegionsSectionCorrupt(t *testing.T) {
	t.Parallel()
	raw := mustMarshal(t, mustCheckpoint(t, regionsWorld(t, regionsStatuses)))
	good := regionsBody([4]uint64{0, 1, 1, 10_000}, [4]uint64{0, 2, 1, 20_000}, [4]uint64{1, 2, 0, 15_000})
	tests := []struct {
		name string
		body []byte
	}{
		{"a pair that is not an edge", regionsBody([4]uint64{0, 0, 1, 10_000})},
		{"a region out of range", regionsBody([4]uint64{0, 9, 1, 10_000})},
		{"a pair of regions with no edge between them", regionsBody([4]uint64{2, 0, 1, 10_000})},
		{"rows out of order", regionsBody([4]uint64{1, 2, 0, 15_000}, [4]uint64{0, 1, 1, 10_000})},
		{"a row repeated", regionsBody([4]uint64{0, 1, 1, 10_000}, [4]uint64{0, 1, 1, 10_000})},
		{"a row at the default status", regionsBody([4]uint64{0, 1, 0, 10_000})},
		{"a factor of 0", regionsBody([4]uint64{0, 1, 1, 0})},
		{"a factor above the bound", regionsBody([4]uint64{0, 1, 0, economysim.MaxEdgeFactorBP + 1})},
		{"a boolean that is neither", regionsBody([4]uint64{0, 1, 2, 10_000})},
		{"a count above the bytes left", append([]byte{50}, good[1:]...)},
		{"bytes after the last row", append(slices.Clone(good), 0)},
		{"a section that ends early", good[:len(good)-1]},
		{"a count with no rows", []byte{1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := reframe(t, raw, func(s []section) []section {
				return editSection(s, 11, func([]byte) []byte { return tt.body })
			})
			_, err := economysim.RestoreWorld(regionsCfg(), c)
			if !errors.Is(err, economysim.ErrCheckpointCorrupt) || !strings.Contains(err.Error(), "section 11") {
				t.Errorf("RestoreWorld(%s) error = %v, want ErrCheckpointCorrupt naming section 11", tt.name, err)
			}
		})
	}
	// The good body itself restores, so each case above fails by its own fault.
	c := reframe(t, raw, func(s []section) []section { return editSection(s, 11, func([]byte) []byte { return good }) })
	if _, err := economysim.RestoreWorld(regionsCfg(), c); err != nil {
		t.Errorf("RestoreWorld(the good body) error = %v, want nil", err)
	}
}

// TestFork_Regions verifies the fork-at-N of pack three with its trade edges and two
// closed edges set between ticks: at each N of the matrix the original and two
// restores give the same events, snapshot, audit and checkpoint bytes for 30 ticks.
// The pack is three with the edges of the overlay trade-three.toml (spec-regions.md 12.4).
//
// Requirement: FR-REG-052, FR-CHK-050
func TestFork_Regions(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packLayers["three"]...)
	cfg.Edges = tradeEdges(t, "three")
	// far_south (0), north (1), south (2) sort by ID; close south>north both ways.
	closeLink := func(w *economysim.World) {
		g := w.Graph()
		a, _ := g.Lookup("south")
		b, _ := g.Lookup("north")
		changes := []economysim.EdgeStatus{{From: a, To: b, Closed: true, FactorBP: 10_000}, {From: b, To: a, FactorBP: 14_000}}
		if err := economysim.SetEdgeStatusesForTest(w, changes, economysim.CauseConsumer); err != nil {
			t.Fatalf("SetEdgeStatusesForTest() error = %v, want nil", err)
		}
	}
	for seed := uint64(1); seed <= uint64(invariantSeeds(2)); seed++ {
		for _, cell := range forkCells {
			t.Run(fmt.Sprintf("seed %d %s", seed, cell.name), func(t *testing.T) {
				t.Parallel()
				forkAtN(t, forkCase{Name: "three with edges", Cfg: cfg, Seed: seed, N: cell.n, M: forkM, Before: closeLink})
			})
		}
	}
}

// TestSnapshot_EdgesAndComponents verifies Snapshot.Components on the packs: 0 for
// the one-region pack one, so the header of its goldens does not move, 3 for three
// with no edge, 1 for three and five with the edges of the trade overlay, and the
// count and the rows after a closure: closing south and north in three splits it in
// two with the two statuses of the link, and a closure of centre and north in five
// leaves one component through east.
//
// Requirement: FR-OBS-091, FR-REG-027, FR-REG-052
func TestSnapshot_EdgesAndComponents(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cases := []struct {
		pack       string
		edges      bool
		components int64
		link       [2]string
		after      int64
	}{
		{"one", false, 0, [2]string{}, 0},
		{"three", false, 3, [2]string{}, 0},
		{"three", true, 1, [2]string{"south", "north"}, 2},
		{"five", true, 1, [2]string{"centre", "north"}, 1},
	}
	for _, tt := range cases {
		t.Run(fmt.Sprintf("%s edges %v", tt.pack, tt.edges), func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, packLayers[tt.pack]...)
			if tt.edges {
				cfg.Edges = tradeEdges(t, tt.pack)
			}
			w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
			if err != nil {
				t.Fatalf("NewWorld() error = %v, want nil", err)
			}
			if s := mustSnapshot(t, w); s.Components != tt.components || s.Edges != nil {
				t.Errorf("Snapshot() Components = %d, Edges = %v, want %d and none", s.Components, s.Edges, tt.components)
			}
			if tt.link == [2]string{} {
				return
			}
			g := w.Graph()
			a, _ := g.Lookup(tt.link[0])
			b, _ := g.Lookup(tt.link[1])
			changes := []economysim.EdgeStatus{{From: a, To: b, Closed: true, FactorBP: 10_000}, {From: b, To: a, Closed: true, FactorBP: 10_000}}
			if err := economysim.SetEdgeStatusesForTest(w, changes, economysim.CauseConsumer); err != nil {
				t.Fatalf("SetEdgeStatusesForTest() error = %v, want nil", err)
			}
			s := mustSnapshot(t, w)
			if s.Components != tt.after || len(s.Edges) != 2 || !s.Edges[0].Closed || !s.Edges[1].Closed || s.Edges[0].FactorBP != 10_000 {
				t.Errorf("Snapshot() after the closure: Components = %d, Edges = %+v, want %d and two closed rows", s.Components, s.Edges, tt.after)
			}
			if s.Edges[0].From > s.Edges[1].From || (s.Edges[0].From == s.Edges[1].From && s.Edges[0].To >= s.Edges[1].To) {
				t.Errorf("Snapshot() Edges = %+v, want ascending (From, To)", s.Edges)
			}
		})
	}
}
