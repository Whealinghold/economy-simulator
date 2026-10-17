package welfare

// WorkShare returns the share in basis points of a household's executions that
// the target-income rule requests (spec-welfare-and-wages.md 5): 10 000 below
// targetLo, 0 at or above T_hi = targetLo * (10 000 + restBand) / 10 000, and
// (T_hi - ratio) * 10 000 / (T_hi - T_lo) between, truncated. With restBand 0,
// or a T_hi that truncates to targetLo, the band is empty and the share is
// 10 000 below the target and 0 at or above it, never a division by zero.
func WorkShare(ratioBP, targetLo, restBand int64) int64 {
	if ratioBP < targetLo {
		return bp
	}
	hi := satMulDiv(targetLo, satAdd(bp, restBand), bp)
	if ratioBP >= hi {
		return 0
	}
	// targetLo <= ratioBP < hi, so hi - targetLo is above 0.
	return satMulDiv(satSub(hi, ratioBP), bp, satSub(hi, targetLo))
}

// Executions returns the executions a household cohort requests,
// ceil(weight * shareBP / 10 000) (spec-welfare-and-wages.md 5).
func Executions(weight, shareBP int64) int64 {
	return satMulDivCeil(weight, shareBP, bp)
}

// FloorBinds reports whether the work floor holds an agent to work: the
// working ticks of its window are fewer than minShareBP * window / 10 000 in
// integer division, window the effective window and not the records held, so a
// missing tick is not a working tick (D65, 5.3). A minShareBP of 0 turns the
// floor off.
func FloorBinds(working, window, minShareBP int64) bool {
	if minShareBP == 0 {
		return false
	}
	return working < satMulDiv(minShareBP, window, bp)
}

// TargetRange returns the lowest and highest target of an agent,
// targetRatio * (10 000 - spread) / 10 000 and targetRatio * (10 000 + spread)
// / 10 000, truncated (5.1). The draw is the caller's: it takes
// lo + draw(hi - lo + 1) from its own stream.
func TargetRange(targetRatio, spread int64) (lo, hi int64) {
	return satMulDiv(targetRatio, satSub(bp, spread), bp), satMulDiv(targetRatio, satAdd(bp, spread), bp)
}
