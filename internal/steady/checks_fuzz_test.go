package steady_test

import (
	"context"
	"math/big"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/steady"
)

// fuzzBytes hands out the bytes of a fuzz input and then zeros.
type fuzzBytes struct {
	data []byte
	at   int
}

// next returns the next byte, or 0 past the end.
func (b *fuzzBytes) next() int {
	if b.at >= len(b.data) {
		return 0
	}
	b.at++
	return int(b.data[b.at-1])
}

// graph decodes up to 5 commodities and 6 candidate activities.
func (b *fuzzBytes) graph() (int, []steady.Candidate) {
	n := 1 + b.next()%5
	var cs []steady.Candidate
	for i, k := 0, b.next()%7; i < k; i++ {
		a := steady.Activity{Starts: 1 + int64(b.next()%3), Duration: 1}
		a.FactorSum = a.Starts * int64(b.next()%4) * 5000
		for j, m := 0, b.next()%3; j < m; j++ {
			a.Inputs = append(a.Inputs, steady.Item{Commodity: b.next() % n, Qty: 1 + int64(b.next()%5)})
		}
		for j, m := 0, 1+b.next()%2; j < m; j++ {
			a.Outputs = append(a.Outputs, steady.Item{Commodity: b.next() % n, Qty: 1 + int64(b.next()%5), Discard: b.next()%4 == 0})
		}
		if b.next()%4 == 0 {
			a.Tools = append(a.Tools, steady.Tool{Commodity: b.next() % n, Wear: 1 + int64(b.next()%3), Durability: 1 + int64(b.next()%4)})
		}
		cs = append(cs, steady.Candidate{Recipe: b.next() % 4, Activity: a})
	}
	return n, cs
}

// year decodes up to 4 runs.
func (b *fuzzBytes) year() []steady.Run {
	var y []steady.Run
	for i, k := 0, b.next()%5; i < k; i++ {
		y = append(y, steady.Run{Length: int64(b.next() % 40), FactorBP: int64(b.next()%3) * 5000})
	}
	return y
}

// runChecks runs every Phase 4 check on one decoded input and returns a
// comparable digest and a description of each violated property.
func runChecks(data []byte) (digest []any, problems []string) {
	b := &fuzzBytes{data: data}
	n, cs := b.graph()
	acts := make([]steady.Activity, len(cs))
	var makers []steady.Maker
	for i, c := range cs {
		acts[i] = c.Activity
		m := steady.Maker{Region: 0, Listed: i%2 == 0}
		for _, in := range c.Inputs {
			m.Needs = append(m.Needs, in.Commodity)
		}
		for _, o := range c.Outputs {
			m.Makes = append(m.Makes, o.Commodity)
		}
		makers = append(makers, m)
	}
	realm := steady.Realm{Commodities: n, Regions: 1, Makers: makers}
	digest = append(digest, steady.Producible(realm), steady.Unproducible(realm, []int{0}))
	for _, comp := range steady.Components(steady.MaterialGraph(n, acts)) {
		digest = append(digest, comp)
	}
	cyc, err := steady.Cycles(context.Background(), n, cs, 1<<16, 1000)
	if err != nil {
		problems = append(problems, "Cycles error: "+err.Error())
	}
	digest = append(digest, cyc)
	for _, c := range cyc {
		if gs, ok := steady.CycleGains(c.Commodities, cs, 500); ok {
			digest = append(digest, gs)
		}
	}
	y := b.year()
	var total int64
	for _, r := range y {
		total += r.Length
	}
	d := 1 + int64(b.next()%7)
	gap := steady.Gap(total, []steady.Producer{steady.NewProducer(d, y)})
	if gap < 0 || (total > 0 && gap > total) {
		problems = append(problems, "gap out of [0, year]")
	}
	res := steady.Reserve(total, steady.Frac{Num: big.NewInt(int64(b.next() % 9)), Den: big.NewInt(1)},
		[]steady.Source{{Duration: d, Flow: steady.Frac{Num: big.NewInt(int64(b.next() % 4)), Den: big.NewInt(1)}, Yield: int64(b.next() % 9), Year: y}})
	if res.Num.Sign() < 0 || res.Den.Sign() <= 0 {
		problems = append(problems, "negative reserve")
	}
	return append(digest, gap, res.Num.String(), res.Den.String()), problems
}

// FuzzChecks verifies that the Phase 4 checks never panic on a small random
// graph and year, that their results are deterministic (the same input twice
// gives deeply equal results) and that the gap and the reserve stay in range.
//
// Requirement: FR-VIA-004, FR-VIA-006, FR-VIA-008, NFR-VIA-004
func FuzzChecks(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{1, 1, 0, 3, 1, 1, 0, 0, 1, 1, 0, 0, 0, 0, 2, 2, 20, 1, 20, 2, 3, 4})
	f.Add([]byte{4, 5, 1, 2, 3, 1, 1, 2, 3, 0, 0, 1, 1, 1, 2, 1, 1, 2, 1, 1, 1, 0, 2, 1, 1, 1, 1, 0, 3, 2, 30, 1, 5, 2, 7, 7, 3})
	f.Fuzz(func(t *testing.T, data []byte) {
		first, problems := runChecks(data)
		second, _ := runChecks(data)
		if !reflect.DeepEqual(first, second) {
			problems = append(problems, "two runs differ")
		}
		for _, p := range problems {
			t.Errorf("runChecks(%v): %s", data, p)
		}
	})
}
