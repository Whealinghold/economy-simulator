package economysim

import (
	"math/rand/v2" // nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used -- only the *rand.Rand type of the injected, seeded simulation source, never a secret (architecture.md 7.3, 7.7)
	"slices"

	"github.com/Whealinghold/economy-sim/internal/policy"
)

// This file is the built-in policies that need no mailbox (spec-policy.md 8.2, 8.3):
// schedule and rule, built from an actor's row and registered by the world under their
// reserved names. They are not exported (N-481): a consumer never builds one. Each is a
// function of the view alone, takes no draw and keeps no state of its own beyond the
// scratch it returns its settings in; the primitives are in internal/policy.

// schedulePolicy is the fixed schedule of one actor (8.2): at every decision it returns
// the value of every lever that has one.
type schedulePolicy struct {
	steps []policy.Step
	cycle int64
	out   []LeverSetting
}

var _ Policy = (*schedulePolicy)(nil)

// newSchedulePolicy resolves the steps of row a to the index of their lever.
func newSchedulePolicy(a *PolicyActor) *schedulePolicy {
	p := &schedulePolicy{cycle: a.Cycle, out: make([]LeverSetting, 0, len(a.Levers))}
	for _, s := range a.Schedule {
		lever := slices.IndexFunc(a.Levers, func(l Lever) bool { return l.ID == s.Lever })
		p.steps = append(p.steps, policy.Step{Lever: lever, At: s.At, BP: s.BP, Closed: s.Closed})
	}
	return p
}

// Decide returns, for each lever in ascending LeverID, the value the schedule gives at
// the tick, and nothing for a lever it holds. Idempotent: the checks turn a value the
// lever already has into an unchanged entry (8.2.2).
func (p *schedulePolicy) Decide(v PolicyView, _ *rand.Rand) Settings {
	p.out = p.out[:0]
	for i := range v.Levers {
		if bp, closed, ok := policy.ScheduleValue(p.steps, i, p.cycle, v.Tick); ok {
			p.out = append(p.out, LeverSetting{Lever: LeverID(i), BP: bp, Closed: closed})
		}
	}
	return Settings{Levers: p.out}
}

// ruleRow is one reaction resolved to its lever and, for the price gauge, its commodity.
type ruleRow struct {
	lever     LeverID
	gauge     Gauge
	commodity CommodityID
	maxAge    int64
	reaction  policy.Reaction
}

// ruleMove is one lever a rule moved, for the Debug line "rule moved lever".
type ruleMove struct {
	lever    LeverID
	reading  int64
	from, to int64
}

// rulePolicy is the reactive rule of one actor (8.3). After a Decide, moves, inBand and
// noReading hold what that call did, which the world reads for the Debug lines and the
// counts of PolicyDecided; the next call resets them.
type rulePolicy struct {
	rows      []ruleRow
	out       []LeverSetting
	moves     []ruleMove
	inBand    int64
	noReading int64
	ratios    []policy.Ratio
	prices    []policy.PriceRow
}

var _ Policy = (*rulePolicy)(nil)

// newRulePolicy resolves the reactions of row a to the index of their lever and the ID of
// their commodity.
func newRulePolicy(c *Config, a *PolicyActor) *rulePolicy {
	p := &rulePolicy{out: make([]LeverSetting, 0, len(a.Rules)), moves: make([]ruleMove, 0, len(a.Rules))}
	for _, r := range a.Rules {
		row := ruleRow{
			lever: LeverID(slices.IndexFunc(a.Levers, func(l Lever) bool { return l.ID == r.Lever })), gauge: r.Gauge, maxAge: r.MaxAge,
			reaction: policy.Reaction{Target: r.Target, Deadband: r.Deadband, Span: r.Span, Move: r.Move, Raise: r.Raise},
		}
		if r.Gauge == GaugePrice {
			row.commodity = CommodityID(slices.IndexFunc(c.Commodities, func(x Commodity) bool { return x.ID == r.Commodity }))
		}
		p.rows = append(p.rows, row)
	}
	return p
}

// Decide takes, for each reaction in ascending LeverID, the reading of its gauge and asks
// the rule's step for the value; a reaction at rest or with no reading returns no entry
// and is counted. The entry carries the lever's Closed as the view has it, so the rule
// neither closes nor opens an edge (8.3.1).
func (p *rulePolicy) Decide(v PolicyView, _ *rand.Rand) Settings {
	p.out, p.moves, p.inBand, p.noReading = p.out[:0], p.moves[:0], 0, 0
	for i := range p.rows {
		row := &p.rows[i]
		reading, ok := p.read(row, &v)
		lv := &v.Levers[row.lever]
		next, hold := policy.RuleStep(row.reaction, reading, ok, lv.BP)
		switch hold {
		case policy.InBand:
			p.inBand++
		case policy.NoReading:
			p.noReading++
		default:
			p.out = append(p.out, LeverSetting{Lever: row.lever, BP: next, Closed: lv.Closed})
			p.moves = append(p.moves, ruleMove{lever: row.lever, reading: reading, from: lv.BP, to: next})
		}
	}
	return Settings{Levers: p.out}
}

// read is the reading of the gauge of row over the view and whether it is defined
// (8.3.2).
func (p *rulePolicy) read(row *ruleRow, v *PolicyView) (int64, bool) {
	switch row.gauge {
	case GaugeTreasury:
		return policy.TreasuryReading(int64(v.Treasury), v.HasTreasury)
	case GaugeUnfedShare:
		return p.unfedShare(v)
	case GaugeMeanRatio:
		return p.meanRatio(v)
	}
	return p.price(row, v)
}

// unfedShare is the share of households that went unfed over the covered regions.
func (p *rulePolicy) unfedShare(v *PolicyView) (int64, bool) {
	var unfed, households int64
	for i := range v.Regions {
		households += v.Regions[i].Region.Households
		for _, r := range v.Regions[i].Roles {
			unfed += r.Unfed
		}
	}
	return policy.UnfedShare(unfed, households)
}

// meanRatio is the household-weighted mean welfare ratio over the role rows that have one.
func (p *rulePolicy) meanRatio(v *PolicyView) (int64, bool) {
	p.ratios = p.ratios[:0]
	for i := range v.Regions {
		for _, r := range v.Regions[i].Roles {
			if r.HasMeanRatio {
				p.ratios = append(p.ratios, policy.Ratio{BP: r.MeanRatioBP, Households: r.RatioHouseholds})
			}
		}
	}
	return policy.MeanRatio(p.ratios)
}

// price is the household-weighted price of the reaction's commodity over the covered
// regions that have a market row of it.
func (p *rulePolicy) price(row *ruleRow, v *PolicyView) (int64, bool) {
	p.prices = p.prices[:0]
	for i := range v.Regions {
		rs := &v.Regions[i]
		if int(row.commodity) < len(rs.Markets) {
			m := &rs.Markets[row.commodity]
			p.prices = append(p.prices, policy.PriceRow{Households: rs.Region.Households, Price: int64(m.State.Price), Age: m.State.Age})
		}
	}
	return policy.Price(p.prices, row.maxAge)
}
