package steady

import (
	"math/big"
	"sort"
)

// Item is a quantity of a commodity, named by index. Discard marks an output
// that counts for nothing (spec-viability.md 3.1).
type Item struct {
	Commodity int
	Qty       int64
	Discard   bool
}

// Tool is a required tool: each use wears Wear of a Durability, a flow of
// Wear/Durability units per execution, not a stock.
type Tool struct {
	Commodity  int
	Wear       int64
	Durability int64
}

// Draw is the quantity an execution takes from a resource of its region
// (spec-resources.md 3.1.3): Resource is the caller's index of the resource, the
// one an Endow of the Model names.
type Draw struct {
	Resource int
	Qty      int64
}

// Activity is a recipe in a region, one column of the reference programme. The
// caller has already counted the starts (Starts) of one household in the year:
// Starts is J and FactorSum the sum of the factors at them, in basis points.
// An activity with Draws also has its mean factor multiplied by AbundanceBP over
// 10 000, the abundance at the plan share, the least over its draws
// (spec-resources.md 11.2); AbundanceBP is read only with Draws, since 0 is a
// legal abundance.
type Activity struct {
	Region      int
	Duration    int64
	Starts      int64
	FactorSum   int64
	Inputs      []Item
	Outputs     []Item
	Tools       []Tool
	Draws       []Draw
	AbundanceBP int64
}

// meanFactor returns the mean yield factor of the activity as num/den, where the
// factor is over 10 000: FactorSum / (10 000 * Starts), times AbundanceBP over
// 10 000 for an activity with draws. Starts below 1 panics (a programmer
// invariant).
func (a Activity) meanFactor() (num, den *big.Int) {
	if a.Starts < 1 {
		panic("steady: invariant violated: activity with no start")
	}
	num = big.NewInt(a.FactorSum)
	den = new(big.Int).Mul(big.NewInt(10000), big.NewInt(a.Starts))
	if len(a.Draws) > 0 {
		num.Mul(num, big.NewInt(a.AbundanceBP))
		den.Mul(den, big.NewInt(10000))
	}
	return num, den
}

// NetEntry is the net units of one commodity per execution of an activity.
type NetEntry struct {
	Commodity int
	Val       *big.Rat
}

// Net returns the net vector of one execution, in commodity order and without
// zero entries: E * q / 10 000 for each non-discard output (E also times the
// abundance for an activity with draws), minus the input quantities, minus
// Wear/Durability per tool (FR-VIA-001). The products are formed in big, since
// FactorSum times a quantity can exceed int64. Starts below 1 or a Durability
// below 1 is a violated programmer invariant and panics.
func (a Activity) Net() []NetEntry {
	fnum, fden := a.meanFactor()
	acc := map[int]*big.Rat{}
	add := func(c int, v *big.Rat) {
		if acc[c] == nil {
			acc[c] = new(big.Rat)
		}
		acc[c].Add(acc[c], v)
	}
	for _, o := range a.Outputs {
		if o.Discard {
			continue
		}
		num := new(big.Int).Mul(fnum, big.NewInt(o.Qty))
		add(o.Commodity, new(big.Rat).SetFrac(num, fden))
	}
	for _, in := range a.Inputs {
		add(in.Commodity, new(big.Rat).SetInt64(-in.Qty))
	}
	for _, tl := range a.Tools {
		if tl.Durability < 1 {
			panic("steady: invariant violated: tool durability below 1")
		}
		add(tl.Commodity, big.NewRat(-tl.Wear, tl.Durability))
	}
	keys := make([]int, 0, len(acc))
	for c := range acc {
		keys = append(keys, c)
	}
	sort.Ints(keys)
	var out []NetEntry
	for _, c := range keys {
		if acc[c].Sign() != 0 {
			out = append(out, NetEntry{Commodity: c, Val: acc[c]})
		}
	}
	return out
}
