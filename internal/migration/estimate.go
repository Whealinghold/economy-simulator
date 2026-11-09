package migration

import "github.com/Whealinghold/economy-sim/internal/checked"

// Line is a quantity of a good and its price at the destination; a price of 0 or
// below means the destination has no price for it (spec-migration.md 6.2).
type Line struct {
	Qty, Price int64
}

// Tool is a durable a recipe needs: the wear of one use against the durability
// of the item, and its price at the destination.
type Tool struct {
	Wear, Durability, Price int64
}

// Recipe is one recipe of the household's role as the destination prices it
// (spec-migration.md 6.3). Duration is the ticks of one execution, 1 or more.
// Yield is the table value S of the recipe at the destination in the season of
// the arrival tick, in basis points of a whole yield. Outputs are the
// non-discard outputs. A recipe with no Inputs is a source.
type Recipe struct {
	Duration int64
	Yield    int64
	Outputs  []Line
	Inputs   []Line
	Tools    []Tool
}

// Cost returns the sum of Qty * Price over the lines (the upkeep cost U of a
// household at a place, or the inputs of a recipe). ok is false on overflow.
func Cost(lines []Line) (total int64, ok bool) {
	for _, l := range lines {
		p, pok := checked.Mul(l.Qty, l.Price)
		if !pok {
			return 0, false
		}
		if total, pok = checked.Add(total, p); !pok {
			return 0, false
		}
	}
	return total, true
}

// Priced reports whether every line has a price.
func Priced(lines []Line) bool {
	for _, l := range lines {
		if l.Price < 1 {
			return false
		}
	}
	return true
}

// priced reports whether every input, tool and output of r has a price.
func (r *Recipe) priced() bool {
	if !Priced(r.Inputs) || !Priced(r.Outputs) {
		return false
	}
	for _, t := range r.Tools {
		if t.Price < 1 {
			return false
		}
	}
	return true
}

// net returns net_r = value_r - cost_r per execution (spec-migration.md 6.3):
// value_r is the sum over the outputs of floor(Qty * Yield * Price / 10 000),
// cost_r the inputs at their prices plus, for each tool, ceil(Wear * Price /
// Durability). ok is false on overflow or a tool of durability 0.
func (r *Recipe) net() (net int64, ok bool) {
	var value int64
	for _, o := range r.Outputs {
		qs, ok := checked.Mul(o.Qty, r.Yield)
		if !ok {
			return 0, false
		}
		v, ok := checked.MulDivFloor(qs, o.Price, wholeBP)
		if !ok {
			return 0, false
		}
		if value, ok = checked.Add(value, v); !ok {
			return 0, false
		}
	}
	cost, ok := Cost(r.Inputs)
	if !ok {
		return 0, false
	}
	for _, t := range r.Tools {
		w, ok := checked.MulDivCeil(t.Wear, t.Price, t.Durability)
		if !ok {
			return 0, false
		}
		if cost, ok = checked.Add(cost, w); !ok {
			return 0, false
		}
	}
	return checked.Sub(value, cost)
}

// Income returns the household's rumoured income per working tick at a
// destination (spec-migration.md 6.3): the greatest of net_r / Duration
// truncated toward zero over the eligible recipes, and 0 if there is none. A
// recipe is eligible when it is fully priced, its Yield is above 0 and either
// net_r is above 0 or it is a source (no Inputs). priced reports whether any
// recipe is fully priced, which is the test of stage 1: a household whose
// recipes are all priced but none eligible has an income of 0 and falls at a
// later stage. A negative income is possible, for a source recipe whose tool
// wear exceeds its value. ok is false on overflow or a Duration below 1.
func Income(recipes []Recipe) (income int64, priced, ok bool) {
	best, found := int64(0), false
	for i := range recipes {
		r := &recipes[i]
		if r.Duration < 1 {
			return 0, false, false
		}
		if !r.priced() {
			continue
		}
		priced = true
		if r.Yield < 1 {
			continue
		}
		net, nok := r.net()
		if !nok {
			return 0, false, false
		}
		if net <= 0 && len(r.Inputs) > 0 {
			continue
		}
		if per := net / r.Duration; !found || per > best {
			best, found = per, true
		}
	}
	return best, priced, true
}

// Vhat returns income * 10 000 / cost, truncated toward zero, the ratio a
// household would reach if it worked every tick (spec-migration.md 6.3), and
// the same shape for the home figure V_a = observed wage * 10 000 / U_h
// (6.3.2). ok is false when cost is not above 0 or the quotient overflows.
func Vhat(income, cost int64) (v int64, ok bool) {
	if cost < 1 {
		return 0, false
	}
	return checked.MulDiv(income, wholeBP, cost)
}
