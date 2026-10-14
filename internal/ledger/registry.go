package ledger

// Kind is the kind of an account, numbered in the order of the table of
// spec-ledger.md 3.1 (the checkpoint saves the number, CHK 4.2 tag 2).
type Kind uint8

// The account kinds. All seven are declared because the order is their
// number; the pairing rules of the built-in rows name Agent, Estate and
// Boundary only, and a module appends the others to the rows it owns.
const (
	KindAgent Kind = iota
	KindEstate
	KindEscrow
	KindTransit
	KindOutside
	KindTreasury
	KindBoundary
	numKinds
)

// Class is the class of a reason: internal (both accounts non-boundary),
// faucet (from a boundary) or sink (to a boundary), spec-ledger.md 7.1.
type Class uint8

// The classes of a reason.
const (
	Internal Class = iota
	Faucet
	Sink
)

// Reason names a flow. It is an internal number, not a string (the ledger
// knows no identifiers, 2.7), so a checkpoint must never save one raw. The
// values are the built-in rows of spec-ledger.md 7.2.2 in the order of the
// table of 7.2, which is the order of the BoundaryFlow rows.
type Reason uint8

// The built-in reasons.
const (
	Trade Reason = iota
	Initial
	RecipeInput
	RecipeOutput
	Disposal
	Consumption
	Spoilage
	Wear
	EstateMoney
	EstateGoods
	StartMoney
	Split
	Merge
	Births
	Starvation
	// LoanOut, LoanRepay and LoanInterest are the loan reasons of the credit
	// module (spec-credit.md 3.3.1). They are declared rows, not built-in
	// ones: a ledger has them only when Config.Declared names them (plan 1-1
	// A1), and a reason that is not declared is refused with ErrReasonMismatch.
	LoanOut
	LoanRepay
	LoanInterest
	// NumReasons is the number of reasons the ledger knows.
	NumReasons
)

// firstDeclared is the first reason of the declared group: a reason at or
// above it has a row and a flow slot only if Config.Declared lists it.
const firstDeclared = LoanOut

// Money is the commodity index that names money where a flow slot or a
// boundary account is addressed by (reason, thing).
const Money = -1

// The two things a reason can move.
const (
	thingMoney = 0
	thingGoods = 1
)

// kindSet is a set of Kinds.
type kindSet uint8

func (s kindSet) has(k Kind) bool { return k < numKinds && s&(1<<k) != 0 }

const (
	setBoundary = kindSet(1 << KindBoundary)
	setAgent    = kindSet(1 << KindAgent)
	setEstate   = kindSet(1 << KindEstate)
	setAE       = setAgent | setEstate
	// setHolder is every non-boundary kind: "any holder" of 7.2.
	setHolder = kindSet(1<<KindBoundary - 1)
)

// row is one row of the registry for one thing: its class and the kinds it
// runs between.
type row struct {
	ok       bool
	class    Class
	from, to kindSet
	// distinct requires the two kinds to differ: "agent to estate, or back".
	distinct bool
}

// fits reports whether the row runs from a kind to a kind.
func (rw row) fits(from, to Kind) bool {
	return rw.ok && rw.from.has(from) && rw.to.has(to) && (!rw.distinct || from != to)
}

// table is the registry of spec-ledger.md 7.2 for the built-in rows (plan
// 1-1 A4), indexed by reason then thing. It is never mutated (architecture.md
// 7.1). A reason with no row for a thing cannot move that thing.
var table = [NumReasons][2]row{
	Trade:        {thingMoney: {true, Internal, setAE, setAE, false}, thingGoods: {true, Internal, setAE, setAE, false}},
	Initial:      {thingMoney: {true, Faucet, setBoundary, setAE, false}, thingGoods: {true, Faucet, setBoundary, setAE, false}},
	RecipeInput:  {thingGoods: {true, Sink, setAgent, setBoundary, false}},
	RecipeOutput: {thingGoods: {true, Faucet, setBoundary, setAgent, false}},
	Disposal:     {thingGoods: {true, Sink, setAgent, setBoundary, false}},
	Consumption:  {thingGoods: {true, Sink, setAgent, setBoundary, false}},
	Spoilage:     {thingGoods: {true, Sink, setHolder, setBoundary, false}},
	Wear:         {thingGoods: {true, Sink, setAgent, setBoundary, false}},
	EstateMoney:  {thingMoney: {true, Internal, setAgent, setEstate, false}},
	EstateGoods:  {thingGoods: {true, Internal, setAgent, setEstate, false}},
	StartMoney:   {thingMoney: {true, Internal, setEstate, setAgent, false}},
	Split:        {thingMoney: {true, Internal, setAE, setAE, true}, thingGoods: {true, Internal, setAE, setAE, true}},
	Merge:        {thingMoney: {true, Internal, setAE, setAE, true}, thingGoods: {true, Internal, setAE, setAE, true}},
	Births:       {thingMoney: {true, Internal, setEstate, setAgent, false}},
	Starvation:   {thingMoney: {true, Internal, setAE, setAE, true}, thingGoods: {true, Internal, setAE, setAE, true}},
	// The declared rows: loan_out runs between agents, and the two payments
	// from an agent to an agent or an estate (spec-ledger.md 7.2, FR-LED-062).
	LoanOut:      {thingMoney: {true, Internal, setAgent, setAgent, false}},
	LoanRepay:    {thingMoney: {true, Internal, setAgent, setAE, false}},
	LoanInterest: {thingMoney: {true, Internal, setAgent, setAE, false}},
}

// rowOf returns the row of a reason for money or goods.
func rowOf(r Reason, goods bool) (row, bool) {
	if r >= NumReasons {
		return row{}, false
	}
	t := thingMoney
	if goods {
		t = thingGoods
	}
	rw := table[r][t]
	return rw, rw.ok
}

// ClassOf returns the class of reason r for money (goods false) or for goods,
// and false when the reason has no row for that thing. The root reads it to
// choose the faucet and sink reasons of BoundaryFlow.
func ClassOf(r Reason, goods bool) (Class, bool) {
	rw, ok := rowOf(r, goods)
	return rw.class, ok
}

// Allowed reports whether reason r moves money (goods false) or goods from an
// account of kind from to an account of kind to, by the table of
// spec-ledger.md 7.2. It is the check of ErrReasonMismatch on the kinds.
func Allowed(r Reason, goods bool, from, to Kind) bool {
	rw, _ := rowOf(r, goods)
	return rw.fits(from, to)
}
