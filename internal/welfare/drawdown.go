package welfare

// Drawdown returns the deepest fall from a running peak of the cumulative
// Income - Cost over recs, oldest first (spec-saving-and-investment.md 4.1.1):
//
//	C_0 = 0, C_i = C_(i-1) + (Income_i - Cost_i)
//	Peak_0 = 0, Peak_i = max(Peak_(i-1), C_i)
//	Drawdown = max over i of (Peak_i - C_i), at least 0
//
// It is 0 for no records and for a run that never fell. The cost is O(len(recs))
// and it allocates nothing. Each record field is clamped to plus or minus
// MaxInt64 / maxWindow by the ring, so a window's net can still reach twice
// MaxInt64; every step saturates at the nearest extreme instead of failing,
// because a drawdown above the buffer cap changes no result and the caller has
// nowhere to put an error (N-491).
func Drawdown(recs []Record) int64 {
	var cum, peak, worst int64
	for i := range recs {
		cum = satAdd(cum, satSub(recs[i].Income, recs[i].Cost))
		if cum > peak {
			peak = cum
		}
		if d := satSub(peak, cum); d > worst {
			worst = d
		}
	}
	if worst < 0 {
		return 0
	}
	return worst
}

// DrawdownRing is Drawdown of the records a ring holds, read through the
// scratch the caller owns: it appends to scratch[:0] and returns the drawdown
// and the scratch, so a steady caller allocates nothing once the scratch has
// grown to the window.
func DrawdownRing(r *Ring, scratch []Record) (int64, []Record) {
	scratch = r.AppendRecords(scratch[:0])
	return Drawdown(scratch), scratch
}
