package ledger

import "errors"

// The sentinels of the ledger (spec-ledger.md 3.3.1, 4.2, 6). Every refusal
// leaves the ledger unchanged (FR-LED-007), and a caller matches one with
// errors.Is. The root maps them to public errors and rejection reasons with
// their first users (N-075); none is exported from the module.
var (
	// ErrInsufficientFunds means the debited account does not have the amount
	// available: its balance less what is reserved.
	ErrInsufficientFunds = errors.New("ledger: insufficient funds")
	// ErrBalanceOverflow means the result would exceed MaxMoney in a
	// non-boundary account, or int64 in a boundary account or a flow total.
	ErrBalanceOverflow = errors.New("ledger: balance overflow")
	// ErrReasonMismatch means the reason is unknown, does not move the thing,
	// or does not fit the kinds of the two accounts (spec-ledger.md 7.2).
	ErrReasonMismatch = errors.New("ledger: reason does not fit the accounts")
	// ErrUnknownAccount means an account ID is outside the ledger.
	ErrUnknownAccount = errors.New("ledger: unknown account")
	// ErrClosedAccount means the operation names a closed account.
	ErrClosedAccount = errors.New("ledger: closed account")
	// ErrInvalidAmount means an amount below 1.
	ErrInvalidAmount = errors.New("ledger: invalid amount")
	// ErrBoundaryReserve means a boundary account was asked to reserve, or a
	// transfer was marked from reserved on a boundary account.
	ErrBoundaryReserve = errors.New("ledger: a boundary account cannot reserve")
	// ErrReservationExceeds means a release, or a transfer marked from
	// reserved, asked for more than is reserved.
	ErrReservationExceeds = errors.New("ledger: amount exceeds the reservation")
	// ErrOpenReservation means a Close found a reservation on the account.
	ErrOpenReservation = errors.New("ledger: account holds a reservation")
	// ErrNotEmpty means a Close found money or goods on the account.
	ErrNotEmpty = errors.New("ledger: account holds money or goods")
	// ErrInvalidKind means a kind that the operation does not apply to: Open
	// of a boundary or of a number that is no kind, Close of a boundary.
	ErrInvalidKind = errors.New("ledger: invalid account kind")
	// ErrUnknownRegion means a region outside the ledger's regions.
	ErrUnknownRegion = errors.New("ledger: unknown region")
	// ErrUnknownCommodity means a commodity index outside the ledger's table.
	ErrUnknownCommodity = errors.New("ledger: unknown commodity")
	// ErrDuplicateBoundary means the boundary account of a reason and thing
	// already exists: there is one per reason and per thing (3.4).
	ErrDuplicateBoundary = errors.New("ledger: duplicate boundary account")
	// ErrNotConstruction means an operation that is valid only in the
	// construction phase (5.2.4) was called after it.
	ErrNotConstruction = errors.New("ledger: not in the construction phase")
	// ErrInsufficientStock means the account does not have the quantity
	// available: its holding less what is earmarked.
	ErrInsufficientStock = errors.New("ledger: insufficient stock")
	// ErrUnknownLot means a named lot (spec-ledger.md 6.7) is not held.
	ErrUnknownLot = errors.New("ledger: unknown lot")
	// ErrInvalidLot means a stated lot key (5.2.4) outside the ranges of
	// FR-LED-043.
	ErrInvalidLot = errors.New("ledger: invalid lot key")
	// ErrEmptyTransform means a Transform with no inputs and no outputs
	// (spec-ledger.md 8.2.1).
	ErrEmptyTransform = errors.New("ledger: empty transform")
	// ErrNotDurable means a wear of a commodity that does not wear.
	ErrNotDurable = errors.New("ledger: commodity does not wear")
	// ErrInvalidConfig means New was given a table it cannot use.
	ErrInvalidConfig = errors.New("ledger: invalid configuration")
	// ErrUnknownClaim means a claim ID is not in the open-claim list: it
	// was never formed, or it has ended (spec-credit.md 3.2).
	ErrUnknownClaim = errors.New("ledger: unknown claim")
	// ErrClaimShape means a claim that could not be a claim: the same account
	// on both sides, a due tick not after the tick of formation, an account
	// of a kind a claim does not run between, or an overdue tick below the due
	// tick (spec-credit.md 3.2).
	ErrClaimShape = errors.New("ledger: invalid claim shape")
	// ErrTooManyClaims means MaxClaims claims are open already.
	ErrTooManyClaims = errors.New("ledger: too many open claims")
	// ErrOpenClaim means a Close found the account the creditor or the
	// debtor of an open claim (spec-ledger.md 3.3.1).
	ErrOpenClaim = errors.New("ledger: account is party to an open claim")
	// ErrAuditFailed means Audit found an identity false. The error is an
	// *AuditError that names it, and matches this sentinel.
	ErrAuditFailed = errors.New("ledger: audit failed")
)
