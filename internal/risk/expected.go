package risk

// Edge is one row met on a path: its effective probability and its severity,
// both in basis points.
type Edge struct {
	P, SeverityBP int64
}

// Expected returns the expected loss of a route, in basis points, over the rows
// met along its path in path order (spec-risk.md 8.1.1):
//
//	K = 10 000; for each row: f = 10 000 - rhe(P * Severity, 10 000),
//	K = rhe(K * f, 10 000); LossBP = 10 000 - K
//
// It is the integer approximation of 1 - the product of (1 - p_i s_i). A
// common row has the same expectation as a unit row, so the caller passes both
// the same way. No row gives 0. ok is false when a value is outside 0 to
// 10 000.
func Expected(rows []Edge) (int64, bool) {
	kept := int64(BP)
	for _, e := range rows {
		if e.P < 0 || e.P > BP || e.SeverityBP > BP {
			return 0, false
		}
		hit, ok := rhe(e.P, e.SeverityBP)
		if !ok || hit > BP {
			return 0, false
		}
		if kept, ok = rhe(kept, BP-hit); !ok {
			return 0, false
		}
	}
	return BP - kept, true
}
