package risk

// Fold returns the kept share after one struck row of severityBP, rhe(kept *
// (10 000 - severityBP), 10 000) (spec-risk.md 4.1.1). ok is false when kept is
// outside 0 to 10 000 or severityBP outside 0 to 10 000.
func Fold(kept, severityBP int64) (int64, bool) {
	if kept < 0 || kept > BP || severityBP < 0 || severityBP > BP {
		return 0, false
	}
	return rhe(kept, BP-severityBP)
}

// Combine returns the loss share LossBP of a shipment whose path met the struck
// rows with the given severities, in path order: kept starts at 10 000, each
// row folds into it, and the result is 10 000 - kept (spec-risk.md 4.1.1 and
// 4.3.1). No struck row gives 0 and a severity of 10 000 gives 10 000. The
// order fixes the rounding. ok is false when a severity is outside 0 to 10 000.
func Combine(severities []int64) (lossBP int64, ok bool) {
	kept := int64(BP)
	for _, s := range severities {
		if kept, ok = Fold(kept, s); !ok {
			return 0, false
		}
	}
	return BP - kept, true
}
