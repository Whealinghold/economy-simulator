package tenure_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/tenure"
)

// FuzzTiers verifies Tiers never panics on up to eight holders and eight
// requests built from the input, that the units taken never exceed the stock,
// that a grant never exceeds its request, that the let units paid for equal the
// rent times the units let, and that the reversed listing gives the same result.
//
// Requirement: NFR-LND-003, FR-LND-011
func FuzzTiers(f *testing.F) {
	f.Add(int64(100), int64(3), false, []byte{5, 5, 5, 3, 1, 9, 2, 2, 1, 4, 3, 1})
	f.Add(int64(7), int64(0), true, []byte{1, 2, 3, 4, 5, 6})
	f.Add(int64(0), int64(1), false, []byte{})
	f.Fuzz(func(t *testing.T, stock, rent int64, closed bool, raw []byte) {
		if stock < 0 || stock > 1<<20 || rent < 0 || rent > 1<<20 {
			return
		}
		p, holders, reqs := fuzzEndowment(stock, rent, closed, raw)
		var s tenure.Scratch
		res, ok := s.Tiers(p, slices.Clone(holders), reqs)
		if !ok {
			return
		}
		checkTiers(t, p, res, reqs)
		rev := slices.Clone(reqs)
		slices.Reverse(rev)
		again, ok := s.Tiers(p, slices.Clone(holders), rev)
		if !ok || again != res || !slices.Equal(byRef(rev), byRef(reqs)) {
			t.Fatalf("Tiers(%+v) reversed gave %+v, want %+v", p, again, res)
		}
	})
}

// checkTiers asserts the grants and the unit counts of a Tiers result.
func checkTiers(t *testing.T, p tenure.Params, res tenure.Result, reqs []tenure.Request) {
	t.Helper()
	var units int64
	for _, r := range reqs {
		if !grantOK(r) {
			t.Fatalf("Tiers(%+v) granted %+v", p, r)
		}
		units += (r.Own + r.Commons + r.Let) * r.Draw
	}
	if units > p.Stock || units != res.OwnUnits+res.CommonsUnits+res.LetUnits || res.LetUnits > res.Supply {
		t.Fatalf("Tiers(%+v) = %+v, units %d", p, res, units)
	}
	if rentPaid := res.LetUnits * p.Rent; rentPaid < 0 || (p.Rent > 0 && rentPaid/p.Rent != res.LetUnits) {
		t.Fatalf("Tiers(%+v) = %+v, rent paid %d is not rent times units let", p, res, rentPaid)
	}
}

// grantOK reports whether a request's grants are within its request.
func grantOK(r tenure.Request) bool {
	return r.Own >= 0 && r.Commons >= 0 && r.Let >= 0 && r.Own+r.Commons+r.Let <= r.Executions
}

// fuzzEndowment builds an endowment from the bytes: the first part of the bytes
// are holder entitlements (a quarter of the stock at most each), the rest are
// requests of four bytes each (executions, draw, key, balance).
func fuzzEndowment(stock, rent int64, closed bool, raw []byte) (tenure.Params, []tenure.Holder, []tenure.Request) {
	var holders []tenure.Holder
	left := stock
	for i := 0; i < len(raw) && i < 8 && i < len(raw)/3; i++ {
		e := min(int64(raw[i]), left)
		holders = append(holders, tenure.Holder{ID: int64(i + 1), Entitled: e})
		left -= e
	}
	var reqs []tenure.Request
	for i := len(holders); i+3 < len(raw) && len(reqs) < 8; i += 4 {
		id := int64(len(reqs) + 1)
		reqs = append(reqs, tenure.Request{
			Ref: len(reqs), ID: id, Principal: int64(raw[i]%10) + 1, Executions: int64(raw[i+1] % 12), Draw: int64(raw[i+2]%4) + 1,
			Key: uint64(raw[i+3] % 3), Balance: int64(raw[i+3]), Exempt: raw[i]%7 == 0,
		})
	}
	return tenure.Params{Stock: stock, Commons: left, Rent: rent, Closed: closed}, holders, reqs
}

// FuzzApportion verifies Apportion never panics, sums to the stock whenever it
// reports ok with a positive weight, gives each part within one unit of its exact
// share, and gives the same result for the reversed listing.
//
// Requirement: NFR-LND-003, FR-LND-007
func FuzzApportion(f *testing.F) {
	f.Add(int64(80), []byte{100, 100, 100, 50})
	f.Add(int64(3), []byte{25, 25, 25, 25})
	f.Add(int64(0), []byte{})
	f.Fuzz(func(t *testing.T, stock int64, raw []byte) {
		if stock < 0 || stock > 1<<40 || len(raw) > 16 {
			return
		}
		parts := make([]tenure.Part, len(raw))
		var weight int64
		for i, b := range raw {
			parts[i] = tenure.Part{Ref: i, ID: int64(i + 1), Share: int64(b), Key: uint64(b % 3), Commons: i == 0}
			weight += int64(b)
		}
		out := make([]int64, len(parts))
		if !tenure.Apportion(stock, slices.Clone(parts), out) {
			if weight != 0 && stock >= 0 {
				t.Fatalf("Apportion(%d, %+v) refused a valid input", stock, parts)
			}
			return
		}
		checkApportion(t, stock, parts, out, weight)
		rev := slices.Clone(parts)
		slices.Reverse(rev)
		again := make([]int64, len(parts))
		tenure.Apportion(stock, rev, again)
		if !slices.Equal(out, again) {
			t.Fatalf("Apportion(%d, %+v) = %v, reversed %v", stock, parts, out, again)
		}
	})
}

// checkApportion asserts the units sum to the stock and each part is within one
// unit of its exact share.
func checkApportion(t *testing.T, stock int64, parts []tenure.Part, out []int64, weight int64) {
	t.Helper()
	if sum(out) != stock {
		t.Fatalf("Apportion(%d, %+v) = %v, sums to %d", stock, parts, out, sum(out))
	}
	if weight == 0 {
		return
	}
	for i, p := range parts {
		if exact := stock * p.Share / weight; out[i] < exact || out[i] > exact+1 {
			t.Fatalf("Apportion(%d, %+v) = %v, part %d is %d, exact %d", stock, parts, out, i, out[i], exact)
		}
	}
}

// FuzzStep verifies Step never panics, moves the posted rent by at most the step
// whenever it started inside the band, never moves it with a StepBP of 0 or a
// closed endowment, and returns the posted rent when it refuses. The bits of
// flags are closed, unserved above 0, idle above 0 and cut above 0.
//
// Requirement: NFR-LND-003, FR-LND-019
func FuzzStep(f *testing.F) {
	f.Add(int64(40), int64(40), int64(250), int64(5_000), int64(20_000), uint8(0b0010))
	f.Add(int64(80), int64(40), int64(0), int64(0), int64(0), uint8(0b1111))
	f.Fuzz(func(t *testing.T, posted, base, stepBP, floorBP, ceilBP int64, flags uint8) {
		in := tenure.StepIn{
			Posted: posted, Base: base, StepBP: stepBP, FloorBP: floorBP, CeilBP: ceilBP, Closed: flags&1 != 0,
			Unserved: int64(flags >> 1 & 1), Idle: int64(flags >> 2 & 1), Cut: int64(flags >> 3 & 1),
		}
		closed := in.Closed
		got, ok := tenure.Step(in)
		if !ok {
			if got != posted {
				t.Fatalf("Step(%+v) refused and returned %d, want the posted rent", in, got)
			}
			return
		}
		if (stepBP == 0 || closed) && got != posted {
			t.Fatalf("Step(%+v) = %d, moved a rent that must not move", in, got)
		}
		if got < 0 || movedPastStep(in, got) {
			t.Fatalf("Step(%+v) = %d, negative or moved by more than the step", in, got)
		}
	})
}

// movedPastStep reports whether a posted rent that started inside the band moved
// by more than the step. Inputs too large for the plain arithmetic are not judged.
func movedPastStep(in tenure.StepIn, got int64) bool {
	if in.Base >= 1<<30 || in.StepBP >= 1<<20 || in.FloorBP >= 1<<20 || in.CeilBP >= 1<<20 {
		return false
	}
	if in.Posted < in.Base*in.FloorBP/10_000 || in.Posted > in.Base*in.CeilBP/10_000 {
		return false
	}
	step := max(1, in.Base*in.StepBP/10_000)
	return got-in.Posted > step || in.Posted-got > step
}

// FuzzAttribute verifies the supply and the attribution never panic and, for a
// let supply and tenants whose units fit it, give triples that sum per tenant to
// the units it took and per lessor to the supply it was apportioned, whatever
// the listing.
//
// Requirement: NFR-LND-003, FR-LND-013
func FuzzAttribute(f *testing.F) {
	f.Add([]byte{3, 2}, []byte{6, 2})
	f.Add([]byte{}, []byte{})
	f.Fuzz(func(t *testing.T, takes, unused []byte) {
		if len(takes) > 12 || len(unused) > 12 {
			return
		}
		var taken, capacity int64
		tenants := make([]tenure.Taker, len(takes))
		for i, b := range takes {
			tenants[i] = tenure.Taker{ID: int64(i + 1), Units: int64(b)}
			taken += int64(b)
		}
		parts := make([]tenure.Part, len(unused))
		for i, b := range unused {
			parts[i] = tenure.Part{Ref: i, ID: int64(i + 1), Share: int64(b), Key: uint64(b % 4)}
			capacity += int64(b)
		}
		supply := make([]int64, len(parts))
		if !tenure.Supply(taken, parts, supply) {
			if taken <= capacity {
				t.Fatalf("Supply(%d, %+v) refused units within the let supply %d", taken, parts, capacity)
			}
			return
		}
		lessors := make([]tenure.Taker, len(parts))
		for i, p := range parts {
			lessors[i] = tenure.Taker{ID: p.ID, Units: supply[i]}
		}
		triples, ok := new(tenure.Scratch).Attribute(nil, tenants, lessors)
		if !ok {
			t.Fatalf("Attribute(%+v, %+v) refused matching sums", tenants, lessors)
		}
		checkAttribute(t, triples, tenants, lessors)
	})
}

// checkAttribute asserts the triples sum per tenant to its units and per lessor
// to its supply, and that Self marks the pairs of one agent.
func checkAttribute(t *testing.T, triples []tenure.Triple, tenants, lessors []tenure.Taker) {
	t.Helper()
	gotT, gotL := make([]int64, len(tenants)), make([]int64, len(lessors))
	for _, tr := range triples {
		gotT[tr.Tenant] += tr.Units
		gotL[tr.Lessor] += tr.Units
		if tr.Self != (tenants[tr.Tenant].ID == lessors[tr.Lessor].ID) {
			t.Fatalf("Attribute triple %+v has the wrong Self", tr)
		}
	}
	for i := range tenants {
		if gotT[i] != tenants[i].Units {
			t.Fatalf("Attribute gave tenant %d %d units, took %d", i, gotT[i], tenants[i].Units)
		}
	}
	for i := range lessors {
		if gotL[i] != lessors[i].Units {
			t.Fatalf("Attribute took %d from lessor %d, supply %d", gotL[i], i, lessors[i].Units)
		}
	}
}
