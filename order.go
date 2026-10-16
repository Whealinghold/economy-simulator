package economysim

// LotKey names the lot an ask is made from: the commodity's born tick and
// remaining durability (spec-clearing.md 3.1.2). The zero value names no lot.
// It is a plain comparable struct, so an Order and a Trade are comparable.
type LotKey struct {
	// Named is true when the key names a lot; false means any lot.
	Named bool
	// Born is the tick the lot was made, 0 for a commodity that does not spoil.
	Born int64
	// Durability is the remaining durability of the lot, 0 for a commodity
	// that does not wear.
	Durability int64
}

// Order is one admitted order of one market (spec-clearing.md 3.1). The market
// is the region and the commodity of the book, so the order carries no region.
type Order struct {
	// Agent is the agent that placed the order.
	Agent AgentID
	// Commodity is the commodity of the order.
	Commodity CommodityID
	// Side is Bid or Ask.
	Side Side
	// Price is the limit price, as admitted.
	Price Price
	// Quantity is the units, at least 1.
	Quantity Quantity
	// Lot is the lot an ask names; the zero value names none.
	Lot LotKey
}

// Trade is a trade of one buyer and one seller (spec-clearing.md 3.1.2).
type Trade struct {
	// Commodity is the commodity traded.
	Commodity CommodityID
	// Buyer is the agent that holds the bid.
	Buyer AgentID
	// Seller is the agent that holds the ask.
	Seller AgentID
	// Price is the price per unit, between the ask's limit and the bid's.
	Price Price
	// Quantity is the units traded, at least 1.
	Quantity Quantity
	// Lot is the lot of the ask the trade filled.
	Lot LotKey
}
