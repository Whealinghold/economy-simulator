package indicator_test

import (
	"errors"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// isSentinel reports whether err is one of the six sentinels of the package.
func isSentinel(err error) bool {
	for _, s := range []error{indicator.ErrConfigMismatch, indicator.ErrWindowOrder, indicator.ErrUnknownClass, indicator.ErrRange, indicator.ErrNegativeValue, indicator.ErrOverflow} {
		if errors.Is(err, s) {
			return true
		}
	}
	return false
}

// rowsOfBytes decodes up to 12 household rows of weight 1 to 7 and total 0 to 255
// (a negative total when neg is set, to reach the sentinel).
func rowsOfBytes(data []byte, neg bool) []indicator.Household {
	var rows []indicator.Household
	for i := 0; i+1 < len(data) && len(rows) < 12; i += 2 {
		total := int64(data[i+1])
		if neg && data[i]&0x80 != 0 {
			total = -total
		}
		rows = append(rows, indicator.Household{Weight: int64(data[i]%7) + 1, Total: total})
	}
	return rows
}

// FuzzRowStatistics verifies, for any rows, that no statistic panics; that the
// result does not depend on the order of the rows (FR-IND-010); that the Gini of
// non-negative rows is between 0 and (W - 1) / W (FR-IND-014) and that a row of
// weight w is w rows of weight 1 (FR-IND-015); that quantiles do not decrease
// (FR-IND-024); and that a share is between 0 and 10 000.
//
// Requirement: NFR-IND-003, FR-IND-010, FR-IND-014, FR-IND-015, FR-IND-024
func FuzzRowStatistics(f *testing.F) {
	f.Add([]byte{1, 5, 1, 9, 2, 0}, uint8(3))
	f.Add([]byte{}, uint8(0))
	f.Add([]byte{0x81, 4, 3, 4}, uint8(200))
	f.Add([]byte{255, 255, 255, 255, 0, 0}, uint8(9))
	f.Fuzz(func(t *testing.T, data []byte, arg uint8) {
		rows := rowsOfBytes(data, true)
		rev := append([]indicator.Household(nil), rows...)
		for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
			rev[i], rev[j] = rev[j], rev[i]
		}
		for _, op := range []indicator.StatOp{indicator.StatSum, indicator.StatMean, indicator.StatQuantile, indicator.StatShareBelow, indicator.StatGini} {
			a := int64(arg) * 40
			if op == indicator.StatQuantile {
				a = int64(arg) * 39 // 0 to 9 945 of 10 000
			}
			got, err := indicator.Stat(rows, op, a)
			if err != nil {
				if !isSentinel(err) {
					t.Fatalf("Stat(%v, %v) error = %v, want a sentinel", rows, op, err)
				}
				continue
			}
			again, err := indicator.Stat(rev, op, a)
			if err != nil || again != got {
				t.Fatalf("Stat(reversed rows, %v) = %+v, %v, want %+v: the order of rows must not matter", op, again, err, got)
			}
			if op == indicator.StatShareBelow && (got.Value < 0 || got.Value > 10_000) {
				t.Errorf("Stat(%v, share below %d) = %d, want within [0, 10 000]", rows, a, got.Value)
			}
			if op == indicator.StatGini && got.OK && (got.Value < 0 || got.Value > 10_000*(got.Households-1)/got.Households+1) {
				t.Errorf("Stat(%v, gini) = %d over %d households, want within [0, (W - 1) / W]", rows, got.Value, got.Households)
			}
			if op == indicator.StatGini {
				var expanded []indicator.Household
				for _, r := range rows {
					for k := int64(0); k < r.Weight; k++ {
						expanded = append(expanded, indicator.Household{Weight: 1, Total: r.Total / r.Weight})
					}
				}
				// A weight divides its total only when it is exact; compare on rows that are.
				if exact(rows) {
					if e, err := indicator.Stat(expanded, op, 0); err != nil || e.Value != got.Value || e.OK != got.OK {
						t.Errorf("Stat(%v, gini) = %+v, but the same households as rows of weight 1 give %+v, %v", rows, got, e, err)
					}
				}
			}
		}
		var prev int64
		seen := false
		for _, q := range []int64{0, 1000, 5000, 9000, 10_000} {
			got, err := indicator.Stat(rows, indicator.StatQuantile, q)
			if err != nil || !got.OK {
				continue
			}
			if seen && got.Value < prev {
				t.Errorf("Stat(%v, quantile %d) = %d after %d, want quantiles not to decrease", rows, q, got.Value, prev)
			}
			prev, seen = got.Value, true
		}
	})
}

// exact reports whether every row's total divides by its weight.
func exact(rows []indicator.Household) bool {
	for _, r := range rows {
		if r.Total%r.Weight != 0 {
			return false
		}
	}
	return true
}

// FuzzIndexFunctions verifies, for any prices and quantities of up to four
// commodities in one region, that PriceIndex never panics and returns only the
// sentinels; that an index against its own base is 10 000 where defined
// (FR-IND-020); that Fisher is the nearest integer to the square root of
// Laspeyres times Paasche (FR-IND-021); and that a chain never exceeds the clamp
// and never recovers from an undefined link (FR-IND-022).
//
// Fix: the corpus entry 0d202d7b28aa0501 (all prices and volumes 0) pins the
// bound of the Fisher check at 0: a Fisher of 0 for a Laspeyres and a Paasche of
// 0 is exact, and the first form of the bound, (2 * 0 - 1)^2, refused it.
//
// Requirement: NFR-IND-003, FR-IND-020, FR-IND-021, FR-IND-022
func FuzzIndexFunctions(f *testing.F) {
	f.Add([]byte{10, 4, 12, 5, 20, 2, 18, 3})
	f.Add([]byte{})
	f.Add([]byte{0, 0, 0, 0, 255, 255, 255, 255})
	f.Fuzz(func(t *testing.T, data []byte) {
		n := min(len(data)/4, 4)
		if n == 0 {
			return
		}
		h := [32]byte{1}
		p0c, p1c := make([]economysim.Price, n), make([]economysim.Price, n)
		q0, q1 := make([]int64, n), make([]int64, n)
		w := indicator.Window{ConfigHash: h, Regions: 1, Commodities: n, Markets: make([]indicator.MarketTally, n)}
		for i := 0; i < n; i++ {
			p0c[i], p1c[i] = economysim.Price(data[4*i]), economysim.Price(data[4*i+1])
			q0[i], q1[i] = int64(data[4*i+2]), int64(data[4*i+3])
			w.Markets[i].SumV = q1[i]
		}
		p0, err0 := indicator.NewPrices(h, 0, 1, 1, n, p0c)
		p1, err1 := indicator.NewPrices(h, 0, 1, 1, n, p1c)
		if err0 != nil || err1 != nil {
			t.Fatalf("NewPrices() errors = %v, %v, want nil", err0, err1)
		}
		base := indicator.Base{Prices: p0, Traded: q0}
		got, err := indicator.PriceIndex(w, base, indicator.WeightsTraded, p1, indicator.CellSel{})
		if err != nil {
			if !isSentinel(err) {
				t.Fatalf("PriceIndex() error = %v, want a sentinel", err)
			}
			return
		}
		l, p, fi := int64(got.Laspeyres.Value), int64(got.Paasche.Value), int64(got.Fisher.Value)
		if got.Laspeyres.OK && got.Paasche.OK && got.Fisher.OK && !got.Laspeyres.Clamped && !got.Paasche.Clamped && !got.Fisher.Clamped {
			lo, hi := int64(0), (2*fi+1)*(2*fi+1)
			if fi > 0 {
				lo = (2*fi - 1) * (2*fi - 1)
			}
			if 4*l*p < lo || 4*l*p > hi {
				t.Errorf("PriceIndex() Fisher = %d for Laspeyres %d and Paasche %d, want the nearest integer to the square root of their product", fi, l, p)
			}
		}
		same, err := indicator.PriceIndex(indicator.Window{ConfigHash: h, Regions: 1, Commodities: n, Markets: markets(q0)}, base, indicator.WeightsTraded, p0, indicator.CellSel{})
		if err != nil {
			t.Fatalf("PriceIndex(own base) error = %v, want nil", err)
		}
		for _, x := range []indicator.Index{same.Laspeyres, same.Paasche, same.Fisher} {
			if x.OK && x.Value != 10_000 {
				t.Errorf("PriceIndex(own base) = %+v, want 10 000 where defined", same)
			}
		}
		links := []indicator.Index{got.Laspeyres, same.Fisher, got.Paasche}
		undefined := false
		for _, c := range indicator.Chain(links) {
			if c.Value > indicator.MaxIndexBP {
				t.Errorf("Chain(%v) holds %d, want at most the clamp", links, c.Value)
			}
			if undefined && c.OK {
				t.Errorf("Chain(%v) recovered from an undefined link", links)
			}
			undefined = undefined || !c.OK
		}
	})
}

// markets returns market rows of the given traded volumes.
func markets(q []int64) []indicator.MarketTally {
	out := make([]indicator.MarketTally, len(q))
	for i, v := range q {
		out[i].SumV = v
	}
	return out
}

// FuzzAccounts verifies, for the worked example with any one flow total or stock
// replaced, that GDP, Sectors, ValueAdded and TradeBalance never panic and return
// only the sentinels, and that the realm is the sum of the regions in the terms
// that add (FR-IND-017, FR-IND-018).
//
// Requirement: NFR-IND-003, FR-IND-017, FR-IND-018
func FuzzAccounts(f *testing.F) {
	f.Add(uint8(0), int32(7), uint8(1), int32(40), int32(2))
	f.Add(uint8(200), int32(-1), uint8(0), int32(0), int32(0))
	f.Add(uint8(3), int32(2147483647), uint8(7), int32(-5), int32(2147483647))
	f.Fuzz(func(t *testing.T, which uint8, total int32, mkt uint8, held, pending int32) {
		w, open, closing := workedWindow(t)
		w.Flows[int(which)%len(w.Flows)].Total = int64(total)
		m := &closing.Markets[int(mkt)%len(closing.Markets)]
		m.HeldByAgents, m.Pending = economysim.Quantity(held), economysim.Quantity(pending)
		p := valleyTable(t, w, alphaP, betaP)
		g, err := indicator.GDP(w, open, closing, p)
		if err != nil && !isSentinel(err) {
			t.Fatalf("GDP() error = %v, want a sentinel", err)
		}
		if err == nil {
			var out, inter, va int64
			for _, r := range g.Regions {
				out, inter, va = out+r.Output, inter+r.Intermediate, va+r.ValueAdded
			}
			if g.Realm.Output != out || g.Realm.Intermediate != inter || g.Realm.ValueAdded != va {
				t.Errorf("GDP() realm = %+v, want the sum of the regions (output %d, intermediate %d, value added %d)", g.Realm, out, inter, va)
			}
			if _, err := indicator.TradeBalance(w, p, g); err != nil && !isSentinel(err) {
				t.Fatalf("TradeBalance() error = %v, want a sentinel", err)
			}
		}
		s, err := indicator.Sectors(w, open, closing, p)
		if err != nil && !isSentinel(err) {
			t.Fatalf("Sectors() error = %v, want a sentinel", err)
		}
		if err == nil {
			var money int64
			for _, r := range s.Regions {
				money += r.Money
			}
			if s.Realm.Money != money {
				t.Errorf("Sectors() realm money = %d, want the sum of the regions, %d", s.Realm.Money, money)
			}
		}
		if _, err := indicator.ValueAdded(w, valleyConfig(), open, closing, p, indicator.NewClasses(valleyConfig())); err != nil && !isSentinel(err) {
			t.Fatalf("ValueAdded() error = %v, want a sentinel", err)
		}
	})
}
