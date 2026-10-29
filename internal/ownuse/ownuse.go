package ownuse

import (
	"math"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Good is one covered commodity of an agent: its whole stock, its effective
// upkeep for the whole agent (the weight times the upkeep of one household,
// spec-subsistence.md 5.2, U_c) and the published price of the commodity in the
// agent's region. Upkeep and Price are at least 1 for a valid good.
type Good struct {
	Stock, Upkeep, Price int64
}

// Params are the tunables of the mode: the entry threshold EnterTicks, the exit
// cover ExitTicks and the saturation of the secure ticks MaxOwnUseTicks
// (spec-subsistence.md 12, 13.1).
type Params struct {
	Enter, Exit, Max int64
}

// Result is the answer of Step for one agent: its three figures, its mode and
// counter after the step, and whether the step entered or left the mode.
type Result struct {
	Cover, Afford, Secure int64
	On                    bool
	Jobs                  int64
	Entered, Left         bool
}

// Cover returns the ticks of upkeep held in stock, the least over the goods of
// floor(Stock / Upkeep) (spec-subsistence.md 5.2). ok is false for no good, a
// negative stock or an upkeep below 1.
func Cover(goods []Good) (cover int64, ok bool) {
	if len(goods) == 0 {
		return 0, false
	}
	cover = math.MaxInt64
	for _, g := range goods {
		if g.Stock < 0 || g.Upkeep < 1 {
			return 0, false
		}
		if c := g.Stock / g.Upkeep; c < cover {
			cover = c
		}
	}
	return cover, true
}

// Cost returns the money a tick of the covered upkeep costs at the going
// price, the sum of Upkeep * Price over the goods (spec-subsistence.md 5.2),
// every product and the sum checked. ok is false on overflow, a price or an
// upkeep below 1, or no good.
func Cost(goods []Good) (cost int64, ok bool) {
	if len(goods) == 0 {
		return 0, false
	}
	for _, g := range goods {
		if g.Upkeep < 1 || g.Price < 1 {
			return 0, false
		}
		p, ok := checked.Mul(g.Upkeep, g.Price)
		if !ok {
			return 0, false
		}
		if cost, ok = checked.Add(cost, p); !ok {
			return 0, false
		}
	}
	return cost, true
}

// Afford returns the ticks of upkeep the money buys at the going price,
// floor(money / cost), and 0 for an agent that went unfed (hungry), whatever
// its money (spec-subsistence.md 5.2.1). ok is false for a negative money or a
// cost below 1 when the agent is not hungry.
func Afford(hungry bool, money, cost int64) (ticks int64, ok bool) {
	if hungry {
		return 0, true
	}
	if money < 0 || cost < 1 {
		return 0, false
	}
	return money / cost, true
}

// Secure returns the ticks of upkeep secured, cover plus afford, saturating at
// limit (spec-subsistence.md 5.2). A sum beyond int64 saturates too.
func Secure(cover, afford, limit int64) int64 {
	s, ok := checked.Add(cover, afford)
	if !ok || s > limit {
		return limit
	}
	return s
}

// Step evaluates one agent for the mode (spec-subsistence.md 5.2, 5.3): it
// computes the cover, afford and secure ticks of its goods, its money and its
// hunger, then enters the mode (counter 0) when the agent is out and its secure
// ticks are below Enter, and leaves it (counter 0) when the agent is in and its
// cover reaches Exit and in no other case. on and jobs are the mode and the
// counter before the step.
//
// ok is false when a figure cannot be formed (no good, a bad stock, upkeep or
// price, a negative money, an overflow). The mode and counter are then
// returned unchanged and the cover and the secure ticks saturated at Max, so a
// household that cannot be evaluated is never taken into the mode by the step.
func Step(p Params, on bool, jobs int64, goods []Good, hungry bool, money int64) (r Result, ok bool) {
	r = Result{Cover: p.Max, Afford: p.Max, Secure: p.Max, On: on, Jobs: jobs}
	cover, ok := Cover(goods)
	if !ok {
		return r, false
	}
	cost, ok := Cost(goods)
	if !ok {
		return r, false
	}
	afford, ok := Afford(hungry, money, cost)
	if !ok {
		return r, false
	}
	r.Cover, r.Afford, r.Secure = cover, afford, Secure(cover, afford, p.Max)
	switch {
	case !on && r.Secure < p.Enter:
		r.On, r.Jobs, r.Entered = true, 0, true
	case on && cover >= p.Exit:
		r.On, r.Jobs, r.Left = false, 0, true
	}
	return r, true
}
