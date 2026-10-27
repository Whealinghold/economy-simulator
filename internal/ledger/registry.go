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
	// WantConsumption is the goods sink of the demand module (spec-demand.md
	// 4.7): the consumption of a want beyond upkeep. It is a declared row, and
	// the first one past the Stage 1 size (numStage1Reasons).
	WantConsumption
	// NumReasons is the number of reasons the ledger knows.
	NumReasons
)

// numStage1Reasons is the number of reasons of the Stage 1 ledger. Ledger.off
// keeps this size, so that a ledger built with no module has the layout of
// Stage 1 (spec-ledger.md 7.2.2, plan 2-5 D311); the offsets of a later reason
// live in a slice that is nil while no such reason is declared.
const numStage1Reasons = int(LoanInterest) + 1

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
	// declared marks a row a module owns: it has a flow slot only if
	// Config.Declared lists its reason. It is a bit of the row, not a
	// threshold on the number of the reason (spec-ledger.md 7.2.2).
	declared bool
	// rank is the position of the row in the table of spec-ledger.md 7.2, the
	// order of the declared rows among themselves in BoundaryFlow
	// (spec-observability.md 4.4.1.2); it is 0 for a built-in row.
	rank uint8
}

// fits reports whether the row runs from a kind to a kind.
func (rw row) fits(from, to Kind) bool {
	return rw.ok && rw.from.has(from) && rw.to.has(to) && (!rw.distinct || from != to)
}

// table is the registry of spec-ledger.md 7.2 for the built-in rows (plan
// 1-1 A4), indexed by reason then thing. It is never mutated (architecture.md
// 7.1). A reason with no row for a thing cannot move that thing.
var table = [NumReasons][2]row{
	Trade:        {thingMoney: {true, Internal, setAE, setAE, false, false, 0}, thingGoods: {true, Internal, setAE, setAE, false, false, 0}},
	Initial:      {thingMoney: {true, Faucet, setBoundary, setAE, false, false, 0}, thingGoods: {true, Faucet, setBoundary, setAE, false, false, 0}},
	RecipeInput:  {thingGoods: {true, Sink, setAgent, setBoundary, false, false, 0}},
	RecipeOutput: {thingGoods: {true, Faucet, setBoundary, setAgent, false, false, 0}},
	Disposal:     {thingGoods: {true, Sink, setAgent, setBoundary, false, false, 0}},
	Consumption:  {thingGoods: {true, Sink, setAgent, setBoundary, false, false, 0}},
	Spoilage:     {thingGoods: {true, Sink, setHolder, setBoundary, false, false, 0}},
	Wear:         {thingGoods: {true, Sink, setAgent, setBoundary, false, false, 0}},
	EstateMoney:  {thingMoney: {true, Internal, setAgent, setEstate, false, false, 0}},
	EstateGoods:  {thingGoods: {true, Internal, setAgent, setEstate, false, false, 0}},
	StartMoney:   {thingMoney: {true, Internal, setEstate, setAgent, false, false, 0}},
	Split:        {thingMoney: {true, Internal, setAE, setAE, true, false, 0}, thingGoods: {true, Internal, setAE, setAE, true, false, 0}},
	Merge:        {thingMoney: {true, Internal, setAE, setAE, true, false, 0}, thingGoods: {true, Internal, setAE, setAE, true, false, 0}},
	Births:       {thingMoney: {true, Internal, setEstate, setAgent, false, false, 0}},
	Starvation:   {thingMoney: {true, Internal, setAE, setAE, true, false, 0}, thingGoods: {true, Internal, setAE, setAE, true, false, 0}},
	// The declared rows: loan_out runs between agents, and the two payments
	// from an agent to an agent or an estate (spec-ledger.md 7.2, FR-LED-062).
	LoanOut:      {thingMoney: {true, Internal, setAgent, setAgent, false, true, 25}},
	LoanRepay:    {thingMoney: {true, Internal, setAgent, setAE, false, true, 26}},
	LoanInterest: {thingMoney: {true, Internal, setAgent, setAE, false, true, 27}},
	// want_consumption is a goods sink from an agent to a boundary (spec-demand.md
	// 4.7); its rank is its position in the table of 7.2.
	WantConsumption: {thingGoods: {true, Sink, setAgent, setBoundary, false, true, 6}},
}

// isDeclared reports whether reason r is a declared reason: one whose row a
// module owns.
func isDeclared(r Reason) bool {
	return r < NumReasons && (table[r][thingMoney].declared || table[r][thingGoods].declared)
}

// Rank returns the position of the declared reason r in the table of
// spec-ledger.md 7.2, the order of the declared rows among themselves, and false
// for a reason that is built in or unknown. The root orders the BoundaryFlow rows
// of the declared module reasons by it, not by the number of the reason.
func Rank(r Reason) (int, bool) {
	if !isDeclared(r) {
		return 0, false
	}
	return int(rankOf(r)), true
}

// FlowReason returns the reason of the money flow at position i of
// Config.Flows, 0-based, the flows being in ID order. A flow reason is a
// dynamic reason: its number follows the built-in and declared ones, so it is
// never saved raw (checkpoint.go saves pinnedRows plus its position).
func FlowReason(i int) Reason { return NumReasons + Reason(i) }

// flowRow is the row of a money flow of the given class: a faucet runs from
// its boundary to an agent or an estate, a sink the other way round
// (spec-money-supply.md 4.5.1). It moves money only.
func flowRow(c Class) row {
	if c == Faucet {
		return row{ok: true, class: Faucet, from: setBoundary, to: setAE}
	}
	return row{ok: true, class: Sink, from: setAE, to: setBoundary}
}

// rowOf returns the row of a reason of the table for money or goods. A flow
// reason is not in the table: Ledger.rowOf knows it.
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

// DeclaredByRank appends to dst the declared reasons in the order of their
// rank, the position of their row in the table of spec-ledger.md 7.2, and
// returns it. It allocates only when dst has no room.
func DeclaredByRank(dst []Reason) []Reason {
	start := len(dst)
	for r := Reason(0); r < NumReasons; r++ {
		if !isDeclared(r) {
			continue
		}
		dst = append(dst, r)
		// Insertion by rank; the ranks are unique, so the order is total.
		for i := len(dst) - 1; i > start && rankOf(dst[i]) < rankOf(dst[i-1]); i-- {
			dst[i], dst[i-1] = dst[i-1], dst[i]
		}
	}
	return dst
}

// rankOf returns the rank of a declared reason.
func rankOf(r Reason) uint8 { return max(table[r][thingMoney].rank, table[r][thingGoods].rank) }
