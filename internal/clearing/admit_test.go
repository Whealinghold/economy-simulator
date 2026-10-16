package clearing_test

import (
	"encoding/binary"
	"math"
	"math/big"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/clearing"
)

const (
	maxQty   = 1_000_000_000             // MaxQuantity of bounds.go
	maxMoney = 1_000_000_000_000_000_000 // MaxMoney of bounds.go
)

// okBid and okAsk are facts that pass every row.
func okBid() clearing.Facts {
	return clearing.Facts{
		AgentKnown: true, SideAllowed: true, CommodityKnown: true, Tradable: true,
		Bid: true, Price: 10, Qty: 5, MaxPrice: maxPrice, MaxQuantity: maxQty, MaxMoney: maxMoney,
		Available: 50, Held: 0,
	}
}

func okAsk() clearing.Facts {
	f := okBid()
	f.Bid, f.Available, f.Balance = false, 5, 0
	return f
}

// TestAdmit_Rows breaks exactly one row of CLR 3.2 per case and asserts the
// code, then two rows at once and asserts the first of the table.
//
// Requirement: FR-CLR-002, FR-CLR-003, FR-CLR-004, FR-CLR-005, FR-CLR-006, FR-CLR-028, FR-CLR-030, FR-CLR-031
func TestAdmit_Rows(t *testing.T) {
	t.Parallel()
	mut := func(base clearing.Facts, f func(*clearing.Facts)) clearing.Facts { f(&base); return base }
	tests := []struct {
		name string
		in   clearing.Facts
		want clearing.Code
	}{
		{"bid admitted", okBid(), clearing.Admitted},
		{"ask admitted", okAsk(), clearing.Admitted},
		{"unknown agent", mut(okBid(), func(f *clearing.Facts) { f.AgentKnown = false }), clearing.UnknownAgent},
		{"side not allowed", mut(okBid(), func(f *clearing.Facts) { f.SideAllowed = false }), clearing.SideNotAllowed},
		{"unknown commodity", mut(okBid(), func(f *clearing.Facts) { f.CommodityKnown = false }), clearing.UnknownCommodity},
		{"not tradable", mut(okBid(), func(f *clearing.Facts) { f.Tradable = false }), clearing.NotTradable},
		{"price 0", mut(okBid(), func(f *clearing.Facts) { f.Price = 0 }), clearing.InvalidPrice},
		{"price 1", mut(okBid(), func(f *clearing.Facts) { f.Price = 1 }), clearing.Admitted},
		{"price max", mut(okAsk(), func(f *clearing.Facts) { f.Price = maxPrice }), clearing.Admitted},
		{"price max+1", mut(okBid(), func(f *clearing.Facts) { f.Price = maxPrice + 1 }), clearing.InvalidPrice},
		{"quantity 0", mut(okBid(), func(f *clearing.Facts) { f.Qty = 0 }), clearing.InvalidQuantity},
		{"quantity 1", mut(okBid(), func(f *clearing.Facts) { f.Qty = 1 }), clearing.Admitted},
		{"quantity max", mut(okAsk(), func(f *clearing.Facts) { f.Qty, f.Available = maxQty, maxQty }), clearing.Admitted},
		{"quantity max+1", mut(okBid(), func(f *clearing.Facts) { f.Qty = maxQty + 1 }), clearing.InvalidQuantity},
		{"duplicate", mut(okBid(), func(f *clearing.Facts) { f.Duplicate = true }), clearing.DuplicateOrder},
		// A bid and an ask of one agent are two keys: the caller sets Duplicate
		// per (owner, lot, side), so the second side of one agent is admitted.
		{"other side of an owner is not a duplicate", okAsk(), clearing.Admitted},
		// 10^9 * 10^9 = 10^18 = MaxMoney exactly, funds suffice.
		{"product exactly MaxMoney", mut(okBid(), func(f *clearing.Facts) {
			f.Price, f.Qty, f.Available = maxPrice, maxQty, maxMoney
		}), clearing.Admitted},
		// Bounds raised so that the int64 product overflows: 4e9 * 4e9 = 1.6e19 > 9.2e18.
		{"product overflows int64", mut(okBid(), func(f *clearing.Facts) {
			f.MaxPrice, f.MaxQuantity, f.Price, f.Qty = 4e9, 4e9, 4e9, 4e9
		}), clearing.Overflow},
		{"insufficient funds", mut(okBid(), func(f *clearing.Facts) { f.Available = 49 }), clearing.InsufficientFunds},
		{"funds exactly", mut(okBid(), func(f *clearing.Facts) { f.Available = 50 }), clearing.Admitted},
		{"insufficient stock", mut(okAsk(), func(f *clearing.Facts) { f.Available = 4 }), clearing.InsufficientStock},
		// D34 bid: held + qty exactly MaxQuantity is admitted, one more is not.
		{"bid holding at the edge", mut(okBid(), func(f *clearing.Facts) { f.Held = maxQty - 5 }), clearing.Admitted},
		{"bid holding one over", mut(okBid(), func(f *clearing.Facts) { f.Held = maxQty - 4 }), clearing.HoldingLimit},
		{"bid holding above max", mut(okBid(), func(f *clearing.Facts) { f.Held = maxQty + 1 }), clearing.HoldingLimit},
		// D34 ask: worst receipt 5 * 1e9 = 5e9; balance + tally + 5e9 == MaxMoney is admitted.
		{"ask headroom exactly", mut(okAsk(), func(f *clearing.Facts) {
			f.Balance, f.Tally = maxMoney-5*maxPrice-3, 3
		}), clearing.Admitted},
		{"ask headroom one short", mut(okAsk(), func(f *clearing.Facts) {
			f.Balance, f.Tally = maxMoney-5*maxPrice-2, 3
		}), clearing.ReceiptLimit},
		{"ask tally beyond int64", mut(okAsk(), func(f *clearing.Facts) {
			f.Balance, f.Tally = math.MaxInt64, math.MaxInt64
		}), clearing.ReceiptLimit},
		{"ask balance above MaxMoney", mut(okAsk(), func(f *clearing.Facts) { f.Balance = maxMoney + 1 }), clearing.ReceiptLimit},
		{"first of two failures", mut(okBid(), func(f *clearing.Facts) { f.Price, f.Qty, f.Available = 0, 0, 0 }), clearing.InvalidPrice},
		{"price beats quantity", mut(okBid(), func(f *clearing.Facts) { f.Price, f.Qty = 0, 0 }), clearing.InvalidPrice},
		{"duplicate with bad price", mut(okBid(), func(f *clearing.Facts) { f.Duplicate, f.Price = true, 0 }), clearing.InvalidPrice},
		{"duplicate with bad quantity", mut(okBid(), func(f *clearing.Facts) { f.Duplicate, f.Qty = true, 0 }), clearing.InvalidQuantity},
		{"not tradable beats price", mut(okBid(), func(f *clearing.Facts) { f.Tradable, f.Price = false, 0 }), clearing.NotTradable},
		{"duplicate beats overflow and funds", mut(okBid(), func(f *clearing.Facts) {
			f.Duplicate, f.Held, f.Available = true, maxQty, 0
		}), clearing.DuplicateOrder},
		{"holding limit beats funds", mut(okBid(), func(f *clearing.Facts) { f.Held, f.Available = maxQty, 0 }), clearing.HoldingLimit},
		// A negative fact is the root's error and is refused as overflow, never
		// wrapped into a sum (the root panics before it gets here).
		{"negative holding", mut(okBid(), func(f *clearing.Facts) { f.Held = -1 }), clearing.Overflow},
		{"negative balance", mut(okAsk(), func(f *clearing.Facts) { f.Balance = -1 }), clearing.Overflow},
		{"negative tally", mut(okAsk(), func(f *clearing.Facts) { f.Tally = -1 }), clearing.Overflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := clearing.Admit(tt.in); got != tt.want {
				t.Errorf("Admit(%+v) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

// refAdmit restates CLR 3.2 with math/big, row by row, independent of the
// package's checked arithmetic.
func refAdmit(f clearing.Facts) clearing.Code {
	bi := big.NewInt
	prod := new(big.Int).Mul(bi(f.Price), bi(f.Qty))
	fits := prod.IsInt64()
	headroom := refHeadroom(f)
	if c := refIdentity(f); c != clearing.Admitted {
		return c
	}
	switch {
	case f.Price < 1 || f.Price > f.MaxPrice:
		return clearing.InvalidPrice
	case f.Qty < 1 || f.Qty > f.MaxQuantity:
		return clearing.InvalidQuantity
	case f.Duplicate:
		return clearing.DuplicateOrder
	case !fits:
		return clearing.Overflow
	case headroom != clearing.Admitted:
		return headroom
	case f.Bid && prod.Cmp(bi(f.Available)) > 0:
		return clearing.InsufficientFunds
	case !f.Bid && f.Qty > f.Available:
		return clearing.InsufficientStock
	}
	return clearing.Admitted
}

// refIdentity is the first four rows of CLR 3.2.
func refIdentity(f clearing.Facts) clearing.Code {
	switch {
	case !f.AgentKnown:
		return clearing.UnknownAgent
	case !f.SideAllowed:
		return clearing.SideNotAllowed
	case !f.CommodityKnown:
		return clearing.UnknownCommodity
	case !f.Tradable:
		return clearing.NotTradable
	}
	return clearing.Admitted
}

// refHeadroom is the D34 rule with math/big and the code of the bound it
// breaks; a negative fact is refused as Overflow, never wrapped.
func refHeadroom(f clearing.Facts) clearing.Code {
	bi := big.NewInt
	switch {
	case f.Held < 0 || f.Balance < 0 || f.Tally < 0:
		return clearing.Overflow
	case f.Bid:
		if bi(f.Held).Add(bi(f.Held), bi(f.Qty)).Cmp(bi(f.MaxQuantity)) > 0 {
			return clearing.HoldingLimit
		}
		return clearing.Admitted
	}
	sum := new(big.Int).Add(bi(f.Balance), bi(f.Tally))
	sum.Add(sum, new(big.Int).Mul(bi(f.Qty), bi(f.MaxPrice)))
	if sum.Cmp(bi(f.MaxMoney)) > 0 {
		return clearing.ReceiptLimit
	}
	return clearing.Admitted
}

// decodeFacts reads Facts from bytes, padding with zeros.
func decodeFacts(b []byte) clearing.Facts {
	buf := make([]byte, 2+9*8)
	copy(buf, b)
	n := func(i int) int64 { return int64(binary.LittleEndian.Uint64(buf[2+i*8:])) }
	return clearing.Facts{
		AgentKnown: buf[0]&1 != 0, SideAllowed: buf[0]&2 != 0, CommodityKnown: buf[0]&4 != 0,
		Tradable: buf[0]&8 != 0, Duplicate: buf[0]&16 != 0, Bid: buf[0]&32 != 0,
		Price: n(0), Qty: n(1), MaxPrice: n(2), MaxQuantity: n(3), MaxMoney: n(4),
		Available: n(5), Held: n(6), Balance: n(7), Tally: n(8),
	}
}

func encodeFacts(f clearing.Facts) []byte {
	b := make([]byte, 2+9*8)
	for i, v := range []bool{f.AgentKnown, f.SideAllowed, f.CommodityKnown, f.Tradable, f.Duplicate, f.Bid} {
		if v {
			b[0] |= 1 << i
		}
	}
	for i, v := range []int64{f.Price, f.Qty, f.MaxPrice, f.MaxQuantity, f.MaxMoney, f.Available, f.Held, f.Balance, f.Tally} {
		binary.LittleEndian.PutUint64(b[2+i*8:], uint64(v))
	}
	return b
}

// FuzzAdmit decodes arbitrary Facts and asserts that Admit never panics and
// agrees with an independent math/big restatement of CLR 3.2: admitted only
// when every row holds, otherwise the first failing row of the table.
//
// Requirement: FR-CLR-002, FR-CLR-003, FR-CLR-004, FR-CLR-005, FR-CLR-006, FR-CLR-007, FR-CLR-028, FR-CLR-030, FR-CLR-031
//
// Fix: the corpus files testdata/fuzz/FuzzAdmit/seed-* pin three hand-written
// edges, not crashes (the same inputs are given with f.Add): seed-bid-edge, a
// bid whose holding plus quantity is exactly MaxQuantity; seed-ask-tally-max,
// an ask whose balance and tally are MaxInt64 and must be refused, not
// wrapped; seed-int64-product, a price times quantity beyond int64.
func FuzzAdmit(f *testing.F) {
	f.Add(encodeFacts(okBid()))
	f.Add(encodeFacts(okAsk()))
	edge := okBid()
	edge.Held = maxQty - 5
	f.Add(encodeFacts(edge))
	edge = okAsk()
	edge.Balance, edge.Tally = math.MaxInt64, math.MaxInt64
	f.Add(encodeFacts(edge))
	edge = okBid()
	edge.MaxPrice, edge.MaxQuantity, edge.Price, edge.Qty = math.MaxInt64, math.MaxInt64, math.MaxInt64, 2
	f.Add(encodeFacts(edge))
	f.Fuzz(func(t *testing.T, data []byte) {
		in := decodeFacts(data)
		if got, want := clearing.Admit(in), refAdmit(in); got != want {
			t.Errorf("Admit(%+v) = %d, want %d", in, got, want)
		}
	})
}

// TestAdmit_Headroom asserts the boundaries of the funds, stock and receipt
// rows: an order with plenty of cash or stock is admitted, and an ask whose
// worst-case receipt does not fit int64, or whose seller's balance and tally
// already fill the headroom, is ReceiptLimit.
//
// Requirement: FR-CLR-004, FR-CLR-005, FR-CLR-028
func TestAdmit_Headroom(t *testing.T) {
	t.Parallel()
	rich := okBid()
	rich.Available = 500 // product 10*5 = 50
	stocked := okAsk()
	stocked.Available = 50 // quantity 5
	wrap := okAsk()
	// 1 * Qty fits, but Qty * MaxPrice = 3 * (2^63-1)/2 does not.
	wrap.Price, wrap.Qty, wrap.MaxPrice, wrap.MaxQuantity = 1, math.MaxInt64/2, 3, math.MaxInt64
	wrap.Available = math.MaxInt64
	full := okAsk()
	full.Balance, full.Tally = math.MaxInt64, math.MaxInt64
	tests := []struct {
		name string
		f    clearing.Facts
		want clearing.Code
	}{
		{"bid with ample funds", rich, clearing.Admitted},
		{"ask with ample stock", stocked, clearing.Admitted},
		{"worst-case receipt overflows int64", wrap, clearing.ReceiptLimit},
		{"balance and tally beyond the headroom", full, clearing.ReceiptLimit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := clearing.Admit(tt.f); got != tt.want {
				t.Errorf("Admit(%+v) = %d, want %d", tt.f, got, tt.want)
			}
		})
	}
}
