package ledger

import (
	"fmt"
	"math"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/checked"
)

// Commodity is what the ledger needs to know about one commodity, as
// primitives (spec-ledger.md 2.7). The index in Config.Commodities is the
// commodity index of every operation.
type Commodity struct {
	// ShelfLife is the ticks a lot lasts, or 0 for a commodity that does not
	// spoil.
	ShelfLife int64
	// Durability is the full durability of an item, or 0 for a commodity that
	// does not wear.
	Durability int64
}

// Config is what New needs. The bounds are parameters because an internal
// package never reads the root's constants (architecture.md 2.9, 9.6).
type Config struct {
	// Commodities is the table of commodities, in commodity index order.
	Commodities []Commodity
	// Regions is the number of regions, at least 1.
	Regions int
	// MaxMoney bounds a non-boundary money balance.
	MaxMoney int64
	// MaxQuantity bounds a non-boundary quantity of one commodity.
	MaxQuantity int64
	// Declared lists the declared reasons (LoanOut, LoanRepay, LoanInterest,
	// WantConsumption) this ledger has, because the module that owns them is on. A reason not
	// listed has no row and no flow slot. It names a reason at most once.
	Declared []Reason
	// MaxClaims is the most claims that may be open at once (spec-credit.md
	// 6.4). 0 means no claim may be opened.
	MaxClaims int
	// Flows lists the money flows of the money supply module in flow-ID order,
	// each as the class of its reason, Faucet for a grant or Sink for a levy
	// (spec-money-supply.md 4.5.1). Flow i has the reason FlowReason(i), a
	// money-only row, and no state is added to a ledger whose Flows is nil
	// (plan 2-6 D323).
	Flows []Class
}

// account is one slot of the account table. A closed account keeps its kind
// and flag only (spec-ledger.md 11).
type account struct {
	kind     Kind
	closed   bool
	owner    int
	region   int
	balance  int64
	reserved int64
	// lots is the goods of a non-boundary account in canonical form
	// (spec-ledger.md 5.1.3): sorted by key, no zero quantity, no equal keys.
	lots []Lot
	// hint is a lower bound of the earliest tick at which any lot expires,
	// valid when hasHint (spec-ledger.md 5.3.2). It is derived, never saved,
	// and can be early (after a removal) but never late.
	hint    int64
	hasHint bool
	// reason and thing identify a boundary account: the reason and Money or
	// the commodity index. They are zero for any other kind.
	reason Reason
	thing  int
}

// Ledger holds all the money and goods and is the only place that changes
// them. It is not safe for concurrent use (architecture.md 12.1). The zero
// value is not usable; build one with New.
type Ledger struct {
	maxMoney int64
	maxQty   int64
	comms    []Commodity
	regions  int

	// stride is the number of flow slots of one region and off the offset of
	// each (reason, thing) in a region's block, -1 for a pair no row moves.
	stride int
	off    [numStage1Reasons][2]int
	// later holds the offsets of the reasons past the Stage 1 size, indexed by
	// reason minus numStage1Reasons. It is nil while none of them is declared,
	// so a ledger built with no module has the layout of Stage 1 (offOf reads
	// both).
	later [][2]int
	// flowClass is the class of each flow reason and flowOff the offset of its
	// money slot; both are nil in a ledger with no flow.
	flowClass []Class
	flowOff   []int

	accounts []account
	// boundary is the account ID of each (reason, thing), indexed like a
	// region's block of flow slots, -1 where none was opened.
	boundary []int

	// totals are the flow totals of the current tick (spec-ledger.md 7.4)
	// and bucket the construction bucket (7.4.1), in the same layout:
	// region-major, then reason, then money before goods, then commodity.
	totals       []int64
	bucket       []int64
	construction bool
	// base is each boundary account's balance when the totals were last
	// cleared, indexed like boundary (A7 of plan 1-1), and cleared whether
	// they have been cleared since construction: until then the bucket is
	// part of the movement the audit explains.
	base    []int64
	cleared bool

	// claims is the open-claim list in ascending ID, nextClaim the next ID and
	// maxClaims the limit on len(claims) (claims.go).
	claims    []Claim
	nextClaim int64
	maxClaims int
	claimTot  ClaimTotals

	// tick is the current tick, which creation reads (5.2.3). It is 0 in
	// construction.
	tick int64

	// wear and wearBucket are the wear units per commodity of the current
	// tick and of the construction phase (5.4.2, 7.4).
	wear, wearBucket []int64
	// relocs are the pending relocation totals (checkpoint.go), cleared with the
	// flow totals.
	relocs []reloc
	// spoil and sel are scratch, empty between calls: expired units per
	// (region, commodity) for Age, and the selection of WearItems.
	spoil []int64
	sel   []wearSel
}

// View is a read-only copy of one account.
type View struct {
	Kind   Kind
	Closed bool
	// Owner is the opaque owner reference, and Region the account's region.
	// Both are 0 for a closed account and Region is -1 for a boundary.
	Owner, Region int
	// Balance is signed for a boundary and is never negative otherwise.
	Balance int64
	// Reserved is the money earmarked by a reservation.
	Reserved int64
}

// Flow is one non-zero total of the flow totals or of the construction
// bucket. Commodity is Money for money.
type Flow struct {
	Region    int
	Reason    Reason
	Commodity int
	Total     int64
}

// validate checks cfg in the order of New's errors and returns the set of
// declared reasons.
func validate(cfg Config) (declared [NumReasons]bool, err error) {
	switch {
	case cfg.Regions < 1:
		return declared, fmt.Errorf("%w: %d regions, want at least 1", ErrInvalidConfig, cfg.Regions)
	case cfg.MaxMoney < 1 || cfg.MaxQuantity < 1:
		return declared, fmt.Errorf("%w: bounds must be at least 1", ErrInvalidConfig)
	}
	for i, c := range cfg.Commodities {
		if c.ShelfLife < 0 || c.Durability < 0 {
			return declared, fmt.Errorf("%w: commodity %d has a negative shelf life or durability", ErrInvalidConfig, i)
		}
	}
	if cfg.MaxClaims < 0 {
		return declared, fmt.Errorf("%w: %d claims, want at least 0", ErrInvalidConfig, cfg.MaxClaims)
	}
	if err := validateFlows(cfg.Flows); err != nil {
		return declared, err
	}
	return declaredSet(cfg.Declared)
}

// validateFlows checks that every flow is a faucet or a sink and that the
// reasons fit the Reason type.
func validateFlows(flows []Class) error {
	if len(flows) > math.MaxUint8-int(NumReasons) {
		return fmt.Errorf("%w: %d flows do not fit the reason numbers", ErrInvalidConfig, len(flows))
	}
	for i, c := range flows {
		if c != Faucet && c != Sink {
			return fmt.Errorf("%w: flow %d is neither a faucet nor a sink", ErrInvalidConfig, i)
		}
	}
	return nil
}

// declaredSet returns the set of reasons, or an error for one that is not
// declarable or is listed twice.
func declaredSet(reasons []Reason) (declared [NumReasons]bool, err error) {
	for _, r := range reasons {
		if !isDeclared(r) || declared[r] {
			return declared, fmt.Errorf("%w: reason %d is not a declarable reason, or is declared twice", ErrInvalidConfig, r)
		}
		declared[r] = true
	}
	return declared, nil
}

// layout assigns the flow-total offsets of every reason and thing that has a
// row, and sets l.stride. A declared row has an offset only if its reason is in
// declared.
func (l *Ledger) layout(declared *[NumReasons]bool) {
	for r := Reason(0); r < NumReasons; r++ {
		for t := range table[r] {
			o := -1
			if table[r][t].ok && (!table[r][t].declared || declared[r]) {
				o = l.stride
				if t == thingMoney {
					l.stride++
				} else {
					l.stride += len(l.comms)
				}
			}
			l.setOff(r, t, o)
		}
	}
	// The flow slots follow every row of the table, so a ledger with no flow
	// has the layout it had before (plan 2-6 D323).
	for range l.flowClass {
		l.flowOff = append(l.flowOff, l.stride)
		l.stride++
	}
}

// reasons is the number of reason numbers the ledger knows: the table's and
// its flows'.
func (l *Ledger) reasons() int { return int(NumReasons) + len(l.flowClass) }

// rowOf returns the row of a reason for money or goods, flows included. A flow
// moves money only.
func (l *Ledger) rowOf(r Reason, goods bool) (row, bool) {
	if r < NumReasons {
		return rowOf(r, goods)
	}
	if i := int(r) - int(NumReasons); !goods && i < len(l.flowClass) {
		return flowRow(l.flowClass[i]), true
	}
	return row{}, false
}

// setOff records the offset of (r, t). The slice of later reasons is made when
// the first of them has an offset, so it stays nil in a ledger with no module.
func (l *Ledger) setOff(r Reason, t, o int) {
	if int(r) < numStage1Reasons {
		l.off[r][t] = o
		return
	}
	if o >= 0 && l.later == nil {
		l.later = make([][2]int, int(NumReasons)-numStage1Reasons)
		for i := range l.later {
			l.later[i] = [2]int{-1, -1}
		}
	}
	if l.later != nil {
		l.later[int(r)-numStage1Reasons][t] = o
	}
}

// offOf returns the offset of (r, t) in a region's block of flow slots, and -1
// for a pair no row moves, a declared row that was not declared, or a reason out
// of range.
func (l *Ledger) offOf(r Reason, t int) int {
	switch {
	case int(r) < numStage1Reasons:
		return l.off[r][t]
	case r >= NumReasons:
		if i := int(r) - int(NumReasons); t == thingMoney && i < len(l.flowOff) {
			return l.flowOff[i]
		}
		return -1
	case l.later == nil:
		return -1
	}
	return l.later[int(r)-numStage1Reasons][t]
}

// New builds an empty ledger in its construction phase. It returns an error
// wrapping ErrInvalidConfig for a table it cannot use.
func New(cfg Config) (*Ledger, error) {
	declared, err := validate(cfg)
	if err != nil {
		return nil, err
	}
	l := &Ledger{
		maxClaims:    cfg.MaxClaims,
		maxMoney:     cfg.MaxMoney,
		maxQty:       cfg.MaxQuantity,
		comms:        append([]Commodity(nil), cfg.Commodities...),
		regions:      cfg.Regions,
		construction: true,
	}
	if len(cfg.Flows) > 0 {
		l.flowClass = slices.Clone(cfg.Flows)
	}
	l.layout(&declared)
	n, ok := checked.Mul(int64(l.stride), int64(cfg.Regions))
	if !ok || n > int64(^uint(0)>>1) {
		return nil, fmt.Errorf("%w: flow totals do not fit", ErrInvalidConfig)
	}
	l.totals = make([]int64, n)
	l.bucket = make([]int64, n)
	l.wear = make([]int64, len(l.comms))
	l.wearBucket = make([]int64, len(l.comms))
	l.spoil = make([]int64, len(l.comms)*cfg.Regions)
	l.base = make([]int64, l.stride)
	l.boundary = make([]int, l.stride)
	for i := range l.boundary {
		l.boundary[i] = -1
	}
	return l, nil
}

// Len returns the number of accounts, closed ones included. Accounts are
// iterated in ID order, 0 to Len-1 (FR-LED-008).
func (l *Ledger) Len() int { return len(l.accounts) }

// Account returns a copy of account id, closed or not.
func (l *Ledger) Account(id int) (View, error) {
	a, err := l.at(id)
	if err != nil {
		return View{}, err
	}
	v := View{Kind: a.kind, Closed: a.closed, Owner: a.owner, Region: a.region, Balance: a.balance, Reserved: a.reserved}
	if a.kind == KindBoundary {
		v.Region = -1
	}
	return v, nil
}

// at returns the account or ErrUnknownAccount.
func (l *Ledger) at(id int) (*account, error) {
	if id < 0 || id >= len(l.accounts) {
		return nil, ErrUnknownAccount
	}
	return &l.accounts[id], nil
}

// open returns the account if it exists and is open.
func (l *Ledger) open(id int) (*account, error) {
	a, err := l.at(id)
	if err != nil {
		return nil, err
	}
	if a.closed {
		return nil, ErrClosedAccount
	}
	return a, nil
}

// Open creates an account of a non-boundary kind and returns its ID, the next
// in creation order. owner is an opaque reference the ledger never reads.
func (l *Ledger) Open(kind Kind, owner, region int) (int, error) {
	if kind >= KindBoundary {
		return 0, ErrInvalidKind
	}
	if region < 0 || region >= l.regions {
		return 0, ErrUnknownRegion
	}
	l.accounts = append(l.accounts, account{kind: kind, owner: owner, region: region})
	return len(l.accounts) - 1, nil
}

// slotOf returns the index of (reason, commodity) in a region's block and
// false when no row moves that thing or the commodity is out of range.
func (l *Ledger) slotOf(r Reason, commodity int) (int, error) {
	if int(r) >= l.reasons() {
		return 0, ErrReasonMismatch
	}
	if commodity == Money {
		if o := l.offOf(r, thingMoney); o >= 0 {
			return o, nil
		}
		return 0, ErrReasonMismatch
	}
	if commodity < 0 || commodity >= len(l.comms) {
		return 0, ErrUnknownCommodity
	}
	if o := l.offOf(r, thingGoods); o >= 0 {
		return o + commodity, nil
	}
	return 0, ErrReasonMismatch
}

// OpenBoundary creates the boundary account of a reason and a thing (Money or
// a commodity index) and returns its ID. Only a faucet or a sink has one
// (spec-ledger.md 3.4.1), and there is one per reason and thing.
func (l *Ledger) OpenBoundary(r Reason, commodity int) (int, error) {
	i, err := l.slotOf(r, commodity)
	if err != nil {
		return 0, err
	}
	if rw, _ := l.rowOf(r, commodity != Money); rw.class == Internal {
		return 0, ErrReasonMismatch
	}
	if l.boundary[i] >= 0 {
		return 0, ErrDuplicateBoundary
	}
	l.accounts = append(l.accounts, account{kind: KindBoundary, region: -1, reason: r, thing: commodity})
	l.boundary[i] = len(l.accounts) - 1
	return l.boundary[i], nil
}

// Boundary returns the ID of the boundary account of a reason and a thing,
// and false if none was opened.
func (l *Ledger) Boundary(r Reason, commodity int) (int, bool) {
	i, err := l.slotOf(r, commodity)
	if err != nil || l.boundary[i] < 0 {
		return 0, false
	}
	return l.boundary[i], true
}

// Close closes an account whose owner has ceased to exist. It checks, in this
// order, and returns the first that holds (spec-ledger.md 3.3.1): unknown,
// closed, a reservation of money or an earmark of goods (ErrOpenReservation),
// money or goods (ErrNotEmpty), a claim held or owed (ErrOpenClaim). A
// boundary is never closed (ErrInvalidKind).
// The slot stays, so IDs stay stable.
func (l *Ledger) Close(id int) error {
	a, err := l.open(id)
	if err != nil {
		return err
	}
	switch {
	case a.kind == KindBoundary:
		return ErrInvalidKind
	case a.reserved != 0 || a.earmarked():
		return ErrOpenReservation
	case a.balance != 0 || len(a.lots) != 0:
		return ErrNotEmpty
	case l.inClaim(id):
		return ErrOpenClaim
	}
	*a = account{kind: a.kind, closed: true}
	return nil
}

// Tick returns the current tick, 0 until the first AdvanceTick.
func (l *Ledger) Tick() int64 { return l.tick }

// AdvanceTick moves the current tick on by one. The world calls it at the end
// of a tick (P8.6); creation stamps new lots with the tick it reads.
func (l *Ledger) AdvanceTick() { l.tick++ }

// InConstruction reports whether the ledger is in its construction phase
// (spec-ledger.md 5.2.4), which it is from New until EndConstruction.
func (l *Ledger) InConstruction() bool { return l.construction }

// EndConstruction ends the construction phase: from now on moves are totalled
// in the flow totals, and the construction bucket is read-only. It is called
// when the first Step begins, and is refused with ErrNotConstruction after.
func (l *Ledger) EndConstruction() error {
	if !l.construction {
		return ErrNotConstruction
	}
	l.construction = false
	return nil
}

// ClearTotals zeroes the flow totals and the wear units, at the end of a tick
// (P8.6). The construction bucket is cleared by nothing.
func (l *Ledger) ClearTotals() {
	clear(l.totals)
	clear(l.wear)
	l.relocs = l.relocs[:0]
	for i, id := range l.boundary {
		if id >= 0 {
			l.base[i] = l.accounts[id].balance
		}
	}
	l.cleared = true
}

// AppendFlows appends the non-zero totals of the current tick, or of the
// construction bucket, to dst and returns it. The order is ascending region,
// then the built-in reasons, the flows in flow order and the declared reasons
// in number order, money before goods, then commodity. It allocates only when dst
// has no room, so the world can reuse a slice. It reports internal reasons
// too: the root keeps the faucets and sinks (spec-ledger.md 12.1).
func (l *Ledger) AppendFlows(dst []Flow, bucket bool) []Flow {
	src := l.totals
	if bucket {
		src = l.bucket
	}
	for region := 0; region < l.regions; region++ {
		base := src[region*l.stride:]
		for r := Reason(0); r < NumReasons; r++ {
			if !isDeclared(r) {
				dst = l.appendReason(dst, base, region, r)
			}
		}
		for i := range l.flowClass {
			dst = l.appendReason(dst, base, region, FlowReason(i))
		}
		for r := Reason(0); r < NumReasons; r++ {
			if isDeclared(r) {
				dst = l.appendReason(dst, base, region, r)
			}
		}
	}
	return dst
}

// appendReason appends the non-zero totals of reason r in one region, money
// before goods, to dst. base is the region's slice of the totals.
func (l *Ledger) appendReason(dst []Flow, base []int64, region int, r Reason) []Flow {
	if o := l.offOf(r, thingMoney); o >= 0 && base[o] != 0 {
		dst = append(dst, Flow{region, r, Money, base[o]})
	}
	if o := l.offOf(r, thingGoods); o >= 0 {
		for c := range l.comms {
			if v := base[o+c]; v != 0 {
				dst = append(dst, Flow{region, r, c, v})
			}
		}
	}
	return dst
}

// flows returns the slice the current phase adds to: the bucket while in
// construction, else the totals.
func (l *Ledger) flows() []int64 {
	if l.construction {
		return l.bucket
	}
	return l.totals
}
