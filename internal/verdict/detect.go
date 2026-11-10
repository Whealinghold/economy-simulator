package verdict

// Series is a window series with the flag IND sets on a clamped value
// (IND 5.3.7): Clamped[i] reports that Points[i] is clamped. Clamped may be
// shorter than Points, or nil, when nothing is clamped.
type Series struct {
	Points  []Point
	Clamped []bool
}

func (s Series) clampedAt(i int) bool { return i < len(s.Clamped) && s.Clamped[i] }

func (s Series) anyClamped() bool {
	for _, c := range s.Clamped {
		if c {
			return true
		}
	}
	return false
}

// Input is everything a detector reads: plain series copied by the caller from
// the results of the indicator presets, with no world, snapshot or event
// (D581). A detector reads only the fields its row of STY 6.2 names; the zero
// value of a field is "absent". Window series carry one point per whole year
// window counted from tick 0 (or per season or year window the caller passes,
// R6); one-tick series carry one point per tick.
type Input struct {
	// Links are the Fisher chain links in basis points, one point per window
	// after the first (10 000 is no change): the whole economy first, then
	// each class (R1); R9 reads the first. A chain of k links covers k+1 windows.
	Links []Series
	// M is Sectors.M at each close and MInit its start (R1).
	M     []Point
	MInit int64
	// Stale is the worst stale share of the indices, in basis points (R1, R9).
	Stale int64
	// Volume is Turnover.Volume, in units, per window (R2, R4, R9).
	Volume []Point
	// MoneyOut is (M_p + M_c + M_t) / M in basis points at each close, Unsold
	// the unsold share of each commodity with consumption, and Top the
	// Inequality.Top(1 000) share of household money (R2).
	MoneyOut []Point
	Unsold   [][]Point
	Top      []Point
	// Households holds one count series per realm or region with a start (R3,
	// R4); Roles one per role, each of one point per tick (R3).
	Households [][]Point
	Roles      [][]Point
	// Rate is the speed of STY-S02 in basis points a year and TicksPerYear the
	// calendar, from which R3 takes t_min (D585).
	Rate, TicksPerYear int64
	// Pinned is Pinned.Share in basis points per tick of each market (R5a);
	// Frozen has a non-zero value on a tick of a market with unmatched units
	// on both sides and no volume (R5b); Outcomes is the count of outcomes the
	// shares were read from (R5a, R7).
	Pinned   [][]Point
	Frozen   [][]Point
	Outcomes int64
	// Ranges are the PriceRange ratios in basis points per window of each
	// market and Swings the RoleShares swing in basis points per window of each
	// role (R6); Active is the share of active ticks in basis points (R6, R8).
	Ranges []Series
	Swings [][]Point
	Active int64
	// Dispersion is the Dispersion per window of each market, and Completions
	// the completions of each recipe per tick over the first ticks of the run
	// (R7).
	Dispersion  [][]Point
	Completions [][]Point
	// Offsets are the Offset in basis points per window of each market, Unmet
	// the Unfed share of household-ticks per window, HasReference reports a
	// steady-state price (R8).
	Offsets      []Series
	Unmet        []Point
	HasReference bool
}

// Detect runs one detector (STY 6.2) over the input with the thresholds. It is
// a pure function: the same input and thresholds give the same verdict, and
// its time and allocations depend on the length of the series and not on the
// number of agents of the run (NFR-STY-001, FR-STY-080). A detector with an
// unset key returns Unknown "threshold unset". Detect panics only for a
// Detector that is not one of Detectors, a programmer error.
func Detect(d Detector, in Input, t Thresholds) Verdict {
	v, ok := detectFirst(d, in, t)
	if !ok {
		v = detectSecond(d, in, t)
	}
	v.Detector = d
	return v
}

// detectFirst dispatches R1 to R5b; ok is false for any other detector.
func detectFirst(d Detector, in Input, t Thresholds) (Verdict, bool) {
	switch d {
	case R1:
		return detectR1(in, t), true
	case R2:
		return detectR2(in, t), true
	case R3:
		return detectR3(in, t), true
	case R4:
		return detectR4(in, t), true
	case R5a:
		return detectR5a(in, t), true
	case R5b:
		return detectR5b(in, t), true
	}
	return Verdict{}, false
}

// detectSecond dispatches R6 to R9 and panics on anything else.
func detectSecond(d Detector, in Input, t Thresholds) Verdict {
	switch d {
	case R6:
		return detectR6(in, t)
	case R7:
		return detectR7(in, t)
	case R8:
		return detectR8(in, t)
	case R9:
		return detectR9(in, t)
	}
	panic("verdict: programmer invariant violated: Detect called with a detector that is not in Detectors")
}

func unknownV(reason string) Verdict { return Verdict{Class: Unknown, Reason: reason} }

func detectedV(onset, value, threshold int64) Verdict {
	return Verdict{Class: Detected, Onset: onset, Value: value, Threshold: threshold}
}

// need returns the values of the named keys, and false if any is unset.
func (t Thresholds) need(names ...string) ([]int64, bool) {
	out := make([]int64, len(names))
	for i, n := range names {
		v, ok := t.Get(n)
		if !ok {
			return nil, false
		}
		out[i] = v
	}
	return out, true
}

func satAdd(a, b int64) int64 {
	switch {
	case b > 0 && a > maxInt64-b:
		return maxInt64
	case b < 0 && a < minInt64-b:
		return minInt64
	}
	return a + b
}

func satSub(a, b int64) int64 {
	if b == minInt64 {
		if a < 0 {
			return a - b // no overflow: a is negative and b is the minimum
		}
		return maxInt64
	}
	return satAdd(a, -b)
}

// scanRuns walks n indexes and returns the longest run of consecutive indexes
// at which cond holds and the first index at which a run reached need
// (-1 if none did). A need below 1 is read as 1.
func scanRuns(n int, need int64, cond func(i int) bool) (longest int64, reach int) {
	need = max(need, 1)
	reach = -1
	var run int64
	for i := 0; i < n; i++ {
		if !cond(i) {
			run = 0
			continue
		}
		run++
		longest = max(longest, run)
		if run >= need && reach < 0 {
			reach = i
		}
	}
	return longest, reach
}

// ---- R1 ----

// detectR1: the Fisher chain link of the whole economy or of a class is at
// least r1.rise_bp above 10 000 in each of r1.windows consecutive windows, or
// Sectors.M is at least r1.m_factor_bp of its start at two successive closes,
// or a link is clamped (STY 6.2, FR-STY-061). Stale prices turn a none into
// unknown, never a detected into unknown.
func detectR1(in Input, t Thresholds) Verdict {
	k, ok := t.need(kR1Rise, kR1Windows, kR1MFactor, kMinWindows, kStaleMax)
	if !ok {
		return unknownV(ReasonThresholdUnset)
	}
	rise, wins, mFactor, minW, staleMax := k[0], k[1], k[2], k[3], k[4]
	short := len(in.Links) == 0
	for _, c := range in.Links {
		short = short || int64(len(c.Points))+1 < minW // not evaluable, but it must not hide a rise or a clamp
		if v, hit := chainRise(c, rise, wins); hit {
			return v
		}
	}
	if v, hit := moneyRun(in.M, in.MInit, mFactor); hit {
		return v
	}
	if short {
		return unknownV(ReasonWindows)
	}
	if in.Stale > staleMax {
		return unknownV(ReasonStale)
	}
	return Verdict{}
}

func chainRise(c Series, rise, wins int64) (Verdict, bool) {
	_, reach := scanRuns(len(c.Points), wins, func(i int) bool { return satSub(c.Points[i].Value, bpScale) >= rise })
	for i, p := range c.Points {
		if c.clampedAt(i) && (reach < 0 || i < reach) {
			return detectedV(p.Tick, satSub(p.Value, bpScale), rise), true
		}
	}
	if reach < 0 {
		return Verdict{}, false
	}
	p := c.Points[reach]
	return detectedV(p.Tick, satSub(p.Value, bpScale), rise), true
}

// moneyCloses is the "two successive closes" of R1's money clause.
const moneyCloses = 2

func moneyRun(m []Point, mInit, factorBP int64) (Verdict, bool) {
	if mInit <= 0 {
		return Verdict{}, false
	}
	_, reach := scanRuns(len(m), moneyCloses, func(i int) bool { return cmpMul(m[i].Value, bpScale, mInit, factorBP) >= 0 })
	if reach < 0 {
		return Verdict{}, false
	}
	return detectedV(m[reach].Tick, m[reach].Value, factorBP), true
}

// ---- R9 ----

// detectR9: the link of the whole economy is at most 10 000 - r9.fall_bp in
// each of r9.windows consecutive windows and the later window's volume is
// below the earlier's (STY 6.2, FR-STY-069). A fall with growing volume is the
// pressure of MON 8.4 and not a failure. Volume is never made unknown by low
// activity.
func detectR9(in Input, t Thresholds) Verdict {
	k, ok := t.need(kR9Fall, kR9Windows, kMinWindows, kStaleMax)
	if !ok {
		return unknownV(ReasonThresholdUnset)
	}
	fall, wins, minW, staleMax := k[0], k[1], k[2], k[3]
	if len(in.Links) == 0 {
		return unknownV(ReasonWindows)
	}
	c := in.Links[0]
	n := len(c.Points)
	if int64(n)+1 < minW || len(in.Volume) < n+1 {
		return unknownV(ReasonWindows)
	}
	_, reach := scanRuns(n, wins, func(i int) bool {
		return satSub(bpScale, c.Points[i].Value) >= fall && in.Volume[i+1].Value < in.Volume[i].Value
	})
	if reach >= 0 {
		p := c.Points[reach]
		return detectedV(p.Tick, satSub(bpScale, p.Value), fall)
	}
	if c.anyClamped() {
		return unknownV(ReasonClamped)
	}
	if in.Stale > staleMax {
		return unknownV(ReasonStale)
	}
	return Verdict{}
}

// ---- R2 ----

// detectR2: the volume of the last window is below r2.vol_bp of the first and
// one of the pool, escrow and treasury share, the unsold share or the top
// decile share is at or above its threshold (STY 6.2, FR-STY-062; the top
// decile conjunct is open item O7). Low activity never makes it unknown: a
// fall of activity is its subject.
func detectR2(in Input, t Thresholds) Verdict {
	k, ok := t.need(kR2Vol, kR2Share, kR2Unsold, kR2Top, kR2Windows, kMinWindows)
	if !ok {
		return unknownV(ReasonThresholdUnset)
	}
	volBP, share, unsold, top, wins, minW := k[0], k[1], k[2], k[3], k[4], k[5]
	if int64(len(in.Volume)) < minW || len(in.Volume) == 0 {
		return unknownV(ReasonWindows)
	}
	first, last := in.Volume[0], in.Volume[len(in.Volume)-1]
	if cmpMul(last.Value, bpScale, first.Value, volBP) >= 0 {
		return Verdict{} // volume did not fall
	}
	if v, hit := atOrAboveRun(in.MoneyOut, share, wins); hit {
		return v
	}
	for _, s := range in.Unsold {
		if v, hit := risingRun(s, unsold, wins); hit {
			return v
		}
	}
	if v, hit := atOrAboveRun(in.Top, top, wins); hit {
		return v
	}
	return Verdict{}
}

// atOrAboveRun finds wins consecutive points at or above thr.
func atOrAboveRun(s []Point, thr, wins int64) (Verdict, bool) {
	_, reach := scanRuns(len(s), wins, func(i int) bool { return s[i].Value >= thr })
	if reach < 0 {
		return Verdict{}, false
	}
	return detectedV(s[reach].Tick, s[reach].Value, thr), true
}

// risingRun finds wins consecutive points at or above thr, each above the
// one before it (the "rising over windows" of R2's unsold share).
func risingRun(s []Point, thr, wins int64) (Verdict, bool) {
	_, reach := scanRuns(len(s), wins, func(i int) bool { return s[i].Value >= thr && i > 0 && s[i].Value > s[i-1].Value })
	if reach < 0 {
		return Verdict{}, false
	}
	return detectedV(s[reach].Tick, s[reach].Value, thr), true
}

// ---- R3 ----

// detectR3: households at or below 10 000 - r3.fall_bp of their start that are
// not back at 10 000 - r3.recover_bp within max(r3.horizon, r3.horizon_factor
// times t_min) ticks of the minimum, or a role with a positive start that has
// none for r3.role_ticks ticks (STY 6.2, FR-STY-063). t_min is the speed limit
// of STY-S02 from the trough, by RegrowTicks at Input.Rate. A run that ends
// before the horizon is unknown, "needs a longer run".
func detectR3(in Input, t Thresholds) Verdict {
	k, ok := t.need(kR3Fall, kR3Recover, kR3Factor, kR3Horizon, kR3RoleTicks)
	if !ok {
		return unknownV(ReasonThresholdUnset)
	}
	if len(in.Households) == 0 && len(in.Roles) == 0 {
		return unknownV(ReasonWindows)
	}
	res := Verdict{}
	for _, r := range in.Roles {
		if v, hit := roleEmpty(r, k[4]); hit {
			return v
		}
	}
	for _, h := range in.Households {
		v := collapse(h, in, k)
		switch {
		case v.Class == Detected:
			return v
		case v.Class == Unknown && res.Class == None:
			res = v
		}
	}
	return res
}

// roleEmpty: a role whose series starts positive and then holds 0 for need
// consecutive points.
func roleEmpty(r []Point, need int64) (Verdict, bool) {
	if len(r) == 0 || r[0].Value <= 0 {
		return Verdict{}, false
	}
	_, reach := scanRuns(len(r), need, func(i int) bool { return r[i].Value == 0 })
	if reach < 0 {
		return Verdict{}, false
	}
	return detectedV(r[reach].Tick, 0, need), true
}

// lowest returns the index of the first minimum of a non-empty series.
func lowest(h []Point) int {
	lo := 0
	for i, p := range h {
		if p.Value < h[lo].Value {
			lo = i
		}
	}
	return lo
}

// fellTo reports whether the series fell to at most 10 000 - fall of a
// positive start.
func fellTo(h []Point, fall int64) bool {
	if len(h) == 0 || h[0].Value <= 0 {
		return false
	}
	return cmpMul(h[lowest(h)].Value, bpScale, h[0].Value, satSub(bpScale, fall)) <= 0
}

// collapse judges one household series; k holds the five keys of R3.
func collapse(h []Point, in Input, k []int64) Verdict {
	fall, recover, factor, horizon := k[0], k[1], k[2], k[3]
	if len(h) < 2 {
		return unknownV(ReasonWindows)
	}
	if !fellTo(h, fall) {
		return Verdict{}
	}
	h0, li := h[0].Value, lowest(h)
	low := h[li]
	target, _ := mulDivFloor(h0, max(satSub(bpScale, recover), 0), bpScale)
	for _, p := range h[li:] {
		if p.Value >= target {
			return Verdict{} // recovered
		}
	}
	if tm, ok := RegrowTicks(low.Value, h0, recover, in.Rate, in.TicksPerYear); ok {
		if x, ok := mulDivFloor(tm, factor, bpScale); ok {
			horizon = max(horizon, x)
		} else {
			horizon = maxInt64
		}
	}
	if satSub(h[len(h)-1].Tick, low.Tick) < horizon {
		return unknownV(ReasonNeedsLongerRun)
	}
	left, _ := mulDivFloor(low.Value, bpScale, h0)
	return detectedV(low.Tick, bpScale-left, fall)
}

// ---- R4 ----

// detectR4: volume below r4.vol_bp of the first window in each of r4.windows
// consecutive windows while households stay above 10 000 - r3.fall_bp of their
// start; a collapse is R3's and R4 is then none (STY 6.2, FR-STY-064). Low
// activity never makes it unknown.
func detectR4(in Input, t Thresholds) Verdict {
	k, ok := t.need(kR4Vol, kR4Windows, kR3Fall, kMinWindows)
	if !ok {
		return unknownV(ReasonThresholdUnset)
	}
	volBP, wins, fall, minW := k[0], k[1], k[2], k[3]
	if int64(len(in.Volume)) < minW || len(in.Volume) == 0 {
		return unknownV(ReasonWindows)
	}
	if len(in.Households) == 0 {
		return unknownV(ReasonReferenceMissing)
	}
	for _, h := range in.Households {
		if fellTo(h, fall) {
			return Verdict{}
		}
	}
	v := in.Volume
	_, reach := scanRuns(len(v), wins, func(i int) bool { return i > 0 && cmpMul(v[i].Value, bpScale, v[0].Value, volBP) < 0 })
	if reach < 0 {
		return Verdict{}
	}
	share, _ := mulDivFloor(v[reach].Value, bpScale, v[0].Value)
	return detectedV(v[reach].Tick, share, volBP)
}

// ---- R5a, R5b ----

// detectR5a: in one-tick windows Pinned.Share of a market is at or above
// r5.pinned_bp for at least r5.pinned_ticks consecutive ticks (STY 6.2,
// FR-STY-065). A series shorter than r5.pinned_ticks is unknown, and a share
// of no outcomes is unknown "outcomes".
func detectR5a(in Input, t Thresholds) Verdict {
	k, ok := t.need(kR5Pinned, kR5PinnedTick)
	if !ok {
		return unknownV(ReasonThresholdUnset)
	}
	if len(in.Pinned) == 0 {
		return unknownV(ReasonWindows)
	}
	if in.Outcomes <= 0 {
		return unknownV(ReasonOutcomes)
	}
	return tickRuns(in.Pinned, k[1], func(p Point) bool { return p.Value >= k[0] })
}

// detectR5b: in one-tick windows a market has unmatched units on both sides
// and no volume for at least r5.freeze_ticks consecutive ticks (STY 6.2,
// FR-STY-065). A series shorter than the key is unknown. It is never made
// unknown by low activity.
func detectR5b(in Input, t Thresholds) Verdict {
	k, ok := t.need(kR5Freeze)
	if !ok {
		return unknownV(ReasonThresholdUnset)
	}
	if len(in.Frozen) == 0 {
		return unknownV(ReasonWindows)
	}
	return tickRuns(in.Frozen, k[0], func(p Point) bool { return p.Value != 0 })
}

// tickRuns is the shared rule of R5a and R5b over the one-tick series of the
// markets: Detected when any market holds cond for count consecutive points;
// otherwise Unknown if a series is shorter than count, else None.
func tickRuns(series [][]Point, count int64, cond func(Point) bool) Verdict {
	short := false
	for _, s := range series {
		if int64(len(s)) < count {
			short = true
		}
		longest, reach := scanRuns(len(s), count, func(i int) bool { return cond(s[i]) })
		if reach >= 0 {
			return detectedV(s[reach].Tick, longest, count)
		}
	}
	if short {
		return unknownV(ReasonWindows)
	}
	return Verdict{}
}

// ---- R6 ----

// swingWindows is the "two windows" of R6's role-share swing.
const swingWindows = 2

// detectR6: the PriceRange ratio of the season or year window is above
// r6.factor and above the previous window's by more than r6.growth_bp in
// r6.windows successive windows, or a role's RoleShares swing is at least
// r6.swing_bp in two windows (STY 6.2, FR-STY-066). The caller passes year or
// season windows, never a bare window shorter than the season (4.1.3). Low
// activity makes it unknown.
func detectR6(in Input, t Thresholds) Verdict {
	k, ok := t.need(kR6Factor, kR6Growth, kR6Windows, kR6Swing, kMinWindows, kActiveMin)
	if !ok {
		return unknownV(ReasonThresholdUnset)
	}
	factor, growth, wins, swing := k[0], k[1], k[2], k[3]
	if reason := r6Shape(in, k[4], k[5]); reason != "" {
		return unknownV(reason)
	}
	for _, s := range in.Ranges {
		if v, hit := rangeBlowUp(s.Points, factor, growth, wins); hit {
			return v
		}
	}
	for _, s := range in.Swings {
		if v, hit := atOrAboveRun(s, swing, swingWindows); hit {
			return v
		}
	}
	for _, s := range in.Ranges {
		if s.anyClamped() {
			return unknownV(ReasonClamped)
		}
	}
	return Verdict{}
}

// r6Shape returns the reason R6 cannot be evaluated, or "".
func r6Shape(in Input, minW, activeMin int64) string {
	switch {
	case in.Active < activeMin:
		return ReasonActive
	case len(in.Ranges) == 0 && len(in.Swings) == 0:
		return ReasonWindows
	}
	for _, s := range in.Ranges {
		if int64(len(s.Points)) < minW {
			return ReasonWindows
		}
	}
	return ""
}

func rangeBlowUp(r []Point, factor, growth, wins int64) (Verdict, bool) {
	_, reach := scanRuns(len(r), wins, func(i int) bool {
		return i > 0 && r[i].Value > factor && cmpMul(r[i].Value, bpScale, r[i-1].Value, satAdd(bpScale, growth)) > 0
	})
	if reach < 0 {
		return Verdict{}, false
	}
	return detectedV(r[reach].Tick, r[reach].Value, factor), true
}

// ---- R7 ----

// detectR7: the lowest Dispersion of a market in a window is below r7.disp_bp,
// or the completions of a recipe on one tick of the first r7.first_ticks ticks
// exceed r7.spike_factor / 10 000 times their mean (STY 6.2, FR-STY-067). The
// dispersion part is unknown without outcomes; the completions part is unknown
// for a series shorter than r7.first_ticks.
func detectR7(in Input, t Thresholds) Verdict {
	k, ok := t.need(kR7Disp, kR7Spike, kR7First)
	if !ok {
		return unknownV(ReasonThresholdUnset)
	}
	if len(in.Dispersion) == 0 && len(in.Completions) == 0 {
		return unknownV(ReasonWindows)
	}
	res := Verdict{}
	if len(in.Dispersion) > 0 {
		v := lowDispersion(in, k[0])
		if v.Class == Detected {
			return v
		}
		res = v
	}
	for _, c := range in.Completions {
		v := completionSpike(c, k[1], k[2])
		switch {
		case v.Class == Detected:
			return v
		case v.Class == Unknown && res.Class == None:
			res = v
		}
	}
	return res
}

func lowDispersion(in Input, disp int64) Verdict {
	if in.Outcomes <= 0 {
		return unknownV(ReasonOutcomes)
	}
	for _, s := range in.Dispersion {
		for _, p := range s {
			if p.Value < disp {
				return detectedV(p.Tick, p.Value, disp)
			}
		}
	}
	return Verdict{}
}

func completionSpike(c []Point, spike, first int64) Verdict {
	if int64(len(c)) < first || first < 1 {
		return unknownV(ReasonWindows)
	}
	c = c[:first]
	var sum int64
	for _, p := range c {
		sum = satAdd(sum, p.Value)
	}
	for _, p := range c {
		// completions > spike/10 000 times the mean sum/len
		if cmpMul(p.Value, first*bpScale, spike, sum) > 0 {
			return detectedV(p.Tick, p.Value, spike)
		}
	}
	return Verdict{}
}

// ---- R8 ----

// detectR8: the absolute Offset of a market is at least r8.offset_bp and the
// Unfed share of household-ticks of the same window is at least r8.unmet_bp
// (STY 6.2, FR-STY-068). It needs a steady-state price and activity.
func detectR8(in Input, t Thresholds) Verdict {
	k, ok := t.need(kR8Offset, kR8Unmet, kMinWindows, kActiveMin)
	if !ok {
		return unknownV(ReasonThresholdUnset)
	}
	offset, unmet := k[0], k[1]
	if reason := r8Shape(in, k[2], k[3]); reason != "" {
		return unknownV(reason)
	}
	for i, u := range in.Unmet {
		if u.Value < unmet {
			continue
		}
		if off, hit := offsetAt(in.Offsets, i, offset); hit {
			return detectedV(u.Tick, off, offset)
		}
	}
	for _, s := range in.Offsets {
		if s.anyClamped() {
			return unknownV(ReasonClamped)
		}
	}
	return Verdict{}
}

// r8Shape returns the reason R8 cannot be evaluated, or "".
func r8Shape(in Input, minW, activeMin int64) string {
	switch {
	case !in.HasReference:
		return ReasonReferenceMissing
	case in.Active < activeMin:
		return ReasonActive
	case int64(len(in.Unmet)) < minW || len(in.Offsets) == 0:
		return ReasonWindows
	}
	return ""
}

// offsetAt returns the magnitude of the first market offset at window i that
// is at least thr.
func offsetAt(offsets []Series, i int, thr int64) (int64, bool) {
	for _, s := range offsets {
		if i >= len(s.Points) {
			continue
		}
		m := magnitude(s.Points[i].Value)
		if thr <= 0 || m >= uint64(thr) {
			return int64(min(m, uint64(maxInt64))), true
		}
	}
	return 0, false
}
