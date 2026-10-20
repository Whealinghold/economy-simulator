package steady

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"testing"
)

// byteReader hands out the bytes of a fuzz input and then zeros.
type byteReader struct {
	data []byte
	at   int
}

// next returns the next byte, or 0 past the end.
func (b *byteReader) next() byte {
	if b.at >= len(b.data) {
		return 0
	}
	b.at++
	return b.data[b.at-1]
}

// ratio returns a small ratio: a numerator in [lo, lo+10] and a denominator in
// 1..4.
func (b *byteReader) ratio(lo int64) Ratio {
	v := b.next()
	return Ratio{Num: lo + int64(v%11), Den: 1 + int64(v/11%4)}
}

// randomProgramme decodes up to 4 rows and 4 columns of small rationals.
func randomProgramme(b *byteReader) Programme {
	m, n := 1+int(b.next()%4), 1+int(b.next()%4)
	p := Programme{Rows: m}
	for i := 0; i < m; i++ {
		p.RHS = append(p.RHS, b.ratio(0))
	}
	for j := 0; j < n; j++ {
		c := Column{Cost: b.ratio(-5)}
		for i := 0; i < m; i++ {
			if v := b.ratio(-5); v.Num != 0 {
				c.Entries = append(c.Entries, Entry{Row: i, Val: v})
			}
		}
		p.Cols = append(p.Cols, c)
	}
	return p
}

// transportCase decodes a complete 2x2 transport with unit value per unit moved
// and returns the programme and its known optimum, min(supply, demand).
func transportCase(b *byteReader) (Programme, int64) {
	s := [2]int64{int64(b.next() % 10), int64(b.next() % 10)}
	d := [2]int64{int64(b.next() % 10), int64(b.next() % 10)}
	one := Ratio{Num: 1, Den: 1}
	p := Programme{Rows: 4}
	for _, v := range append(s[:], d[:]...) {
		p.RHS = append(p.RHS, Ratio{Num: v, Den: 1})
	}
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			p.Cols = append(p.Cols, Column{Cost: one, Entries: []Entry{{Row: i, Val: one}, {Row: 2 + j, Val: one}}})
		}
	}
	return p, min(s[0]+s[1], d[0]+d[1])
}

// FuzzSolve verifies, over small random programmes and random unit-cost
// transport cases, that the solver never panics, that the only failure it
// reports is an unbounded programme, that every result passes the certificate
// and is the same when solved again, and that a transport case reaches its
// known optimum.
//
// Requirement: NFR-VIA-004, FR-VIA-014, FR-VIA-016, FR-VIA-020
func FuzzSolve(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{3, 3, 4, 9, 3, 10, 2, 8, 7, 5, 3, 1, 6, 9, 2})
	f.Add([]byte{0, 0, 5, 5, 5, 5, 5})
	f.Add([]byte{2, 1, 0, 0, 200, 7, 7, 1, 1})
	f.Add([]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255})
	f.Fuzz(func(t *testing.T, data []byte) {
		p := randomProgramme(&byteReader{data: data})
		res, err := Solve(context.Background(), p, 100_000)
		switch {
		case errors.Is(err, ErrUnbounded):
		case err != nil:
			t.Fatalf("Solve(%v) error = %v, want nil or ErrUnbounded", p, err)
		default:
			if cerr := certify(p, res); cerr != nil {
				t.Fatalf("certify(Solve(%v)) = %v", p, cerr)
			}
			again, _ := Solve(context.Background(), p, 100_000)
			if a, b := fmt.Sprint(res.Primal, res.Dual, res.Basis), fmt.Sprint(again.Primal, again.Dual, again.Basis); a != b {
				t.Fatalf("Solve(%v) twice = %s then %s", p, a, b)
			}
		}

		tp, want := transportCase(&byteReader{data: data})
		tres, err := Solve(context.Background(), tp, 100_000)
		if err != nil {
			t.Fatalf("Solve(transport %v) error = %v", tp.RHS, err)
		}
		if got := new(big.Rat).SetFrac(tres.Objective.Num, tres.Objective.Den); got.Cmp(big.NewRat(want, 1)) != 0 {
			t.Fatalf("Solve(transport %v) = %v, want %d", tp.RHS, got, want)
		}
	})
}

// roundReference computes the three roundings from truncating division, a
// different path from the one under test.
func roundReference(num, den *big.Int) (even, floor, ceil *big.Int) {
	n, d := new(big.Int).Set(num), new(big.Int).Set(den)
	if d.Sign() < 0 {
		n.Neg(n)
		d.Neg(d)
	}
	q, r := new(big.Int).QuoRem(n, d, new(big.Int)) // toward zero
	floor, ceil = new(big.Int).Set(q), new(big.Int).Set(q)
	if r.Sign() < 0 {
		floor.Sub(floor, big.NewInt(1))
	}
	if r.Sign() > 0 {
		ceil.Add(ceil, big.NewInt(1))
	}
	twice := new(big.Int).Abs(r)
	twice.Lsh(twice, 1)
	switch c := twice.Cmp(d); {
	case c < 0:
		even = q
	case c > 0:
		even = floor
		if r.Sign() > 0 {
			even = ceil
		}
	default: // an exact half: the even neighbour
		even = floor
		if floor.Bit(0) == 1 {
			even = ceil
		}
	}
	return even, floor, ceil
}

// FuzzRound verifies RoundHalfEven, Floor and Ceil against a reference built on
// truncating big.Int division, including whether the result fits int64.
//
// Requirement: NFR-VIA-004, FR-VIA-015
func FuzzRound(f *testing.F) {
	f.Add([]byte{5}, []byte{2}, false, false)
	f.Add([]byte{7}, []byte{2}, true, false)
	f.Add([]byte{1, 0, 0, 0, 0, 0, 0, 0, 0}, []byte{1}, false, false)
	f.Add([]byte{255, 255, 255, 255, 255, 255, 255, 127}, []byte{1}, true, true)
	f.Add([]byte{}, []byte{3}, false, false)
	f.Fuzz(func(t *testing.T, numBytes, denBytes []byte, negNum, negDen bool) {
		num, den := new(big.Int).SetBytes(numBytes), new(big.Int).SetBytes(denBytes)
		if den.Sign() == 0 {
			t.Skip("a zero denominator is a programmer invariant")
		}
		if negNum {
			num.Neg(num)
		}
		if negDen {
			den.Neg(den)
		}
		even, floor, ceil := roundReference(num, den)
		for name, c := range map[string]struct {
			f    func(a, b *big.Int, lo, hi int64) (int64, bool)
			want *big.Int
		}{"RoundHalfEven": {RoundHalfEven, even}, "Floor": {Floor, floor}, "Ceil": {Ceil, ceil}} {
			got, ok := c.f(num, den, math.MinInt64, math.MaxInt64)
			if ok != c.want.IsInt64() || (ok && got != c.want.Int64()) || (!ok && got != 0) {
				t.Fatalf("%s(%s, %s) = %d, %v, want %s", name, num, den, got, ok, c.want)
			}
		}
	})
}
