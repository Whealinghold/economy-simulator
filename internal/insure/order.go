package insure

import (
	"cmp"
	"slices"
)

// Request is one admitted request of a round: the limit of a lane request is its
// highest loading, that of a store request its highest store rate, both in basis
// points (spec-insurance.md 7.3.1). The caller sets Ref to the index of the request in
// its own slice.
type Request struct {
	// Ref is the caller's index of the request, returned in a Fill.
	Ref int
	// Party is the insured's ID: the tie-break after Key and the guard that an offer
	// is never matched with the request of its own party.
	Party int64
	// Limit is the highest loading or store rate the request pays, in basis points.
	Limit int64
	// MinCover is the lowest cover share the request accepts, in basis points.
	MinCover int64
	// Key is the stateless tie-break key K(t, party), smaller first.
	Key uint64
	// Seq is the position of the request in its party's plan, the last tie-break.
	Seq int
}

// Offer is one admitted offer of a round. Room is what the underwriter may still
// take on as maximum payout; Match lowers it in place. The same offers are held in
// two orders, by Loading for the lane requests and by StoreRate for the store
// requests, so the caller sorts one slice with SortLoadings, matches the lane
// requests, sorts it again with SortStoreRates and matches the store requests: Room
// carries over between the two passes.
type Offer struct {
	// Ref is the caller's index of the offer, returned in a Fill.
	Ref int
	// Party is the underwriter's ID, the tie-break after Key.
	Party int64
	// Loading is the loading of a route cover, in basis points of the expected payout.
	Loading int64
	// StoreRate is the rate of a store cover, in basis points of the sum.
	StoreRate int64
	// CoverBP is the share of a loss the underwriter pays, in basis points.
	CoverBP int64
	// Room is the remaining maximum liability.
	Room int64
	// Key is the stateless tie-break key K(t, party), smaller first.
	Key uint64
}

// SortRequests orders requests by limit descending, equal limits by Key ascending,
// then by Party and by Seq (spec-insurance.md 7.3.1, FR-INS-033). Rows equal in all
// four, which the round never builds, end in Ref order so the order is total
// (architecture.md 7.6).
func SortRequests(requests []Request) {
	slices.SortFunc(requests, func(a, b Request) int {
		return cmp.Or(cmp.Compare(b.Limit, a.Limit), cmp.Compare(a.Key, b.Key), cmp.Compare(a.Party, b.Party), cmp.Compare(a.Seq, b.Seq), cmp.Compare(a.Ref, b.Ref))
	})
}

// SortLoadings orders offers by Loading ascending, equal loadings by Key ascending,
// then by Party and by Ref (spec-insurance.md 7.3.1, FR-INS-033): the order the lane
// requests are matched in.
func SortLoadings(offers []Offer) {
	slices.SortFunc(offers, func(a, b Offer) int {
		return cmp.Or(cmp.Compare(a.Loading, b.Loading), cmp.Compare(a.Key, b.Key), cmp.Compare(a.Party, b.Party), cmp.Compare(a.Ref, b.Ref))
	})
}

// SortStoreRates orders offers by StoreRate ascending, equal rates by Key ascending,
// then by Party and by Ref (spec-insurance.md 7.3.1, FR-INS-033): the order the store
// requests are matched in.
func SortStoreRates(offers []Offer) {
	slices.SortFunc(offers, func(a, b Offer) int {
		return cmp.Or(cmp.Compare(a.StoreRate, b.StoreRate), cmp.Compare(a.Key, b.Key), cmp.Compare(a.Party, b.Party), cmp.Compare(a.Ref, b.Ref))
	})
}
