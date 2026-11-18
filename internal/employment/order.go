package employment

import (
	"cmp"
	"slices"
)

// Offer is a worker's admitted labour offer in one lane (spec-employment.md
// 4.1): Households of the agent are offered at a Reserve, the least wage per
// household-tick it accepts. Key is the stateless tie key K(t, agent ID) that
// the root computes (spec-world.md 10.1.1); the round takes no random word
// (5.3.1). Ref is the caller's index and the last tie-break, so the order is
// total (architecture.md 7.6).
type Offer struct {
	Ref        int
	Households int64
	Reserve    int64
	Key        uint64
}

// Vacancy is an employer's admitted vacancy in one lane: Positions households
// for a recipe of Duration ticks, at most Bid per household-tick, with the
// ShareBP of the employer's block. Key and Ref are as for an Offer.
type Vacancy struct {
	Ref       int
	Positions int64
	Bid       int64
	Duration  int64
	ShareBP   int64
	Key       uint64
}

// SortOffers sorts offers in place by Reserve ascending, equal reserves by Key
// and then Ref, both ascending (spec-employment.md 5.3.3). The comparison is a
// total order when Refs are distinct, so the result does not depend on the
// order of the input.
func SortOffers(offers []Offer) {
	slices.SortFunc(offers, func(a, b Offer) int {
		return cmp.Or(
			cmp.Compare(a.Reserve, b.Reserve),
			cmp.Compare(a.Key, b.Key),
			cmp.Compare(a.Ref, b.Ref),
		)
	})
}

// SortVacancies sorts vacancies in place by Bid descending, equal bids by Key
// and then Ref, both ascending (spec-employment.md 5.3.3). It is a total order
// when Refs are distinct.
func SortVacancies(vacancies []Vacancy) {
	slices.SortFunc(vacancies, func(a, b Vacancy) int {
		return cmp.Or(
			cmp.Compare(b.Bid, a.Bid),
			cmp.Compare(a.Key, b.Key),
			cmp.Compare(a.Ref, b.Ref),
		)
	})
}
