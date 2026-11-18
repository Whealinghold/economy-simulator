package employment_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/employment"
)

// build makes a scenario of n offers and n vacancies from fuzz bytes. Refs are
// distinct and keys come from the bytes, so ties are common.
func build(data []byte) ([]employment.Offer, []employment.Vacancy) {
	var offers []employment.Offer
	var vacancies []employment.Vacancy
	for i := 0; i+3 < len(data) && i < 64; i += 4 {
		offers = append(offers, employment.Offer{Ref: i, Households: int64(data[i]%4) + 1, Reserve: int64(data[i+1] % 12), Key: uint64(data[i+2] % 3)})
		vacancies = append(vacancies, employment.Vacancy{Ref: 1000 + i, Positions: int64(data[i+3]%6) + 1, Bid: int64(data[i+2] % 16), Duration: 2, ShareBP: int64(data[i]) * 40, Key: uint64(data[i+1] % 3)})
	}
	return offers, vacancies
}

// FuzzRound verifies the invariants of a round over random lanes: every offer
// is hired at most once, a vacancy's positions never go negative and are
// conserved, each wage lies in [reserve, bid], and the result is the same
// whatever the order the offers and vacancies were passed in.
//
// Requirement: FR-EMP-019, NFR-EMP-003
func FuzzRound(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12})
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	f.Add([]byte{255, 255, 255, 255, 3, 1, 4, 1, 5, 9, 2, 6})
	f.Fuzz(func(t *testing.T, data []byte) {
		offers, vacancies := build(data)
		lane := employment.Lane{Open: 0, MaxOpen: int64(len(data)%5) + 1}
		check := func(o employment.Offer, v employment.Vacancy, wage int64, formed []employment.Hire) (int, employment.Close) {
			if (o.Ref+v.Ref)%7 == 0 {
				return employment.ReasonCheck, employment.CloseFit
			}
			return employment.ReasonNone, employment.CloseFit
		}
		var a employment.Result
		employment.Round(&a, slices.Clone(offers), slices.Clone(vacancies), lane, check)

		seen := map[int]bool{}
		for _, h := range a.Hires {
			if seen[h.Offer] {
				t.Fatalf("offer %d hired twice", h.Offer)
			}
			seen[h.Offer] = true
			if h.Wage < h.Reserve || h.Wage > h.Bid || h.Wage < 1 {
				t.Fatalf("hire %+v: wage outside [reserve, bid] or below 1", h)
			}
		}
		if int64(len(a.Hires)) > lane.MaxOpen {
			t.Fatalf("%d hires formed under MaxOpen %d", len(a.Hires), lane.MaxOpen)
		}
		positions := map[int]int64{}
		for _, v := range vacancies {
			positions[v.Ref] = v.Positions
		}
		for _, v := range a.Vacancies {
			if v.Left < 0 || v.Filled+v.Left != positions[v.Ref] {
				t.Fatalf("vacancy %+v: left negative or positions not conserved (%d)", v, positions[v.Ref])
			}
		}

		slices.Reverse(offers)
		slices.Reverse(vacancies)
		var b employment.Result
		employment.Round(&b, offers, vacancies, lane, check)
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("result depends on the input order:\n%+v\n%+v", a, b)
		}
	})
}

// BenchmarkRound reports the cost of one round of a lane of 200 offers and 50
// vacancies, with the result reused (NFR-EMP-004).
func BenchmarkRound(b *testing.B) {
	offers := make([]employment.Offer, 200)
	vacancies := make([]employment.Vacancy, 50)
	var r employment.Result
	b.ReportAllocs()
	for b.Loop() {
		for i := range offers {
			offers[i] = employment.Offer{Ref: i, Households: 1, Reserve: int64(i % 20), Key: uint64(i * 7919)}
		}
		for i := range vacancies {
			vacancies[i] = employment.Vacancy{Ref: 1000 + i, Positions: 4, Bid: int64(5 + i%30), Duration: 3, ShareBP: 5000, Key: uint64(i * 104729)}
		}
		employment.Round(&r, offers, vacancies, employment.Lane{MaxOpen: 1000}, nil)
	}
}

// BenchmarkOrder reports the cost of ordering 1 000 offers and 1 000
// vacancies (NFR-EMP-004).
func BenchmarkOrder(b *testing.B) {
	offers := make([]employment.Offer, 1000)
	vacancies := make([]employment.Vacancy, 1000)
	b.ReportAllocs()
	for b.Loop() {
		for i := range offers {
			offers[i] = employment.Offer{Ref: i, Reserve: int64((i * 31) % 97), Key: uint64(i * 7919)}
			vacancies[i] = employment.Vacancy{Ref: i, Bid: int64((i * 17) % 89), Key: uint64(i * 104729)}
		}
		employment.SortOffers(offers)
		employment.SortVacancies(vacancies)
	}
}
