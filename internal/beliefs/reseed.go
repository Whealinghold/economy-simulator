package beliefs

// Seed is what Reseed needs for one commodity of a tradable set (spec-beliefs.md
// 8.3): its reference price and the caller's unit cost, the market's window mean
// (Empty when the window holds no price) and the market price, and, for an agent
// that changed role, the belief it already holds when the commodity is in the
// tradable sets of both roles.
type Seed struct {
	Ref, UnitCost int64
	Mean          int64
	Empty         bool
	Price         int64
	Kept          bool
	Belief        Belief
}

// Reseed appends to dst one belief per seed, in the order of the input (the
// canonical commodity order), and returns it (spec-beliefs.md 8.2, 8.3). A seed
// with Kept holds the stored belief with its centre and width re-clamped to the
// bounds of the new unit cost. Any other starts at the window mean, or at the
// market price while the window is empty, clamped to the bounds, with width
// WidthInit. The seed price of an initial agent is passed as Price with Empty
// set. It takes no random word: it consumes none.
func Reseed(dst []Belief, seeds []Seed, p Params) []Belief {
	for _, s := range seeds {
		bd := Limits(s.Ref, s.UnitCost, p)
		b := s.Belief
		if !s.Kept {
			b = Belief{Centre: s.Mean, Width: p.WidthInit}
			if s.Empty {
				b.Centre = s.Price
			}
		}
		b, _ = Clamp(b, bd, p)
		dst = append(dst, b)
	}
	return dst
}
