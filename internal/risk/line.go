package risk

// Line returns the units lost of a line that holds held units at a loss share
// of lossBP, rhe(held * lossBP, 10 000), half to even (spec-risk.md 5.1.1,
// 5.3.1). A line of one unit loses it if and only if lossBP is above 5 000.
// ok is false when held is negative or lossBP is outside 0 to 10 000.
func Line(held, lossBP int64) (int64, bool) {
	if lossBP > BP {
		return 0, false
	}
	return rhe(held, lossBP)
}

// Account returns, for each commodity of a trip account, the units lost at
// lossBP, with the commodity at index skip left out (spec-risk.md 5.4.1: the
// mode's Vehicle is not cargo), appended to dst[:0]. A skip below 0 or beyond
// the account leaves nothing out. ok is false when a held count is negative or
// lossBP is outside 0 to 10 000, and the result is then nil.
func Account(dst, held []int64, skip int, lossBP int64) ([]int64, bool) {
	dst = dst[:0]
	for i, h := range held {
		var lost int64
		if i != skip {
			var ok bool
			if lost, ok = Line(h, lossBP); !ok {
				return nil, false
			}
		}
		dst = append(dst, lost)
	}
	return dst, true
}
