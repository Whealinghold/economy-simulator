package steady

// Run is a stretch of consecutive ticks of the year with one yield factor, in
// basis points. A year is a slice of runs in tick order; the package knows no
// season, the caller turns the calendar into runs (A3 of the plan).
type Run struct {
	Length   int64
	FactorBP int64
}

// StartCount is the greedy start sequence of one household over a year.
type StartCount struct {
	// Starts is J, the number of jobs started.
	Starts int64
	// FactorSum is the sum of the yield factors, in basis points, at the starts;
	// the mean factor E is FactorSum / Starts.
	FactorSum int64
	// Completions holds the tick at which each start completes, start plus
	// duration minus 1, in order and not reduced modulo the year: a job begun late
	// in the year completes past its end (spec-viability.md 4.5).
	Completions []int64
}

// Starts counts the jobs one household can start in a year, greedily from tick
// 0: start at the first tick whose factor is positive, be busy for duration
// ticks, repeat. A start counts at every tick of the open window, whether or not
// the job completes inside the year (spec-viability.md 3.1, FR-VIA-002). It
// works run by run, so its cost is the number of runs plus the starts.
//
// A duration below 1, a negative length or a negative factor is a violated
// programmer invariant and panics.
func Starts(duration int64, year []Run) StartCount {
	if duration < 1 {
		panic("steady: invariant violated: duration below 1")
	}
	var sc StartCount
	var t, base int64 // t is the first tick a start is possible
	for _, r := range year {
		if r.Length < 0 || r.FactorBP < 0 {
			panic("steady: invariant violated: negative run")
		}
		end := base + r.Length
		base = end
		if t >= end {
			continue // busy through the whole run
		}
		if r.FactorBP == 0 {
			t = end
			continue
		}
		n := (end - t + duration - 1) / duration
		for i := int64(0); i < n; i++ {
			sc.Completions = append(sc.Completions, t+i*duration+duration-1)
		}
		sc.Starts += n
		sc.FactorSum += n * r.FactorBP
		t += n * duration
	}
	return sc
}
