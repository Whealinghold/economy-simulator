package reduce

// Moments accumulates the count, the sum and the sum of squares of int64
// readings in 128 bits. The zero value is empty. Each Add that would overflow
// 128 bits leaves the accumulator unchanged and returns false.
type Moments struct {
	n          int64
	sum, sumSq Int128
}

// Add folds one reading in, and returns false when the sum or the sum of squares
// would overflow (never for 10^6 readings of magnitude 2^40, spec-experiments.md
// 8.3.2).
func (m *Moments) Add(v int64) bool {
	sq, _ := FromInt64(v).Mul(FromInt64(v)) // |v|^2 <= 2^126: fits
	sum, ok1 := m.sum.Add(FromInt64(v))
	ss, ok2 := m.sumSq.Add(sq)
	if !ok1 || !ok2 {
		return false
	}
	m.n, m.sum, m.sumSq = m.n+1, sum, ss
	return true
}

// Count returns the number of readings added.
func (m Moments) Count() int64 { return m.n }

// Sum returns the exact sum of the readings.
func (m Moments) Sum() Int128 { return m.sum }

// Mean returns the sum over the count rounded half to even, and false for no
// readings.
func (m Moments) Mean() (int64, bool) {
	if m.n == 0 {
		return 0, false
	}
	q, ok := RDiv(m.sum, FromInt64(m.n))
	if !ok {
		return 0, false
	}
	return q.Int64() // a mean of int64 values is an int64
}

// Variance returns the sample variance, rounded half to even once:
// RDiv(n*SS - S^2, n*(n-1)), exact in 128 bits. It returns false for fewer than
// two readings and for an overflow, which the 2^40 bound on a reading rules out.
func (m Moments) Variance() (Int128, bool) {
	if m.n < 2 {
		return Int128{}, false
	}
	n := FromInt64(m.n)
	a, ok1 := n.Mul(m.sumSq)
	b, ok2 := m.sum.Mul(m.sum)
	num, ok3 := a.Sub(b)
	den, ok4 := n.Mul(FromInt64(m.n - 1))
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return Int128{}, false
	}
	return RDiv(num, den)
}

// Std returns the sample standard deviation, IsqrtNearest of the rounded
// variance: two roundings, the variance and then the root (spec-experiments.md
// 8.3). It returns false for fewer than two readings and for a root beyond int64.
func (m Moments) Std() (int64, bool) {
	v, ok := m.Variance()
	if !ok {
		return 0, false
	}
	r := IsqrtNearest(v)
	return int64(r), r <= 1<<63-1
}
