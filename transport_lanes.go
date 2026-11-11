package economysim

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/transit"
)

// This file derives the lanes of the viability solver from the transport module
// (spec-transport.md 9.4, FR-TRN-041). It is a pure function of the
// configuration: it builds no world, draws nothing and logs nothing.

// routeLossBP is the expected loss of a lane in basis points. It is the seam of
// the risk module (spec-risk.md 8.3): 0 until that module exists (N-180).
func routeLossBP(*Config, RegionID, RegionID, *Mode) int64 { return 0 }

// haulRecipe is the cheapest recipe that makes a haul commodity: its ticks per
// job and the units one job makes.
type haulRecipe struct {
	duration, output int64
	ok               bool
}

// cheaper reports whether a job of duration d making q units beats best: a lower
// Duration / Output by cross-multiplication, ties to the lower recipe ID.
func (best haulRecipe) cheaper(d, q int64, id, bestID string) bool {
	if !best.ok {
		return true
	}
	l, lok := checked.Mul(d, best.output)
	r, rok := checked.Mul(best.duration, q)
	return lok && rok && (l < r || (l == r && id < bestID))
}

// cheapest returns the recipe with the lowest Duration / Output among those that
// make the commodity, ties by recipe ID.
func cheapest(c *Config, commodity string) haulRecipe {
	var best haulRecipe
	bestID := ""
	for i := range c.Recipes {
		rc := &c.Recipes[i]
		for _, o := range rc.Outputs {
			if o.Commodity == commodity && o.Quantity > 0 && best.cheaper(rc.Duration, int64(o.Quantity), rc.ID, bestID) {
				best, bestID = haulRecipe{rc.Duration, int64(o.Quantity), true}, rc.ID
			}
		}
	}
	return best
}

// laneKey orders the lanes by (From, To, Commodity, Labour).
func laneKey(a, b Lane) int {
	return cmp.Or(cmp.Compare(a.From, b.From), cmp.Compare(a.To, b.To), cmp.Compare(a.Commodity, b.Commodity), cmp.Compare(a.Labour, b.Labour))
}

// TransportLanes returns the lanes the transport module offers the viability
// solver (spec-transport.md 9.4): for each hired mode, ordered pair of regions
// with a route at the neutral transit factor, and commodity, one lane whose labour
// is the carriers' household-ticks per 10 000 units. The list is sorted by
// (From, To, Commodity, Labour), without duplicates, and at most MaxLanes long
// (ErrOutOfBounds otherwise). It is nil with the module off or no hired mode, and
// an error with no partial list when the region graph is invalid, a hired mode's
// haul commodity has no recipe that makes it (ErrUnproducible) or the arithmetic
// overflows (ErrOverflow). Self-carried modes give no lanes in version 1.
func (cfg Config) TransportLanes() ([]Lane, error) {
	if cfg.Modules.Transport == nil {
		return nil, nil
	}
	g, err := NewGraph(cfg)
	if err != nil {
		return nil, err
	}
	c := cfg.canonical()
	var out []Lane
	for i := range c.Modules.Transport.Modes {
		md := &c.Modules.Transport.Modes[i]
		if !md.hired() {
			continue
		}
		if out, err = appendModeLanes(out, &c, g, md); err != nil {
			return nil, err
		}
	}
	slices.SortFunc(out, laneKey)
	return slices.CompactFunc(out, func(a, b Lane) bool { return a == b }), nil
}

// appendModeLanes adds the lanes of one hired mode.
func appendModeLanes(out []Lane, c *Config, g Graph, md *Mode) ([]Lane, error) {
	hr := cheapest(c, md.Haul)
	if !hr.ok {
		return nil, fmt.Errorf("economysim: mode %q: no recipe makes the haul commodity %q: %w", md.ID, md.Haul, ErrUnproducible)
	}
	for from := range c.Regions {
		for to := range c.Regions {
			d, ok := routeDistance(g, md, RegionID(from), RegionID(to))
			if !ok {
				continue
			}
			loss := routeLossBP(c, RegionID(from), RegionID(to), md)
			for ci := range c.Commodities {
				labour, lok := transit.Labour(c.Commodities[ci].Mass, c.Commodities[ci].Bulk, md.UnitMass, md.UnitBulk, md.HaulSpan, d, hr.duration, hr.output)
				if !lok {
					return nil, fmt.Errorf("economysim: mode %q lane %s to %s of %q: %w", md.ID, c.Regions[from].ID, c.Regions[to].ID, c.Commodities[ci].ID, ErrOverflow)
				}
				out = append(out, Lane{From: c.Regions[from].ID, To: c.Regions[to].ID, Commodity: c.Commodities[ci].ID, Labour: labour, LossBP: loss})
			}
			if err := tooManyLanes(len(out), MaxLanes); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// tooManyLanes refuses a list of n lanes above the limit (counted before the final
// dedup, as a memory guard), a parameter so a test can
// reach the bound with a short list.
func tooManyLanes(n, limit int) error {
	if n <= limit {
		return nil
	}
	return &ValidationError{Path: []string{"modules", "transport", "lanes"}, Rule: ErrOutOfBounds, Value: i64(int64(n)), Bound: i64(int64(limit))}
}

// routeDistance returns the distance of a route of the mode: two different
// regions with a path and a travel time within MaxTravelTicks at the neutral
// transit factor (the test of World.travel).
func routeDistance(g Graph, md *Mode, from, to RegionID) (int64, bool) {
	d, ok := g.Distance(from, to)
	if from == to || !ok {
		return 0, false
	}
	k, ok := transit.Ticks(d, md.Speed, wholeBP)
	return d, ok && k <= MaxTravelTicks
}
