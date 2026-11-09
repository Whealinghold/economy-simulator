package measure

// sticky chains checked 128-bit operations and remembers the first overflow, so
// a formula of several steps is written once and checked once. After an overflow
// every later result is meaningless and over stays true.
type sticky struct{ over bool }

// mul returns a * m.
func (s *sticky) mul(a Uint128, m uint64) Uint128 {
	r, ok := a.MulU64(m)
	s.over = s.over || !ok
	return r
}

// add returns a + b.
func (s *sticky) add(a, b Uint128) Uint128 {
	r, ok := a.Add(b)
	s.over = s.over || !ok
	return r
}

// mulBP returns a * 10 000, guarded as MulBPU is.
func (s *sticky) mulBP(a Uint128) Uint128 {
	r, ok := MulBPU(a)
	s.over = s.over || !ok
	return r
}
