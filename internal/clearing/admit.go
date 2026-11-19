package clearing

import (
	"github.com/Whealinghold/economy-sim/internal/checked"
	"github.com/Whealinghold/economy-sim/internal/institution"
)

// Code is a small integer the root maps to a rejection reason (Admit) or to
// the failure of a list (Check). Zero means admitted or valid.
type Code uint8

// Admission codes, numbered in the order of the table of CLR 3.2, then the rows of
// the modules in the order they were built, each appended. The row of not_permitted
// (the institutions module) is the last code, NotPermitted, but its place in the order
// of the checks is after NotTradable; the legal clamp of 3.2.1 is no row and has no
// code: it replaces the limit between ReceiptLimit and InsufficientFunds. no_room of
// the storage module is the code before NotPermitted, after InsufficientStock.
//
// Overflow is the product of limit and quantity not fitting int64. HoldingLimit
// and ReceiptLimit are the two settlement bounds of CLR 3.2.2 (a bid that would
// push the buyer's holding over MaxQuantity, an ask whose worst-case receipt
// would not fit the seller's balance under MaxMoney); they have codes of their
// own so that an operator can tell the two causes apart.
const (
	Admitted Code = iota
	UnknownAgent
	SideNotAllowed
	UnknownCommodity
	NotTradable
	InvalidPrice
	InvalidQuantity
	DuplicateOrder
	Overflow
	HoldingLimit
	ReceiptLimit
	InsufficientFunds
	InsufficientStock
	NoRoom
	NotPermitted
)

// Facts is what the root resolved for one order. The bounds are passed in so
// the package imports no root constant (architecture.md 9.6).
type Facts struct {
	AgentKnown     bool // the agent exists and is in the market's region
	SideAllowed    bool // a pseudo-agent that places one side only placed it
	CommodityKnown bool
	Tradable       bool // in the role's tradable set; true for a pseudo-agent
	Duplicate      bool // an order of this (owner, commodity, lot, side) exists
	Bid            bool // the side of the order
	Price, Qty     int64
	MaxPrice       int64
	MaxQuantity    int64
	MaxMoney       int64
	Available      int64 // a bid: available money; an ask: available units
	Held           int64 // a bid: the buyer's holding of the commodity (D34)
	Balance        int64 // an ask: the seller's money balance (D34)
	Tally          int64 // an ask: worst-case receipt of its asks admitted so far
	// Limited says the bidder's account has a room (storage module): FreeMass and
	// FreeBulk are its free room and SizeMass and SizeBulk the size of one unit of
	// the commodity. They are read for a bid only.
	Limited            bool
	FreeMass, FreeBulk int64
	SizeMass, SizeBulk int64
	// NotPermitted says a right in force bars the agent from this activity on the
	// commodity (institutions module, spec-institutions.md 5.6.2). It stays false for a
	// pseudo-agent and with the module off.
	NotPermitted bool
	// Cap and Floor are the legal range of the market in minor units, 0 for none
	// (spec-institutions.md 6.3.1). A bid above Cap is admitted at Cap and an ask below
	// Floor at Floor. They stay 0 with the module off.
	Cap, Floor int64
}

// Admit returns the code of the first failing row of CLR 3.2, in the order of
// the table, or Admitted. Nothing wraps: the product and every sum use checked
// arithmetic or the subtraction form. It is AdmitClamp without the price it admits.
func Admit(f Facts) Code {
	c, _, _ := AdmitClamp(f)
	return c
}

// AdmitClamp is Admit with the legal clamp of spec-institutions.md 6.3.1. It returns the
// code of the first failing row, the limit price the order is admitted with and whether
// the clamp replaced it. The clamp sits after the overflow row and the two settlement
// bounds and before the funds row, so an order the earlier rows refuse is not rescued by
// it and the funds are tested at the replaced limit. The price is f.Price for a refused
// order and for an order inside the range, and changed is true only for an admitted
// order whose limit was replaced. NotPermitted is tested right after NotTradable.
func AdmitClamp(f Facts) (code Code, price int64, changed bool) {
	if c := identityRow(f); c != Admitted {
		return c, f.Price, false
	}
	if f.NotPermitted {
		return NotPermitted, f.Price, false
	}
	if c := limitsRow(f); c != Admitted {
		return c, f.Price, false
	}
	if _, ok := checked.Mul(f.Price, f.Qty); !ok {
		return Overflow, f.Price, false
	}
	if c := headroomRow(f); c != Admitted {
		return c, f.Price, false
	}
	price, changed = institution.Clamp(f.Price, f.Bid, f.Cap, f.Floor)
	// The replaced limit is at most the original for a bid and the cap is at most MaxPrice
	// (the floor too), so the product fits whenever the first one did.
	product, ok := checked.Mul(price, f.Qty)
	if !ok {
		return Overflow, f.Price, false
	}
	if c := fundsRow(f, product); c != Admitted {
		return c, f.Price, false
	}
	if c := roomRow(f); c != Admitted {
		return c, f.Price, false
	}
	return Admitted, price, changed
}

// roomRow is the no_room row of CLR 3.2 (spec-storage.md 5.2.1, FR-STO-008): a bid
// of a limited account whose quantity times each size exceeds the free room in
// either dimension. It is the last row, so a bid that fails for two reasons reports
// the older one. An ask and an account that is not limited are not tested. A load
// that does not fit int64 is above any room.
func roomRow(f Facts) Code {
	if !f.Bid || !f.Limited {
		return Admitted
	}
	mass, okm := checked.Mul(f.Qty, f.SizeMass)
	bulk, okb := checked.Mul(f.Qty, f.SizeBulk)
	if !okm || !okb || mass > f.FreeMass || bulk > f.FreeBulk {
		return NoRoom
	}
	return Admitted
}

// limitsRow is the price, quantity and duplicate rows of the table of CLR 3.2.
func limitsRow(f Facts) Code {
	switch {
	case f.Price < 1 || f.Price > f.MaxPrice:
		return InvalidPrice
	case f.Qty < 1 || f.Qty > f.MaxQuantity:
		return InvalidQuantity
	case f.Duplicate:
		return DuplicateOrder
	}
	return Admitted
}

// fundsRow is the insufficient_funds and insufficient_stock rows of CLR 3.2.
func fundsRow(f Facts, product int64) Code {
	if f.Bid && product > f.Available {
		return InsufficientFunds
	}
	if !f.Bid && f.Qty > f.Available {
		return InsufficientStock
	}
	return Admitted
}

// identityRow is the first four rows of the table of CLR 3.2.
func identityRow(f Facts) Code {
	switch {
	case !f.AgentKnown:
		return UnknownAgent
	case !f.SideAllowed:
		return SideNotAllowed
	case !f.CommodityKnown:
		return UnknownCommodity
	case !f.Tradable:
		return NotTradable
	}
	return Admitted
}

// headroomRow returns the code of the settlement bound the order would break
// (CLR 3.2.2, D34), or Admitted, in the subtraction form so no sum wraps: a
// bid that would take the buyer's holding over MaxQuantity is HoldingLimit, an
// ask whose worst-case receipt does not fit under MaxMoney is ReceiptLimit.
//
// Held, Balance and Tally are never negative in a ledger; a negative one is a
// fact the root got wrong and is refused as Overflow rather than wrapped. The
// root panics on such a fact before it calls Admit (clearing.go, readFacts),
// so this refusal is only the total function's own guard.
func headroomRow(f Facts) Code {
	if f.Held < 0 || f.Balance < 0 || f.Tally < 0 {
		return Overflow
	}
	if f.Bid {
		if f.Qty > f.MaxQuantity-f.Held {
			return HoldingLimit
		}
		return Admitted
	}
	head, ok := checked.Sub(f.MaxMoney, f.Balance)
	if ok {
		head, ok = checked.Sub(head, f.Tally)
	}
	worst, ok2 := checked.Mul(f.Qty, f.MaxPrice)
	if !ok || !ok2 || worst > head {
		return ReceiptLimit
	}
	return Admitted
}
