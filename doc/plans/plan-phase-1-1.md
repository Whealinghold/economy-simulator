# Plan: Stage 1, row 1: the ledger

**Status: Implemented (2026-10-13), uncommitted: Phases 1 to 7 done; the gate
of Phase 7 is clean.** Sequences the first row of Stage 1 of
[roadmap-v1.md](../roadmap-v1.md), "`internal/ledger` (accounts, the transfer
primitive, reasons, lots, reservations, `Transform`, `Wear`, ageing, flow
totals, the audit) and `ledger.go`", with the claims list and the balance
sheet of LED 9.2 and 10 added at the owner's request (decision D8), into seven
phases. It implements
[spec-ledger.md](../specs/spec-ledger.md) (LED) for the slice. It is built
against `doc/rules/architecture.md` (the `internal/` boundary of 2.9, the
fixed-point and overflow rule of 3.5, the determinism rules of section 7, the
layout of 2.6), `doc/rules/testing.md` and
`doc/rules/observability-and-logging.md`. Where this plan and the rules or the
specs disagree, they win and this plan is corrected.

A plan decides nothing. Every decision below comes from a spec, a rule, an ADR
or the decisions of [plan-phase-0-foundation.md](plan-phase-0-foundation.md)
(D3 and D5 above all). Where a decision is still owed, Phase 1 makes it in the
document that owns it, before the code that depends on it.

No PRD exists for this row. The requirements source is the roadmap row, LED
and the owner's request of 2026-10-13: build the ledger row alone, ahead of
the other rows of Stage 1.

Spec codes: LED ledger, GDS goods, WLD world, OBS observability, CFG config,
CHK checkpoint, CLR clearing, PRO production, AGT agents, EST estates, SCL
scale, SPN spawner, REG regions, CRD credit, STO storage, MON money supply,
DEM demand, TRN transport, RSK risk, IND indicators. `FR-`/`NFR-` IDs are the
requirement IDs of those specs.

## Scope

**In scope.** What the slice (one region, the reference chain, no module)
needs of LED:

- `internal/ledger`: accounts and boundary accounts (LED 3), money and
  `Transfer` (4), lots, `Move`, creation and the stated lot key of
  construction (5.1, 5.2), ageing and `Expiring` (5.3), `Wear` and `WearItems`
  (5.4), reservations, earmarks and named lots (6.1 to 6.7), the built-in
  reason registry for the slice (7.1, 7.2), flow totals, wear units and the
  construction bucket (7.4), `Transform` (8), the audit of the identities of
  9.1 and 9.3 (12.4), and its sentinels.
- Claims (LED 10; CRD 3.1 to 3.3, 3.6): the `Claim` record and the open-claim
  list, `Lend`, `Settle`, `WriteOff`, `Reassign` and `Obligate`, the seven
  cumulative totals, `ErrOpenClaim`, and the claim identities of the audit.
  The credit and insurance modules that call them are Stage 3.
- The balance sheet of a non-boundary account (LED 9.2): money, goods valued
  at a caller's price table, claims held and owed, net worth, checked.
- `ledger.go`: the part of the root that needs no `World`: the conversion of
  the configuration into the ledger's primitives, the mapping between the
  root's `Reason` constants and the ledger's reasons, and the rows of
  `BoundaryFlow` built from the flow totals in the order of OBS 4.4.1.2.
- The glossary terms these files use, in `doc.go`, before they use them
  (architecture.md 3.2).
- The spec, note and roadmap edits those files depend on (Phase 1).

**Out of scope**, with where each is tracked:

| Item | Tracked in |
|------|-----------|
| `World.Audit`, `WithAudit` behaviour, `ErrAuditFailed`, the step positions P8.1, P8.2, P8.4 and P8.6, the emission of `BoundaryFlow`, the construction steps of WLD 5.1 and 5.2 | Roadmap Stage 1, `world.go` row. This plan gives them the functions they call |
| Opening and closing the accounts of agents and estates; the weight-change transfers of FR-LED-027 and FR-LED-051 | Roadmap Stage 1, `agent.go`, `estates.go` and `spawner.go` rows (LED 16: "the root package, in the specs that own them") |
| The public sentinels the root maps the ledger's errors to (N-075), and the mapping to rejection reasons (CLR 3.2) | Each with its first user (plan 0, Phase 3: "a sentinel with its first user"): the clearing, production and world rows |
| The credit and insurance modules that use the claims list: formation, interest, settlement and its pot, `ForgiveClaims`, the payout claim's rules, and the root declaring the three loan reasons and `MaxClaims` when the module is on | Roadmap Stage 3, `credit.go` and `insurance.go` rows |
| A reader of the balance sheet (the engine never reads one, LED 9.2) | Roadmap Stage 2, `indicator/` row |
| The room of a limited account (LED 6.8; FR-STO-005 to 010, 040) | Roadmap Stage 3, `storage.go` row |
| `Relocate` and the relocation totals (FR-LED-054) | Roadmap Stage 2, `regions.go` row |
| The account kinds Escrow, Transit, Outside and Treasury as anything but a number, and the reasons a module owns (`relief`, `want_consumption`, `deaths`, the escrow, wage, loan, transit, outside, rent, premium and `loss` rows, a money or fiscal flow ID); FR-LED-022, 050, 055 to 061, 065 | Each module's own PR, appended (architecture.md 2.10.5) |
| The checkpoint encoding of the ledger's state (LED 11; FR-LED-029, 030 beyond the hint rebuild) | Roadmap Stage 2, `checkpoint.go` row |
| The `Debug` line of a refused transfer (LED 12.2, a MAY) | The root call site that turns the refusal into a rejection (clearing row). `internal/ledger` takes no logger |
| The full-`Step` half of NFR-LED-001 | Roadmap Stage 1 "Done when" (the two allocation limits of the no-op path) |
| CI | Open item O3 of plan 0; the commands of `AGENTS.md` are run by hand |

## Decisions this plan rests on

Decided by documents already:

- `internal/ledger` is standard library only, never imports the root, and
  sees accounts, commodity indexes, reasons and numbers. It receives shelf
  life and durability as a table of primitives at construction (LED 2.7,
  architecture.md 2.9).
- Its homes are fixed by LED 16: the operations, the flow totals, the
  construction bucket, `Audit` and the sentinels in `internal/ledger`; the
  built-in registry a fixed table there; the reason names exported constants
  of the root (`event.go`).
- All state is slices of structs with no map, closure or package variable
  (LED 2.4; architecture.md 7.1, 7.5). Every addition and multiplication is
  checked through `internal/checked` (LED 2.5), which exists (`Add`, `Sub`,
  `Mul`).
- Every operation applies completely or changes nothing (LED 2.6).
- A violated internal invariant found outside an audit panics with a message
  that says it is a programmer error; the ledger logs nothing above `Debug`
  (LED 12.2, FR-LED-032; architecture.md 8.1).
- Slice only, appended later, and an append that is off leaves no trace (D3
  and D5 of plan 0; architecture.md 2.10.5).
- The flow totals are cleared at P8.6, after the events and the audit, not at
  the start of a tick (WLD 6.12 ruling C18).
- Bounds come from `bounds.go` (`MaxMoney`, `MaxQuantity`, `MaxCommodities`),
  passed in as parameters (architecture.md 9.6).

Decided by the owner on 2026-10-13, for this plan:

- **D6.** A reason a module owns has no row and no boundary account while the
  module is off. The built-in rows are those of LED 7.2 the slice can reach
  (A4), created in the order of 7.2, money before goods, goods in ascending
  commodity index, one account per commodity for a goods reason. This is the
  answer the owed question below asked for; its spec edit is Phase 1.
- **D7.** `Close` refuses an account that still holds something, in a fixed
  order of checks, with `ErrOpenReservation` (the name REG already uses for a
  relocation, N-075 allows it to be ledger-level) for a reservation,
  `ErrNotEmpty` (new, ledger-level) for money or goods, and `ErrOpenClaim`
  (LED 3.3, CRD 3.2.1) for a claim. FR-LED-009 and LED 3.3 are completed with
  these names in Phase 1.
- **D8.** The claims list and the balance sheet are built in this row, not
  left to Stages 2 and 3, so the checkpoint and the balance sheets do not
  change shape when credit arrives (LED 10).

- **D9.** The two recommendations of deliverable 1b are accepted: a sixth
  ledger operation `MarkOverdue(claim, tick)` sets `Overdue`, and `Settle`
  skips the transfer of an amount of 0 and is refused with `ErrInvalidAmount`
  when both are 0.

Resolved by D6, kept for the record (it was open item O1):

- **Which reasons have a boundary account in a world with every module off,
  and the creation order within the built-in group.** LED 3.4 says one
  boundary account per reason and per thing, WLD 5.2 says "the registry order
  of LED 7.2", and FR-LED-050 and 055 say the registry "MUST contain"
  `want_consumption`, `haul_use` and `transit_use`, while FR-LED-052 and
  FR-LED-057 say a module's reasons are declared when it is on. The answer
  fixes the account ID of every estate and agent, which the checkpoint saves.

## Assumptions this plan makes (flagged, not decided)

Sequencing and implementer-level calls. Challenge any of them in review.

- **A1.** The claims list is built with no module on, as LED 10 asks, and
  stays empty and unusable until a module sets a limit. `MaxClaims` is a
  constructor parameter (architecture.md 9.6) and 0 means no claim may be
  opened, so `Lend` and `Obligate` return `ErrTooManyClaims` in every world of
  Stage 1. The three loan reasons (`loan_out`, `loan_repay`, `loan_interest`)
  are declared rows passed to the constructor by the root only when the
  credit module is on (D6), so with them undeclared `Lend` and `Settle` return
  `ErrReasonMismatch` (A5). `Obligate`, `WriteOff` and `Reassign` need no
  reason.
- **A2.** The balance sheet takes the price table as a slice of `int64`
  indexed by commodity (a price of the caller's choosing, LED 9.2), and
  returns one value of plain integers. It refuses a boundary account.
  Sector totals are not a separate operation: the audit computes them (9.3).
- **A3.** `internal/ledger` names a reason by a small integer constant and
  holds no reason string. `ledger.go` maps the constant to the root's
  `Reason`. This keeps LED 2.7 ("knows no identifiers") and the "fixed table"
  of LED 16 together.
- **A4.** The built-in table holds the rows the slice can reach: `trade`,
  `initial` (money and goods), `recipe_input`, `recipe_output`, `disposal`,
  `consumption`, `spoilage`, `wear`, `estate_money`, `estate_goods`,
  `start_money`, `split`, `merge`, `births`, `starvation`. All seven account
  kinds are declared as constants in the order of LED 3.1, because the order
  is their number (CHK 4.2 tag 2); the pairing rules name Agent, Estate and
  Boundary only. The constructor takes a list of declared optional rows (A1
  uses it for the credit reasons; MON and the modules append theirs).
- **A5.** A reason value outside the table is refused with
  `ErrReasonMismatch` and does not panic.
- **A6.** The signatures of `internal/ledger` (how "from reserved" and an
  optional lot key are passed, the names of the tick-end and construction-end
  calls, the shape of the read accessors) are the implementer's, within LED
  and architecture.md 2.9. LED fixes the operation names, their refusals and
  the `Lot` struct.
- **A7.** The audit's per-tick identity (LED 9.1 item 3) is checked against a
  baseline of the boundary balances that the ledger records when the totals
  are cleared. LED 9.1 describes that baseline for a restore; it is one
  `int64` per boundary account.
- **A8.** `ledger.go` is tested from an internal test with a small
  configuration literal, as the options were in Stage 0 (A7 of plan 0), until
  `NewWorld` exists. It reads `Commodity.ShelfLife` and `Commodity.Durability`
  from `config.go` directly, so it does not wait for `commodity.go`.
- **A9.** One PR per phase, in phase order, on a branch from `stage-1-1`.
  Phases 2 to 4 may start while Phase 1 is in review; Phase 5 (claims) and
  Phase 7 do not start before Phase 1 is merged, because the first needs the
  claim operations settled there and the second creates the boundary
  accounts.

## Phase 1: settle the owed spec lines

**Why first:** the order in which boundary accounts are created is part of
every account ID. Phase 7 writes it into code. This phase changes a spec and
notes, so it starts by reading `doc/notes.md` in full and quoting its tag, it
handles the open notes tagged LED (N-021, N-107, N-125, N-130, N-159, N-209)
and `ALL`, and `doc/notes.md` gets a fresh tag (`AGENTS.md`).

Deliverables, each in the document that owns it:

1. **LED 3.4 and 7.2, with WLD 5.2 row 1** (D6). State, for a world with
   every module off: which rows are built in; that a reason a module owns has
   no row and no boundary account while the module is off (D5 applied to the
   registry, and what FR-LED-052 and FR-LED-057 already say); and the
   creation order inside the built-in group. Reword FR-LED-050 and
   FR-LED-055 to match, keeping their IDs (testing.md 4.1). Add a worked
   count of `B` for the pack `one`.
1a. **The refusal of a `Close`** (D7). Write the three names and their order
   into LED 3.3 and FR-LED-009.
1b. **Two lines the claim operations lack** (D9). (i) What sets `Overdue`: CRD 8.5
   has the settlement set it, and none of the five operations of CRD 3.2
   does. Recommended: a sixth operation `MarkOverdue(claim, tick)`, atomic,
   refused with `ErrUnknownClaim`, which changes nothing else; FR-LED-063 and
   CRD 3.2 then say six. (ii) `Settle` with one amount 0: recommended, the
   transfer of that part is skipped and both 0 is `ErrInvalidAmount`, because
   a transfer of 0 is refused by LED 4.2. Both are CRD's text to amend: edit
   CRD 3.2, FR-LED-063 and the CRD requirement rows that count the
   operations, and raise a note against CRD in the same change.
2. **A stage line in LED 16** for what the first build leaves out (the room
   and `Relocate`), pointing at the roadmap rows of
   the table above, so the spec and the code do not appear to disagree.
3. **The roadmap.** The Stage 0 rows read "done (uncommitted)" and commit
   `023cf97` holds them: set the commit. Link this plan from Stage 1, set the
   ledger row to **in progress**, add the claims list and the balance sheet to
   its description (D8), and record the items of the out-of-scope
   table that move a requirement of LED to another stage, here and in
   `vision.md` section 4 and `spec-roadmap.md` as the roadmap requires.
4. **Notes.** N-021 and N-075 updated with what this row applies and what
   stays owed (the public sentinels). A new note, tagged LED and CHK, that the
   ledger's reason constants are internal numbers, so a checkpoint must not
   save one raw. Any further gap found is a note in the same change.
5. **`doc/documentation_index.md`**: the row of this plan is already there;
   correct it if the phases change.

**Done (2026-10-13).** Deliverables 1 to 5 are applied, uncommitted, in the files they
name: `spec-ledger.md` 3.3.1, 3.4.1 to 3.4.3, 7.2.2, 10, 16.1 and the rows
FR-LED-009, 050, 052, 055 and 063; `spec-world.md` 5.2 rows 1 and 3a;
`spec-credit.md` 3.2, 3.2.1, 8.5 and FR-CRD-003, 032, 034; `roadmap-v1.md`,
`vision.md` section 4, `spec-roadmap.md`, `documentation_index.md`; and
`notes.md` (N-021, N-075, N-107, N-125, N-130, N-159, N-209 and N-217 updated,
N-016, N-027, N-103, N-126, N-128 and N-163 left open unchanged, N-225 to N-227 new, a
fresh tag). The count for the pack `one` is `B = 1 + 7 * 9 = 64`; the creation order
is `initial` (money), then commodity-major (the same shape as the `outside_*` goods
reasons), which is not the `BoundaryFlow` order. Two points go beyond the text of
this plan and are for the owner to confirm: `MarkOverdue` is also refused with
`ErrClaimShape` for a `tick` below the claim's `Due` (N-227), and the boundary
accounts of the module reasons `want_consumption`, `haul_use`, `transit_use` and
`loss` are given a position, row 3a of `spec-world.md` 5.2 (N-226).

**Tests:** none; documents only.

**Observability and rollback:** none, because no code changes. A wrong edit
is reverted like any document change; nothing depends on it until Phase 7.

**Acceptance:** LED states the built-in set and the creation order with the
count for `one`; FR-LED-050 and 055 agree with FR-LED-052; the roadmap rows
carry their status; the notes carry a fresh tag; O1 is closed in this plan.

## Phase 2: accounts, money, the registry and the flow totals

**Why here:** every later operation is a transfer between two accounts with a
reason, and every one adds to a flow total. Lots, wear and ageing reuse the
account table, the registry check and the totals built here.

- The package `internal/ledger` and its doc comment. The `Ledger` type, built
  by a constructor that takes the commodity table (`ShelfLife`, `Durability`
  per commodity index), the bounds and the number of regions, and returns an
  error for a table it cannot use.
- Accounts (LED 3): the seven kind constants (A4), `Open(kind, owner,
  region)` returning the next ID, `Close`, the closed flag, iteration in ID
  order. A boundary has no region and may go negative; no other account may.
- The registry (LED 7.1, 7.2): the reason constants and the fixed table of
  A4, each row with its kind (internal, faucet, sink), whether it moves money,
  goods or both, and the account kinds it may run between.
- `Transfer` (LED 4.2) with its six refusals, in a fixed order of checks so
  that one call has one answer. The marked form "from reserved" (4.3).
- `ReserveMoney` and `ReleaseMoney` (LED 6.1 to 6.3) with
  `ErrBoundaryReserve` and `ErrReservationExceeds`.
- Flow totals (LED 7.4): one preallocated slot per (region, reason, money or
  commodity) that the reason can move (a money-only reason has no goods
  slots), filled by plain integer additions, attributed by FR-LED-053. The
  layout is dense, as LED 12.3 asks; it is sized by the configuration, which
  is one region in the slice, and its size at many regions is measured in the
  `regions.go` row before anything is changed;
  the construction flag, the construction bucket (7.4.1) and the call that
  ends construction; the call that clears the totals.
- The sentinels this phase returns, each with its first user:
  `ErrInsufficientFunds`, `ErrBalanceOverflow`, `ErrReasonMismatch`,
  `ErrUnknownAccount`, `ErrClosedAccount`, `ErrInvalidAmount`,
  `ErrBoundaryReserve`, `ErrReservationExceeds`, and the two of D7 that this
  phase can return, `ErrOpenReservation` and `ErrNotEmpty`.
- `doc.go`: the glossary gains ledger, account, boundary account, transfer,
  reservation, flow totals, construction phase and construction bucket,
  before the code uses them; `notYetTerms` of the glossary test loses any it
  listed.

**Tests:** unit only, in the external package `ledger_test`; the package is
pure.

- `Transfer`: a table with one case per refusal that breaks only that rule,
  asserted with `errors.Is`, and the ledger compared before and after with
  `reflect.DeepEqual` (FR-LED-005, 006, 007). The boundaries: amount 1,
  exactly the available balance, exactly `MaxMoney`, one above; a boundary
  that goes negative; a boundary balance at the edge of `int64` (FR-LED-003,
  006).
- The registry: for every row and every ordered pair of the built kinds, the
  pairing is accepted or refused as the table of LED 7.2 says; the table is
  read from the spec by a plain text parse, as the catalogue tests of Stage 0
  read theirs (FR-LED-004, 037, 038). A `Guard:` test that every built row is
  a row of LED 7.2 and that the unbuilt ones are named in a `notBuilt` list
  in the staged form of architecture.md 9.7.
- Accounts: IDs in creation order, none reused after a close, a close refused
  while money or a reservation is held, an operation on a closed or unknown
  account refused (FR-LED-008, 009 in part).
- Reservations of money: reserve, release, over-reserve, over-release, a
  boundary cannot reserve, an unmarked transfer cannot spend an earmark, a
  marked one draws it down and fails if it is too small (FR-LED-017, 018,
  019).
- Flow totals: region attribution for a faucet, a sink and an internal flow;
  a move in construction is in the bucket and not in the totals; the bucket
  survives a clear (FR-LED-023, 045, 053).
- Zero total after every case of every table (FR-LED-002).

**Observability and rollback:** none, because the package logs nothing, emits
nothing and holds no global state; nothing imports it yet. Rollback is
reverting the PR.

**Acceptance:** `gofmt -l .` prints nothing; `go vet`, `staticcheck` and
`go test -race ./...` pass; the import guard and the determinism guard of
Stage 0 pass over the new package unchanged; `internal/ledger` is at 90%
statement coverage or more, read from `go tool cover -func=coverage.out`.

## Phase 3: lots, moving goods and goods reservations

**Why here:** it needs accounts, the registry and the totals. `Transform`,
wear and ageing (Phase 4) are all built from a move of lots in canonical
order.

- `Lot` as LED 5.1.1 gives it, and canonical form (5.1.3): merge equal keys,
  drop a zero quantity, sort by (commodity, born, durability). One internal
  function restores it, and every operation that changes lots ends in it.
- `Move` (LED 5.2.1, 5.2.2): oldest first, then lowest durability; the moved
  lots keep `Born` and durability; its refusals, `ErrInsufficientStock`
  among them. The marked form "from reserved".
- Creation (5.2.3): a move from a boundary gives `Born` the current tick for
  a commodity that spoils and the full durability for one that wears, 0
  otherwise.
- The stated lot key (5.2.4): accepted under `initial` in construction only,
  within the ranges of FR-LED-043, else `ErrInvalidLot` or
  `ErrNotConstruction`.
- `ReserveGoods` and `ReleaseGoods` as earmarks inside the lots (6.4), in
  canonical order.
- Named lots (6.7): the optional key on `ReserveGoods`, `ReleaseGoods` and
  `Move`, with `ErrUnknownLot`.
- `Close` also refuses an account that holds goods.
- `doc.go`: the glossary gains lot, born, earmark and, if absent, durable.

**Tests:** unit.

- Canonical form: the same holdings reached by two different sequences of
  moves are `reflect.DeepEqual`; a commodity that neither spoils nor wears is
  one lot (FR-LED-010, 011).
- `Move`: a table whose expected lots carry their arithmetic in a comment
  (testing.md 5.5): oldest first, then most worn; a split of a lot; a move
  that empties a lot; age and wear travel (FR-LED-012). One case per refusal,
  each leaving the ledger unchanged (FR-LED-005, 006, 007).
- Creation: born in the tick, full durability, the boundary more negative by
  the quantity (FR-LED-013, 038).
- The stated key: each edge of both ranges accepted, one outside each
  refused, a stated key after construction refused, a negative `Born` kept in
  canonical order (FR-LED-043, 044 in part).
- Earmarks: taken in canonical order; an unmarked move skips them and fails
  when only earmarked units are left; a marked move consumes them; a release
  returns them (FR-LED-019, 034). Named lots: exactly that lot, too few
  units, no such lot (FR-LED-033).

**Observability and rollback:** none, for the reasons of Phase 2. Rollback is
reverting the PR.

**Acceptance:** the commands of Phase 2 pass; coverage holds at 90%.

## Phase 4: transform, wear, ageing and the end of a tick

**Why here:** each is a composition of Phase 3's moves with one of the four
reasons the ledger itself uses (`recipe_input`, `recipe_output`, `wear`,
`spoilage`).

- `Transform(account, inputs, outputs)` (LED 8.1, 8.2.1): all or nothing, an
  empty side allowed, both empty `ErrEmptyTransform`.
- `WearItems(account, commodity, n, units)` and `Wear` as its case `n = 1`
  (5.4): the `n` most worn available items selected before any is worn,
  excess not carried, a destroyed item moved to `wear`, survivors re-merged,
  the wear units actually removed counted by commodity.
- Ageing (5.3.1, 5.3.2): accounts in ID order, lots in canonical order, every
  expired lot whole to `spoilage`; the earliest-expiry hint per account, kept
  up to date by every operation that adds a lot, and a function that rebuilds
  the hints from the lots (the restore path of FR-LED-030).
- `Expiring(account, commodity, tick)` (5.3.3), read-only, using the hint.
- The tick: the counter the ages are measured against, and the two calls the
  world will make at P8.1 (age) and P8.6 (advance the tick, then clear the
  totals and the wear units).

**Tests:** unit.

- `Transform`: inputs and outputs applied together; a failing output (an
  overflow in the account) leaves the inputs untouched; each empty side;
  both empty (FR-LED-021, 039). Inputs are taken oldest first and outputs
  born as in 5.2.3.
- Wear: the most worn item first; destruction at exactly 0 and below 0; no
  carry to the next item; `n` items over several lots with no item worn
  twice; survivors merged into the lot of their new durability; an earmarked
  item never worn; each refusal leaves the ledger unchanged; `Wear` equals
  `WearItems` with `n = 1` on a copy; the units counted are the durability
  removed (FR-LED-015, 016, 040, 041, 042).
- Ageing: a lot made in tick `t` with shelf life `S` leaves at the end of
  tick `t + S - 1` and not before; a negative `Born`; a commodity that does
  not spoil never leaves; the order of the moves is the same on every run
  (FR-LED-014, 044). The hint: an account whose hint is in the future is not
  visited, shown by a rebuilt-hints comparison after every operation of a
  scripted run (NFR-LED-002, FR-LED-030 in part).
- `Expiring`: 0 for a commodity that does not spoil and for a past tick; the
  earmarked units left out; the sum over commodities for the current tick
  equals what ageing then moves, when nothing is earmarked; the two account
  errors (FR-LED-048, 049).
- The tick end: the totals of a call made after the clear are in the next
  tick's totals (WLD 8.1.3).

**Observability and rollback:** none, for the reasons of Phase 2. Rollback is
reverting the PR.

**Acceptance:** the commands of Phase 2 pass; coverage holds at 90%.

## Phase 5: claims and the balance sheet

**Why here:** claims are transfers of money plus a record, so they need
Phases 2 to 4's accounts, registry and `Close`. The audit of Phase 6 checks
them, so they come first. It needs Phase 1's amendments (1b, D9).

- The `Claim` record exactly as CRD 3.1.1 gives it, the open-claim list in
  ascending ID, the next claim ID, `MaxClaims` as a constructor parameter (A1)
  and the seven cumulative totals of CRD 3.3.2.
- `Lend`, `Settle`, `WriteOff`, `Reassign` and `Obligate` (CRD 3.2), each
  atomic and each with the refusals of its row; `MarkOverdue` (D9). A claim with nothing owed is removed at once (CRD 3.1.2). Interest
  is paid before principal, with `loan_interest` then `loan_repay`.
- The loan reasons as declared rows (A1): pairings per LED 7.2 and
  FR-LED-062 (`loan_out` agent to agent; `loan_repay` and `loan_interest`
  agent to agent or to an Estate account).
- `Close` refuses an account that is the creditor or debtor of an open claim,
  `ErrOpenClaim` (D7, CRD 3.2.1).
- The balance sheet (LED 9.2, A2): money, goods at the supplied prices,
  claims held and owed (the amount owed, undiscounted), net worth; every sum
  through `internal/checked`; refused with `ErrUnknownAccount`,
  `ErrClosedAccount` or the boundary refusal.
- The sentinels, each with its first user: `ErrUnknownClaim`,
  `ErrClaimShape`, `ErrTooManyClaims`, `ErrOpenClaim`.
- `doc.go`: the glossary gains claim, creditor, debtor and balance sheet.

**Tests:** unit.

- `MarkOverdue`: sets the field and changes nothing else; an unknown claim
  is refused. `Settle` with interest 0 or principal 0 skips that transfer;
  both 0 is `ErrInvalidAmount` (D9).
- One case per refusal of each operation in the CRD 3.2 table that breaks
  only that rule, asserted with `errors.Is`, the ledger compared with a copy
  (FR-LED-063, FR-CRD-004). The total of money over non-boundary accounts is
  equal before and after every operation (FR-LED-063).
- `Lend`: the transfer and the claim together, with the next ID; a refused
  `Lend` leaves neither (the overflow of `principal + interest`, the debtor
  at `MaxMoney`). `Settle`: interest first, a partial payment, both parts
  paid and the claim gone, an amount above the owed. `Obligate` moves no
  money and counts in `Awarded`. `WriteOff` returns what was owed and counts
  in `LostPrincipal` and `LostInterest`. `Reassign` to an Estate account only.
- The totals identity of CRD 3.6 item 1 after every case of a table, by exact
  equality, with `Lent`, `Awarded`, `Repaid`, `LostPrincipal`,
  `InterestFixed`, `InterestPaid` and `LostInterest` (FR-LED-028; FR-CRD-004
  and 008 in part: IDs ascending and never reused).
- `Close` with each of the four things held, in the order of D7; no new
  account is opened to carry a claim (FR-LED-009).
- Balance sheet: a hand-computed case with money, two commodities, a claim
  held and a claim owed, the arithmetic in a comment (testing.md 5.5); an
  overflowing valuation refused, not wrapped; a closed, unknown and boundary
  account (FR-LED-026).

**Observability and rollback:** none, for the reasons of Phase 2: the
operations are returned results and the credit module's own events are
Stage 3's. Rollback is reverting the PR.

**Acceptance:** the commands of Phase 2 pass; coverage holds at 90%.

## Phase 6: the audit, the invariant tests and the cost

**Why here:** the audit checks the whole of what Phases 2 to 5 built, and the
invariant tests need every operation to exist before they can drive them at
random.

- `Audit` (LED 12.4, 9.1, 9.3): the zero total of money and of each
  commodity; no negative non-boundary balance or quantity; reserved at most
  the balance; canonical form; conservation by reason over the tick against
  the baseline of A7; the sectoral identity over the four sectors by a
  kind-to-sector table (households, public, contract, treasury; a term is
  zero until a module opens an account of its kind); the four identities of
  CRD 3.6 and the principal identity with `Awarded` (LED 10), naming the ID
  of the first claim that fails (FR-LED-064); the construction bucket while
  in construction (FR-LED-046). It returns an error that names the identity and the account
  or commodity of the first failure, and changes nothing.
- A read that reports whether any reservation is held, for the world's check
  at a tick boundary (FR-LED-020).
- The panic of a violated internal invariant, with a message that says it is
  a programmer error (FR-LED-032).

**Tests:**

- Unit: one case per identity, each on a ledger whose state the test breaks
  directly from an internal test (the only way to reach an inconsistent
  state), asserting the identity and the account or commodity named
  (FR-LED-035 in part). Audit leaves the ledger `reflect.DeepEqual` to a copy
  (FR-LED-036 in part).
- Invariant (testing.md 6.3), still at the unit level because one package is
  under test: a scripted sequence of several thousand operations drawn from a
  test-local `math/rand/v2` source with a fixed seed, over a range of seeds,
  the failing seed in the message (testing.md 6.2). After every operation the
  audit passes and the lots are canonical; after every refusal the ledger
  equals its copy; at every tick end no reservation is held once the script
  has released them (FR-LED-001, 002, 003, 007, 011, 024, 025 in part).
- The invariant test and `FuzzOps` include the claim operations, with a
  `MaxClaims` above 0 and the loan reasons declared in the test.
- `FuzzOps`: a byte string decoded into operations, asserting the same
  properties and that nothing panics. Not required by testing.md 7.4 (the
  ledger parses no external input); it is here because atomicity is the
  property a hand-written table misses. Corpus in
  `internal/ledger/testdata/fuzz/`.
- Cost, as ordinary tests with `testing.AllocsPerRun` (testing.md 8.2): a
  transfer, a move between existing lots, a reserve and release, a transform
  in a steady state, a `Settle` that does not remove its claim, an ageing
  pass with nothing due, and `Expiring` allocate nothing (NFR-LED-001 in part, NFR-LED-004).
- Benchmarks with `b.ReportAllocs()` and `b.Loop()`: transfer, move, ageing,
  transform, `WearItems`, `Expiring` (NFR-LED-003, 004). They report; they
  gate nothing.

**Observability and rollback:** none emitted. The audit is the ledger's one
diagnostic and is a returned error, never a log line (architecture.md 8.3).
Rollback is reverting the PR.

**Acceptance:** the invariant test passes over its seed range; `FuzzOps` runs
with its seed corpus and for five minutes by hand with no crasher (any
crasher is committed with a `Fix:` comment, testing.md 7.5); every allocation
test passes at zero; coverage holds at 90%.

## Phase 7: `ledger.go` and the exit gate

**Why last:** it writes the creation order of Phase 1 into code and checks the
row as a whole. It needs nothing of the other Stage 1 rows (A8).

- `ledger.go`, unexported, converting inline at the call (architecture.md
  2.9):
  - the ledger built from a validated `Config`: the commodity table from
    `Commodity.ShelfLife` and `Durability` in commodity-ID order, the bounds
    from `bounds.go`, and the boundary accounts of the built-in reasons in
    the order Phase 1 fixed;
  - the mapping between the ledger's reason constants and the root's
    `Reason`, both ways;
  - the rows of `BoundaryFlow` for a set of totals: faucet and sink reasons
    only, non-zero only, ascending region, then reason in the order of LED
    7.2, money before goods, ascending commodity, written into a slice the
    caller owns so that the world can reuse it (OBS 4.4.1; FR-LED-031 in
    part). The same function serves the construction bucket with `Tick` -1.
- `event.go`, appended: the `Reason` constants of the internal reasons of A4
  (`trade`, `estate_money`, `estate_goods`, `start_money`, `split`, `merge`,
  `births`, `starvation`), which LED 16 homes there.
- The exit gate, on the whole module: `gofmt -l .`, `go vet ./...`,
  `staticcheck ./...`, `go test -short -race ./...`, the full run with
  `-covermode=atomic -coverpkg=./...`, `go mod tidy`; the per-scope coverage
  read and recorded; the traceability check of testing.md 4.6 by text search
  (every ID in a `Requirement:` line exists in LED, and every requirement the
  table below lists as covered in full is cited).
- The roadmap row set to **done** with its commit. N-021 and N-075 updated.

Not gating, worth doing if time allows: a mutation pass over
`internal/ledger` (testing.md 9.4, a SHOULD), with `gomu` inside Docker only,
`--cpus=4 --memory=16g --memory-swap=16g`, `GOMAXPROCS` and `--workers` equal
to the CPU cap.

**Tests:**

- Unit, internal test (A8): the commodity table equals the configuration's;
  the boundary accounts are created in the order of Phase 1, with the count
  LED now states for the pack `one`; every built reason maps to a `Reason`
  and back, and every faucet or sink `Reason` constant of `event.go` has a
  ledger row (a `Guard:` test, so a constant cannot be added without a row).
- Unit: the eight new constants join the table of the `Reason` string test
  in `event_test.go`.
- Unit: the `BoundaryFlow` rows of a scripted set of totals, in order, with
  internal reasons and zero totals left out, `NoCommodity` for money, and the
  bucket form with `Tick` -1; the function allocates nothing when the slice
  it is given has room (`testing.AllocsPerRun`).
- No integration or end-to-end test: there is no `World`. The integration
  tests that cite FR-LED-031, 035, 036, 045, 046 and 047 through the public
  API are the world row's (testing.md 2.3), and the table below says so.

**Observability and rollback:** the checklist of
observability-and-logging.md section 9 is answered in the PR. No log line and
no metric is added. No event type is added; `BoundaryFlow` exists since Stage
0 and is still emitted by nothing. Rollback is reverting the PR; nothing
outside `ledger.go` calls it until the world row.

**Acceptance:** every command of the gate is clean; the boundary-order test
passes against the count in LED; the traceability check finds no ID without a
spec row and no "covered in full" requirement without a test; the roadmap row
reads done.

## Security review

The module is a library with no network, no file access, no credentials and
no persistence. This row adds no trust boundary: `internal/ledger` is called
only by the root, with numbers the root derived from a validated
configuration and from the tick.

- **Who can reach it.** A consumer reaches the ledger only through the world
  (later rows): orders, `AddAgent`, the slots. Each of those paths validates
  first and turns a ledger refusal into a rejection or an error. Nothing here
  is exported from the module.
- **Integer overflow.** Every sum and product goes through
  `internal/checked`; a result above `MaxMoney` or `MaxQuantity`, or outside
  `int64` for a boundary, is refused and changes nothing (FR-LED-006). A wrap
  would be a silent wrong result; the boundary cases of Phases 2 and 3 and
  the invariant test guard it.
- **Partial application.** A refused operation that had already moved
  something would create or destroy money. Atomicity (LED 2.6) is tested per
  refusal and by `FuzzOps`.
- **Panics.** The ledger panics only on an internal invariant that no input
  can cause (architecture.md 8.1). An out-of-range account, reason or
  commodity is a returned error (A5), never an index panic; the tests include
  negative and too-large indexes.
- **Resource exhaustion.** The lots of an account are bounded by its
  commodities times the distinct (born, durability) pairs, which the
  configuration bounds (`MaxCommodities`, shelf life, durability). Ageing
  visits only accounts with a lot due. `WearItems` works per lot, not per
  item, so a large `n` costs no more than the account's lots. Closed slots
  are never compacted (LED open question 6); the growth is one slot per
  removal and is CHK's to revisit.
- **Not applicable:** authentication, authorisation, injection, secrets,
  cross-tenant exposure, supply chain (no dependency is added).

## Compatibility and rollout

- **No code contract exists to break.** There is no release and no consumer.
  Nothing in this row is exported from the module: `internal/ledger` is
  internal and `ledger.go` adds unexported functions only. The eight `Reason`
  constants appended to `event.go` are new exported identifiers; they are
  appended, not inserted (architecture.md 2.10.5).
- **Document contracts.** Phase 1 rewords FR-LED-050 and 055 and renumbers
  nothing (testing.md 4.1). `doc/notes.md` gets a fresh tag.
- **What becomes hard to change, and when.** The creation order of the
  boundary accounts and the order of the `BoundaryFlow` rows enter the golden
  files at the end of Stage 1 and the checkpoint in Stage 2. Until the first
  golden file is recorded they may be corrected in a normal PR. The ledger's
  reason constants are internal numbers and stay free to change, provided a
  checkpoint never saves one raw (the note of Phase 1).
- **Append-only stays true.** A module adds its reasons, kinds and
  operations in its own PR. With the module off it adds no row and no
  boundary account (Phase 1), so `testdata/golden/off/` does not move.
- **Rollout** is one PR per phase (A9). Each leaves the branch passing every
  command of `AGENTS.md`. A phase is rolled back by reverting its PR; later
  phases depend only on earlier ones, and no other row of Stage 1 has started.
- **Dependency failure.** None: no dependency, no run-time service.
- **The Go toolchain.** `go.mod` says `go 1.25`; the checks run on the
  installed toolchain. Nothing here depends on a version-specific order: no
  unstable sort is used where equal keys can occur, since a lot key is unique
  within an account (architecture.md 7.6).

## Testing (summary, testing.md section 11)

The pyramid for this row is all base: unit tests in `internal/ledger` and in
the root's internal test of `ledger.go`, one invariant test and one fuzz
test over the package, and allocation tests. There is no integration or
end-to-end test because there is no world; the world row adds them.

| Phase | Covered in full | Covered in part (completed in) |
|-------|-----------------|--------------------------------|
| 2 | FR-LED-004, 005, 008, 017, 018, 023, 037, 053 | FR-LED-002, 003, 006, 007 (money here, goods in Phases 3 and 4); FR-LED-009 (the claim case in Phase 5); FR-LED-019 (money here, goods in Phase 3); FR-LED-038 (money here, goods in Phase 3); FR-LED-045 (the bucket here, its checkpoint half in Stage 2); FR-LED-052 (the built-in rows; the declared rows with MON) |
| 3 | FR-LED-010, 011, 012, 013, 019, 033, 034, 038, 043 | FR-LED-044 (ageing in Phase 4, the checkpoint in Stage 2) |
| 4 | FR-LED-014, 015, 016, 021, 039, 040, 041, 042, 048, 049; NFR-LED-002 | FR-LED-030 (the hint rebuild here, the restore in Stage 2) |
| 5 | FR-LED-009, 026, 028, 062, 063 | FR-LED-052 (the declared loan rows; the root declares them with the credit module) |
| 6 | FR-LED-001, 002, 003, 006, 007, 024, 046, 064; NFR-LED-003, 004 | FR-LED-020, 035, 036 (the ledger's half; `World.Audit`, `WithAudit` and the boundary check in the world row); FR-LED-025 (all four sectors by kind; the accounts of the contract and treasury kinds are opened by their modules); FR-LED-032 (the panic; "no log above `Debug`" holds by the package taking no logger); NFR-LED-001 (per operation; a full `Step` in the world row) |
| 7 | none | FR-LED-031 (the rows and their order; the emission in the world row); FR-LED-052 (the creation order of the built-in boundary accounts) |

"Covered in full" means at the lowest level that can verify it (testing.md
2.1). The second half of testing.md 2.3, the same behaviour through the public
API at the integration level, is the world row's for every consumer-visible
requirement above: the invariants of testing.md 6.3 over the reference chain
and a range of seeds (FR-LED-002, 003, 024, 025) are in Stage 1's "Done
when".

Deferred with their code: FR-LED-027 and 051
(`spawner.go`, `scale.go`); FR-LED-047 (needs a job, `production.go`);
FR-LED-029 (CHK); FR-LED-054 (REG); FR-LED-022, 050, 055 to 061, 065 and the
FR-STO and FR-INS IDs of LED 13 (their modules).

Boundaries and error paths: every sentinel has a case that breaks only its
rule and is asserted with `errors.Is` (testing.md 3.9). Golden files: none
exists yet. Fuzz: `FuzzOps`. Each PR states its coverage and that its tests
were run against a broken version (testing.md 9.3).

## Observability (summary, observability-and-logging.md section 9)

None added, because: a ledger refusal is a returned error and is handled once
by its caller (1); the ledger's transitions are reported in aggregate by
`BoundaryFlow`, which exists and whose rows Phase 7 builds, and by the events
of the specs that ask for the transfer (2); the ledger makes no decision an
operator needs explained (3); its loops log nothing and build no event (4);
no event, metric, unit or attribute changes (5, 6); neither `Error` nor
`Info` is used (7); no error is logged and returned (8); the tests of section
8 that apply here are the allocation tests of Phases 5 and 6, and the rest
need a world (9).

## Cross-plan remarks (added by the consistency review, 2026-10-13)

Additive; no phase, decision or built code above is changed.

- **`Discard` and `Holding`.** [plan-phase-1-5.md](plan-phase-1-5.md) Phase 2
  (D54, D58) adds two operations in new files of `internal/ledger`
  (`discard.go`): `Discard(account, commodity, quantity)`, create under
  `recipe_output` and sink under `disposal` with no room check, and
  `Holding(account, commodity) (held, reserved, err)`, which plan 1-3 also
  uses. They edit no file of Phases 2 to 7 here. The audit of Phase 6 needs no
  change: a `Discard` moves both boundary accounts and both flow totals by the
  same quantity, so the zero-total and by-reason identities hold; plan 1-5
  adds its audit assertion once Phase 6 has merged. An optional cross-reference
  to `Discard` in the doc comment of `Transform` is welcome when Phase 4 is
  closed.
- **Names other rows read.** Plans 1-3 and 1-5 call the operations by the names
  in the tree (`Transfer`, `TransferReserved`, `ReserveMoney`, `ReleaseMoney`,
  `ReserveGoods`, `ReleaseGoods`, `ReserveLot`, `ReleaseLot`, `Move`,
  `MoveReserved`, `MoveLot`, `MoveLotReserved`, `MoveStated`, `AppendLots`,
  `Account` with `Balance` and `Reserved`, `Boundary`, `AppendFlows`, `Tick`,
  `AdvanceTick`, `ClearTotals`) and, for plan 1-5 Phase 6, by the A6 shapes of
  `Transform` and `WearItems`; a rename is a compile error there, so rename
  only with the same change to the plans' contract tables.
- **Reason constants of Phase 7.** The eight constants are appended to
  `event.go` before those of plan 1-4 Phase 5 (belief reasons) and plan 1-5
  Phase 5 (start refusals) if this row merges first, and their Go names and
  strings are distinct from theirs. The `Guard:` test of Phase 7 ("every
  faucet or sink `Reason` constant has a ledger row") is over the group of
  ledger-reason constants, so the non-ledger `Reason` values those rows append
  need no ledger row.
- **After Phase 6.** Once Phase 6 has landed, the owner of this plan folds
  `Holding` and `Discard` (plan 1-5 Phase 2, built there by ruling) into the
  package doc, `FuzzOps` and the audit tests (coordinator, 2026-10-13).
- **Decision IDs.** D6 to D9 here are this plan's; plan 0 has D1 to D5 and
  ADR 0008 has its own D1 to D10, which other plans cite with the file name.
  Cite these as "D6 of plan-phase-1-1".

## Open items

- **O1.** Closed (2026-10-13). D6 to D9 are decided and their spec edits are made
  (Phase 1): the built-in rows, the creation order and the count `B` are in
  `spec-ledger.md` 3.4 and 7.2.2, the refusal order of `Close` in 3.3.1, and
  `MarkOverdue` and the `Settle` of 0 in `spec-credit.md` 3.2. Phases 5 and 7 and
  `Close` no longer wait for a document.
- **O2.** N-075 stays open: which ledger sentinels become public errors of
  the root is decided with each first user, not here.
- **O3.** No CI exists (O3 of plan 0). The gate of Phase 7 and the timed fuzz
  run are done by hand, on one architecture.
