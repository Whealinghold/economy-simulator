package employment_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/employment"
)

// TestOrder_OffersByReserveThenKeyThenRef verifies the order of the offers of a
// lane: the least reservation first, equal reservations by the stateless key and
// then the reference.
//
// Requirement: FR-EMP-018, FR-EMP-019
func TestOrder_OffersByReserveThenKeyThenRef(t *testing.T) {
	t.Parallel()
	offers := []employment.Offer{
		{Ref: 1, Reserve: 5, Key: 10},
		{Ref: 2, Reserve: 3, Key: 90}, // the least reserve is first whatever the key
		{Ref: 3, Reserve: 5, Key: 2},  // the same reserve as Ref 1: the smaller key first
		{Ref: 4, Reserve: 5, Key: 2},  // the same reserve and key as Ref 3: the smaller Ref first
	}
	employment.SortOffers(offers)
	var got []int
	for _, o := range offers {
		got = append(got, o.Ref)
	}
	if want := []int{2, 3, 4, 1}; !slices.Equal(got, want) {
		t.Errorf("SortOffers order = %v, want %v", got, want)
	}
}

// TestOrder_VacanciesByBidDescThenKeyThenRef verifies the order of the
// vacancies: the highest bid first, equal bids by key and then reference.
//
// Requirement: FR-EMP-018, FR-EMP-019
func TestOrder_VacanciesByBidDescThenKeyThenRef(t *testing.T) {
	t.Parallel()
	vacancies := []employment.Vacancy{
		{Ref: 1, Bid: 7, Key: 10},
		{Ref: 2, Bid: 9, Key: 90}, // the highest bid is first whatever the key
		{Ref: 3, Bid: 7, Key: 2},
		{Ref: 4, Bid: 7, Key: 2},
	}
	employment.SortVacancies(vacancies)
	var got []int
	for _, v := range vacancies {
		got = append(got, v.Ref)
	}
	if want := []int{2, 3, 4, 1}; !slices.Equal(got, want) {
		t.Errorf("SortVacancies order = %v, want %v", got, want)
	}
}

// TestOrder_TotalOrder_NoTies verifies that the order does not depend on the
// order of the input: every permutation of a set with equal reservations, keys
// and bids sorts to the same sequence, because the reference ends the order.
//
// Requirement: FR-EMP-019
//
// Guard: architecture.md 7.6. The sort is not stable; an order with a tie left
// would change with the Go version.
func TestOrder_TotalOrder_NoTies(t *testing.T) {
	t.Parallel()
	base := []employment.Offer{
		{Ref: 4, Reserve: 1, Key: 5}, {Ref: 1, Reserve: 1, Key: 5},
		{Ref: 3, Reserve: 1, Key: 5}, {Ref: 2, Reserve: 1, Key: 5},
	}
	want := []int{1, 2, 3, 4}
	for shift := range base {
		in := append(slices.Clone(base[shift:]), base[:shift]...)
		employment.SortOffers(in)
		var got []int
		for _, o := range in {
			got = append(got, o.Ref)
		}
		if !slices.Equal(got, want) {
			t.Errorf("SortOffers(rotation %d) = %v, want %v", shift, got, want)
		}
	}
	vbase := []employment.Vacancy{{Ref: 3, Bid: 2, Key: 1}, {Ref: 1, Bid: 2, Key: 1}, {Ref: 2, Bid: 2, Key: 1}}
	for shift := range vbase {
		in := append(slices.Clone(vbase[shift:]), vbase[:shift]...)
		employment.SortVacancies(in)
		var got []int
		for _, v := range in {
			got = append(got, v.Ref)
		}
		if want := []int{1, 2, 3}; !slices.Equal(got, want) {
			t.Errorf("SortVacancies(rotation %d) = %v, want %v", shift, got, want)
		}
	}
}
