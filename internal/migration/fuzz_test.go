package migration_test

import (
	"math"
	"math/big"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/migration"
)

// bigFits reports whether v fits in int64.
func bigFits(v *big.Int) bool { return v.IsInt64() }

// floorDiv and ceilDiv divide exactly by a non-zero b, rounding toward negative
// and positive infinity.
func floorDiv(a, b *big.Int) *big.Int {
	q, r := new(big.Int).QuoRem(a, b, new(big.Int))
	if r.Sign() != 0 && (a.Sign() < 0) != (b.Sign() < 0) {
		q.Sub(q, big.NewInt(1))
	}
	return q
}

func ceilDiv(a, b *big.Int) *big.Int {
	q, r := new(big.Int).QuoRem(a, b, new(big.Int))
	if r.Sign() != 0 && (a.Sign() < 0) == (b.Sign() < 0) {
		q.Add(q, big.NewInt(1))
	}
	return q
}

// FuzzEstimate checks the arithmetic of the estimate against exact integers:
// whatever the inputs nothing panics, a result reported with ok true equals the
// exact value, and an overflow is reported and never wrapped.
//
// Requirement: NFR-MIG-004, NFR-MIG-003
func FuzzEstimate(f *testing.F) {
	f.Add(int64(70), int64(20), int64(14_376), int64(2), int64(180), int64(6_393), int64(5_957), int64(1042), int64(1_000), int64(5_000))
	f.Add(int64(0), int64(1), int64(-7), int64(1), int64(10), int64(-1), int64(0), int64(0), int64(0), int64(0))
	f.Add(int64(math.MaxInt64), int64(1), int64(math.MaxInt64), int64(1), int64(2), int64(math.MinInt64), int64(math.MaxInt64), int64(-1), int64(math.MaxInt64), int64(math.MaxInt64))
	f.Fuzz(func(t *testing.T, dist, speed, vhat, k, horizon, home, margin, id, gain, width int64) {
		bi := big.NewInt
		if got, ok := migration.Ticks(dist, speed); ok {
			want := ceilDiv(bi(dist), bi(speed))
			if want.Cmp(bi(1)) < 0 {
				want = bi(1)
			}
			if speed < 1 || dist < 0 || got != want.Int64() {
				t.Errorf("Ticks(%d, %d) = %d, true, want %v", dist, speed, got, want)
			}
		} else if speed >= 1 && dist >= 0 {
			t.Errorf("Ticks(%d, %d) refused a valid input", dist, speed)
		}

		got, ok := migration.Discount(vhat, k, horizon)
		valid := horizon >= 1 && k >= 0 && k < horizon
		switch {
		case ok != valid:
			t.Errorf("Discount(%d, %d, %d) ok = %v, want %v", vhat, k, horizon, ok, valid)
		case ok:
			prod := new(big.Int).Mul(bi(vhat), bi(horizon-k))
			if want := new(big.Int).Quo(prod, bi(horizon)); got != want.Int64() {
				t.Errorf("Discount(%d, %d, %d) = %d, want %v", vhat, k, horizon, got, want)
			}
		}

		span := new(big.Int).Mul(bi(int64(uint32(id)*2654435769)), bi(width))
		wantG := new(big.Int).Add(bi(gain), floorDiv(span, bi(1<<32)))
		if g, ok := migration.Margin(id, gain, width); ok != (width >= 0 && bigFits(wantG)) || (ok && g != wantG.Int64()) {
			t.Errorf("Margin(%d, %d, %d) = %d, %v, want %v, ok when width >= 0 and the sum fits", id, gain, width, g, ok, wantG)
		}

		if clears, ok := migration.Clears(vhat, home, margin); ok {
			lhs := new(big.Int).Mul(bi(vhat), bi(10_000))
			rhs := new(big.Int).Mul(bi(home), new(big.Int).Add(bi(10_000), bi(margin)))
			if margin < 0 || clears != (lhs.Cmp(rhs) >= 0) {
				t.Errorf("Clears(%d, %d, %d) = true-ok %v, want %v", vhat, home, margin, clears, lhs.Cmp(rhs) >= 0)
			}
		}

		if v, ok := migration.Vhat(home, speed); ok {
			want := new(big.Int).Quo(new(big.Int).Mul(bi(home), bi(10_000)), bi(speed))
			if speed < 1 || v != want.Int64() {
				t.Errorf("Vhat(%d, %d) = %d, true, want %v", home, speed, v, want)
			}
		} else if speed >= 1 {
			want := new(big.Int).Quo(new(big.Int).Mul(bi(home), bi(10_000)), bi(speed))
			if bigFits(want) {
				t.Errorf("Vhat(%d, %d) refused a quotient that fits: %v", home, speed, want)
			}
		}

		// One recipe: output (q, p), input (qi, pi), tool (wear, durability, tp).
		r := migration.Recipe{
			Duration: k, Yield: horizon,
			Outputs: []migration.Line{{Qty: dist, Price: speed}},
			Inputs:  []migration.Line{{Qty: vhat, Price: home}},
			Tools:   []migration.Tool{{Wear: margin, Durability: id, Price: gain}},
		}
		income, priced, ok := migration.Income([]migration.Recipe{r})
		inDomain := k >= 1 && horizon >= 1 && dist >= 0 && speed >= 1 && vhat >= 0 && home >= 1 && margin >= 0 && id >= 1 && gain >= 1
		if ok && inDomain {
			value := floorDiv(new(big.Int).Mul(new(big.Int).Mul(bi(dist), bi(horizon)), bi(speed)), bi(10_000))
			cost := new(big.Int).Add(new(big.Int).Mul(bi(vhat), bi(home)), ceilDiv(new(big.Int).Mul(bi(margin), bi(gain)), bi(id)))
			net := new(big.Int).Sub(value, cost)
			wantPriced := home >= 1 && speed >= 1 && gain >= 1
			want := int64(0)
			if net.Sign() > 0 {
				want = new(big.Int).Quo(net, bi(k)).Int64()
			}
			// With an input the recipe is eligible only at a positive net.
			if priced != wantPriced || income != want {
				t.Errorf("Income(%+v) = %d, %v, want %d, %v", r, income, priced, want, wantPriced)
			}
		}
	})
}

// FuzzAdmit checks the admission of a decoded set of households: nothing panics,
// every household ends in exactly one outcome, the results are in the order of
// need, no cap is passed by more than one household's weight, the outcomes agree
// with the inputs that cause them, and a second run gives the same results.
//
// Requirement: NFR-MIG-004, FR-MIG-025
func FuzzAdmit(f *testing.F) {
	f.Add([]byte{3, 3, 2, 5, 9, 4, 1, 200, 3, 1, 1, 2, 2, 7, 3, 9, 2, 4, 1, 3, 3, 6, 5, 2, 0, 1})
	f.Add([]byte{1, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20})
	f.Add([]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255})
	f.Fuzz(func(t *testing.T, data []byte) {
		pos := 0
		next := func(mod int) int {
			if pos >= len(data) {
				return 0
			}
			v := int(data[pos]) % mod
			pos++
			return v
		}
		n := 1 + next(4)
		caps := migration.Caps{Out: make([]int64, n), In: make([]int64, n), Pipeline: make([]int64, n)}
		for i := range n {
			caps.Out[i], caps.In[i], caps.Pipeline[i] = int64(next(12)), int64(next(12)), int64(next(12))
		}
		var agents []migration.Agent
		for i, count := 0, next(14); i < count; i++ {
			a := migration.Agent{Ref: i, ID: int64(next(30)), Weight: int64(1 + next(4)), EarnedBP: int64(next(5)) - 1,
				Origin: next(n), Bound: next(8) == 0, HoldTicks: int64(next(6))}
			for r, rk := 0, next(4); r < rk && r < n; r++ {
				a.Ranked = append(a.Ranked, migration.Ranked{Ref: r, K: int64(1 + next(5)), VEff: 1})
			}
			agents = append(agents, a)
		}
		var s migration.Scratch
		got := migration.Admit(nil, agents, caps, &s)
		again := migration.Admit(nil, agents, caps, &migration.Scratch{})
		if !slices.Equal(got, again) {
			t.Fatalf("two runs differ: %v and %v", got, again)
		}
		if len(got) != len(agents) {
			t.Fatalf("%d results for %d agents", len(got), len(agents))
		}
		maxW := int64(0)
		outW, inW := make([]int64, n), make([]int64, n)
		seen := make([]bool, len(agents))
		for i, r := range got {
			if seen[r.Ref] {
				t.Fatalf("agent %d has two results", r.Ref)
			}
			seen[r.Ref] = true
			a := agents[r.Ref]
			maxW = max(maxW, a.Weight)
			if i > 0 {
				p := agents[got[i-1].Ref]
				if p.EarnedBP > a.EarnedBP || (p.EarnedBP == a.EarnedBP && p.ID > a.ID) {
					t.Errorf("results %d and %d are out of the order of need", got[i-1].Ref, r.Ref)
				}
			}
			switch r.Outcome {
			case migration.Started:
				k := int64(-1)
				for _, c := range a.Ranked {
					if c.Ref == r.Dest {
						k = c.K
					}
				}
				if a.Bound || k < 0 || k != r.K || r.K > a.HoldTicks {
					t.Errorf("agent %d started to %d in %d ticks: bound %v, ranking %v, hold %d", r.Ref, r.Dest, r.K, a.Bound, a.Ranked, a.HoldTicks)
				}
				outW[a.Origin] += a.Weight
				inW[r.Dest] += a.Weight
			case migration.Bound:
				if !a.Bound {
					t.Errorf("agent %d refused as bound but is not", r.Ref)
				}
			case migration.NoDestination:
				if len(a.Ranked) != 0 {
					t.Errorf("agent %d has a ranking and no destination", r.Ref)
				}
			case migration.Unprovisioned, migration.InCap:
				anyHeld := false
				for _, c := range a.Ranked {
					anyHeld = anyHeld || c.K <= a.HoldTicks
				}
				if a.Bound || len(a.Ranked) == 0 || anyHeld != (r.Outcome == migration.InCap) {
					t.Errorf("agent %d outcome %d with hold %d and ranking %v", r.Ref, r.Outcome, a.HoldTicks, a.Ranked)
				}
			case migration.OutCap:
			default:
				t.Errorf("agent %d has outcome %d, which admission never gives", r.Ref, r.Outcome)
			}
		}
		for i := range n {
			if outW[i] > 0 && outW[i] >= caps.Out[i]+maxW {
				t.Errorf("origin %d admitted weight %d past its cap %d by more than %d", i, outW[i], caps.Out[i], maxW)
			}
			if caps.Out[i] == 0 && outW[i] != 0 {
				t.Errorf("origin %d with cap 0 admitted weight %d", i, outW[i])
			}
			if inW[i] > 0 && (caps.Pipeline[i] >= caps.In[i] || caps.Pipeline[i]+inW[i] >= caps.In[i]+maxW) {
				t.Errorf("destination %d admitted weight %d on a pipeline of %d, cap %d, max weight %d", i, inW[i], caps.Pipeline[i], caps.In[i], maxW)
			}
		}
	})
}
