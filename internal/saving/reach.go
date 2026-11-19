package saving

import "github.com/Whealinghold/economy-sim/internal/checked"

// Need returns max(0, outlay - held + floor - money): what the household still
// lacks for the project (spec 5.6.5). It is 0 for a funded household.
func Need(outlay, held, floor, money int64) int64 {
	return nonneg(satSub(satAdd(satSub(outlay, nonneg(held)), nonneg(floor)), money))
}

// Reach reports need * n <= max(0, cn) * t in 128 bits (spec 5.6.5,
// FR-SAV-045): the household can save what it lacks, at its own mean surplus,
// within the t ticks of service the test counted. n is the records of the ring
// and cn the cumulative income - cost over them; with n of 0 only a need of 0
// is reachable.
func Reach(need, n, cn, t int64) bool {
	need, n, t = nonneg(need), nonneg(n), nonneg(t)
	if n == 0 {
		return need == 0
	}
	cmp, ok := checked.Cmp3(need, n, 1, nonneg(cn), t, 1)
	return ok && cmp <= 0
}
