package verdict

import (
	"errors"
	"strconv"
)

// Relation says how a band stands to the regime threshold it backs (STY
// 6.5.2): a regime is a gross failure and a failed fact a smaller one, so a
// pack cannot be in regime while its facts pass.
type Relation uint8

// The relations of OrderingPairs.
const (
	// AtMost: the band value is at most the threshold (an upper bound).
	AtMost Relation = iota
	// AtLeast: the band value is at least the threshold (a floor).
	AtLeast
	// AtLeastComplement: the band value is at least 10 000 minus the
	// threshold (x4.households_bp against r3.fall_bp).
	AtLeastComplement
	// PinnedRunShare: a pinned run of threshold ticks is at least the band's
	// share of a year, by integer division (the implication pair).
	PinnedRunShare
)

// Pair is one pair of STY 6.5.2: a band key and the threshold it backs.
type Pair struct {
	Band, Threshold string
	Rel             Relation
}

// ticksPerYearRef is the year of the reference packs, for the implication pair
// (60 * 10 000 / 365 = 1 643, STY 6.5.2).
const ticksPerYearRef = 365

// OrderingPairs returns the pairs of STY 6.5.2 in the order of the spec. The
// pair of s14.recover_ticks against r3.horizon_factor needs the speed limit
// t_min and is checked apart, by CheckOrdering's tMin argument.
func OrderingPairs() []Pair {
	return []Pair{
		{"s07.freeze_ticks", kR5Freeze, AtMost},
		{"s03.pinned_ticks", kR5PinnedTick, AtMost},
		{"s04.pinned_ticks", kR5PinnedTick, AtMost},
		{"s05.range_factor", kR6Factor, AtMost},
		{"s12.range_factor", kR6Factor, AtMost},
		{"x4.range_factor", kR6Factor, AtMost},
		{"s05.range_growth_bp", kR6Growth, AtMost},
		{"s06.swing_bp", kR6Swing, AtMost},
		{"s01.offset_bp", kR8Offset, AtMost},
		{"h02.offset_bp", kR8Offset, AtMost},
		{"s01.unmet_bp", kR8Unmet, AtMost},
		{"p07.drift_bp", kR1Rise, AtMost},
		{"p07.drift_bp", kR9Fall, AtMost},
		{"s10.dispersion_min_bp", kR7Disp, AtLeast},
		{"m05.pool_share_bp", kR2Share, AtMost},
		{"x4.households_bp", kR3Fall, AtLeastComplement},
		{"s16.pinned_share_bp", kR5PinnedTick, PinnedRunShare},
		{"x4.pinned_share_bp", kR5PinnedTick, PinnedRunShare},
	}
}

// ErrOrdering is the error of CheckOrdering: a band is looser than the
// threshold it backs.
var ErrOrdering = errors.New("verdict: a band is looser than the regime threshold it backs")

// OrderingError names the pair that fails.
type OrderingError struct {
	Pair            Pair
	Band, Threshold int64
}

// Error names the keys and the values.
func (e *OrderingError) Error() string {
	return ErrOrdering.Error() + ": " + e.Pair.Band + " " + strconv.FormatInt(e.Band, 10) +
		" against " + e.Pair.Threshold + " " + strconv.FormatInt(e.Threshold, 10)
}

// Unwrap lets errors.Is find ErrOrdering.
func (e *OrderingError) Unwrap() error { return ErrOrdering }

// CheckOrdering checks the pairs of STY 6.5.2 over the values that exist (a
// pair with a missing side is skipped: an unset key asserts nothing) and, when
// tMin is positive and both are present, that s14.recover_ticks, which is
// 4 * t_min, is at most r3.horizon_factor / 10 000 times tMin. It returns the
// first violation in the order of OrderingPairs (FR-STY-072).
func CheckOrdering(values map[string]int64, tMin int64) error {
	for _, p := range OrderingPairs() {
		b, okB := values[p.Band]
		th, okT := values[p.Threshold]
		if !okB || !okT || holds(p.Rel, b, th) {
			continue
		}
		return &OrderingError{p, b, th}
	}
	rt, okR := values["s14.recover_ticks"]
	f, okF := values[kR3Factor]
	if tMin > 0 && okR && okF && !mulLE(rt, 10_000, f, tMin) {
		return &OrderingError{Pair{"s14.recover_ticks", kR3Factor, AtMost}, rt, f}
	}
	return nil
}

// holds reports whether the band value b stands in relation rel to the
// threshold th.
func holds(rel Relation, b, th int64) bool {
	switch rel {
	case AtLeast:
		return b >= th
	case AtLeastComplement:
		return b >= 10_000-min(max(th, -1<<40), 1<<40) // clamped: no overflow on a forged th
	case PinnedRunShare:
		// th ticks of a year of ticksPerYearRef, in basis points.
		return th >= 0 && b <= satMul(th, 10_000)/ticksPerYearRef
	}
	return b <= th
}

// mulLE reports whether a*b <= c*d, by the 128-bit compare of arith.go.
func mulLE(a, b, c, d int64) bool { return cmpMul(a, b, c, d) <= 0 }

// satMul multiplies two non-negative values, saturating at the largest int64.
func satMul(a, b int64) int64 {
	if a != 0 && b > maxInt64/a {
		return maxInt64
	}
	return a * b
}

// Step is the step of STY 10.3 in which a key is set. Step 1 (the healthy
// baseline) and step 0 set no key.
type Step struct {
	Key  string
	Step int
}

// stepKeys lists the band keys by the step of 10.3 that sets them, with the
// facts of 3.6 and 5. Keys of the regime file are all step 2; a key that no
// step names (the keys of REF's scenarios that STY does not set) has no step.
func stepKeys() []Step {
	steps := []Step{}
	add := func(step int, keys ...string) {
		for _, k := range keys {
			steps = append(steps, Step{k, step})
		}
	}
	add(2, ThresholdNames()...)
	add(3,
		"s03.pinned_ticks", "s04.pinned_ticks", "s07.freeze_ticks", "s07.spoiled_bp", "s16.pinned_share_bp", "x4.pinned_share_bp",
		"s05.range_factor", "s05.range_growth_bp", "s12.range_factor", "x4.range_factor",
		"s01.offset_bp", "s01.unmet_bp", "p07.drift_bp", "x4.trade_share_bp", "d01.unfed_bp",
		"s06.share_band_bp", "s06.swing_bp", "s19.ratio_band_bp", "m05.pool_share_bp", "s19.stock_bound_bp", "s20.ticks",
		"a1.price_band_bp", "h02.offset_bp", "h02.money_bp", "s17.spike_factor", "s17.control_gap_bp")
	add(4, "s01.band_bp", "s01.recover_ticks", "s08.band_bp", "s08.settle_ticks",
		"s14.recover_band_bp", "s14.recover_ticks", "s14.fall_bp", "s04.elasticity_max_bp")
	add(5, "s10.tolerance_bp", "s10.dispersion_min_bp")
	add(6, "x4.households_bp", "w01.mean_low_bp", "w01.mean_high_bp", "s06.free_tool_tol_bp")
	add(7, "x1.margin_bp", "x1.worst_bp", "x1.warmup", "s28.band_bp", "s28.reopen_ticks", "x2.share_band_bp")
	add(8, "t03.quote_band_bp", "t03.pull_bp")
	add(9, "w02.n_band_bp", "w02.ratio_band_bp", "w06.fixed_point_bp", "w06.money_cap_bp")
	return steps
}

// StepOrder returns the keys of the steps of STY 10.3 in step order.
func StepOrder() []Step { return stepKeys() }

// ErrStepOrder: a key of a later step has a value while a key of an earlier
// step is unset (FR-STY-091).
var ErrStepOrder = errors.New("verdict: a key of a later step is set while a key of an earlier step is unset")

// CheckSteps checks the order of STY 10.3. declared maps every key that some
// band file declares to whether it has a value; a key no file declares is not
// unset, it does not exist. It returns an error naming the first key of a
// later step with a value and the earlier key that is unset.
func CheckSteps(declared map[string]bool) error {
	steps := stepKeys()
	for _, later := range steps {
		if !declared[later.Key] {
			continue
		}
		for _, earlier := range steps {
			set, there := declared[earlier.Key]
			if there && !set && earlier.Step < later.Step {
				return errors.Join(ErrStepOrder, errors.New(later.Key+" (step "+strconv.Itoa(later.Step)+") is set, "+earlier.Key+" (step "+strconv.Itoa(earlier.Step)+") is not"))
			}
		}
	}
	return nil
}
