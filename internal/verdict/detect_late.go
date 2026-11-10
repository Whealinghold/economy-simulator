package verdict

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
