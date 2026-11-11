package economysim

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// This file holds the lanes of the viability solver (spec-viability.md 9.4): the
// list a consumer gives to WithLanes, its checks, its resolution against the
// configuration and the components that make the programmes. The arithmetic of a
// lane is in internal/steady (lane.go); nothing here draws a random number, logs
// or reads a map in an order that reaches a result.

// laneErr is the refusal of the lane at index i.
func laneErr(i int, rule error, value, bound string) error {
	return &ValidationError{Path: []string{"solve_options", fmt.Sprintf("lanes[%d]", i)}, Rule: rule, Value: value, Bound: bound}
}

// checkLaneList applies the rules that need no configuration (FR-VIA-063). limit
// is MaxLanes, a parameter so a test can reach the bound with a short list.
func checkLaneList(lanes []Lane, limit int) error {
	if len(lanes) > limit {
		return &ValidationError{Path: []string{"solve_options", "lanes"}, Rule: ErrOutOfBounds, Value: i64(int64(len(lanes))), Bound: i64(int64(limit))}
	}
	seen := make(map[Lane]bool, len(lanes)) // membership only: no order reaches a result
	for i, ln := range lanes {
		switch {
		case ln.From == ln.To:
			return laneErr(i, ErrOutOfBounds, ln.From, "from differs from to")
		case ln.Labour < 0:
			return laneErr(i, ErrOutOfBounds, i64(ln.Labour), "0")
		case ln.LossBP < 0 || ln.LossBP > wholeBP-1:
			return laneErr(i, ErrOutOfBounds, i64(ln.LossBP), i64(wholeBP-1))
		case seen[ln]:
			return laneErr(i, ErrDuplicateID, ln.From+" "+ln.To+" "+ln.Commodity, "")
		}
		seen[ln] = true
	}
	return nil
}

// viaLane is a lane resolved against the configuration.
type viaLane struct {
	from, to  RegionID
	commodity CommodityID
	labour    int64
	lossBP    int64
}

// resolveLanes names the regions and commodities of the lanes by their index and
// returns the lanes in canonical order: by From, To, Commodity, Labour and LossBP
// (FR-VIA-061). A lane that names a region or commodity the configuration does
// not have is ErrUnknownRegion or ErrUnknownCommodity. cfg is canonical and valid.
func resolveLanes(cfg *Config, lanes []Lane) ([]viaLane, error) {
	var out []viaLane
	for i, ln := range lanes {
		from, ok1 := cfg.regionID(ln.From)
		to, ok2 := cfg.regionID(ln.To)
		c, ok3 := cfg.commodityID(ln.Commodity)
		switch {
		case !ok1:
			return nil, laneErr(i, ErrUnknownRegion, ln.From, "")
		case !ok2:
			return nil, laneErr(i, ErrUnknownRegion, ln.To, "")
		case !ok3:
			return nil, laneErr(i, ErrUnknownCommodity, ln.Commodity, "")
		}
		out = append(out, viaLane{from: from, to: to, commodity: c, labour: ln.Labour, lossBP: ln.LossBP})
	}
	slices.SortFunc(out, func(a, b viaLane) int {
		return cmp.Or(cmp.Compare(a.from, b.from), cmp.Compare(a.to, b.to), cmp.Compare(a.commodity, b.commodity),
			cmp.Compare(a.labour, b.labour), cmp.Compare(a.lossBP, b.lossBP))
	})
	return out, nil
}

// laneGroups returns the weakly connected components of the lane graph over the
// regions in have (ascending): each component's regions ascending, the components
// ordered by their first region. A region with no lane is a component of one, and
// a lane with an end outside have is ignored (FR-VIA-061).
func laneGroups(have []RegionID, lanes []viaLane) [][]RegionID {
	parent := make(map[RegionID]RegionID, len(have)) // a lookup only; no order reaches a result
	for _, r := range have {
		parent[r] = r
	}
	find := func(r RegionID) RegionID {
		for parent[r] != r {
			parent[r] = parent[parent[r]]
			r = parent[r]
		}
		return r
	}
	for _, ln := range lanes {
		if _, ok := parent[ln.from]; !ok {
			continue
		}
		if _, ok := parent[ln.to]; !ok {
			continue
		}
		a, b := find(ln.from), find(ln.to)
		parent[max(a, b)] = min(a, b) // the least region of a component is its root
	}
	var groups [][]RegionID
	at := map[RegionID]int{}
	for _, r := range have {
		root := find(r)
		if _, seen := at[root]; !seen {
			at[root] = len(groups)
			groups = append(groups, nil)
		}
		groups[at[root]] = append(groups[at[root]], r)
	}
	return groups
}

// lanesIn returns the lanes with both ends in regions as steady lanes, the region
// being its index in regions; nil when there are none.
func (m *viaModel) lanesIn(regions []RegionID) []steady.Lane {
	var out []steady.Lane
	idx := make(map[RegionID]int, len(regions))
	for i, r := range regions {
		idx[r] = i
	}
	for _, ln := range m.lanes {
		from, fok := idx[ln.from]
		to, tok := idx[ln.to]
		if fok && tok {
			out = append(out, steady.Lane{From: from, To: to, Commodity: int(ln.commodity), Labour: ln.labour, LossBP: ln.lossBP})
		}
	}
	return out
}

// price returns the dual of the goods row of commodity c in region k of the
// programme: the commodity's own row, or the row of the region in a lane form
// (spec-viability.md 9.4, FR-VIA-062).
func (p *viaProgramme) price(k, c int) steady.Frac {
	if len(p.md.Lanes) > 0 {
		return p.pr.Price[k*p.md.Commodities+c]
	}
	return p.pr.Price[c]
}

// laneWantsTerm is wantsTerm for a lane programme: the fixed term of each region
// in its own goods rows, the row of commodity c in region k at k*commodities+c.
func laneWantsTerm(c *Config, regions []RegionID) ([]steady.Ratio, error) {
	n := len(c.Commodities)
	var out []steady.Ratio
	for k, r := range regions {
		w, err := wantsTerm(c, []RegionID{r})
		if err != nil || w == nil {
			return nil, err // the module is off, or an overflow
		}
		if out == nil {
			out = make([]steady.Ratio, len(regions)*n)
			for i := range out {
				out[i] = steady.Ratio{Num: 0, Den: 1}
			}
		}
		copy(out[k*n:], w)
	}
	return out, nil
}
