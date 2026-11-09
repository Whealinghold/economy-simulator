package indicator_test

import (
	"errors"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// TestPrices_ConfigMismatch verifies that a price table carries the hash of the
// world it came from and that a table of another configuration is
// ErrConfigMismatch, while a table of the same one is compatible
// (spec-indicators.md 4.3.5).
//
// Requirement: FR-IND-028
func TestPrices_ConfigMismatch(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	w := newOne(t)
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	p, err := indicator.PricesOfSnapshot(s)
	if err != nil {
		t.Fatalf("PricesOfSnapshot() error = %v", err)
	}
	if err := p.Compatible(w.ConfigHash()); err != nil {
		t.Errorf("Compatible(the world's hash) = %v, want nil", err)
	}
	other := w.ConfigHash()
	other[0] ^= 0xff
	if err := p.Compatible(other); !errors.Is(err, indicator.ErrConfigMismatch) {
		t.Errorf("Compatible(another hash) = %v, want ErrConfigMismatch", err)
	}
}

// TestPricesOfSnapshot_IsTheMarketPrice verifies that the table of a snapshot
// holds, for every market row, the published price of the last step, that its
// tick and hash are the snapshot's, and that it still does after the world has
// stepped and the prices moved.
//
// Requirement: FR-IND-028
func TestPricesOfSnapshot_IsTheMarketPrice(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	w := newOne(t)
	for range 6 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v", err)
		}
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	p, err := indicator.PricesOfSnapshot(s)
	if err != nil {
		t.Fatalf("PricesOfSnapshot() error = %v", err)
	}
	if p.ConfigHash != s.ConfigHash || p.From != s.Tick || p.To != s.Tick || p.Regions != len(s.Regions) || p.Commodities != len(s.Names.Commodities) {
		t.Errorf("table = hash %x from %d to %d shape %dx%d, want the snapshot's %x, tick %d, %dx%d", p.ConfigHash[:4], p.From, p.To, p.Regions, p.Commodities, s.ConfigHash[:4], s.Tick, len(s.Regions), len(s.Names.Commodities))
	}
	if len(s.Markets) == 0 {
		t.Fatal("the snapshot has no market row")
	}
	for _, m := range s.Markets {
		if got := p.At(m.State.Region, m.State.Commodity); got != m.State.Price || got < 1 {
			t.Errorf("At(%d, %d) = %d, want the published price %d (at least 1)", m.State.Region, m.State.Commodity, got, m.State.Price)
		}
	}
}

// TestPricesOfSnapshot_RejectsAnOutOfRangeRow verifies that a market row naming
// a region or commodity outside the names table is ErrRange and not an index
// panic.
//
// Requirement: FR-IND-028
func TestPricesOfSnapshot_RejectsAnOutOfRangeRow(t *testing.T) {
	t.Parallel()
	base := economysim.Snapshot{Regions: make([]economysim.RegionRow, 1), Names: economysim.Names{Commodities: []string{"a"}}}
	tests := []struct {
		name string
		row  economysim.MarketState
		ok   bool
	}{
		{"in range", economysim.MarketState{Region: 0, Commodity: 0, Price: 7}, true},
		{"region past the table", economysim.MarketState{Region: 1}, false},
		{"negative region", economysim.MarketState{Region: -1}, false},
		{"commodity past the table", economysim.MarketState{Commodity: 1}, false},
		{"negative commodity", economysim.MarketState{Commodity: -1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := base
			s.Markets = []economysim.MarketRow{{State: tt.row}}
			p, err := indicator.PricesOfSnapshot(s)
			if tt.ok != (err == nil) || (!tt.ok && !errors.Is(err, indicator.ErrRange)) {
				t.Errorf("PricesOfSnapshot(%+v) error = %v, want ok %v", tt.row, err, tt.ok)
			}
			if tt.ok && p.At(0, 0) != 7 {
				t.Errorf("At(0, 0) = %d, want 7", p.At(0, 0))
			}
		})
	}
}

// TestPricesOfReference_RegionOverride verifies the effective reference price of
// spec-indicators.md 4.3.4: a commodity's ReferencePrice, or the region's
// override where it has one, in the configuration's own order of regions and
// commodities, whatever order the override rows are written in.
//
// Requirement: FR-IND-028
func TestPricesOfReference_RegionOverride(t *testing.T) {
	t.Parallel()
	cfg := economysim.Config{
		Regions:     []economysim.Region{{ID: "north"}, {ID: "south"}},
		Commodities: []economysim.Commodity{{ID: "a", ReferencePrice: 10}, {ID: "b", ReferencePrice: 20}},
		RegionPrices: []economysim.RegionPrice{
			{Region: "south", Commodity: "b", Price: 35},
			{Region: "north", Commodity: "a", Price: 12},
		},
	}
	p := indicator.PricesOfReference(cfg)
	tests := []struct {
		r    economysim.RegionID
		c    economysim.CommodityID
		want economysim.Price
	}{
		{0, 0, 12}, {0, 1, 20}, {1, 0, 10}, {1, 1, 35},
		{2, 0, 0}, {0, 2, 0}, {-1, 0, 0}, {0, -1, 0}, // outside the table
	}
	for _, tt := range tests {
		if got := p.At(tt.r, tt.c); got != tt.want {
			t.Errorf("At(%d, %d) = %d, want %d", tt.r, tt.c, got, tt.want)
		}
	}
	if p.ConfigHash != economysim.HashConfig(cfg) || p.From != 0 || p.To != 0 {
		t.Errorf("hash %x from %d to %d, want HashConfig(cfg) and a table of no window", p.ConfigHash[:4], p.From, p.To)
	}
}

// TestPricesOfReference_ThePackAgreesWithTheWorld verifies that the table of a
// real configuration equals World.ReferencePrice for every region and commodity
// and carries the world's hash, so a table of the reference prices is compatible
// with the snapshots of that world.
//
// Requirement: FR-IND-028
func TestPricesOfReference_ThePackAgreesWithTheWorld(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	w := newOne(t)
	p := indicator.PricesOfReference(w.Config())
	if err := p.Compatible(w.ConfigHash()); err != nil {
		t.Errorf("Compatible(the world's hash) = %v, want nil", err)
	}
	for r := range p.Regions {
		for c := range p.Commodities {
			if got, want := p.At(economysim.RegionID(r), economysim.CommodityID(c)), w.ReferencePrice(economysim.RegionID(r), economysim.CommodityID(c)); got != want {
				t.Errorf("At(%d, %d) = %d, want World.ReferencePrice %d", r, c, got, want)
			}
		}
	}
}

// TestNewPrices_ChecksTheShape verifies the constructor of a consumer's own
// table: the shape must match, a negative price is ErrRange, zero is allowed
// (the value of a quantity at it is 0), and the cells are copied.
//
// Requirement: FR-IND-028
func TestNewPrices_ChecksTheShape(t *testing.T) {
	t.Parallel()
	var hash [32]byte
	tests := []struct {
		name  string
		r, c  int
		cells []economysim.Price
		from  int64
		to    int64
		ok    bool
	}{
		{"two by two", 2, 2, []economysim.Price{1, 2, 3, 4}, 0, 10, true},
		{"a zero price", 1, 2, []economysim.Price{0, 4}, 5, 5, true},
		{"empty", 0, 0, nil, 0, 0, true},
		{"short", 2, 2, []economysim.Price{1, 2, 3}, 0, 0, false},
		{"long", 1, 1, []economysim.Price{1, 2}, 0, 0, false},
		{"negative price", 1, 2, []economysim.Price{1, -2}, 0, 0, false},
		{"negative count", -1, 2, nil, 0, 0, false},
		{"count product wraps int", 1 << 32, 1 << 32, nil, 0, 0, false},
		{"window reversed", 1, 1, []economysim.Price{1}, 10, 5, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, err := indicator.NewPrices(hash, tt.from, tt.to, tt.r, tt.c, tt.cells)
			if tt.ok != (err == nil) || (!tt.ok && !errors.Is(err, indicator.ErrRange)) {
				t.Fatalf("NewPrices(%dx%d, %v) error = %v, want ok %v", tt.r, tt.c, tt.cells, err, tt.ok)
			}
			if tt.ok && tt.r > 0 && tt.c > 0 {
				tt.cells[0] = 99
				if p.At(0, 0) == 99 {
					t.Errorf("At(0, 0) followed the caller's slice, want a copy")
				}
			}
		})
	}
	var zero indicator.Prices
	if zero.At(0, 0) != 0 {
		t.Errorf("zero Prices At(0, 0) = %d, want 0", zero.At(0, 0))
	}
	if broken := (indicator.Prices{Regions: 2, Commodities: 2, Cells: []economysim.Price{1}}); broken.At(1, 1) != 0 {
		t.Errorf("a table whose cells do not match its shape: At(1, 1) = %d, want 0 and no panic", broken.At(1, 1))
	}
}
