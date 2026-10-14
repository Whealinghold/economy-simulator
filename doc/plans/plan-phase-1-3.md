# Plan: Stage 1, row 3: clearing

**Status: Proposed (2026-10-13); no phase started.** Sequences the third row of
Stage 1 of [roadmap-v1.md](../roadmap-v1.md), "`internal/clearing` and
`clearing.go` (`Clearing`, `AverageClearing`, admission, reservation,
settlement) with `WithClearing`, and the name check `ErrUnknownClearing`", into
six phases. It implements [spec-clearing.md](../specs/spec-clearing.md) (CLR),
the registration rules of `spec-world.md` 4.1 to 4.3 (WLD) and the name rule of
`spec-config.md` 7.1.4 and 7.4.1 (CFG) for the slice. It is built against
`doc/rules/architecture.md` (the `internal/` boundary of 2.9, the extension
rules of 4.3 to 4.5 and 7.7, the fixed-point and overflow rule of 3.5, the
tie-break rule of 7.6, the module-file rule of 2.10), `doc/rules/testing.md` and
`doc/rules/observability-and-logging.md`. Where this plan and the rules or the
specs disagree, they win and this plan is corrected.

A plan decides nothing. Every decision below comes from a spec, a rule, an ADR
or the decisions of [plan-phase-0-foundation.md](plan-phase-0-foundation.md)
(D2, D3 and D5 above all). Where a decision is still owed, Phase 1 makes it in
the document that owns it, before the code that depends on it, and where it is
not this row's to make it is an open item or a question to the coordinator.

No PRD exists for this row. The requirements source is the roadmap row, CLR,
and the parts of WLD and CFG named above. The row is built after the ledger row
([plan-phase-1-1.md](plan-phase-1-1.md), in progress: money and goods
operations are in the tree) and after, or beside, the goods and role tables of
[plan-phase-1-2.md](plan-phase-1-2.md). It needs no `World`: every function
here is tested against a real `internal/ledger` and a hand-built book.

Spec codes: CLR clearing, LED ledger, WLD world, CFG config, OBS
observability, BEL beliefs, AGT agents, EST estates, STO storage, ORG
institutions, FIS fiscal, CHK checkpoint, SCL scale. `FR-`/`NFR-` IDs are the
requirement IDs of those specs.

## What Stage 0 and the ledger row already give

| Concern | Where it is |
|---------|-------------|
| `Price`, `Quantity`, `Money` with checked `Price.Times(Quantity)`, `Add`, `Sub`; `CommodityID`, `RegionID`, `AgentID` | `value.go` |
| `MaxPrice`, `MaxQuantity`, `MaxMoney` (the largest trade is `MaxMoney`) | `bounds.go` |
| `World.Clearing` (default `"average"`, hashed as a name, no check in `Validate`) | `config.go`, `defaults.go`, `hash.go`, the `registry_test.go` row |
| The events `OrderPlaced`, `OrderRejected`, `TradeExecuted`, `MarketCleared`, `Side`, the rejection `Reason` constants of the slice (the ledger's `trade` reason is the internal `ledger.Trade`; its root `Reason` constant is appended by plan 1-1 Phase 7 and this row does not use it) | `event.go` |
| `Observer`, the delivery loop with the "supplied" flag, `discardObserver` | `observer.go` |
| `Option`, `options`, `applyOptions`, `WithLogger`, `WithObserver`, `WithAudit`, `WithTargetSpread`, `WithSeed` | `option.go` |
| The keyed stream, `siteClearing` and the pair key `(tick, region, commodity)` | `random.go` (`keyedPair`, `word`) |
| `ErrDuplicateID`, `ErrInvalidID`, `ErrOutOfBounds`, `*ValidationError` with `Path`, `Rule`, `Value` | `errors.go`, `validate.go` (`validID`) |
| `checked.Add`, `Sub`, `Mul`, `MulDiv*` including `MulDivHalfEven` | `internal/checked` |
| `ledger.Trade`, `TransferReserved`, `ReserveMoney`, `ReleaseMoney`, `ReserveGoods`, `ReleaseGoods`, `ReserveLot`, `ReleaseLot`, `MoveReserved`, `MoveLotReserved`, `Key`, `Account` | `internal/ledger` (uncommitted; plan-phase-1-1 A6 leaves the names to its implementer) |

What is left for this row is the rule of the market: the matching algorithm, the
admission checks, the check of a list of trades, the market report, the three
public types `Order`, `Trade` and `Clearing` with its default, the registration
option and name check, and the composition that turns a book into trades and
ledger moves and events.

## Scope

**In scope.**

- `internal/clearing` (stdlib and `internal/checked` only, no random source, no
  root type): `Offer`, `Match`, `Average` (CLR 4, 5.1, 5.2, 8.3); the ordered
  admission checks as a pure function over primitive facts (CLR 3.2); the check
  of a list of trades (CLR 8.2); the market report (CLR 5.4, 5.5).
- `internal/checked`: one 128-bit accumulator for the volume-weighted mean
  (CLR 5.4), so no second copy of the 128-bit helper appears (N-019).
- `order.go`: `Order` and `Trade`. `clearing.go`: `Clearing`, `AverageClearing`
  and its assertion; the unexported book, admission, reservation, the trade-list
  check, settlement, release and the per-market step that emits the four events
  and the two `Debug` lines (CLR 3, 6, 8, 9). `option.go`: `WithClearing`,
  the registered-name list, and the name check the world calls. `errors.go`:
  `ErrInvalidTrade`, `ErrNilExtension`, `ErrUnknownClearing`.
- The glossary terms these files use, in `doc.go`, before they use them
  (architecture.md 3.2).
- The spec, note, rule and roadmap edits these files depend on (Phase 1).

**Out of scope**, with where each is tracked:

| Item | Tracked in |
|------|-----------|
| `World`, `NewWorld` step order, `Step`, P3.2 plan execution, P4.1 iteration over markets, `ErrWorldFailed`, `TickEnded` with `Failed`, the agent table and the `AgentID` to account lookup | Roadmap Stage 1, `world.go` row. This plan gives it `clearMarket`, `book.admit` and `resolveClearing` |
| Holding the per-market state across ticks (`marketReport` per market, the price window ring), the seed price, pushing the window at P5.1, `World.Market` | Roadmap Stage 1, `world.go` row (`market.go`); the ring is BEL's. This plan defines the report type and computes it (D35) |
| The outcome per order and `Revise` (FR-CLR-025), the window push (FR-CLR-032) | `world.go` row (the outcome loop and the push; plan 1-4 gives it `Gate`, `Change`, `Tally` and `Window`); `Revise` itself is the agent row's |
| How an agent sizes an order, `not_on_role`, the strategy's tradable-set refusal while the world builds orders (FR-CLR-030) | `strategy.go` row; this plan takes the tradable fact as an input |
| The estate's asks, the pricing of lots (FR-CLR-029 needs only the lot key) | `estates.go` row |
| The room: `no_room` and the settlement of a bid from reserved room (CLR 3.2, 6 step 5; FR-STO-008, 009) | Roadmap Stage 3, `storage.go` row |
| `not_permitted` and the legal clamp (CLR 3.2.1, FR-CLR-035; FR-ORG-024) | Roadmap Stage 3, `institutions.go` row |
| Pseudo-agents other than the estate: the counterparty and the treasury (their sides and one bid plus one ask) | `external.go` and `fiscal.go` rows; the admission facts below carry "side allowed" and the duplicate key already names the side |
| The checkpoint encoding of the market report (FR-CLR-034) | Roadmap Stage 2, `checkpoint.go` row (N-125) |
| The full-`Step` half of NFR-CLR-002 and the clearing half of the Stage 1 "Done when" invariants | `world.go` row and Stage 1 "Done when" |
| CI | Open item O3 of plan 0; the commands of `AGENTS.md` are run by hand |

## Decisions this plan rests on

Decided by documents already:

- Admission and matching are separate. Admission is the same for every
  `Clearing`; matching is the extension (CLR 2.3). The world validates the whole
  list of trades before it settles any, and an invalid list fails the step with
  `ErrInvalidTrade` (CLR 8.2, WLD 3.3.3).
- `AverageClearing` takes one `uint64` per order from the `*rand.Rand` it is
  given, bids first then asks, each in input order, before sorting; the internal
  package takes the keys and no random source (CLR 4.2, 8.3; WLD 7.3 row D7).
  The stream is keyed by `(siteClearing, tick, region, commodity)` by the
  world; nothing here reseeds it (ADR 0008).
- The order is a total order ending in the input position (architecture.md
  7.6), the trade price is the midpoint of the two limits rounded half to
  even, and the published price is the volume-weighted mean rounded half to
  even computed in 128 bits (CLR 5.1, 5.2, 5.4; architecture.md 3.5).
- A bid reserves at its limit, an ask reserves its quantity; settlement is a
  ledger transfer and move marked from reserved with the reason `trade`; the
  unspent part of the bid's reservation is released (CLR 3.3, 6; LED 4.3, 6.3).
- `internal/clearing` defines its own small structs and the root converts
  inline at the call (architecture.md 2.9). The internal package does not know
  what an agent, a commodity or a lot is: it sees opaque numbers.
- `Clearing` is one of the four extension interfaces; it ships with a default
  and a compile-time assertion, and its doc comment states the randomness and
  `Stateful` contracts (architecture.md 4.3, 4.5, 7.7).
- A registration name is hashed and its code is not; `average` is built in;
  a name given twice, or a built-in name, is `ErrDuplicateID`; a nil extension
  is `ErrNilExtension`; an unknown or empty configured name is
  `ErrUnknownClearing` as a `*ValidationError` with the path of the field,
  checked in `NewWorld` after `Validate` succeeds (WLD 3.2.2, 4.1, 4.3; CFG
  7.1.4, FR-CFG-038, FR-CFG-081).
- With no observer supplied no event is built; with the default logger a
  `Debug` line costs nothing; the clearing path emits no `Info`, `Warn` or
  `Error` (CLR 9.2; observability-and-logging.md 4.5; FR-CLR-033).

Decisions of this plan (D30 to D39; D30, D34 and D36 were answered by the
coordinator on 2026-10-13, as marked; the others are sequencing-level and open
to challenge):

- **D30. File homes.** CLR 15 puts `Order`, `Trade`, `Clearing`,
  `AverageClearing` in `order.go` and `clearing.go` and admission, reservation
  and the price in `market.go`; the roadmap puts admission, reservation and
  settlement in `clearing.go` and gives `order.go` and `market.go` to the world
  row. This row builds `order.go` (the two types the extension signature needs)
  and puts admission, reservation, settlement and the market report *type and
  computation* in `clearing.go`; the world row's `market.go` holds the report
  *state* per market and the window ring and calls these functions. Phase 1
  edits the roadmap rows and CLR 15 to say so. Answered by coordinator
  2026-10-13 (Q2, option a): `market.go` stays whole with the world row (the
  report state, the window ring, the market per region and commodity); this
  row builds `order.go` (`Order`, `LotKey`, `Trade`, because the `Clearing`
  interface cannot exist without them) and the report type and its
  computation in `clearing.go`. The world-row plan must drop `order.go`.
- **D31. The shape of `internal/clearing`.** CLR 8.3 gives `Offer{Ref, Price,
  Qty, Key}`, `Match{BidRef, AskRef, Price, Qty}` and `Average(bids, asks)` and
  says the final form is decided with the code. This row fixes: `Average(dst
  []Match, bids, asks []Offer) []Match` sorts the two slices in place (they are
  scratch the root owns) and appends to `dst`; `Offer` gains the opaque fields
  `Owner int64` and `Lot` (a flag and two `int64`) that `Check` needs and
  `Average` ignores; reasons of rejection are small integer codes the root maps
  to its `Reason` constants, as the ledger's are (plan-phase-1-1 A3). `Check`
  takes caller-owned scratch (`sums`, `index`) so it allocates nothing and
  needs no scan of the matches per fill: `Check(dst []Match, sc Scratch,
  commodity int64, bids, asks []Offer, fills []Fill) ([]Match, Code)`, with
  `Scratch{Sums []int64; Index []int32}` (Phase 3). Phase 1
  writes it into CLR 8.3. The example of architecture.md 2.9 omits the key and
  the random source; that rule edit is not made here: it is recorded in N-235
  for the one combined architecture.md edit (coordinator, 2026-10-13).
- **D32. `Order`, `Trade`, `Clearing` shapes.** `Order{Agent AgentID, Commodity
  CommodityID, Side Side, Price Price, Quantity Quantity, Lot LotKey}` and
  `Trade{Commodity CommodityID, Buyer, Seller AgentID, Price Price, Quantity
  Quantity, Lot LotKey}`; `LotKey` is a plain comparable struct `{Named bool;
  Born, Durability int64}` whose zero value names no lot (CLR 3.1, 3.1.2: a
  trade says which ask it filled, so it carries the ask's lot). `Clearing`
  is `Clear(bids, asks []Order, rng *rand.Rand) []Trade` (CLR 8, 8.1), as the
  spec sketches; `AverageClearing` is the empty struct with a value receiver.
  The list a `Clearing` returns is read, not owned, by the world.
- **D33. Settlement is a sequence of ledger operations after a complete
  check.** For each trade in order: `TransferReserved(buyer, seller, p*q,
  trade)`, `ReleaseMoney(buyer, (limit-p)*q)` only when `(limit-p)*q` is above
  0, then `MoveReserved` (or `MoveLotReserved` for a named lot) of `q` units
  seller to buyer. The ledger refuses an amount below 1 (`ErrInvalidAmount`
  from `ReleaseMoney`, `ReleaseGoods` and `ReleaseLot`), and 0 is a normal
  value here (a trade at the bid's own limit; a fully filled order), so **every
  release is skipped when its amount is 0**: the per-trade money release, and
  at the end of the market the remaining money of each bid, the remaining units
  of each ask and the remaining units of each named-lot ask. A refusal after
  the whole list has passed the check is a violated invariant and panics with a
  message that says so (architecture.md 8.1, LED 12.2); admission and the check
  make it unreachable (D34). At the end of the market the remaining
  reservation of every order that has one is released. All of it happens at
  P4.1 and no reservation is open after it (WLD 6.1.6). Settlement uses the
  book's commodity for the goods it moves, never `Trade.Commodity` (Phase 3,
  sixth failure of the check), and reads the admitted records of the book, never
  the `[]Order` slices handed to `Clear` (CLR 8.2 "as admitted").
- **D34. Admission closes the two overflow cases of settlement.**
  Answered by coordinator 2026-10-13 (Q1, option a; MON is not touched), with
  the bid half corrected by the audit of the same day. A trade moves money into
  the seller and goods into the buyer, and the ledger refuses a balance above
  `MaxMoney` or a stock above `MaxQuantity`; CLR 2.3 says settlement cannot
  fail but CLR 3.2 tests only `limit * quantity` against `int64`. A bid is
  rejected with `overflow` when the buyer's holding of the commodity plus this
  quantity exceeds `MaxQuantity`, tested as `quantity > MaxQuantity - held`
  (the subtraction form). An agent has at most one bid per commodity in a book
  and a second is `duplicate_order`, a check that runs before `overflow`, so
  no per-agent tally of bids exists: the single-order test is already the
  whole rule. An ask is rejected with `overflow` when the seller's balance plus
  the worst-case receipt of all its asks already admitted in the round (any
  market, any commodity or lot: one agent has many asks) plus `quantity *
  MaxPrice` exceeds `MaxMoney`. This one is cumulative: the receipts per ledger
  account live in a small `receiptTally` slice that the world row owns, clears
  at the start of P3.2 and passes in. The sums are computed in the form
  `MaxMoney - balance - tally` compared with `quantity * MaxPrice`, using
  `checked.Add`/`checked.Mul` where a sum is formed (never a naive `+`: a tally
  over many asks can exceed `int64`, architecture.md 13), and a negative
  headroom means rejected. No new reason; `overflow` is widened to "the order
  or what it would settle does not fit" and the doc comment of `ReasonOverflow`
  in `event.go` and the parse of CLR 3.2 in `TestConstants_Values` are edited
  with it (Phase 5). With this, `ErrBalanceOverflow` at settlement is a
  violated invariant and panics with a message that says it is a programmer
  error (architecture.md 8.1), never an input-derived path. Phase 1 writes the
  rule into CLR 3.2 and FR-CLR-007 (the bid half as the single-order check
  `held + quantity > MaxQuantity`, the ask half as the cumulative receipt).
- **D35. The market report is computed here and held by the world.**
  `internal/clearing.Report` takes the offers, the filled units per order, the
  trades, the previous price and age and returns the published price, age,
  volume, unmatched units and best unmatched quotes with their flags (CLR 5.4,
  5.5). The Go type `marketReport` holds exactly the fields of CLR 16 (the
  checkpoint's list) and is defined in `clearing.go`; the world row stores one
  per (region, commodity), seeds it with the seed price and age 0, and the
  checkpoint row saves it (FR-CLR-034).
- **D36. One registration list per extension kind, and no map.** `options`
  holds the clearings as a slice of `{name, Clearing}` searched linearly
  (architecture.md 7.5 forbids a map only where iteration order matters; a
  slice keeps the order given and allocates once). The name check is an
  unexported function `resolveClearing(o *options, name string, path []string)
  (Clearing, error)` that `NewWorld` calls once; it returns the built-in for
  `"average"`. Names follow `validID` of `validate.go`
  (A3); a non-conforming name to `WithClearing` is `ErrInvalidID`. Answered by
  coordinator 2026-10-13 (Q3, option a): the rule is written once in WLD 4.1
  for all four registration options (a name satisfies `validID`, else
  `ErrInvalidID`; a duplicate or built-in name is `ErrDuplicateID`; a nil
  extension is `ErrNilExtension`); only `WithClearing` is built here, with the
  shared `registry[T]`, which the strategy and spawner rows reuse unchanged.
- **D37. Events and logs are emitted by the market step, not by admission.**
  `clearMarket` takes the `*observers` and the `*slog.Logger` of the world and
  emits, in this order, the `TradeExecuted` events in trade order and then
  `MarketCleared`; `book.admit` returns the reason and the caller (the world
  row at P3.2, in the order submitted) emits `OrderPlaced` or `OrderRejected`
  through the same helpers, which this row builds as small unexported methods
  (`emitPlaced`, `emitRejected`). Every event is built only when
  `observers.supplied`; every `Debug` line is guarded by `Enabled`.
- **D38. Reason of the first failing check is the one reported.** The ordered
  table of CLR 3.2 is one function whose rows are numbered in the order of the
  table, with the positions of the module rows marked by a comment and no code.
  How a module row is reached from it is not decided here, because the order
  of the table is the requirement (FR-CLR-031) and cannot be met by appending,
  while architecture.md 2.10.2 and 2.10.4 allow a module outside its file only
  in the catalogue files and the four fan-ins: it is open item O2, settled by
  the first module PR that needs a row, with the rule files.
- **D39. A book belongs to one market.** The book is built with its region and
  commodity. The world routes every order by its commodity, so an order of
  another commodity cannot come from input: `book.admit` **panics** on it, with
  a message that says it is a violated programmer invariant (architecture.md
  8.1). An order whose agent is in another region is input-derived and is the
  returned rejection `unknown_agent` (FR-CLR-003), never a panic. FR-CLR-001 is
  then a property of the type, tested with two books: the panic and the
  rejection are the two asserted behaviours.

## Assumptions this plan makes (flagged, not decided)

Sequencing and implementer-level calls. Challenge any of them in review.

- **A1.** `Average` is an `O(n log n)` sort-and-merge over two slices with the
  comparator `(price desc or asc, key, position)`; it uses `slices.SortFunc`
  because the comparator is a total order (architecture.md 7.6). No other
  matching structure is built (NFR-CLR-001 is a SHOULD). `Check` is `O((n + f)
  log n)` for `n` admitted orders and `f` fills, by a sorted index in caller
  scratch and a binary search per fill (Phase 3); it is not a scan of the
  matches per fill.
- **A2.** `AverageClearing.Clear` holds no state and allocates its offer
  scratch and its result (the `Clearing` interface returns a fresh `[]Trade`,
  so an implementation of the extension point cannot be made allocation-free).
  Everything *else* in the market step uses scratch owned by one unexported
  `clearScratch` struct (defined in `clearing.go`): the slices `bids`, `asks`
  (the book split by side, in admission order), `offers`, `matches`, `fills`,
  `lots`, and the two scratch slices of `Check`: `sums` (`[]int64`, the units
  filled per admitted order, bids then asks) and `index` (`[]int32`, the
  sorted lookup index of the two sides). The world row owns one `clearScratch`
  per world, passes it to `newBook`, and every market step resets the lengths
  to the number of admitted orders and reuses the capacity (spec 9.3 "scratch
  slices held by the world"). `Check` and `Report` take caller-owned `dst`
  and scratch slices and write only to them (D31). `sums` is the one
  per-order tally: `Check` fills it, and `Report` and `book.filled()` read it. The
  zero-allocation test uses a hand-written `Clearing` that returns a
  preallocated slice, so it measures `clearMarket` and not the implementation;
  `AverageClearing.Clear` itself is held by a baseline constant for one `Clear`
  of a small book, recorded in a test beside a benchmark, and the world row's
  full-`Step` baseline is the NFR-CLR-002 gate.
- **A3.** The registration name rule is the ID rule `validID` (lowercase
  letters, digits and `_`, at most `MaxIDLength`), because `World.Clearing` is
  a configuration name and the loader's tables are keyed by such names. No spec
  states it for registrations; D36 (coordinator) makes it the rule for all four.
- **A4.** The ledger operations are called as they stand in the tree on
  2026-10-13: `Transfer`, `TransferReserved`, `ReserveMoney`, `ReleaseMoney`,
  `ReserveGoods`, `ReleaseGoods`, `ReserveLot`, `ReleaseLot`, `Move`,
  `MoveReserved`, `MoveLot`, `MoveLotReserved`, `Account` (a `View` with
  `Balance` and `Reserved`), `AppendLots`, the `Key{Commodity, Born,
  Durability}` and the reason `ledger.Trade`. plan-phase-1-1 A6 leaves the names
  to its implementer, so a rename is a compile error here, not a silent one.
  The read of one account's held and reserved units of a commodity is
  **resolved**: [plan-phase-1-5.md](plan-phase-1-5.md) Phase 2 (D58) adds
  `Holding(account, commodity) (held, reserved, err)`, zero allocation, in new
  files of `internal/ledger`, and Phase 5 uses it for an ordinary ask and for
  the buyer's stock. `Holding` returns commodity totals only, so the **facts of
  an ask that names a lot** (FR-CLR-029) come from a lot-level read, which is
  permanent and not a fallback: `AppendLots(scratch[:0], account)` into
  `clearScratch.lots`, find the `Key{Commodity, Born, Durability}`, and
  `available = Quantity - Reserved`; an absent lot is zero available, so the
  ask is `insufficient_stock` (a nonexistent lot is not a panic at
  `ReserveLot`). `AppendLots` is therefore a permanently consumed symbol. If
  plan 1-5 prefers a one-lot read in the ledger, that is its proposal to make;
  this row needs no more than `AppendLots`. If `Holding` has not merged,
  `AppendLots` also serves the ordinary reads. Still ASSUMED: the
  ledger `Audit` (plan-phase-1-1 Phase 6).
- **A5.** Tests of the root pieces run from an internal test (package
  `economysim`) with small configuration literals and a real `ledger.Ledger`,
  until `NewWorld` exists, as plan 0 A7 and plan-phase-1-1 A8 did.
- **A6.** One PR per phase, in phase order, on a branch from `stage-1-1`.
  Phases 2 and 3 depend only on Phase 1 and may be built in parallel; Phase 4
  needs Phase 2; Phase 5 needs Phases 3 and 4 and the ledger's goods and
  reservation operations, waits for plan 1-5 Phase 2 (`Holding`; or uses the
  `AppendLots` fallback) and, for the audit assertion of its invariant test and
  of `FuzzBook` only, for plan 1-1 Phase 6 (`Audit`). The row shares `doc.go`, `doc/notes.md`, the
  roadmap and `event_test.go`/`glossary_test.go` with the other rows: each PR
  re-reads the shared file before editing it.
- **A7.** `ErrNilExtension` is declared here, being the first row that needs
  it; the strategy and spawner rows reuse it and the small registration helper
  of D36 is written so that they can (a second copy would be a defect).

## Phase 1: settle the owed spec lines

**Why first:** the shapes of `Order`, `Trade` and the internal structs, the
overflow rule of admission and the file homes are what Phases 2 to 5 write into
code, and two of them (the file homes and the overflow rule) touch other
rows. This phase changes specs, notes and the roadmap (no rule file: N-235), so it
starts by reading `doc/notes.md` in full and quoting its tag, it handles the
open notes tagged CLR and ALL (the disposition table below), and
`doc/notes.md` gets a fresh tag (`AGENTS.md`).

Deliverables, each in the document that owns it:

1. **CLR 8 and 8.3 and CLR 15** (D30, D31, D32): replace "a proposed shape" by
   the shapes of D32 and the `internal/clearing` signatures of D31; correct the
   sentence that names a second reserved name "`default`" (WLD 4.1 and
   FR-WLD-002 reserve only `average` for the clearing; CLR 8 is made to agree);
   correct the home table of 15 (the types in `order.go`; admission,
   reservation and settlement in `clearing.go`; the report state in the world
   row's `market.go`); correct the CLR 15 row that names "midpoint, half-to-even
   division, 128-bit volume-weighted mean, checked multiply" as one home in the
   checked-arithmetic helpers: `Midpoint` is in `internal/clearing` (Phase 2),
   the half-to-even division and the 128-bit accumulator are in
   `internal/checked` (Phase 3), as CLR 8.3's last sentence says. No `architecture.md` edit is made in this
   phase: the 2.9 example fix (the key and the random source) and the module
   rows of the admission table (D38, O2) are recorded in N-235, which holds
   them for one combined edit with a fresh RULE-TAG before the first Stage 2
   module PR (`demand.go`).
2. **CLR 3.2 and FR-CLR-007** (D34): the answer to Q1, as a clause of the
   `overflow` row and of the requirement, with its worked case. The bid clause
   is the single-order check `held + quantity > MaxQuantity` (no per-agent bid
   tally: one bid per commodity and book, a second is `duplicate_order`); the
   ask clause is the cumulative worst-case receipt across the round. The
   clause also says when D34 holds: only while no other credit reaches the
   seller's balance or the buyer's stock between P3.2 and P4.1. A later module
   that credits in that window (credit P3.3b, insurance P3.3c, fiscal) feeds
   the `receiptTally` or keeps its own headroom; this phase adds that line to
   N-235 so the module PRs see it, and to O2 below.
3. **WLD 4.1 and 4.3, CFG 7.1.4** (D36, A3): the one rule for all four registration
   options, written once in 4.1: a name satisfies `validID` (`ErrInvalidID`), a
   duplicate or built-in name is `ErrDuplicateID`, a nil extension is
   `ErrNilExtension`; CLR 8's second reserved name `default` is removed (WLD
   reserves `average` only) and a note is raised for the CLR and WLD owner (**N-229**, reserved for this plan by the
   cross-plan review: areas CLR and WLD; it is not in `doc/notes.md` yet, so
   this phase creates it, after re-reading the index for the next free ID and
   with a fresh notes tag); the wording "an empty name is
   `ErrUnknownClearing`" is read as the configured name, in 4.1 as in CLR 8.
4. **The roadmap, `vision.md` section 4 and `spec-roadmap.md`** (D30): Stage 1
   prose links this plan; the clearing row's Create cell gains `order.go` and
   the report type; `order.go` is moved out of the world row's Create cell and
   its `market.go` says "the state of each market's report and the price
   window"; the move is recorded in `vision.md` section 4 and `spec-roadmap.md`
   as the roadmap's "Updating this file" requires, but neither file names
   `order.go` or splits the clearing row today. The implementer re-reads both
   and writes only what is true there: if a file lists files per row, it moves
   `order.go` to the clearing row in the same way; if it does not, nothing is
   written there and this deliverable is closed with that sentence in the PR.
   The row's status stays **planned** until Phase 2 starts. The two Create
   cells of `roadmap-v1.md` were already edited by the cross-plan consistency
   review (2026-10-13).
5. **Notes** (the table below), with one new note if a gap appears.
6. **`doc/documentation_index.md`**: the row of this plan.

**Tests:** none; documents only.

**Observability and rollback:** none, because no code changes. A wrong edit is
reverted like any document change; nothing depends on it until Phase 4.

**Acceptance:** CLR 8, 8.3, 15 and 3.2 state the shapes, the homes and the
overflow rule; WLD 4.1 and CLR 8 agree on the built-in name; the roadmap rows
agree with CLR 15; the notes carry a fresh tag and the dispositions below;
`go test -short ./...` still passes after the spec edits (the tests read the
specs: `TestConstants_Values` parses CLR 3.2, and the registry and catalogue
tests read other documents).

### Notes read for this plan

Rule tags: architecture=905c0e578c4e662908638dd8,
observability=748e621d4300d088eb469349, testing=b3559bc6d47470c221bc4642.

Notes tag: the tag at the end of `doc/notes.md` when the implementer starts
(read the file again and quote that one; `6cc73d3abf2fd311fb034a21` was current
on 2026-10-13 and moves with every note edit). Open notes tagged CLR: N-113, N-125,
N-137, N-185, N-214. Tagged ALL: N-016, N-017, N-027, N-103, N-126, N-128,
N-163, N-217, N-235.

| Note | Disposition |
|------|-------------|
| N-113 A quote is a limit, not a peg | **Leave open.** The midpoint rule is kept and `Order` gets no "trade at my own limit" flag; STY-T03 and the external row's S51 decide. Learned: if the flag is ever added it is one field of `Order` and one branch of the price in `Average`, so the shapes of D31 and D32 leave room |
| N-125 A snapshot is a function of saved state | **Apply in part.** `marketReport` holds exactly the fields of CLR 16 and nothing else (D35), so a checkpoint saves it whole and a snapshot derives from it. Stays open for CHK and OBS |
| N-137 Room is committed at the decision | **Leave open.** `no_room` is STO's; admission keeps its place in the table (D38) and `deferredReasons` of `TestConstants_Values` keeps `no_room` listed |
| N-185 One order per agent, commodity, named lot and side | **Apply for CLR.** The duplicate key of `book` is (agent, commodity, lot, side) (FR-CLR-004); the one-bid-one-ask treasury case is tested with two sides of one agent. CLR leaves the areas; FIS keeps the validation (`ErrFiscalShape`) |
| N-214 Keyed random streams | **Apply for CLR.** D7 is one word per order from the market's stream; Phase 4 pins it with the vector of WLD 7.3 (seed 42, tick 7, markets `(r0, grain)` and `(r0, loaf)`). ADR 0008 open point 2 (a stateless key) is answered by WLD 7.1 and 7.3 and not reopened. CLR leaves the areas |
| N-016 Keep the parameter registry true | **Leave open.** No field is added; the `World.Clearing` row exists and the registry test line stays as it is |
| N-017 Words and names | **Apply.** Terms added to `doc.go` before use: market, order, bid, ask, limit price, trade, admission, reservation, available, tie-break key, crossing, market price, price age, unmatched, market report, clearing. `notYetTerms` of `glossary_test.go` loses `Market`, `Order`, `Trade`, `Clearing`. "Treasury" and "pool" are used in the senses of N-163 only |
| N-027 No default macro constant is stable | **Leave open.** This row sets no default and asserts no number of the economy |
| N-103 Running several spec writers in parallel | **Leave open.** A process note; not for this row. The shared files are re-read before each edit (A6) |
| N-126 A new event or `Warn` adds a row | **Leave open.** No event or `Warn` is added: the four events are declared since Stage 0 and this row first emits them; the staged catalogue test is not changed here |
| N-128 A hosted per-agent field is saved once | **Leave open.** Not applicable: the report is per market |
| N-163 The word "treasury" | **Leave open.** Code comments say "pool" for the region's estate account and "treasury" only for the account kind |
| N-217 D5 binds every module's change | **Apply.** No configuration field, no event field and no off-golden row moves; `TestHashConfig_ReferencePinned` stays as it is. Stays open as a standing rule |
| N-235 One combined architecture.md edit | **Leave open.** This plan adds D31 (the example of 2.9 omits the key and the random source) and D38 (the module rows of the admission table) to it, and the D34 window line of deliverable 2; it makes no `architecture.md` edit |

N-075 (mapping the ledger's sentinels, areas CFG) gets an update: this row maps
`ErrUnknownClearing`, `ErrNilExtension` and `ErrInvalidTrade` as root sentinels
and maps no ledger sentinel to a public error, because every ledger refusal at
settlement is an invariant (D33, D34).

## Phase 2: the matching in `internal/clearing`

**Why here:** everything else in the row either feeds this function or reads
its result, and it has no dependency on the ledger or the root.

- The package `internal/clearing` and its doc comment (stdlib and
  `internal/checked`; no `rand`, no `time`).
- `Offer`, `Match` and `Average(dst, bids, asks)` (D31): sort the bids by
  limit descending, the asks by limit ascending, ties by `Key` then by position
  in the input (the position is a field, `Ref`, so the comparator is total);
  walk with two cursors; a pair trades while the best bid's limit is not below
  the best ask's; the quantity is the smaller remaining; a trade at the
  midpoint of the two limits (CLR 4.1 to 4.4, 5.1).
- `Midpoint(a, b)`: `a + (b-a)/2`, and the half-to-even correction when
  `b-a` is odd (CLR 5.2), stated in its doc comment (architecture.md 3.5).
- The same-owner guard is not in `Average` (a treasury never matches itself by
  the integers of its limits, CLR 3.1.2); the check of Phase 3 stays the guard.

**Tests:** unit, in the external package `clearing_test`; the package is pure.

- The worked example of CLR 11 as one table case with its arithmetic in a
  comment (testing.md 5.5): three trades (2 at 9, 1 at 10, 2 at 10) and the
  stop at 7 against 9 (FR-CLR-009, 011, 012, 013).
- Empty and one-sided books: no bids, no asks, both empty, and a book with
  one side only give no matches and consume no key (zero draws are the
  root's, Phase 4), and a single bid against a single ask that do not cross
  gives none (FR-CLR-009, 012; testing.md 3.8).
- Midpoint: a table over even and odd differences, equal limits, neighbours
  that make the half-to-even choice go down and up (10.5 gives 10, 9.5 gives
  10), and the largest limits (`MaxPrice`, `MaxPrice-1`) (FR-CLR-013, 014).
- Order: equal limits ordered by key then position; equal keys ordered by
  position; the same input in two permutations of equal-limit offers with the
  same keys gives the same matches (FR-CLR-010, architecture.md 7.6).
- Property, with a test-local `math/rand/v2` source, fixed seeds, the failing
  seed in the message (testing.md 6.2): units matched equal the units that
  cross computed independently from the sorted limits (FR-CLR-015); every
  match price lies between its two limits (FR-CLR-014); the units bought equal
  the units sold and no offer is filled beyond its quantity (FR-CLR-017); no
  match has a bid limit below its ask limit (FR-CLR-009).
- Pairing: over many key draws, with two bids and two asks all at one limit,
  each position pairs first about equally often, within a stated tolerance, a
  `Guard:` for STB-L12 (the tolerance is the one place a tolerance is allowed:
  it is a statistic over keys, not money, testing.md 6.3).
- A benchmark over books of 10, 100, 1 000 and 10 000 offers with
  `b.ReportAllocs()` and `b.Loop()` (NFR-CLR-001, NFR-CLR-003); it gates
  nothing.

**Observability and rollback:** none, because the package logs nothing, emits
nothing and holds no state; nothing imports it yet. Rollback is reverting the
PR.

**Acceptance:** `gofmt -l .` prints nothing; `go vet`, `staticcheck`,
`go test -race ./...` and `go mod tidy` pass; the import guard and the
determinism guard of Stage 0 pass over the new package; coverage of
`internal/clearing` is at least 90% from `go tool cover -func`.

## Phase 3: admission, the trade-list check and the report

**Why here:** these are the other three rules of the internal package, and the
root pieces of Phases 4 and 5 call them.

- **Admission checks.** `Admit(f Facts) Code`, `Facts` being the primitive
  values the root resolves for one order: whether the agent is known and in the
  region, whether its side is allowed, whether the commodity is known, whether
  it is in the agent's tradable set, whether this (owner, lot, side) already
  has an order, the limit, the quantity, the bounds (`MaxPrice`, `MaxQuantity`
  and `MaxMoney` are passed in, architecture.md 9.6), the available money or
  stock, and, for D34, the buyer's stock (the bid's single-order check) and the
  seller's balance with its receipt tally (the ask's cumulative check). It returns
  the code of the first failing row of CLR 3.2 in the order of the table
  (D38), or 0 for admitted. The ask's receipt tally of D34 is given in the facts (the
  `receiptTally` holds it; the function adds nothing to a tally, the caller
  does after an admission); the headroom is computed as `MaxMoney - balance -
  tally` with checked arithmetic. The product `limit * quantity` is `checked.Mul`.
  The two module-owned rows (`not_permitted`, `no_room`) and the legal clamp
  have their positions marked and no code (D38).
- **The check of a list** (CLR 8.2): `Check(dst []Match, sc Scratch,
  commodity int64, bids, asks []Offer, fills []Fill) ([]Match, Code)`, `Fill`
  carrying the trade's commodity, the buyer and seller owners, the ask's lot,
  the price and the quantity; it resolves each fill to its bid and its ask by
  (owner) and (owner, lot), and fails with a code for: a commodity other than
  the book's (CLR 8.2 "in this market"), no bid or no ask, a quantity below 1,
  a summed quantity above an order's, a price outside `[ask limit, bid
  limit]`, the same owner on both sides: six failures. `bids` and `asks` are
  the book's own admitted records (limits as admitted, after the clamp), never
  the slices the `Clearing` was given. It appends the matches (the references)
  to `dst`, and writes only to `dst` and to the caller-owned scratch `sc`.
  **Scratch and lookup** (A2): `Scratch{Sums []int64; Index []int32}`, each of
  length `len(bids) + len(asks)` (the caller sets the lengths from
  `clearScratch`; `Check` zeroes `Sums` itself, so a warmed scratch needs no
  reset by the caller). `Sums[i]` is the units filled of order `i` (bids first,
  then asks, in the order given), and is the one per-order tally that `Report`
  and `book.filled()` read afterwards. `Index` holds the positions of each
  side sorted by (owner, lot flag, born, durability), which is a total order
  because admission allows one order per (owner, lot, side) (FR-CLR-004), so
  `slices.SortFunc` needs no tie-break (architecture.md 7.6); a fill is
  resolved by binary search on it. The cost is `O((n + f) log n)`, with no scan
  of the matches per fill, and `Check` allocates nothing with a warmed scratch.
  On a failure `Check` returns `dst` with no match appended (its length as
  passed) and the code; `Sums` is then meaningless.
- **The report** (CLR 5.4, 5.5): `Report(...)` takes the units filled per
  order that `Check` left in `Sums` (no second tally), and computes the volume-weighted
  mean of the trade prices in 128 bits, rounded half to even; the previous price
  carried when nothing traded; the age (0 after a tick with units, else one
  more); the unmatched units per side and the best unmatched limit with a
  presence flag, from the units filled per order.
- **`internal/checked`**: an accumulator of 128-bit products, `AddMul(a, b)`,
  and `DivHalfEven(c) (int64, bool)`, with tests; the clearing package imports
  it and keeps no 128-bit code of its own.

**Tests:** unit.

- Admission: a table with one case per reason that breaks only that rule,
  asserted by code; a case failing two rules gets the first (FR-CLR-031); the
  boundaries: price 0, 1, `MaxPrice`, `MaxPrice+1`; quantity 0, 1,
  `MaxQuantity`, `MaxQuantity+1`; a product of exactly `MaxMoney` accepted and
  one that overflows `int64` rejected, nothing wraps (FR-CLR-002, 003, 004, 005,
  006, 007, 028, 030); the bid case of D34 at its edge (`held + quantity` exactly `MaxQuantity`
  admitted, one more `overflow`, in the subtraction form) and the ask case at
  its edge, again with a receipt tally already in the facts, and with a tally
  large enough that a naive sum would exceed `int64`; a bid and an ask
  of one agent are not a duplicate, two bids are (FR-CLR-004, N-185).
- The check: each of the six failures breaks only that rule (the sixth: a
  trade naming another commodity than the book's, otherwise valid); a valid list
  passes and returns the references; a fill against a named lot resolves to the
  ask of that lot and not another ask of the same owner (FR-CLR-024, 018, 029
  in part). Fills given in any order, several fills of one order that sum to
  exactly its quantity (valid) and to one more (the fourth failure), and an
  empty list (valid, no match) are cases. `Sums` after a valid list equals the
  per-order sums computed independently in the test. With a warmed `Scratch`
  and `dst`, `testing.AllocsPerRun` over a valid and an invalid list is 0
  (testing.md 8.2; observability-and-logging.md 1.3). A benchmark of `Check`
  over 10, 100, 1 000 and 10 000 orders with `b.ReportAllocs()` shows the
  growth beside it (reports, gates nothing).
- The report: the example of CLR 11 gives 48 over 5 units, `9.6`, rounded to
  10; a tie that rounds to the even neighbour; a price below the lowest or
  above the highest trade price is impossible, as a property over seeds; the
  sum of products that exceeds `int64` (many trades at `MaxPrice` and
  `MaxQuantity`) gives the exact mean; no trade carries the previous price, and
  the first tick with none carries the seed passed in; age after no trade, a
  trade, no trade (FR-CLR-019, 020, 021). The "none" cases are explicit: an
  empty book, a book with bids only and one with asks only report no best
  bid or ask (flag false, not 0) and the unmatched units of an empty side as
  0; a market with every order filled reports no best on that side
  (FR-CLR-021 in full).
- `internal/checked`: the accumulator against `math/big` in a test (the one
  place `math/big` is allowed in a test, as a reference oracle; architecture.md
  3.5 bans it from the code, not from a test), at the edges of 128 bits, and a
  fuzz target `FuzzAcc` (the package already fuzzes `MulDiv`).
- **Fuzz targets for the two validators** (testing.md 7.4: every function that
  validates external input; `Admit` validates a consumer's orders and `Check`
  validates the list a consumer's `Clearing` returns, which is untrusted). Each
  has seeds by `f.Add` (the boundary values of the tables above) and its
  corpus in `internal/clearing/testdata/fuzz/<FuzzName>/`; a crasher is
  committed with a `Fix:` comment in the fuzz function (testing.md 7.5).
  - `FuzzCheck`: bytes decode to a small book (orders with owners, lots and
    limits, one order per (owner, lot, side)) and an arbitrary list of fills
    (any commodity, owner, lot, price, quantity, including values outside the
    book). Properties: it never panics; a list it accepts satisfies every
    CLR 8.2 property, checked by an independent plain loop in the test (each
    buyer holds a bid and each seller an ask of that lot, each quantity is 1 to
    the order's, summed per order within the order's, each price within the
    two limits as given, buyer and seller differ, and the commodity is the
    book's); a list it rejects returns `dst` unchanged in length and a code
    that is the failure the independent loop finds first; the result does not
    depend on the order of the fills (FR-CLR-024, 014, 018).
  - `FuzzAdmit`: bytes decode to `Facts` (including every bound and the
    receipt tally and holdings at the D34 edges). Properties: it never panics;
    code 0 only when an independent restatement of every row of CLR 3.2 holds
    (product within `int64` and `MaxMoney`, `held + quantity <= MaxQuantity`,
    the ask's headroom, funds, stock); a nonzero code is the first failing row
    of the table in its order; nothing wraps (FR-CLR-002 to 007, 028, 030,
    031).

**Observability and rollback:** none, for the reasons of Phase 2. Rollback is
reverting the PR.

**Acceptance:** the commands of Phase 2 pass; coverage of `internal/clearing`
and `internal/checked` holds at 90%; `FuzzAcc`, `FuzzCheck` and `FuzzAdmit`
each run five minutes by hand with no crasher (a crasher is committed with a
`Fix:` comment, testing.md 7.5); the `Check` allocation test is 0.

## Phase 4: the types, the extension, the registration and the name check

**Why here:** the public shapes and the option are what the world row and the
strategy and spawner rows read; the draw of the keys is the one place the
root meets the random stream.

- `order.go`: `Order`, `Trade`, `LotKey` (D32), each exported identifier with
  its doc comment.
- `clearing.go`: the `Clearing` interface with its doc comment stating that the
  `rng` is the world's, that an implementation must not create its own source,
  use a global one or keep it after `Clear` returns, and that one that keeps
  state implements `Stateful` and one instance is not shared between worlds
  (architecture.md 7.7; CLR 8.1); `AverageClearing` and `var _ Clearing =
  AverageClearing{}` (architecture.md 4.5); `Clear` converts the orders to
  offers inline, draws one word per order (`rng.Uint64()`), bids first then asks
  in input order, calls `clearing.Average`, and converts the matches to trades
  with the lot of the ask (architecture.md 2.9). It keeps no state, so it does
  not implement `Stateful` (CLR 8).
- `errors.go`: `ErrNilExtension` (first user `WithClearing`) and
  `ErrUnknownClearing` (first user `resolveClearing`), each with its first user
  (plan 0: a sentinel with its first user), and the rows of `errors_test.go`.
  `ErrInvalidTrade` is declared in Phase 5 with its first user, `clearMarket`,
  so the Stage 0 guard on sentinels with no returner needs no exception.
- `option.go`: `WithClearing(name, c)`: additive; `nil` is `ErrNilExtension`; a
  name that is `average` or already given is `ErrDuplicateID`; a name that is
  not a valid ID is `ErrInvalidID` (D36, subject to Q3); a registration is
  stored in `options` in the order given. The shared piece is one small
  unexported generic container `registry[T]` (a slice of name and value;
  `add` applies the nil, built-in and duplicate rules, `find` is a linear
  search) in `option.go`, so that the strategy, spawner and policy rows do not
  write three copies (A7); it is the one use of generics that architecture.md
  4.6 allows (a small container). `resolveClearing` (D36): the built-in
  for `average`, the registered one by name, else a `*ValidationError` with
  path `world, clearing`, rule `ErrUnknownClearing` and the name as value, also
  for the empty name.
- `doc.go`: the glossary terms of the notes table; `notYetTerms` loses the four
  entries.

**Tests:** unit, from the external package where the API is public and from an
internal test for `resolveClearing`.

- `AverageClearing.Clear` on the book of CLR 11 returns the three trades with
  their buyer and seller, price and quantity, and the lot of the ask; with
  `rand.New(rand.NewPCG(...))` given, exactly one `Uint64` per order is
  consumed: a counting source in the test (a hand-written `rand.Source`)
  proves bids first then asks and one word each (FR-CLR-022).
- The vector of WLD 7.3: with `newStream(42)` keyed `(siteClearing, 7, 0, 0)`,
  the first key is `a0043220bd826c9b` and three words are consumed for the
  market `(r0, grain)`, one bid and two asks, bid first; the same for
  `(siteClearing, 7, 0, 1)`, the market `(r0, loaf)` of two bids and one ask,
  giving `9f4b829a3880ffc9` (`spec-world.md` 19.2; FR-CLR-022, FR-WLD-060 in
  part; N-214).
- Determinism: the same seed and orders give the same trades in the same
  order over two runs. A different input order of equal-limit orders may give
  different trades, because each key is drawn by input position (CLR 4.2, 7.2:
  the world supplies a deterministic input order); the test asserts only that
  a given order gives one answer.
- The options: a table with one case per rule of D36 asserted with `errors.Is`
  (a duplicate name, `average`, nil, an invalid ID); two registrations are both
  found; the order given is kept; `resolveClearing` for `average`, a
  registered name, an unknown name, the empty name, and the path and rule of the
  `*ValidationError` (FR-WLD-002 and FR-CFG-038 for the clearing, FR-CFG-081;
  Q3 may change one row).
- A consumer example: an `Example` test of a `Clearing` that wraps
  `AverageClearing` and builds the option with `WithClearing`, to show the
  public API is enough (architecture.md 1.3, 6.5). It does not register the
  clearing in a world: `NewWorld` does not exist and nothing applies the
  option yet, so the example stops at the wrapper and the option; the example
  that runs a world with it is the world row's.

**Observability and rollback:** none emitted. Nothing here logs: a refusal of an
option is a returned error, handled once by its caller (architecture.md 8.3).
Rollback is reverting the PR; nothing calls `resolveClearing` until the world
row.

**Acceptance:** the commands of Phase 2 pass; the glossary test, the errors
test and the guard tests of Stage 0 (N-224) pass with their lists updated, not
loosened; coverage of the root holds at 90%.

## Phase 5: the book, reservation, settlement and the market step

**Why here:** it composes everything and is the only phase that needs the
ledger's goods and reservation operations and the observer.

- `errors.go`: `ErrInvalidTrade` (first user `clearMarket`) and its row of
  `errors_test.go`; `event.go`: the doc comment of `ReasonOverflow` widened as
  D34 says, and `TestConstants_Values` (its parse of CLR 3.2) follows the
  spec line Phase 1 wrote.
- The unexported `clearScratch` (A2) and the `book` of one market (D39): region, commodity, tick, a `*clearScratch`, the
  admitted orders in the order admitted (each with its ledger account, side,
  limit, quantity, lot key and the reservation it holds), and `admit(l, facts,
  order)`: it calls `clearing.Admit`, and on success reserves cash for a bid
  (`ReserveMoney(limit*quantity)`) or goods for an ask (`ReserveGoods`, or
  `ReserveLot` for a named lot) and appends the order. The facts of an ask that
  names a lot come from `AppendLots` (A4): an absent or short lot is
  `insufficient_stock`, never a `ReserveLot` refusal. A ledger refusal at
  reservation after a passed check is an invariant (D33). **Who supplies which
  fact.** The book reads from the ledger, by the order's account: the
  available cash (`Account` view: balance less reserved), the available stock
  of the commodity (`Holding`, or `AppendLots` until it merges), for a bid the
  buyer's held units of the commodity (the D34 bid check), for an ask the
  seller's balance, and for a named lot the lot's available units
  (`AppendLots`). The world passes in `facts` what the ledger does not know:
  the agent is known and in the book's region, the side is allowed for its
  kind, the commodity is known and in the role's tradable set, the ledger
  account of the agent, and the `receiptTally` slot of that account. After an
  ask is admitted `book.admit` adds `quantity * MaxPrice` to that account's
  tally with `checked.Mul` and `checked.Add` (an overflow of the tally is the
  `overflow` rejection, never a wrap): D34's guarantee that settlement cannot
  meet `ErrBalanceOverflow` depends on this update, and the world row's
  contract (O1) is to clear the tally at the start of P3.2 and to feed it
  from any other credit in that window (O2).
- `clearMarket(l *ledger.Ledger, b *book, c Clearing, rng *rand.Rand, prev
  marketReport, tick int64, obs *observers, log *slog.Logger, trades []Trade)
  (marketReport, []Trade, error)` is the one parameter list the world row and
  plan 1-4 build against (CLR 6, 7.2, 8.2, 9; O5 closed): the settled trades are
  appended to the caller-owned `trades` (no allocation when it has room) and
  the book exposes the filled units of each admitted order, in admission order
  (`book.filled()`, read from the `Sums` that `Check` left in `clearScratch`),
  for the outcome of CLR 7.3 and the welfare scratch; the
  world row maps an order to its purpose (`Order` has no `Purpose`, no new
  event, CLR 3.1 unchanged). The step: split the book into bids and asks in admission order (into the
  scratch);
  `c.Clear`; `clearing.Check` with the scratch's `Sums` and `Index` over the
  book's admitted records and the book's
  commodity (a trade naming another commodity is the sixth failure); on a failure release every reservation of the
  book and return an error that wraps `ErrInvalidTrade` and names the tick,
  region and commodity, emitting and settling nothing (FR-CLR-024); otherwise
  settle each trade by D33, emit `TradeExecuted` after its settlement, release
  the remainder of every order that has one (D33: nothing is released
  at 0), compute the report (D35) from the previous one
  the caller passes, emit `MarketCleared` (also with no trade, FR-CLR-027),
  and return the new report and the trades.
- `emitPlaced` and `emitRejected` and the two `Debug` lines "order rejected"
  and "market cleared" with the keys of observability-and-logging.md 3.3.2
  (`commodity`, `agent`, `side`, `reason`; `commodity`, `price`, `quantity`),
  named by the constants of `internal/logkeys` (`logkeys.Commodity`,
  `logkeys.Agent`, `logkeys.Side`, `logkeys.Reason`, `logkeys.Price`,
  `logkeys.Quantity`, all present), never by literals (architecture.md and
  observability 3.3.2: a key is defined once); built and logged only under
  `supplied` and `Enabled` (D37, FR-CLR-033).
- The mapping of the internal rejection codes to the `Reason` constants of
  `event.go`, both ways, with a `Guard:` test that every code has a constant
  and every slice constant of CLR 3.2 has a code.

**Tests:** unit and integration, the second only in the sense of the real
ledger and the real observer wired together (there is no world; testing.md
2.3's public-API half is the world row's). Integration cases begin with the
`testing.Short()` skip.

- Admission against a real ledger: a bid reserves exactly `limit * quantity`
  and the available cash falls by it; an ask reserves its units; a second
  order of the same side is `duplicate_order` and reserves nothing; a rejected
  order changes nothing (ledger compared with a copy by `reflect.DeepEqual`)
  (FR-CLR-004, 005, 006, 030). Two books of two commodities and one ledger:
  an order of the other commodity panics with the invariant message (D39) and
  an order of an agent of another region is `unknown_agent` and reserves
  nothing; no order of one book is ever in the other's trades (FR-CLR-001,
  003).
- Two ticks on one warmed `clearScratch`: a first market with trades and
  unmatched orders, then a second market of the same book size with different
  orders; the second's trades, filled units, report and released reservations
  are exactly those of a fresh scratch, and no order of the first is present
  in the second (FR-CLR-008; a stale `Sums`, `Index` or `bids` length would
  show).
- Settlement: the book of CLR 11 on a ledger with the six accounts: balances,
  stock, the 8 returned to B1, 28 released to B3 and the unmatched asks
  released, with the arithmetic in a comment; money paid equals money received
  and units sold equal units bought (FR-CLR-008, 016, 017); the flow totals
  show the `trade` reason net zero (the per-region trade flow total is an
  `int64` sum at settlement and cannot overflow unless total money passes
  `int64`, which `MaxMoney` per balance and the audit exclude; one line in the
  code comment, no extra check).
- Defeating admission: a buyer holding `MaxQuantity - 3` places one bid of 4
  (`overflow`, nothing reserved) and, in another case, one bid of 3 (admitted,
  and settles with no ledger refusal); a second bid of the same agent in the
  book is `duplicate_order`, not `overflow`. A seller with a balance at
  `MaxMoney` minus `4 * MaxPrice` places several small asks, in one market and
  across two markets, so that no order alone overflows but the sum would; the
  last is `overflow`, and settlement of what was admitted meets no ledger
  refusal (D34).
- Releases at zero: a bid that trades at its own limit (CLR 11 case B2: limit
  10, trade at 10, `(10-10)*2 = 0`) and a fully filled bid, ask and named-lot
  ask do not call a release and the step does not panic; a partially filled
  order releases exactly its remainder (D33).
- A named lot: an ask that names a lot moves that lot and no other of the same
  commodity held by the same seller (FR-CLR-029); an ask that names a lot the
  seller does not hold, or holds fewer units of than the quantity (or whose
  units are already reserved), is `insufficient_stock` and reserves nothing.
- A faulty `Clearing` (a hand-written fake at the extension point, testing.md
  6.1): each of the six breaks of CLR 8.2 (the sixth: a trade that names another
  commodity) returns `ErrInvalidTrade`, settles
  nothing, releases every reservation and leaves the ledger equal to its state
  before the market step except the released reservations (FR-CLR-014 for a
  price outside the limits, FR-CLR-018 for the same agent, FR-CLR-024).
- Trades and fills: the trades `clearMarket` appends equal the `TradeExecuted`
  events in order, and the filled units per admitted order summed over the book
  equal the units of the trades on each side (FR-CLR-016, 017, 026); with a
  hand-written `Clearing` returning a preallocated slice, a `trades` slice
  that has room and a warmed `clearScratch` (its `Sums` and `Index` included,
  so the `Check` inside the step is covered), `clearMarket` allocates nothing,
  for a valid list and for a rejected one (A2; testing.md 8.2, NFR-CLR-002 in
  part; `AverageClearing.Clear` is held by its own baseline constant); a
  failed list appends no trade.
- Order of markets: two books cleared in either order against one ledger give
  the same trades and balances, given the same keys (FR-CLR-023).
- Events: a recording observer sees `TradeExecuted` in matching order and then
  `MarketCleared`, once, also with no orders; `OrderPlaced` and
  `OrderRejected` in the order submitted with the admitted limit and the
  reason; no map or pointer in an event (FR-CLR-026, 027; observability-and-
  logging.md 8.1).
- Logs: a capturing handler sees only `Debug` lines with the constant messages
  and keys; no `Info`, `Warn` or `Error` over a scripted market with an invalid
  list (FR-CLR-033).
- Allocation (observability-and-logging.md 8.2): with the default observer and
  logger, `emitPlaced`, `emitRejected`, the `TradeExecuted` and `MarketCleared`
  call sites and the two `Debug` lines allocate nothing, by
  `testing.AllocsPerRun`; a benchmark with `b.ReportAllocs()` beside it
  (NFR-CLR-002 in part).
- Non-interference: the same book with and without an observer and a logger
  gives the same ledger and report.
- An invariant test over seeds (testing.md 6.3): random books over a small
  commodity set and a small number of agents, a fixed seed range with the
  failing seed in the message; after every market step no reservation is open,
  no balance or stock is negative, the clearing price lies between the matched
  limits, and goods and money are conserved by exact equality, asserted by a
  local sum over the ledger's accounts and boundary balances that needs no
  `Audit`. Only one more assertion waits for plan 1-1 Phase 6 `Audit` (A6):
  that the ledger's audit passes after every step; it is added when `Audit`
  merges and until then the ledger's own checks stand in.
- A fuzz target `FuzzBook` decodes bytes to books and asserts the same local
  properties and that nothing panics, including books with balances near
  `MaxMoney` and stocks near `MaxQuantity` (D34); it also decodes an
  arbitrary list of trades (any buyer, seller, price, quantity, lot) into a
  hand-written fake `Clearing` that returns it, so the whole of the faulty-list
  path is fuzzed through `clearMarket`: a list the step accepts satisfies
  every CLR 8.2 property and conserves goods and money; a list it rejects
  returns `ErrInvalidTrade` and leaves the ledger equal to its state before
  the step except for the released reservations (FR-CLR-024). Corpus in the
  package's `testdata/fuzz/`, crashers with a `Fix:` comment (testing.md 7.5).
  Its seeds include the six hand-written failures of the faulty-`Clearing`
  case. `FuzzCheck` and `FuzzAdmit` of Phase 3 cover the two internal
  validators on their own.

**Observability and rollback:** the checklist of
observability-and-logging.md section 9 is answered in the PR. No metric, no
event type and no log key is added; the four events exist since Stage 0 and
this phase is their first emitter. Rollback is reverting the PR; nothing outside
the file calls it until the world row.

**Acceptance:** the commands of Phase 2 pass; the invariant test passes over its
seed range and `FuzzBook` runs five minutes by hand with no crasher (a crasher
is committed with a `Fix:` comment, testing.md 7.5); only the `Audit`
assertion of both is conditional on plan 1-1 Phase 6 (`Audit`) being merged,
which is ASSUMED, and is added when it lands; every `clearMarket`
and emission allocation test passes at zero (the `AverageClearing.Clear`
baseline constant is recorded, not zero); coverage holds at 90%.

## Phase 6: the exit gate

- The commands on the whole module: `gofmt -l .`, `go vet ./...`,
  `staticcheck ./...`, `go test -short -race ./...`, the full run with
  `-covermode=atomic -coverpkg=./...`, `go mod tidy`; the per-scope coverage
  read and recorded.
- The traceability check of testing.md 4.6 by text search: every ID in a
  `Requirement:` line exists in CLR, WLD or CFG, and every requirement the
  table below lists as covered in full is cited.
- The roadmap row set to **done** with its commit; N-075, N-125, N-185 and
  N-214 updated; the `World.Clearing` registry-test entry unchanged; the
  Cross-plan contract of this plan checked against the world row and the
  strategy and spawner rows.
- Not gating, worth doing: a mutation pass over `internal/clearing` with
  `gomu` inside Docker only, `--cpus=4 --memory=16g --memory-swap=16g`,
  `GOMAXPROCS` and `--workers` equal to the CPU cap (AGENTS.md; testing.md 9.4).

**Acceptance:** every command is clean; the traceability check finds no ID
without a spec row and no "covered in full" requirement without a test; the
roadmap row reads done.

## Cross-plan contract

**Consumes.**

| Symbol or fact | Kind | From | Status |
|----------------|------|------|--------|
| `Price`, `Quantity`, `Price.Times(Quantity) (Money, bool)`, `CommodityID`, `RegionID`, `AgentID`, `MaxPrice`, `MaxQuantity`, `MaxMoney`, `MaxIDLength`, `validID` | root types, constants, function | plan 0 (`value.go`, `bounds.go`, `validate.go`) | built |
| `OrderPlaced`, `OrderRejected`, `TradeExecuted`, `MarketCleared`, `Side`, `Reason` and the ten rejection constants | events and constants | plan 0 (`event.go`) | built |
| `observers` with `supplied` and `deliver`, `options`, `applyOptions` | unexported types | plan 0 (`observer.go`, `option.go`) | built |
| `stream.keyedPair`, `siteClearing`, the stream's `*rand.Rand` | unexported | plan 0 (`random.go`) | built |
| `ErrDuplicateID`, `ErrInvalidID`, `*ValidationError` | sentinels, type | plan 0 | built |
| `checked.MulDivHalfEven`, `checked.Mul` | functions | plan 0 (`internal/checked`) | built |
| `ledger.Ledger`, `ledger.Trade`, `TransferReserved`, `ReserveMoney`, `ReleaseMoney`, `Account` (`View{Balance, Reserved}`) | ledger API | plan-phase-1-1 Phase 2 | built in the tree (read 2026-10-13); a rename is a compile error (its A6) |
| `ReserveGoods`, `ReleaseGoods`, `ReserveLot`, `ReleaseLot`, `MoveReserved`, `MoveLotReserved`, `AppendLots`, `Key{Commodity, Born, Durability}` | ledger API | plan-phase-1-1 Phase 3 | built in the tree (read 2026-10-13) |
| `Ledger.Holding(account, commodity) (held, reserved, err)`, allocating nothing | ledger API | plan-phase-1-5 Phase 2 (D58, new file `internal/ledger/discard.go`; not plan 1-1) | **Resolved**, not yet in the tree; commodity totals only (ordinary asks, the buyer's stock); `AppendLots` serves until it merges |
| `Ledger.AppendLots(dst, account)` read for the lot-level available units of a named-lot ask | ledger API | plan-phase-1-1 Phase 3 | built in the tree; **permanent**, not a fallback (A4) |
| `Ledger.Audit` and a read that no reservation is held | ledger API | plan-phase-1-1 Phase 6 | for the audit assertion of the invariant test and `FuzzBook` only (the local conservation sum needs none); **ASSUMED** shape |
| Every ledger refusal at settlement is an invariant, not a public error | rule | plan-phase-1-1 (N-075 update) | decided here (D33) |
| The tradable set of a role and a commodity's "tradable" flag, as a fact the caller passes in | derived table | plan-phase-1-2 Phase 5 (`role.go`) | Provided: `(*roleRow).isTradable(CommodityID) bool` and `(*roleTable).tradableSomewhere(CommodityID) bool` ("Cross-plan contract" of plan 1-2); this row takes a `bool` and does not read the table |
| `receiptTally` (the per-account worst-case receipt of the round, cleared at the start of P3.2) | unexported | world row (owner and clearer; this row defines the type and the cumulative rule, D34) | **ASSUMED**: the world row owns it |
| `clearScratch` (bids, asks, offers, matches, fills, lots, `sums`, `index` slices) | unexported | world row (owns one per world, passes it to `newBook`, never shrinks it; this row defines the type and resets lengths per market, A2) | **ASSUMED**: the world row owns it |
| `ErrUnknownCommodity` for an order's commodity | sentinel | plan 0 | built; the rejection is the `Reason`, not this sentinel |
| The agent table, `AgentID` to account, region membership, role and pseudo-agent kind | world state | world row (`agent.go`, `world.go`) | **ASSUMED**: resolved by the caller into `Facts` |

**Provides.**

| Symbol or fact | Kind | For |
|----------------|------|-----|
| `Order`, `Trade`, `LotKey` | exported types (D32) | the strategy row (a plan's orders), the estate row (named lots), the external and fiscal rows, consumers |
| `Clearing`, `AverageClearing`, the contract of its doc comment | exported interface and default | consumers, the registration of every later rule, the checkpoint row (`Stateful`) |
| `WithClearing(name, c)` | exported option | the world row, `strategy.Options()` pattern, the checkpoint row (same option type) |
| `ErrInvalidTrade`, `ErrNilExtension`, `ErrUnknownClearing` | exported sentinels | the world row (`Step` and `NewWorld`), the strategy and spawner rows reuse `ErrNilExtension` and the pattern of `resolveClearing` |
| `resolveClearing(o *options, name string, path []string) (Clearing, error)` | unexported | `NewWorld` (world row) at the name checks of WLD 3.2.2 |
| `options.clearings` registration list and the generic `registry[T]` (`add`, `find`) of D36 | unexported | the strategy and spawner rows reuse it unchanged (coordinator Q3); the policy row likewise |
| `clearScratch`, `newBook(region, commodity, tick, *clearScratch)`, `book`, `book.admit(l, facts, order)`, `book.filled()` (units filled per admitted order, admission order), `clearMarket(l, b, c, rng, prev marketReport, tick, obs, log, trades) (marketReport, []Trade, error)` appending the settled trades to a caller-owned slice | unexported | the world row at P3.2 and P4.1; its outcome loop (its own, built from plan 1-4's `Gate`, `Change` and `Tally`; `Revise`, FR-CLR-025, is the agent row's) and the welfare scratch feed (plan 1-6 `addReceipt`, `addInputSpend`) |
| `marketReport` (the fields of CLR 16) and `seedReport(price)` | unexported type and constructor | the world row's `market.go` (state, `World.Market`, snapshot), the checkpoint row (FR-CLR-034), the beliefs row (the report passed to `Revise`) |
| The mapping from internal code to `Reason` | unexported | the world row (rejection events), the strategy row (`not_tradable` raised while it builds orders) |
| `internal/clearing` `Offer`, `Match`, `Average`, `Admit`, `Check`, `Report` | internal API | no other row; `internal/checked` accumulator for any later 128-bit sum |
| Glossary terms listed in the notes table; `notYetTerms` without Market, Order, Trade, Clearing | `doc.go` | the glossary test of every later row |
| Roadmap rows edited by D30: `order.go` is built here and **the world-row plan must drop it**; the world row keeps `market.go` whole (state, window ring, per-market table) | document | the world row's plan |
| Admission is a single ordered function with marked positions for the module rows; how a module reaches it is open (O2) | rule (D38) | STO, ORG and the pseudo-agent rows |

## Security review

The module is a library with no network, file access, credentials or
persistence. This row adds one trust boundary that matters: a consumer's
`Clearing` implementation, which is untrusted code that runs inside the step and
returns a list the world must not believe.

- **A faulty or hostile `Clearing`.** It may return trades with no order behind
  them, quantities above an order's, prices outside the limits, a buyer that is
  the seller, or a list long enough to be a cost. The whole list is checked
  before any trade is settled (D33, FR-CLR-024), so it cannot create or destroy
  money or goods or break a budget; it can only fail the step loudly with
  `ErrInvalidTrade`. The check is `O(n log n)` in orders and trades; a list
  longer than the orders can match is refused at the first excess (the sum per
  order). It may call back into the world only if given a pointer to it, which
  the interface does not provide.
- **The random source.** The `rng` is the world's; the contract forbids keeping
  it or creating another (7.7). The library cannot detect a violation; the doc
  comment and the determinism guard of Stage 0 say so.
- **Integer overflow.** `limit * quantity` is `checked.Mul`; the volume-weighted
  mean is accumulated in 128 bits; a stock above `MaxQuantity` or a balance
  above `MaxMoney` is closed by D34 at admission, so settlement cannot meet it.
  `FuzzBook` includes balances and stocks at those edges.
- **Panics.** Settlement and reservation panic only on a violated invariant that
  no input can cause once D34 holds (architecture.md 8.1); an order of an
  agent of another region, an unknown agent or a bad price is a returned
  rejection, not a panic. The one other panic is a book given an order of
  another commodity (D39): the world routes by commodity, so no input can cause
  it. Every panic message says it is a violated programmer invariant.
- **Resource exhaustion.** One book is bounded by the orders admitted, which the
  agents' number and the one-order-per-key rule bound; scratch slices are
  sized once per market. A consumer's `Clearing` that is slow is the consumer's
  problem (the step is synchronous, architecture.md 12.5).
- **Not applicable:** authentication, authorisation, injection, secrets,
  cross-tenant exposure, supply chain (no dependency is added).

## Compatibility and rollout

- **No code contract exists to break.** There is no release and no consumer.
  This row adds exported identifiers: `Order`, `Trade`, `LotKey`, `Clearing`,
  `AverageClearing`, `WithClearing`, `ErrInvalidTrade`, `ErrNilExtension` and
  `ErrUnknownClearing`. Each is appended, not inserted (architecture.md
  2.10.5), and a change to one after the first release is breaking
  (architecture.md 6.6), so the shapes of D32 are the thing to get right now.
- **Document contracts.** Phase 1 edits CLR 8, 8.3, 15 and 3.2, WLD 4.1 and
  4.3, CFG 7.1.4 and the roadmap rows (no rule file: N-235 holds the
  `architecture.md` edit); it renumbers no requirement (testing.md 4.1). `doc/notes.md`
  gets a fresh tag.
- **What becomes hard to change, and when.** The draw count of D7 and the keys
  by position enter the golden files at the end of Stage 1 and no draw count
  may change after; the order of the checks of CLR 3.2 and the half-even
  rounding enter them the same way. Until the first golden file is recorded
  they may be corrected in a normal PR.
- **Append-only stays true.** The module rows of the admission table are added
  by their modules at their positions (D38); with the module off they add no
  call. `testdata/golden/off/` and the pinned hash do not move (D5); this row
  adds no configuration field and no event field.
- **Rollout** is one PR per phase (A6). Each leaves the branch passing every
  command of `AGENTS.md`. A phase is rolled back by reverting its PR; later
  phases depend only on earlier ones and the other rows of Stage 1 do not call
  this row's functions until the world row.
- **Dependency failure.** None: no dependency and no run-time service. The
  ledger row's names may change under this row; the assumption is A4, and the
  build fails at compile time, never silently.
- **The Go toolchain.** `go.mod` says `go 1.25`. No unstable sort is used where
  equal elements can occur: the comparator ends in the input position
  (architecture.md 7.6).

## Testing (summary, testing.md section 11)

The pyramid is mostly base: unit tests in `internal/clearing` and
`internal/checked`, unit tests of the root pieces with a real ledger, one
invariant test and two fuzz targets, and allocation tests. There is no
end-to-end test because there is no world; the world row adds the integration
tests that run these functions through `Step`.

| Phase | Covered in full | Covered in part (completed in) |
|-------|-----------------|--------------------------------|
| 2 | FR-CLR-009, 011, 012, 013, 015; NFR-CLR-001, 003 | FR-CLR-010 (the total order here, the key draw in Phase 4); FR-CLR-014, 017 (the algorithm here, settlement in Phase 5) |
| 3 | FR-CLR-002, 003, 005, 006, 018, 019, 020, 021, 028, 031 (the validators also by `FuzzAdmit` and `FuzzCheck`) | FR-CLR-004, 007 (the key and the overflow here, the ledger in Phase 5); FR-CLR-024 (the check here, with its scratch and allocation test, the world's response in Phase 5); FR-CLR-030 (the fact here, the refusal at plan building is the strategy row's) |
| 4 | FR-CLR-010, 022; FR-CFG-038 (clearing), FR-CFG-081; FR-WLD-002 (clearing) | none |
| 5 | FR-CLR-001, 004, 007, 008, 014, 016, 017, 023, 024, 026, 027, 029, 033 | NFR-CLR-002 (the call sites here, the full `Step` baseline in the world row) |
| 6 | none | the traceability check |

Deferred with their code: FR-CLR-025 (`beliefs.go`, `world.go`); FR-CLR-032
(`world.go`, BEL); FR-CLR-034 (`checkpoint.go`); FR-CLR-035 and `not_permitted`
(`institutions.go`); `no_room` (`storage.go`); the invariants of testing.md 6.3
through the public API over the reference chain (Stage 1 "Done when").

Boundaries and error paths: every sentinel and every rejection code has a case
that breaks only its rule (testing.md 3.9). Golden files: none exists yet.
Fuzz (testing.md 7.4: the validators of external input): `FuzzCheck` (the
list a consumer's `Clearing` returns) and `FuzzAdmit` (a consumer's orders) in
`internal/clearing`, `FuzzAcc` in `internal/checked`, and `FuzzBook`, which
also drives an arbitrary trade list through a fake `Clearing`. Each PR states its coverage and that its tests
were run against a broken version (testing.md 9.3), for example a flipped
comparison in the crossing test and the midpoint's rounding tie.

## Observability (summary, observability-and-logging.md section 9)

1. **Failure path or rejected input?** A rejected order is an `OrderRejected`
   event with a `reason`, a `Debug` line, and a returned code; an invalid list is
   an error returned from the step and not logged (architecture.md 8.3).
2. **State transition?** Admission, a trade and a market clearing each have an
   event that exists (`OrderPlaced`, `OrderRejected`, `TradeExecuted`,
   `MarketCleared`); none has a map and every value is fixed point.
3. **Decision point?** The rejection `reason` is the operator's explanation;
   the clearing's choices are in the trades; the strategy's choices are the
   strategy row's `Debug` lines.
4. **New loop?** Per order and per trade: only `Debug`, guarded by `Enabled`;
   no `Info` and no `Warn`; events are built only under `supplied`
   (observability-and-logging.md 4.5), proved by `AllocsPerRun`.
5. **Changed event, metric, unit or attribute?** None.
6. **New metric attribute?** None.
7. **`Error` or `Info`?** Neither is used.
8. **An error logged and returned?** No.
9. **Tests?** Section 8 applies and the tests are in Phase 5; the golden
   rendering and the emission order over a step are the world row's.

## Open items

- **O1.** The world-row plan must drop `order.go` (D30), own `market.go`, and own
  and clear the `receiptTally` at the start of P3.2 (D34). The coordinator is
  making the plans consistent.
- **O2.** Where the module rows of the table of CLR 3.2 sit relative to the
  world's plan checker (`checkTradePlan`, ADR 0007 fan-in): `not_permitted` is
  checked "beside `not_tradable` and after it" while building orders, and
  `no_room` after `insufficient_stock` in admission, so the single order of
  FR-CLR-031 spans two functions. The slice has one function and no module
  row; the first module PR that adds one settles it, by the combined
  `architecture.md` edit that N-235 holds (with the fifth fan-in of plan 1-2 O1).
  The same PRs read one more line: D34 holds only while no other credit
  reaches a seller's balance or a buyer's stock between P3.2 and P4.1, so a
  module that credits in that window (credit P3.3b, insurance P3.3c, fiscal)
  feeds the `receiptTally` or uses its own headroom; Phase 1 adds it to N-235.
- **O3.** No CI exists (O3 of plan 0). The gate of Phase 6 and the timed fuzz
  runs are done by hand, on one architecture.
- **O5.** Closed (coordinator, 2026-10-13, ruling 1): `clearMarket` appends the
  settled trades to a caller-owned slice and the book exposes the filled units
  per admitted order in admission order; no `Purpose` on `Order`, no CLR 3.1
  change, no event. The world row owns the mapping of an order to its purpose.
- **O4.** Whether the `LotKey` of an `Order` should be a pointer or the
  flagged struct of D32 is a style call; D32 takes the comparable struct so an
  `Order` is comparable and has no pointer. Revisit with the estates row.

## Answered by the coordinator (2026-10-13)

1. **Q1 (D34):** option (a), cumulative per agent in the round, `ErrBalanceOverflow`
   at settlement an invariant panic, MON untouched.
2. **Q2 (D30):** option (a): `order.go` and the report type here, `market.go` whole
   with the world row; Phase 1 moves `order.go` out of the world row in the
   roadmap and records the move in `vision.md` section 4 and `spec-roadmap.md`.
3. **Q3 (D36):** option (a), written once in WLD 4.1 for all four registration
   options; only `WithClearing` and `registry[T]` are built here. CLR 8's second
   reserved name `default` was checked against WLD 4.1, 4.3 and FR-WLD-002: WLD
   reserves `average` only, so CLR 8 is corrected and a note is raised for the
   CLR and WLD owner.
4. **O5 (cross-plan review, 2026-10-13):** `clearMarket` appends the trades to a
   caller-owned slice and the book exposes filled units per admitted order.
5. **`architecture.md` edits (2026-10-13):** none in this plan; N-235 holds the
   2.9 example and the admission-table touch point for one combined edit.
