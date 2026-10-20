package steady

import (
	"math/big"
	"slices"
)

// Producer is the completion pattern of one existing activity that outputs a
// commodity in a region (spec-viability.md 4.5). Every is set for an activity
// whose start is open on every tick of the year: its households are spread over
// the offsets, so one of them completes at each tick. Otherwise Completions
// holds the ticks of the greedy start sequence, not reduced modulo the year.
type Producer struct {
	Every       bool
	Completions []int64
}

// NewProducer reads the completion pattern of an activity of the given duration
// over a year of runs. The greedy sequence is read only for an activity with a
// closed tick (D122).
func NewProducer(duration int64, year []Run) Producer {
	var total int64
	open := true
	for _, r := range year {
		total += r.Length
		if r.Length > 0 && r.FactorBP == 0 {
			open = false
		}
	}
	if total > 0 && open {
		return Producer{Every: true}
	}
	return Producer{Completions: Starts(duration, year).Completions}
}

// Gap returns the supply gap of a commodity: the longest cyclic run of ticks of
// a year of ticks in which no producer completes (spec-viability.md 4.5,
// FR-VIA-008). A completion at or after the year end is taken modulo the year,
// since the steady state is periodic. It is 0 for a year with no ticks or when
// a producer completes every tick, and the whole year when nothing completes.
// The cost is the completions, never the ticks of the year.
func Gap(year int64, ps []Producer) int64 {
	if year < 1 {
		return 0
	}
	var ticks []int64
	for _, p := range ps {
		if p.Every {
			return 0
		}
		for _, c := range p.Completions {
			ticks = append(ticks, c%year)
		}
	}
	if len(ticks) == 0 {
		return year
	}
	slices.Sort(ticks)
	ticks = slices.Compact(ticks)
	longest := ticks[0] + year - ticks[len(ticks)-1] - 1 // the run across the year end
	for i := 1; i < len(ticks); i++ {
		longest = max(longest, ticks[i]-ticks[i-1]-1)
	}
	return longest
}

// ExceedsShelf reports that no stock can bridge the gap: the shelf life is
// positive (a non-perishable has 0) and below the gap. It is the condition of
// storage_gap for an upkeep good and of storage_gap_input for an input.
func ExceedsShelf(gap, shelfLife int64) bool {
	return shelfLife > 0 && shelfLife < gap
}

// Source is an activity that outputs the commodity whose lean reserve is sought:
// its duration, its flow x1 at lambda 1 (D120) in executions per tick, the
// non-discard quantity of the commodity per execution, and its year of runs.
type Source struct {
	Duration int64
	Flow     Frac
	Yield    int64
	Year     []Run
}

// span is a stretch of production ticks [Lo, Hi) at a constant rate per tick.
type span struct {
	lo, hi int64
	rate   *big.Rat
}

// spans returns the production of a source as constant-rate stretches inside
// [0, year): each open start tick has x1 * year / N_open executions starting
// (N_open counts every open tick, whether or not its job completes inside the
// year), each yielding Yield at the factor of its start tick, credited at the
// completion tick; a completion at or after the year end adds nothing (D123).
func (s Source) spans(year int64) []span {
	var open int64
	for _, r := range s.Year {
		if r.FactorBP > 0 {
			open += r.Length
		}
	}
	if open == 0 || s.Yield < 1 || s.Flow.Num.Sign() == 0 {
		return nil
	}
	base := new(big.Rat).SetFrac(new(big.Int).Mul(s.Flow.Num, new(big.Int).Mul(big.NewInt(year), big.NewInt(s.Yield))), new(big.Int).Mul(s.Flow.Den, big.NewInt(open)))
	var out []span
	var at int64
	for _, r := range s.Year {
		lo, hi := at+s.Duration-1, at+r.Length+s.Duration-1
		at += r.Length
		hi = min(hi, year)
		if r.FactorBP == 0 || lo >= hi {
			continue
		}
		rate := new(big.Rat).Mul(base, big.NewRat(r.FactorBP, 10000))
		out = append(out, span{lo: lo, hi: hi, rate: rate})
	}
	return out
}

// Reserve returns the lean reserve of a commodity in a region, exactly: the
// largest cumulative shortfall of consumption against production over the year
// starting at tick 0, max over t of max(0, sum over u <= t of cons - prod(u)),
// with cons the whole consumption per tick at lambda 1 (D123, spec-viability.md
// 8.4.1). Production is piecewise constant, so the shortfall is linear between
// the ends of the stretches and its maximum is at one of them; the cost is the
// stretches, not the ticks. The split among the groups, the rounding up per
// household and the cut to the shelf life are the table's.
func Reserve(year int64, cons Frac, srcs []Source) Frac {
	var all []span
	cuts := []int64{0, year}
	for _, s := range srcs {
		for _, sp := range s.spans(year) {
			all = append(all, sp)
			cuts = append(cuts, sp.lo, sp.hi)
		}
	}
	slices.Sort(cuts)
	cuts = slices.Compact(cuts)
	level, peak := new(big.Rat), new(big.Rat)
	for i := 0; i+1 < len(cuts); i++ {
		a, e := cuts[i], cuts[i+1]
		delta := frat(cons)
		for _, sp := range all {
			if sp.lo <= a && e <= sp.hi {
				delta.Sub(delta, sp.rate)
			}
		}
		level.Add(level, delta.Mul(delta, new(big.Rat).SetInt64(e-a)))
		if level.Cmp(peak) > 0 {
			peak.Set(level)
		}
	}
	return Frac{Num: peak.Num(), Den: peak.Denom()}
}
