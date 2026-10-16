package clearing

import "github.com/Whealinghold/economy-sim/internal/checked"

// Code is a small integer the root maps to a rejection reason (Admit) or to
// the failure of a list (Check). Zero means admitted or valid.
type Code uint8

// Admission codes, numbered in the order of the table of CLR 3.2. The rows of
// the modules (not_permitted after NotTradable; no_room after
// InsufficientStock) and the legal clamp of 3.2.1 (between ReceiptLimit and
// InsufficientFunds) have their positions marked in Admit and no code yet.
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
}

// Admit returns the code of the first failing row of CLR 3.2, in the order of
// the table, or Admitted. Nothing wraps: the product and every sum use checked
// arithmetic or the subtraction form.
func Admit(f Facts) Code {
	if c := identityRow(f); c != Admitted {
		return c
	}
	// Position of not_permitted (institutions module): no code yet.
	switch {
	case f.Price < 1 || f.Price > f.MaxPrice:
		return InvalidPrice
	case f.Qty < 1 || f.Qty > f.MaxQuantity:
		return InvalidQuantity
	case f.Duplicate:
		return DuplicateOrder
	}
	product, ok := checked.Mul(f.Price, f.Qty)
	if !ok {
		return Overflow
	}
	if c := headroomRow(f); c != Admitted {
		return c
	}
	// Position of the legal clamp (3.2.1, institutions module): no code yet.
	if f.Bid && product > f.Available {
		return InsufficientFunds
	}
	if !f.Bid && f.Qty > f.Available {
		return InsufficientStock
	}
	// Position of no_room (storage module): no code yet.
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
