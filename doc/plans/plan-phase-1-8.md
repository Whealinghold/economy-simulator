# Plan: Stage 1, row 8: the estate, its listings and the pool that funds an entrant

**Status: plan written (2026-10-17, written in parallel with rows 7, 9, 10, 11 and 12; audited, not started).** Sequences the eighth row of Stage 1 of
[roadmap-v1.md](../roadmap-v1.md), "`estates.go`: the estate pseudo-agent and
account, the listings, and the pool with its use to fund an entrant", into five
phases. It implements [spec-estates.md](../specs/spec-estates.md) (EST) for the
slice, and the removal steps 3 and 4 of [spec-agents.md](../specs/spec-agents.md)
3.6 (AGT) that open an estate. It is built against `doc/rules/architecture.md`
(the layout of 2.6, the `internal/` boundary of 2.9, the fixed-point and overflow
rule of 3.5, the determinism rules of section 7), `doc/rules/testing.md` and
`doc/rules/observability-and-logging.md`. Where this plan and the rules or the
specs disagree, they win and this plan is corrected.

Rule tags read for this plan: architecture=905c0e578c4e662908638dd8,
observability=35827912d1e3141a76af3937, testing=b3559bc6d47470c221bc4642.
Notes tag (as read by the round-1 fixer, 2026-10-17): 0613273b12d0bd5281290e37. Notes
N-254 to N-258 and N-266 already exist in `doc/notes.md` (raised with this plan and with plan 1-9).

A plan decides nothing. Every decision below comes from a spec, a rule, an ADR
or the decisions of [plan-phase-0-foundation.md](plan-phase-0-foundation.md)
(D3 and D5 above all). Where a decision is still owed, Phase 1 makes it in the
document that owns it, before the code that depends on it. Decision IDs D100 to
D109 are this plan's (plan 1-6 used D61 to D70; the other rows planned in
parallel have their own ranges); earlier plans' decisions are cited by file and
ID. Each decision that the specs do not settle is flagged for the owner's
override (open item O1).

No PRD exists for this row. The requirements source is the roadmap row, EST and
AGT 3.6. The row is built after the ledger, the goods tables and clearing
([plan-phase-1-1.md](plan-phase-1-1.md), [plan-phase-1-2.md](plan-phase-1-2.md),
[plan-phase-1-3.md](plan-phase-1-3.md)), whose named-lot ask and pseudo-agent
admission facts it uses, and before the world row, which calls it.

Spec codes: EST estates, AGT agents, REG regions, WLD world, LED ledger, CLR
clearing, SPN spawner, SKL skill barriers, GDS goods, CUR currency, OBS
observability, CHK checkpoint, CFG config, VIA viability, MON money supply, CRD
credit, STB stability. `FR-`/`NFR-` IDs are the requirement IDs of those specs.

## What Stage 0 and the earlier rows already give, and what is left

| Concern | Where it already is |
|---------|---------------------|
| `Region.EstateMarkdown` with its default 500, bound 0 to 10 000 and hash field 5; `World.EntryPoolShare`; `WarmStart.Pools` (`WarmPool`) with its validation and canonical order | `config.go`, `defaults.go`, `bounds.go`, `validate.go`, `hash.go` |
| The reasons `estate_money`, `estate_goods`, `start_money`, `births`, `split`, `merge`, `starvation`, `initial`, `trade` with their pairing rules (agent to estate, estate to agent, boundary to estate) and the root names `ReasonEstateMoney`, `ReasonEstateGoods`, `ReasonStartMoney`, `ReasonBirths`, `ReasonSplit`, `ReasonMerge`, `ReasonStarvation` | `internal/ledger/registry.go`, `ledger.go` (`reasonNames`), `event.go` |
| The ledger account kind `KindEstate`, `Open(kind, owner, region)`, `Transfer`, `Move`, `MoveLot` (the key is kept), `AppendLots`, `View.Balance`, `Audit`; the boundary accounts of the built-in group (`newLedger`), so that the first estate account is ID `1 + 7C` | `internal/ledger`, `ledger.go` |
| Named-lot asks: `Order.Lot` (`LotKey`), `orderFacts{AgentKnown, SideAllowed, Tradable, Account, Tally}`, `book.admit` with `ReserveLot`, the duplicate key (agent, lot, side), `clearing.SideNotAllowed` and `ReasonSideNotAllowed`, the settlement of a lot with `MoveLotReserved` | `order.go`, `clearing.go`, `internal/clearing/admit.go` |
| `EstateOpened`, `AgentBankrupted.GoodsListed`, `RoleChanged.GoodsListed` as declared events | `event.go` |
| `resolutionFloor(World) (int64, bool)` = `ceil(10 000 / BeliefStep)`; `cfg.Commodities[id].ReferencePrice`, `.Durability` (the effective reference price in the slice is the commodity's, plan 1-2 A10) | `validate.go`, `config.go` |
| `internal/checked` (`MulDiv` and its rounding forms) | `internal/checked` |
| The glossary entries `Estate`, `Estate pool`, `Named lot`, `Pool policy` | `doc.go` |

What is left for this row is the behaviour EST gives those pieces: the identity
of the estate (account and agent ID), the intake of a removal, the price state
of the listings with the two price rules, the asks of P3.4, the markdown of
P5.3, the pool read, the seed and the draw that funds an entrant, and the ordered
read and validated rebuild of the listing state for the checkpoint.

## Scope

**In scope.**

- `estates.go` (root, unexported; a core concept file, architecture.md 2.6):
  - opening the estate account of each region and its identity: the agent ID
    equal to the `RegionID` (REG 3.6.1, EST 2.7.6);
  - seeding the pool from `WarmStart.Pools` by `initial`;
  - the pool read and the pool draw that funds an entrant (`start_money`,
    `births`), capped at the pool (EST 2.3, 2.8.1, FR-EST-014, 022);
  - the intake of a removal (AGT 3.6 steps 3 and 4: the money by `estate_money`,
    every lot by `estate_goods`) and of the goods of a role change, with the
    counts the events carry;
  - the listing state: one price per held lot, sorted, the two price rules (the
    first price of EST 2.7.3, the markdown of P5.3), the asks of P3.4 and the
    order facts of an estate (asks only), and the "had an ask" record the
    markdown reads;
  - the ordered read and the validated rebuild of the listing state for the
    checkpoint (EST 2.7.5, FR-EST-024);
  - the glossary terms these use, in `doc.go`, before they use them.
- The spec, note and roadmap edits these depend on (Phase 1).

**Out of scope**, with where each is tracked:

| Item | Tracked in |
|------|-----------|
| The `Estates` module part: `Modules.Estates` (the 25th pointer), `Region.PoolRelease` and `Region.ReliefBelow`, relief at P1.6, `ReliefPaid`, the pool as a money-flow target and `ErrModuleDependency` for it | Roadmap Stage 2, the `estates.go` row appended after MON (CFG 6.2.4, FR-CFG-099; EST 2.6, FR-EST-017 to 020). This plan appends nothing to `Config`, and leaves `estates.go` with a marked place for the module's half |
| Calling any of this: the construction order (WLD 5.2 order 4, 5.6 item 5), P3.4, P5.3, the removal fan-in (`removeAgent`, steps 1 to 2g, the close of the account at step 5, `AgentBankrupted` and `EstateOpened`), the entry step P7.1(d) and its budget `PoolMoney * EntryPoolShare / 10 000`, the weight-change and role-change callers, the snapshot rows `MoneyPool`, `HeldByEstate`, `EntryBudget` | Roadmap Stage 1, `world.go` row (11), which holds the spawner executor, `removeAgent` and the role and weight-change composition and is the only caller; `agent.go` (7) and `spawner.go` (9) call nothing here and name these functions in their Consumes only |
| The decision of how many households an entry funds (`min(n, floor(budget / amount))`, whole households, the unfunded count, `EntryLimited`) | `spawner.go` row and the world's executor (SPN 4.3, 7.3). This row gives the draw and the pool read only |
| The waterfall of a removed debtor's money to its creditors (2d), the claims that pass to the estate (2e), titles (2b), hires (2c), covers (2f), facilities (2g) | `credit.go`, `tenure.go`, `employment.go`, `insurance.go`, `infrastructure.go` (Stage 2 and 3). Step 3 here moves "all money that remains" |
| A diverted cargo reaching the estate by `transit_in`, a traveller dying in transit (FR-EST-003), the escrow refund to the estate | `transport.go` and `contract.go` (Stage 2). The estate lists any lot it holds, so no listing code changes when they land |
| Money the estate receives as a creditor (credit), a title holder (land) or a pool-targeted flow | Each module's PR; the pool is the account's balance, so nothing here changes |
| The tool-sale rule of SKL 2.7 beyond the price (the price is EST 2.7.3, built here) and the sale of a tool the holder kept bound | `skills.go` (Stage 3) |
| The checkpoint section 8 encoding, its restore step 7 and the cross-check call | Roadmap Stage 2, `checkpoint.go`. This plan gives it an ordered read and a rebuild |
| The snapshot and `indicator` reads of the estate (`HeldByEstate`, `MoneyPool`, `Lots`) | `snapshot.go` in the world row |
| The estate of an outside node (it holds nothing, `spec-external-sector.md` 3.1.1) and the counterparty IDs that follow the estates | `external.go` (Stage 2). The estate of a region is created for every region, as REG 3.6.1 says |
| CI | Open item O3 of plan 0; the commands of `AGENTS.md` are run by hand |

## Decisions this plan rests on

Decided by documents already:

- One estate account and one pseudo-agent per region, created at construction in
  ascending `RegionID` order before any household and whether or not the region
  has a group or the estates module is on; the pseudo-agent's agent ID is the
  `RegionID`; it never moves region, is never removed and is in no count
  (REG 3.6.1 to 3.6.4, EST 2.7.1, 2.7.4, 2.7.6, FR-EST-004, 005, 010, 011, FR-REG-005, 006).
  In the account order of WLD 5.2 the estates are order 4, after the boundary
  accounts and before the Outside, Treasury and Agent accounts.
- The estate places asks only; a bid is `side_not_allowed`; each ask is backed by
  a reservation of goods and needs no cash; it places one ask for each lot it
  holds, naming the lot (EST 2.7.2, 2.7.3, FR-EST-006, 007). Two lots with the
  same key are one lot (LED 5.1.3, EST 6.1(c)).
- The first price of a lot is the effective reference price, times the lot's
  remaining durability over the full durability for a durable, truncated and
  never below the resolution floor (EST 2.7.3, FR-EST-008). The price is state
  (2.7.5). At P5.3 a lot that had an ask and was not wholly sold falls by
  `EstateMarkdown` basis points of the last price, never below the floor
  (FR-EST-009; WLD ruling C6). An estate's ask has no outcome and takes no
  belief word (WLD C21, FR-WLD-049).
- A pool draw for a new agent of weight `w` is `min(w * Role.StartMoney, pool
  balance)` with the reason `start_money`; households added to an existing agent
  and births use `births`; no draw exceeds the pool; an entrant gets no goods
  (EST 2.3, 2.8.1, FR-EST-014, 022; SPN 4.3). The budget is taken once by the
  spawner executor at P7.1(d) after the deaths (WLD ruling C15). Initial agents
  get their money by `initial`, not from the pool, and the pool starts at
  `WarmStart.Pools` by `initial` into the estate account, or 0 (FR-EST-015).
- The removal moves all money that remains with `estate_money` and all goods with
  `estate_goods` to the estate of the region of the agent's account, then the
  account is closed and `AgentBankrupted` then `EstateOpened` are emitted
  (AGT 3.6 steps 3 to 6, FR-EST-001, 002, 021). The goods of a debtor do not pay
  its creditors (N-167). The estate lists what it receives at the next P3.4.
- `EstateOpened` is core: emitted by every removal, with the pool policy on or
  off (EST 2.5, OBS 6.3.5). `ReliefPaid` is the module's and is not built here.
- The estate listing state in a checkpoint is, per region in ascending
  `RegionID`, the current price of each lot by key (commodity, born, durability)
  in canonical order (CHK 4.2 tag 8, EST 2.7.5, FR-EST-024); restore cross-checks
  that every listing is a lot the estate holds (FR-CHK-023).
- Arithmetic is `int64` through `internal/checked` where a product can overflow;
  no float, no map iteration, no random draw (architecture.md 3.5, 7.1 to 7.6).
  An estate's ask takes a tie-break key in the clearing like any order
  (EST 2.7.4, CLR 4.2); this row draws nothing.
- Module-off leaves no trace (D5): this row appends no field to `Config`, an
  event or a snapshot row, so the pinned configuration hash does not move.

Decided by this plan, to be written into the owning spec in Phase 1 before
code (each is flagged for the owner's override in the review, O1):

- **D100.** What the row is and is not. `estates.go` is the core estate of
  `spec-config.md` 6.2.4 and `spec-regions.md` 3.6.2; it holds no module state
  and no `Modules` pointer, and appends nothing to the configuration. The code
  has an unexported `estates` value (the accounts, the listing rows and the
  scratch of a tick), held by the world as one field. The pure price rules are two
  small functions in `estates.go` over `int64`; no `internal/estate` package is
  created, because architecture.md 2.6 lists none and 2.1 asks for a package only
  with a reason to exist alone (adding one is an architecture change, 14.2). The
  module's half is appended later in the same file after a marked comment line
  (architecture.md 2.10.5).
- **D101.** `GoodsListed` counts **lots** moved to the estate: one per distinct
  key of the agent's holdings, whatever the unit count (the same count as
  `FacilityChanged.GoodsListed`, `spec-infrastructure.md` 700, FR-CAP-047). The
  specs say "the number of goods passed to the estate" and do not say lots or
  units; the row that emits the event is the world row's, but the count is made
  here, by the one function every caller shares (`acceptGoods`), so that
  `AgentBankrupted`, `EstateOpened` and `RoleChanged` agree. **Ruled by the coordinator (2026-10-17): lots.** Units travel separately in
  `WeightChanged.Units` (FR-OBS-019). Phase 1 states it once where the field is
  defined (OBS event table, AGT 14.1, SPN 13.1, EST 2.5) and rewords the comments
  "number of goods" of `AgentBankrupted`, `EstateOpened` and `RoleChanged` in
  `event.go` to "number of lots" (a code-side owed edit, made in Phase 3).
- **D102.** The listing state. Per region a slice of rows `(commodity, born,
  durability, price, asked)` sorted by the key of LED 5.1.3, never a map. A row is
  created the first time a P3.4 walk sees a held lot with no row, at the first
  price of FR-EST-008, so that a lot that arrived by any path (a removal, a role
  change, a weight reduction, a diversion) is priced by one rule. A walk drops
  every row whose lot is no longer held (sold wholly, spoiled, worn out), and
  `markdown` (P5.3) drops the row of a lot that is gone as well, so a lot sold
  wholly loses its price at once: goods that later arrive under the same key
  (a diversion, a between-ticks call) are priced at the first price, as a world
  restored at the boundary prices them (FR-EST-024). A lot lost after P5.3 (spoilage
  at P8) is dropped by the next walk, and `rows` omits it. A lot
  that is held and has a row keeps the row's price when units join it (EST 6.1(c):
  one key is one lot, and the price is the lot's state); the units that join a
  marked-down lot are listed at its current price. The first price is a function
  of the configuration only (the reference price is static in the slice and in
  Stage 2, `spec-regions.md` 5.3.6), so lazy and eager pricing give the same
  value, and a restored world, which rebuilds a held lot with no row at the same
  first price, equals an uninterrupted one. Phase 1 writes it into EST 2.7.3 and
  2.7.5.
- **D103.** The ask. One `Order{Agent: estate, Commodity, Side: Ask, Price,
  Quantity: all units of the lot, Lot: LotKey{Named: true, Born, Durability}}` per
  held lot, per region in ascending key, built from `AppendLots` (every unit is
  free between ticks). The facts for the clearing are `orderFacts{AgentKnown:
  true, SideAllowed: side == Ask, CommodityKnown: true, Tradable: true, Account}`:
  a bid built the same way is refused by the real admission with
  `side_not_allowed`, which is how FR-EST-006 is tested. An ask the clearing
  rejects (`receipt_limit`, `overflow`) is an `OrderRejected` and **has no ask**,
  so its lot is not marked down. Phase 1 writes the quantity (all units of the
  lot) and this consequence into EST 2.7.3.
- **D104.** The markdown. At P5.3 a lot is marked down only if it had an
  **admitted** ask in the tick and is still held (so it was not wholly sold; a
  partly sold lot is held with fewer units). The new price is `max(floor, price -
  trunc(price * EstateMarkdown / 10 000))`: the **reduction** is truncated (the
  reading of "fall by `EstateMarkdown` basis points of the last price,
  truncated", FR-EST-009), the product through `checked.MulDiv`, and a price at
  or below the floor stays where it is. With `EstateMarkdown` 0 nothing falls.
  Phase 1 states the formula in EST 2.7.3 and FR-EST-009. The other reading, a
  truncated new price `trunc(price * (10 000 - md) / 10 000)`, differs by at most
  one unit per tick and reaches the floor at the same bound; it is the owner's
  (O1).
- **D105.** The pool draw is one function for every entrant and every rise of a
  weight: `grant(l, region, to, want, reason) (given, error)` gives `min(want,
  pool)`, where `pool` is the estate account's money balance (the estate places no
  bid, so none is reserved), and does nothing for an amount below 1 (the ledger
  refuses a transfer below 1, `ErrInvalidAmount`). It does not decide the budget
  or the number of whole households: the executor (world and spawner rows) does,
  and passes `want = households * amount`. The reason (`start_money` for a new
  agent, `births` for households added to an existing agent or a birth) is the
  caller's, from SPN 4.3. A reduction of a weight sends money the other way with
  `acceptMoney(l, from, region, amount, reason)` (`split`, `merge`, `starvation`;
  the reason is the caller's, and `deaths` arrives with the population module,
  FR-LED-055), which also does nothing for an amount below 1. A reduction sends
  its goods the same way with `acceptUnits(l, from, region, commodity, qty,
  reason)` (D106), once per commodity held, `qty = x - floor(x * w' / w)`. So the
  minimum of FR-EST-014 and the bound of FR-EST-022 have one home, and neither row
  calls the ledger on the estate account, for money or for goods, in any other way.
- **D106.** The intake. `receive(l, from, region)` is AGT 3.6 steps 3 and 4 in one
  call: the whole balance of `from` by `estate_money` (none if 0), then every lot
  of `from` in key order by `estate_goods` with `MoveLot` (so keys are kept),
  returning `(money, lots, error)`. `acceptGoods(l, from, region, reason, only)`
  is the same for whole lots of the commodities `only` selects (nil: all), for a
  role change (SPN 4.2); both return the lot count of D101 (the count of
  `GoodsListed`, a role change's and a removal's only). A weight **reduction**
  moves a share of each commodity, not whole lots (spec-agents.md 3.7:
  "the difference goes to the estate"; spec-scale.md 4.8; FR-LED-051 reasons
  `split`, `merge`, `starvation`): `acceptUnits(l, from, region, commodity, qty,
  reason) (units int64, err error)` is `l.Move(from, estate, commodity, qty,
  reason)` (the ledger takes the oldest lots first, then the lowest durability,
  and keeps each lot's key), a no-op returning 0 for `qty < 1`, and its returned
  `units` are what the executor sums into `WeightChanged.Units` (units, not lots;
  a reduction has no `GoodsListed`). A lot that joins a held key
  is merged by the ledger; the count is of the agent's lots moved. They run when
  no reservation is open (AGT 3.6: removal runs after clearing and consumption), so
  a refusal is an invariant violation and calls the root `invariant` helper of
  `clearing.go` (its message satisfies architecture.md 8.1 and the guard in
  `determinism_guard_test.go`; no hand-written `panic`), except a balance overflow of the estate account, which is
  returned as `ErrBalanceOverflow` for the world to fail the step as WLD 3.3.3
  says. The close of the agent's account (step 5) and the events are the fan-in's.
- **D107.** FR-EST-010 and the snapshot. The snapshot has no row for the estate:
  OBS 5.1.2 puts pseudo-agents only where a row is about them, REG 3.6.1 and EST
  2.7.6 fix the estate's agent ID as its `RegionID`, FR-CHK-048 gives it no agent
  row and the pool is `Regions[].MoneyPool`. Phase 1 rewords FR-EST-010 to "the
  estate pseudo-agent MUST NOT be counted in the number of agents or households,
  MUST NOT cause an `AgentSpawned` event, and MUST have no row among the agents of
  a snapshot; its identity is the `RegionID` of the region row and its money is
  `MoneyPool`". **Ruled by the coordinator (2026-10-17):** OBS owns the snapshot (OBS 5.1.2,
  FR-OBS-007); there is nothing to mark, and no snapshot marker is added.
- **D108.** Construction. `openEstates(l, regions)` opens the accounts with
  `Open(KindEstate, owner = region, region)` in ascending region order, after the
  boundary accounts and before every other account, and returns the `estates`
  value; `seedPool` creates a region's pool by `initial` while the ledger is in
  construction, in region order, skipping an amount of 0 (WLD 5.6 item 5). The
  account IDs are a function of the commodity and region counts (LED 3.4.3: the
  estate of `north` in pack `one` is ID 64 and the first household 65), and a test
  reads them. `openEstates` requires that every existing account is of kind Boundary (a loop over `l.Account(i).Kind`, else an invariant; not a count, since the declared money-flow and module boundary accounts of FR-LED-052 open before the estates and make `Len()` larger than `1 + 7C` in Stage 2), since FR-EST-005 and LED 3.4.3 depend on it.
- **D109.** The checkpoint read and rebuild. `rows(l, region, dst)` reconciles the
  rows against the lots the ledger holds (`AppendLots`, so it takes the ledger) and
  returns the rows of the **held** lots in key order (a stale row is not saved: a
  lot that sold or spoiled after the last P3.4 has no row to save), and
  `restore(l, region, rows)` rebuilds the rows and refuses, with an unexported
  package sentinel (`errListingRows`, wrapped with `%w` and the offending key) that
  the Stage 2 checkpoint row wraps in `ErrCheckpointCorrupt` (that sentinel does not
  exist yet), rows out of order or repeated, a price below 1 or above `MaxPrice`, and a row whose key is not a lot the estate holds (FR-CHK-023). Neither takes `cfg`: `rows` is **filter-only** (it returns the held lots that have a row, inserts nothing, and a held lot with no row is omitted, which is fine because it is priced at the first price when next listed), and `restore` checks a key by requiring it to be a held lot (a held lot is in the commodity's ranges by the ledger's invariants), so a key outside the ranges is tested as a row for a key not held. Inserting a row and the first price are `asks`' alone (it has `cfg`). The listing's `asked` flag is scratch of the
  tick and is not saved (a checkpoint is taken between ticks). Phase 1 writes the
  "held lots only" rule into CHK 4.2 tag 8 and EST 2.7.5.

## Assumptions this plan makes (flagged, not decided)

Sequencing and implementer-level calls. Challenge any of them in review.

- **A1.** The signatures in this plan and in the Cross-plan contract are the
  implementer's, within EST and architecture.md 2.9; they are indicative of the
  capability. The coordinator asked for exact signatures in Provides, so they are
  given there as the intended ones; a row that calls them checks the code of this
  row once it is merged.
- **A2.** `estates` holds only slices indexed by region (accounts, rows, asked
  marks) and a scratch slice of lots reused across ticks (architecture.md 2.9:
  reuse scratch held by the world); a warmed P3.4 and P5.3 walk allocates nothing
  (NFR-EST-001, tested with `testing.AllocsPerRun`).
- **A3.** As plan 1-6 A8: until `NewWorld` exists the root file's cases run from an
  internal test (package `economysim`) with small configuration literals; what
  needs the pack `one` (the account numbers of LED 3.4.3, the reference prices)
  runs from one external test (package `economysim_test`) that loads the pack
  through `config/` and reaches the unexported functions through the existing
  `export_test.go`. No integration test is possible in this row; the golden file
  of the world row exercises the estate end to end.
- **A4.** One PR per phase, in phase order, on a branch from `stage-1-7`. Phases 3
  and 4 implement spec text of Phase 1 (D101 to D104, D106, D109), so neither is
  merged before Phase 1 is. The row shares `doc.go`, `doc/notes.md`, the roadmap
  and the index with its sibling rows: each edit is a small targeted edit after a
  re-read.
- **A5.** The row's own tests build a real `book` (plan 1-3) with a real ledger to
  admit and clear the estate's asks, so the conservation and the
  `side_not_allowed` tests go through the real admission rather than a fake. The
  buyer in those tests is a plain agent account opened with `Open(KindAgent, …)`;
  no agent type is needed.
- **A6.** The unit of "wholly sold" at P5.3 is the ledger: a lot sold wholly is no
  longer held, so a held lot with an admitted ask this tick was not wholly sold.
  This needs no fill count from the clearing, and it also covers a partial fill
  (the lot is held with fewer units). `book.filled()` is not read.

## Phase 1: settle the owed spec lines and the notes

**Why first:** D101 to D104, D106, D107 and D109 are rules the code implements; a
code change that invents them would put a result-affecting choice in a place the
configuration hash and the checkpoint cannot see (architecture.md 9.5). This
phase changes specs and notes, so it starts by reading `doc/notes.md` and quoting
its tag (`AGENTS.md`), handles every open note tagged EST and ALL, and for the specs it edits
AGT, OBS, CHK and MON (below), and gives `doc/notes.md` a fresh tag.

Deliverables, each in the document that owns it:

1. **EST layer line and 2.6, 3 (N-219).** The layer line says "module for the
   regional pool, starting money and relief"; it becomes "core for the estate
   account, its pseudo-agent, its listings and the use of the pool to fund an
   entrant; module for relief and the pool as a money-flow target" (CFG 6.2.4).
   Section 3 says that `EstateMarkdown` is built with the core and `PoolRelease`
   and `ReliefBelow` are appended to `Region` with the module. The layer line of
   `spec-estates.md` and open question 8 of `spec-money-supply.md` ("the spec
   assumes they do not") are answered in the same change: the account exists
   without the module and the pool target needs it (`ErrModuleDependency` stays).
2. **EST 2.7.3, 2.7.5, FR-EST-007, 008, 009 (D102, D103, D104).** State: the ask is
   for all units of the lot; a row is created at the first walk that sees the lot
   and dropped when the lot is gone (at the P3.4 walk and at P5.3); a lot that receives units keeps its price; an
   ask that the clearing rejects is not an ask and is not marked down; the
   markdown formula with the truncated reduction and the floor; the 128-bit
   product; the first-price arithmetic `trunc(ref * left / full)` through the
   checked helper, `full` the commodity's `Durability`, a commodity with
   `Durability` 0 listed at the reference price with no factor.
3. **EST 2.1, 2.5 and AGT 3.6 steps 3 and 4 (D101, D106).** Say once, where the field is defined (OBS 4.1 event table, AGT 14.1, SPN 13.1, EST 2.5), what is listed: the
   lots (ruled: lots, one count per distinct key of the agent), and that `EstateOpened` and
   `AgentBankrupted` carry the same count; one function moves the money and the
   lots, in key order, keeping the keys. State the invariant failure for a refusal
   with no reservation open.
4. **EST 2.3, 2.8.1, FR-EST-014, 022 (D105).** State that the draw is
   `min(want, pool)` where `want` is the executor's whole households times the
   amount, that a draw below 1 is no transfer, and that the same function serves
   `start_money` and `births`; that the weight-reducing transfers are the same
   function in the other direction, money by `acceptMoney` and goods by
   `acceptUnits` (`floor(x * w' / w)` kept, the difference to the estate, AGT 3.7,
   SCL 4.8), and that a role change's goods are whole lots by `acceptGoods`.
5. **EST FR-EST-010 and 2.7.4 (D107).** The rewording of D107 (ruled); EST 2.7.4 says "the snapshot marks it as a pseudo-agent", which contradicts it and OBS 5.1.2, so that clause is reworded the same way (no `AgentSpawned`, no snapshot row or marker).
6. **`spec-checkpoint.md` 4.2 tag 8, FR-CHK-023 and EST 2.7.5 (D109).** The
   section writes only the rows of held lots; a held lot with no row is priced at
   the first price when the world next lists; restore refuses the cases of D109.
   Re-read the table just before the edit: plans 1-5 and 1-6 and the ledger row
   edit it too.
7. **`spec-world.md`: a check, no edit expected.** Re-read 5.2 order 4, 5.6 item 5,
   P3.4, P5.3 and C6, C15, C21 for agreement with D102 to D105; edit only a
   statement that contradicts them (none is known).
8. **The roadmap and `doc/documentation_index.md`.** Row 8's Plan cell links this
   plan with "5 phases" and its Status says "plan audited"; the index has one row
   for this plan. Nothing else.
9. **Notes.** N-254 to N-258 already exist in `doc/notes.md` (raised with this
   plan, 2026-10-17): N-254 (D102: a lot that gains units keeps its price), N-255
   (D101: what `GoodsListed` counts, with the three events), N-256 (D109: the
   checkpoint writes held-lot rows only and restore refuses a row for a lot not
   held), N-257 (D107: FR-EST-010 against OBS 5.1.2) and N-258 (what the world
   row must call: the order of P3.4 then P5.3, the "had an ask" mark, the tally,
   and that the estate's order is built and admitted through `estates` and
   `book.admit` only); N-266 (plan 1-9) lists the calls row 11 makes for the
   spawner. This phase closes N-219, N-255 and N-257 (the latter two by the
   rulings of 2026-10-17, D101, D107), updates N-254, N-256, N-258 and N-266
   with `acceptUnits` (D106) and with the stale-row drop at P5.3 (D102), and
   writes the disposition table below. The tag is replaced with a fresh random
   value on each edit of the file.

**Notes disposition** (apply, reject or leave open, with what was learned):

| Note | Disposition |
|------|-------------|
| N-082 Relief and entry share the pool in time (EST) | **Leave open.** Relief is the Stage 2 module's. Learned: the entry draw of this row is the only P7.1 draw on the pool; P1.6 relief and P7.1 entry stay in the world row's order (already applied for WLD) |
| N-096 The regional pool serves births, entry, relief and money flows (MON, POP, EST, SPN) | **Leave open** for MON, POP, SPN. Learned: `grant` and `acceptMoney` (D105) are the one place the pool is drawn or paid by an entry, a birth or a weight reduction; relief and a pool-targeted flow append to the same file in Stage 2 |
| N-119 The receipts of a household that feeds itself (WEL, EST) | **Leave open.** Relief is not built here; the provenance question is SUB's and EST's, for the module |
| N-167 The goods of a removed debtor do not pay its creditors (EST, CRD) | **Apply for the slice.** The intake lists every lot at the first price and the proceeds are the pool's (D106, D102); a test shows that a removal moves the goods to the estate with no payment to any claim. Left open for CRD's waterfall (money only) |
| N-219 Edits owed to EST and MON by the estates answer (EST, MON) | **Apply and close.** Deliverable 1 edits the EST layer line and answers `spec-money-supply.md` open question 8 |
| N-254 A lot that gains units keeps its listed price (EST, CHK, CODE) | **Leave open until the code merges** (Phase 4), then close. Learned: D102 drops the row of a lot that is gone at P5.3 as well as at the P3.4 walk, so a same-key arrival after a wholesale sale is priced at the first price, as after a restore |
| N-255 What `GoodsListed` counts (EST, SPN, CAP, OBS) | **Apply and close** with Phase 1 (D101: lots; units are `WeightChanged.Units`) |
| N-256 The checkpoint saves listing rows of held lots only (EST, CHK) | **Leave open until the code merges**, then close. Learned: `rows` takes the ledger to know which lots are held (D109) |
| N-257 FR-EST-010 against OBS 5.1.2 (EST, OBS) | **Apply and close** with Phase 1 (D107) |
| N-258 What the world row calls on the estate (EST, WLD, CODE) | **Leave open** for row 11. Learned: its list gains `acceptUnits` for a weight reduction's goods (D106) |
| N-266 What the world row calls for the spawner (SPN, WLD, AGT, EST) | **Leave open** for row 11. Learned: `acceptMoney`, `acceptUnits` (a reduction's goods, per commodity), `acceptGoods` (a role change's whole lots), `receive` and `grant` match Provides |
| N-016 Keep the parameter registry true (ALL, CFG, WEL) | **Leave open.** The row adds no field and no default (`EstateMarkdown` is a Stage 0 row; `PoolRelease` and `ReliefBelow` arrive with the module) and deletes no `notBuilt` entry; the registry test is Stage 0's staged test |
| N-017 Words and names (ALL) | **Apply.** The terms go to `doc.go` before use: estate pseudo-agent, listing, listed price, first price, markdown (qualified: the estate's markdown), pool (qualified: the estate pool), intake, draw (qualified: the pool draw; "draw" has the random sense), held lot. "Treasury" is not used. The note stays open |
| N-027 No default macro constant is stable yet (ALL, REF, STY) | **Leave open.** `EstateMarkdown` 500 stays a labelled placeholder (N-015); no test asserts a default as stable |
| N-103 Running several spec writers in parallel (ALL) | **Leave open.** Process note; applied: six plans written in parallel with ranges, targeted edits of the shared files and the `'"'"'` search after the edit |
| N-126 A new event or `Warn` adds a row (ALL) | **Apply.** The row declares no event and no `Warn`: `EstateOpened` is a Stage 0 event row and the world row emits it. Left open for later rows |
| N-128 A hosted per-agent field is saved once, in the agent record (ALL, CHK) | **Leave open, not applicable.** The listing prices are per region in CHK section 8, not an agent field; the estate has no agent record (FR-CHK-048) |
| N-163 The word "treasury" (MON, FIS, ALL) | **Apply.** Code and comments say "pool" for the estate account and never "treasury" for it. Left open for its owners |
| N-217 D5 binds every module's change (ALL, CODE) | **Apply.** Nothing appends to the configuration, an event or a snapshot row; the exit gate checks the pinned hash. `testdata/golden/off/` does not exist yet |
| N-235 One combined architecture.md edit for the fan-in and admission touch points (ALL) | **Leave open.** The row adds no fan-in and no admission touch point: the estate reaches the admission as a plain pseudo-agent through `orderFacts`, and `removeAgent` is the world row's. Learned: the estate is the first caller of a named lot, tested through the existing `admit` |
| N-228, N-232, N-236, N-243, N-214, N-216, N-222 (CODE and AGT, PRO) | **Left open, not touched.** The row adds no tradable-set item, no job, no draw and no site |
| The other open notes tagged AGT, OBS, CHK, MON and SPN | **Left open, not read as touched.** The clauses edited are AGT 3.6 steps 3 and 4 (a wording), FR-EST-010 against OBS 5.1.2 (no OBS edit), CHK 4.2 tag 8 and MON open question 8; re-read the index for notes raised since |

**Tests:** none; documents only. **Observability and rollback:** none, because no
code changes; a wrong edit is reverted like any document change.

**Acceptance:** EST states D101 to D105, D107 (FR-EST-010 and 2.7.4 agree) and D109 and the layer line; AGT
3.6 and CHK 4.2 tag 8 carry their clauses; MON open question 8 is answered; every
open note tagged EST or ALL has a row; the roadmap row reads "plan written" and
links this plan; the notes carry the dispositions and a fresh tag; the text search
of `doc/` finds no stray shell-quoting sequence (N-103).

## Phase 2: the identity, the price rules and the listing state

**Why here:** everything after it stores or compares a price or a row, and the two
price rules and the sorted-row operations are closed forms that can be tested and
fuzzed with no ledger.

- **First, `doc.go`**: the terms of N-017 above, added to the glossary before the
  file uses them.
- `estates.go`: the `estates` type, `agentID(r)` and the reverse `regionOf(a)`
  (`a` below the region count), and the row type `listing{commodity, born,
  durability int64-keyed fields as the ledger's Key, price Price, asked bool}`.
- `resolutionFloor(cfg.World)` returning ok=false (a `BeliefStep` out of bound) cannot happen for a validated configuration; `asks` and `markdown` treat it as an invariant at their one caller (the root `invariant` helper), like a `firstPrice` overflow.
- `firstPrice(ref Price, durability, left int64, floor int64) (Price, bool)`: the
  base is `ref` for `durability == 0` and `checked.MulDiv(ref, left, durability)`
  otherwise, and in **both** branches the result is `max(floor, base)` (EST 2.7.3,
  FR-EST-008: never below the resolution floor; a non-tradable or dear commodity can
  have `ref < floor`); `ok` false only on an overflow, which cannot occur for a
  validated configuration (`ref` is bounded by `MaxPrice`, `left <= durability`) and
  is a violated invariant at the one caller (the root `invariant` helper).
- `markedDown(price Price, mdBP, floor int64) Price` of D104.
- The row operations over a sorted slice: find by key (`slices.BinarySearchFunc`,
  the order of LED 5.1.3), insert at the position, the merge-walk that reconciles
  the rows with the sorted lots of the estate account (insert a row for a lot
  with none, drop a row for a lot gone), written once and used by P3.4 (`asks`, which has `cfg` for the first price); `rows` uses the same walk with insertion switched off (filter-only, D109).

**Tests (unit, base of the pyramid):** table tests of both price rules with the
arithmetic in the comment (testing.md 5.5): reference 100, left 3 of 10: 30;
floor binding: reference 60, left 1 of 10, floor 50 gives 50 (not 6); a
non-durable commodity gives the reference price, and with a reference price below
the floor (reference 3, floor 50) gives the floor; markdown 500 on 60 gives 57 (reduction
`60 * 500 / 10 000 = 3`), on 19 gives 19 (reduction truncates to 0), at the floor
stays, 0 basis points stays, 10 000 goes to the floor; the overflow case of
`firstPrice`, built with out-of-contract input (`left > durability`, e.g. `ref =
1<<62, left = 4, durability = 1`), since no in-contract input overflows. Row operations: insert keeps the order, a duplicate key is one row,
the merge-walk with lots {a, b, c} and rows {b, d} gives rows {a, b, c} with b's
price kept. A fuzz target over the merge-walk (any sorted lots and rows give
sorted unique rows over exactly the lots) and over the two rules (never below the
floor, never above the previous price, never panics). Requirements: FR-EST-008,
009 (formula), 007 (rows), 005 (the ID of the estate is the `RegionID`).

**Observability and rollback:** none: no failure path is survived, no event, no
log line. A phase is rolled back by reverting its PR; nothing calls it yet.

**Acceptance:** the table tests, the fuzz seed corpora and the allocation test of the
walk (no allocation with a warmed scratch) pass; `go vet`, `staticcheck` and
`gofmt` are clean.

## Phase 3: the accounts, the pool and the intake

**Why here:** the listing walk reads lots the estate holds, and a test needs lots
in the estate account before it can list; the pool read, seed, draw and intake
are all calls on the ledger and share one set of invariants.

- `openEstates(l *ledger.Ledger, regions int) (*estates, error)` (D108): `Open(KindEstate,
  r, r)` in ascending order; the account IDs are kept in a slice by region.
- `(*estates).seedPool(l, r RegionID, m Money) error`: `Transfer` from the
  `initial` boundary account of money (`l.Boundary(ledger.Initial, ledger.Money)`)
  into the estate with the reason `initial`, only while `l.InConstruction()`
  (otherwise it returns `ledger.ErrNotConstruction`: the ledger itself would accept
  an `initial` transfer during a run, AGT 3.5.4); an amount of 0 is skipped.
- `(*estates).pool(l, r) Money` from `l.Account(acct)` (`(View, error)`: an error
  is an invariant, the account was opened by `openEstates`) `.Balance`.
- `event.go`: the comments of `AgentBankrupted.GoodsListed`, `EstateOpened.GoodsListed`
  and `RoleChanged.GoodsListed` say "number of lots" (D101, ruled); a comment edit only.
- `grant` and `acceptMoney` (D105); `receive` and `acceptGoods` (D106) with the lot
  count (D101). `acceptGoods` takes the commodity selector as a `func(int) bool`
  (nil for all); the closure does not escape. `acceptUnits` (D106) is the
  weight-reduction half: a share of one commodity by `l.Move`, the returned units
  feed `WeightChanged.Units`.
- Reuse, no new code for: the transfer primitive, the reasons and their pairing
  rules (a wrong pairing is `ErrReasonMismatch`, so a caller using a wrong reason
  fails loudly), the `MoveLot` key preservation, `Audit` for the invariant tests.

**Tests:** unit tests on a small real ledger (A5): accounts open in region order
with IDs `1 + 7C + r`; `seedPool` creates the amount by `initial` and shows in the
construction bucket; `grant` gives `min(want, pool)`, nothing for 0 and gives the
whole pool when short; a draw never leaves the pool negative; `receive` moves the
whole balance and every lot with keys kept, returns the lot count of D101, moves
nothing and returns zero for an empty agent; `acceptGoods` with a selector moves
only those commodities; `acceptUnits` moves exactly `qty` (partial quantity of a
lot, across two lots oldest first), returns the units moved, is a no-op returning 0
for `qty < 1`, and conserves goods (`Audit`); `openEstates` on a ledger that already holds a non-boundary account is an invariant, and on one with extra boundary accounts (declared reasons) it succeeds; the goods are not paid for (N-167); the audit passes
after each. **Invariant test** over seeds 1 to 50 (testing.md 6.2: the failing seed is
printed): a script of removals, grants and seeds keeps the conservation of
money and goods by exact equality (`Audit` and the sum agents plus pool).
Boundary and error paths: a refused transfer with a wrong reason is
`ErrReasonMismatch` (`errors.Is`); an estate balance overflow is returned as
`ErrBalanceOverflow`; a seed outside construction is `ledger.ErrNotConstruction` (`errors.Is`). An external test
loads pack `one` and asserts the estate account is ID 64 and the first
household account would be 65 (LED 3.4.3). Requirements: FR-EST-001 (the estate's
part), 004, 005, 013 (in part: the pool read equals the account), 014, 015 (in part:
the pool seed), 016 (no money outside a named reason), 021 (the counts the event
carries), 022; FR-EST-002 is the fan-in's (row 11).

**Observability and rollback:** none: the functions emit nothing (the world row
emits `AgentBankrupted`, `EstateOpened` and `AgentSpawned` behind the observer
flag, so the no-op path allocates nothing, observability-and-logging.md 4.5), and
a refusal is returned or is an invariant, never logged and returned (8.3). Rolled
back by reverting the PR.

**Acceptance:** the tests pass with `-race`; the invariant test passes for 50 seeds;
the pack-`one` account numbers are asserted.

## Phase 4: the asks, the markdown and the checkpoint read

**Why here:** the listing needs the accounts of Phase 3 and the rows of Phase 2, and
the checkpoint read needs the rows to be settled.

- `(*estates).asks(l, cfg *Config, r RegionID, dst []Order) ([]Order, error)`: for
  the estate account's lots (`AppendLots` into the scratch), reconcile the rows
  (Phase 2), clear every `asked` mark, and append one `Order` per held lot as in
  D103, ascending key; the floor from `resolutionFloor(cfg.World)` (read the field `cfg.World`; plan 1-11 renames its type to `WorldParams`, N-284). It allocates
  only when `dst` has no room.
- `(*estates).facts(r RegionID, side Side, tally *receiptTally) orderFacts` (D103).
- `(*estates).admitted(r RegionID, o Order)`: sets the row's `asked` mark, called by
  the world after `book.admit` returns true for the estate's order. No mark is
  set for a rejected ask (D103).
- `(*estates).markdown(l, cfg, r RegionID) error` (P5.3): for every row with `asked`
  whose lot is still held, `price = markedDown(...)` with the region's
  `EstateMarkdown`; drop the row of every lot that is no longer held (D102); clear
  the marks.
- `(*estates).rows(l *ledger.Ledger, r RegionID, dst []listing) ([]listing, error)`
  (filter-only against `AppendLots`: a stale row is not returned and none is inserted; no `cfg`) and `(*estates).restore(l, r, rows) error` (D109; no `cfg`: each key must be a held lot).
- A marked comment line "the pool policy of the Estates module (relief, the pool as
  a money-flow target) is appended here in Stage 2" at the end of the file.

**Tests:** unit and the lowest-level integration the row can make, A5. Two real `book`s (a book is per commodity and `admit` calls `invariant` on another commodity; the asks are routed by commodity as the world does) with a plain buyer and an estate holding three lots (one non-durable lot in the first commodity's book; two lots of a durable, remaining durability 3 and 6 of 10, in the second's): the three asks
are placed with the prices of FR-EST-008 (reference 100 and a fixture `BeliefStep`
of 2500, so the floor is `ceil(10 000 / 2500)` = 4 and binds nowhere: 100, 30, 60;
the default 200 gives floor 50 and the middle lot would list at 50), in key
order, each naming its lot; a bid from the estate is refused with
`side_not_allowed` by the real admission and an ask is admitted with a
reservation of goods and no cash (the estate has 0 money and the pool is not
touched, FR-EST-006); the buyer's bid, in the durable's book, clears against the cheapest ask (30) and the
proceeds are a `trade` transfer into the estate, so the pool read equals the
account (FR-EST-013); an unsold lot falls by the markdown at P5.3, a wholly sold
lot has no row afterwards, a partly sold lot falls, and an ask rejected by the
clearing (a receipt over `MaxMoney`) is not marked down (D103). Units joining a
marked-down lot are listed at its current price (D102). A lot sold wholly, then
goods under the same key arrive before the next walk: the next `asks` lists the
first price, and equals a restored world's (D102, FR-EST-024). `rows(l, ...)` skips
a stale row; `restore` of `rows` gives the same listing and the next `asks` equal an
uninterrupted run's; the refusals of D109 each have a case, asserted with
`errors.Is(err, errListingRows)`. **Determinism and
non-interference:** the asks of two runs are equal and the same estate yields the
same order whatever the insertion order of the lots. **Allocation test**
(NFR-EST-001, observability 8.2): a warmed `asks` plus `markdown` over three
regions allocates nothing. Fuzz: `FuzzRestore` (never panics, a valid `rows`
round-trips, an invalid input is an error). Requirements: FR-EST-006, 007, 008,
009, 011 (the `regionOf` ID test only; the real check, that no `AgentSpawned`,
`AgentBankrupted`, `WeightChanged` or `RoleChanged` subject is an estate, is plan
1-11's), 012 (in part: the order carries the estate's ID; the placement before any
market clears and the tie-break key are the world's and the clearing's), 013 (the
pool read equals the account), 024 (in part: the read and rebuild), NFR-EST-001.

**Observability and rollback:** the events of an estate's order are the clearing's
(`OrderPlaced`, `OrderRejected`, `TradeExecuted` carry the estate's agent ID,
emitted by the world row); the row adds none. A restored lot with no row is priced
at the first price: no `Warn`, since it is expected. Rolled back by reverting the
PR; no other row has started to use it.

**Acceptance:** the tests above pass; the allocation test passes; the fuzz seed corpus
is committed under `testdata/fuzz/`; the restore refusals are asserted with
`errors.Is`.

## Phase 5: the exit gate

- On the whole module: `gofmt -l .`, `go vet ./...`, `staticcheck ./...`,
  `go test -short -race ./...`, the full run with `-covermode=atomic
  -coverpkg=./...`, `go mod tidy`; the per-scope coverage read and recorded (the
  root at least 90%, testing.md 9.1).
- The pinned hash of the all-off configuration did not move (D5): the existing
  `TestHashConfig_ReferencePinned`. `testdata/golden/off/` does not exist yet, so
  that test alone is the gate until the world row records the first golden file.
- The traceability check of testing.md 4.6 by text search: every ID in a
  `Requirement:` line exists in EST, and every requirement the table below lists
  as covered in full is cited.
- A deliberately broken copy (mutate a comparison of the markdown, drop the
  `min` of `grant`, skip the stale-row drop) under the scratchpad shows the
  tests fail (testing.md 9.3, 5.3).
- The roadmap row set to **done** with its commit, and the notes of Phase 1
  updated with what the code settled (D101 and D107 are ruled).
- Not gating, if time allows: a mutation pass over the root files of this row
  with `gomu` inside Docker only, `--cpus=4 --memory=16g --memory-swap=16g`,
  `GOMAXPROCS` and `--workers` equal to the CPU cap (never on the host).

**Tests:** the gate itself. **Observability and rollback:** none; rollback is
reverting the PRs in reverse order, and no other row has started to use them.

**Acceptance:** every command is clean; the traceability check finds no ID without
a spec row and no "covered in full" requirement without a test; the roadmap row
reads done.

## Cross-plan contract

### Consumes

| Symbol, type, sentinel, spec edit or field | Provided by | Status |
|---|---|---|
| `Region.EstateMarkdown`, `World.BeliefStep`, `WarmStart.Pools`, `Commodity.ReferencePrice` and `.Durability`, `resolutionFloor(cfg.World)` (the type becomes `WorldParams` in plan 1-11, N-284; the field stays) | Stage 0, `config.go`, `validate.go` | Built (023cf97) |
| `internal/ledger`: `Open`, `KindEstate`, `Boundary`, `Transfer`, `Move`, `MoveLot`, `AppendLots`, `Account`, `Key`, `Lot`, `InConstruction`, `Audit`; the reasons and their root names | [plan-phase-1-1.md](plan-phase-1-1.md) | Built |
| `Order`, `LotKey`, `Side`, `orderFacts`, `book.admit`, `receiptTally`, `ReasonSideNotAllowed`, the reason of each rejection | [plan-phase-1-3.md](plan-phase-1-3.md) | Built |
| `internal/checked.MulDiv` | Stage 0 | Built |
| The commodity table and `cfg.Commodities[id]` read by `CommodityID` (no wrapper, plan 1-2 A10) | [plan-phase-1-2.md](plan-phase-1-2.md) | Built |
| The world's field that holds the `estates` value, the construction calls (`openEstates` at WLD 5.2 order 4, `seedPool` at 5.6 item 5; `WarmPool.Region` is a `string` (config.go), so row 11 maps it to a `RegionID` before calling `seedPool(l, r RegionID, m)`), P3.4, P5.3, the removal fan-in and the entry step, the owned `receiptTally` and `clearScratch` | `world.go` row (11) | **ASSUMED**: row 11 calls what is listed under Provides, in the positions named there |
| The agent's account and region at removal, the agent type's weight and the close of the account after `receive` | `agent.go` row (7) | **ASSUMED**: the fan-in calls `receive` with the agent's account before it closes it |
| The caller of `grant` (the executor of entry, births and weight rises, with the whole-household budget) and of `acceptMoney`, `acceptUnits` (weight reductions) and `acceptGoods` (role change) | The world executor, `world.go` row (11); rows 7 and 9 name the functions in their Consumes only | **ASSUMED** |
| The rulings on what `GoodsListed` counts (D101: lots) and on FR-EST-010 (D107) | Coordinator | Ruled 2026-10-17 |
| EST layer line, 2.1, 2.3, 2.5, 2.7.3, 2.7.5, FR-EST-007 to 010, 014, 022, AGT 3.6, CHK 4.2 tag 8 and the MON open question 8 edits | This row, Phase 1 | Owed here |

### Provides

All unexported, in package `economysim`, in `estates.go`. Exact intended
signatures (A1):

| What | For | Contract |
|---|---|---|
| `type estates struct{…}`; `openEstates(l *ledger.Ledger, regions int) (*estates, error)` | `world.go` (construction, WLD 5.2 order 4) | Opens one `KindEstate` account per region in ascending `RegionID`, after the boundary accounts and before every other account (an invariant if an account of a kind other than Boundary exists); no pseudo-agent exists for an outside node's missing market, but the estate does (REG 3.6.1) |
| `(*estates).agentID(r RegionID) AgentID`; `regionOf(a AgentID) (RegionID, bool)`; `account(r RegionID) int` | `world.go`, `clearing` callers | The estate's agent ID is the `RegionID`; `regionOf` is true for an ID below the region count; the estate is in no count and gets no strategy call, no belief and no outcome |
| `(*estates).seedPool(l *ledger.Ledger, r RegionID, m Money) error` | `world.go` (WLD 5.6 item 5) | `initial` into the estate during construction only; 0 is skipped |
| `(*estates).pool(l *ledger.Ledger, r RegionID) Money` | `world.go` (`MoneyPool`, `EntryBudget`, the entry budget, and the view builder that fills the spawner view's `PoolMoney`) | The estate account's money balance (nothing in it is reserved) |
| `(*estates).grant(l *ledger.Ledger, r RegionID, to int, want Money, reason ledger.Reason) (Money, error)` | the world's entry executor, births and weight rises | Gives `min(want, pool)` to account `to` with `start_money` or `births`; 0 for `want < 1` or an empty pool; never negative; the caller takes the budget and counts the unfunded households |
| `(*estates).acceptMoney(l *ledger.Ledger, from int, r RegionID, amount Money, reason ledger.Reason) error` | the world executor (11), weight reductions (`split`, `merge`, `starvation`; `deaths` arrives with the population module) | Agent to estate; no transfer for `amount < 1` |
| `(*estates).acceptUnits(l *ledger.Ledger, from int, r RegionID, commodity int, qty int64, reason ledger.Reason) (units int64, err error)` | the world executor (11), the goods of a weight reduction (`x - floor(x * w' / w)` per commodity, AGT 3.7) | `l.Move` of `qty` units to the estate (oldest lots first, keys kept); 0 and no transfer for `qty < 1`; `units` feeds `WeightChanged.Units` |
| `(*estates).receive(l *ledger.Ledger, from int, r RegionID) (money Money, lots int64, err error)` | `removeAgent` (AGT 3.6 steps 3 and 4), `world.go` row (11) | The whole balance by `estate_money`, every lot by `estate_goods` in key order, keys kept; `money` is `AgentBankrupted.Money` (the agent's money passed); `EstateOpened.Balance` is "the money the estate holds" (`event.go`), so the world reads `pool(l, r)` after `receive` for it (O6); `lots` is `GoodsListed` (D101). The caller closes the account and emits |
| `(*estates).acceptGoods(l *ledger.Ledger, from int, r RegionID, reason ledger.Reason, only func(commodity int) bool) (lots int64, err error)` | the world executor (11), role change (SPN 4.2, `RoleChanged.GoodsListed`) | The whole lots of the selected commodities (nil: all) to the estate; the same count as `receive`. A weight reduction's goods are `acceptUnits`, not this |
| `(*estates).asks(l *ledger.Ledger, cfg *Config, r RegionID, dst []Order) ([]Order, error)`; `facts(r RegionID, side Side, tally *receiptTally) orderFacts`; `admitted(r RegionID, o Order)` | `world.go` P3.4 (region ascending, after the agents' orders, before the treasuries', WLD 6.5.1) | One named-lot ask per held lot in key order at its current price; the world admits each through `book.admit` with `facts(r, Ask, tally)` and calls `admitted` for each admitted one; a bid with `facts(r, Bid, …)` is `side_not_allowed`; no outcome, no belief word |
| `(*estates).markdown(l *ledger.Ledger, cfg *Config, r RegionID) error` | `world.go` P5.3 | Marks down the lots with an admitted ask that are still held, by D104; clears the marks |
| `(*estates).rows(l *ledger.Ledger, r RegionID, dst []listing) ([]listing, error)`; `(*estates).restore(l *ledger.Ledger, r RegionID, rows []listing) error` | `checkpoint.go` (Stage 2, tag 8) | The rows of held lots in key order (filter-only: no insertion, a held lot with no row is omitted; no `cfg`); a validated rebuild (D109, a key must be a held lot) that refuses with the unexported `errListingRows` (wrapped by the checkpoint row in `ErrCheckpointCorrupt`) |
| The glossary terms of N-017 in `doc.go` | all later rows | Added before use |
| The marked place in `estates.go` for the module half | the Stage 2 `estates.go` row | `PoolRelease`, `ReliefBelow`, relief at P1.6, `ReliefPaid` and the pool target append after it (architecture.md 2.10.5) |

## Security review

The module is a library with no network, file access, credentials or
persistence. This row adds no trust boundary: `estates.go` is called only by the
world with accounts and numbers it derived from a validated configuration and the
ledger.

- **Who can reach it.** A consumer reaches the estate only through the world's
  queries and the events (later rows). A consumer's `Strategy` never sees the
  estate's ID as an agent, cannot place an order for it and cannot change a price;
  a consumer's `Clearing` sees the estate's ask as an ordinary order (EST 2.7.4)
  and is checked by the trade-list check like any (plan 1-3).
- **Integer overflow.** The one product that can overflow, `ref * left /
  durability`, and the markdown product go through `internal/checked`; the
  estate's balance and a lot's quantity are bounded by the ledger (`MaxMoney`,
  `MaxQuantity`), and a balance overflow of the estate is returned for the world
  to fail the step (D106), never wrapped. An ask whose receipt would exceed
  `MaxMoney` is rejected by admission (`receipt_limit`), not wrapped.
- **Corrupt input.** A restore feeds `restore` rows from outside the process
  (Stage 2): it refuses rows out of order, repeated, out of range or for a lot
  not held (D109), so a hostile checkpoint cannot make the estate list a lot it
  does not hold or a price outside `MaxPrice`.
- **Resource exhaustion.** The rows are at most one per held lot; the lots of an
  estate are bounded by the keys that exist (a non-spoiling, non-wearing
  commodity is one lot; a spoiling one fragments by born tick, bounded by its shelf
  life; a durable by its durability), the same bound as `Markets[].Lots`. A
  consumer cannot grow it faster than it can remove agents.
- **Manipulating the estate.** A strategy could dump goods on the estate by
  starving an agent to remove it; the goods are listed at the reference price
  and fall by the markdown, and the money goes to the pool: this is the designed
  substitute for inheritance (EST 2.4), not an exposure of this row, and its
  stability is STB-L38 and L39.
- **Not applicable:** authentication, authorisation, injection, secrets,
  cross-tenant exposure, supply chain (no dependency is added).

## Compatibility and rollout

- **No code contract exists to break.** There is no release and no consumer.
  Nothing in this row is exported from the module; it adds unexported
  identifiers only.
- **Document contracts.** Phase 1 adds text to EST, AGT 3.6, `spec-checkpoint.md`
  4.2 tag 8 and the MON open question and the notes, and renumbers nothing
  (testing.md 4.1); `doc/notes.md` gets a fresh tag. The checkpoint row of Stage 2
  reads the tag-8 clause.
- **What becomes hard to change, and when.** The first price, the markdown and the
  merged-lot rule (D102, D104) enter the golden files at the end of Stage 1 and
  the checkpoint in Stage 2. Until the first golden file is recorded they may be
  corrected in a normal PR; after it, a change is a behaviour change with the
  explanation of testing.md 7.2. The count of `GoodsListed` (D101) is in the
  golden rendering of the events.
- **Append-only stays true.** The module half appends to `estates.go`, `Config`
  (`Region.PoolRelease`, `Region.ReliefBelow`, written to the hash only when the
  module is on, CFG 8.2.1) and `event.go` (`ReliefPaid`); with the module off
  nothing is added, so the pinned hash and `testdata/golden/off/` do not move
  (architecture.md 2.10.5, D5).
- **Rollout** is one PR per phase (A4). Each leaves the branch passing every
  command of `AGENTS.md`. A phase is rolled back by reverting its PR; later phases
  depend only on earlier ones and no other row has started to use this one.
- **Dependency failure.** None at run time. The schedule risk is the world row's
  calls: Phases 3 and 4 are tested without a world (A3, A5), so the row can finish
  before the world row starts. The coordinator ruled D101 and D107 on 2026-10-17, before any code.
- **The Go toolchain.** `go.mod` says `go 1.25`. The only ordered structure is a
  sorted slice with an explicit total order (LED 5.1.3); no map iteration and no
  unstable sort is used.

## Testing (summary, testing.md section 11)

The pyramid for this row is all base plus the lowest integration the row can
make (a real book and a real ledger, A5): unit tests in the root's internal test
of `estates.go`, fuzz targets over the price rules, the row walk and `restore`,
an invariant test over 50 seeds and allocation tests. There is no end-to-end test
because there is no world; the world and agent rows add them.

| Phase | Covered in full | Covered in part (completed in) |
|-------|-----------------|--------------------------------|
| 2 | FR-EST-008, FR-EST-009 (the formula and the floor) | FR-EST-007 (rows here; the asks in Phase 4), FR-EST-005 (the estate IDs here; the counterparties and treasuries follow in their modules) |
| 3 | FR-EST-022 | FR-EST-004 (accounts open in region order with only boundary accounts before them, by the D108 guard; "before any household" is the world's order, row 11), FR-EST-014 (the cap `min(want, pool)` and no transfer below 1 here; `w * Role.StartMoney` and "no goods" are the executor's, rows 9 and 11), FR-EST-001 (the estate's part: money and goods in; the waterfall is CRD's), FR-EST-002 (`receive` takes the region from its caller; that it is the region of the agent's account is the fan-in's, row 11), FR-EST-013 (the pool read equals the account here; `MoneyPool` in the snapshot is row 11's), FR-EST-015 (the pool's `initial` seed here; initial agents' `initial` money is the world's), 016 (the reasons here; the sum over all flows is the world's), 021 (the counts here; the event is the world row's) |
| 4 | FR-EST-006, 007, 009 (at P5.3 with an admitted ask), NFR-EST-001 (the walk and the markdown) | FR-EST-011 (only the `regionOf` ID test here; the real check, that no strategy path or agent event subject is an estate, is plan 1-11's), FR-EST-010 (the estate has no agent row: the snapshot is the world row's), FR-EST-012 (the order carries the estate's ID; the placement and the tie-break are the world's and the clearing's), FR-EST-024 (the read and rebuild here; the checkpoint section in Stage 2) |

Deferred with their module or row: FR-EST-003 (a traveller or a diverted cargo:
transport, Stage 2), FR-EST-017 to 020 (relief and `ReliefPaid`: the Stage 2
estates module), FR-EST-023 (`EstateMarkdown` built and bounded in Stage 0; the
two module fields arrive with the module).

Boundaries and error paths: each refusal of D109 and each `ErrReasonMismatch` and
overflow of Phase 3 has a case that breaks only its rule; a bid from an estate;
an empty pool; an agent with nothing; a lot sold wholly; a rejected ask. Golden
files: none yet. Fuzz: the rules, the walk and `restore`. Each PR states its
coverage and that its tests were run against a broken version (testing.md 9.3).

## Observability (summary, observability-and-logging.md section 9)

No event, no metric, no log line is added by this row; "None, because" each:
(1) no failure path is survived rather than returned: a refusal is returned or an
invariant (D106), and a rejected ask is the clearing's `OrderRejected`; (2) the
state transitions it makes are covered by events that exist since Stage 0 and
that the world row emits behind the observer flag, so the no-op path allocates
nothing (`AgentBankrupted`, `EstateOpened`, `AgentSpawned.StartMoney`,
`OrderPlaced`, `OrderRejected`, `TradeExecuted`); a markdown is a price change on
state, saved in the checkpoint and visible in the next ask's `OrderPlaced.Limit`,
so it has no event of its own; (3) the decision points (the first price, the
markdown, the pool cap) are rule constants with no strategy choice, and a
`Debug` line per lot would be per-lot volume; the pool read and the unfunded
count (`EntryLimited.Unfunded`, the "entry unfunded" `Warn`) are the spawner's;
(4) the loops (the walk over lots, the markdown) log nothing and build no event;
(5) no event, metric, unit or attribute changes; (6) no new metric attribute;
(7) neither `Error` nor `Info` is used; (8) no error is logged and returned;
(9) the tests of section 8 that apply here are the allocation test of Phase 4
and the emission-order and cardinality tests, which need a world and are the world
row's.

## Open items

- **O1.** The decisions the specs do not settle (D101 lots against units, D102 a
  merged lot keeps its price and lazy pricing, D103 the ask is the whole lot and a
  rejected ask is not marked down, D104 the truncated reduction, D105 one pool
  draw, D106 the invariant for a refusal, D107 FR-EST-010, D109 held rows only)
  are made in Phase 1; the owner may override any of them before the phase that
  implements it: D102 to D104 before Phase 4 (D101 and D107 are ruled).
- **O2.** `GoodsListed` as lots (D101, ruled) is also the count rows 7, 9 and 11 emit in
  `RoleChanged` and `AgentBankrupted`; the comments of the three events in `event.go`
  are reworded in Phase 3.
- **O3.** No CI exists (O3 of plan 0). The gate of Phase 5 and the timed fuzz runs
  are done by hand, on one architecture.
- **O7.** `GoodsListed` (lots) and `WeightChanged.Units` (units) are different
  counts by design: a role change and a removal move whole lots (`acceptGoods`,
  `receive`), a weight reduction moves a share of units (`acceptUnits`).
- **O6.** `EstateOpened.Balance` is documented in `event.go` as "the money the estate
  holds" while `spec-estates.md` 2.5 says only "balance". The plan reads it as the
  estate's pool after the intake (so a world row reads `pool` once after `receive`);
  if the owner means the agent's money passed, it is `AgentBankrupted.Money` again.
  Phase 1 states the reading in EST 2.5 (the owner may override before Phase 3).
- **O4.** The estate's first price is the static effective reference price. If
  experiments show the starting anchor matters, `spec-skill-barriers.md` open
  question 4 proposes the mean of the price window; that is a spec change to EST
  2.7.3 and a change to `firstPrice` only.
- **O5.** A lot that gains units keeps its marked-down price (D102): dumped goods
  of a later removal join an old lot at its current low price. If the S02 or S14
  scenarios show this lowers the local price too much (STB-L38), a rule that
  reprices a growing lot to the higher price is a one-line change.
