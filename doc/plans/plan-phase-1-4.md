# Plan: Stage 1, row 4: price beliefs

**Status: Proposed (2026-10-13); no phase started.** Sequences the fourth row
of Stage 1 of [roadmap-v1.md](../roadmap-v1.md), "`internal/beliefs` and
`beliefs.go`", into six phases. It implements
[spec-beliefs.md](../specs/spec-beliefs.md) (BEL) for the slice. It is built
against `doc/rules/architecture.md` (the `internal/` boundary of 2.9, the
module-file rule of 2.10, the fixed-point and overflow rule of 3.5, the
determinism rules of section 7, the layout of 2.6), `doc/rules/testing.md` and
`doc/rules/observability-and-logging.md`. Where this plan and the rules or the
specs disagree, they win and this plan is corrected.

A plan decides nothing. Every decision below comes from a spec, a rule, an ADR
or the decisions of [plan-phase-0-foundation.md](plan-phase-0-foundation.md)
(D3 and D5 above all). Where a decision is still owed, Phase 1 makes it in the
document that owns it, before the code that depends on it, and where it is not
this row's to make it is an open item. Decision IDs D40 to D49, assumptions
A1 and following, and open items O1 and following are this plan's; the
decisions of the earlier plans are cited by file and ID.

No PRD exists for this row. The requirements source is the roadmap row and
BEL. The row shares no code with the ledger row
([plan-phase-1-1.md](plan-phase-1-1.md)) or the tables row
([plan-phase-1-2.md](plan-phase-1-2.md)): the rule is a pure function of
integers (BEL 5.8, FR-BEL-017) and takes its inputs from the caller, so none
of the three waits for another.

Spec codes: BEL beliefs, CLR clearing, AGT agents, WLD world, OBS
observability, CFG config, CHK checkpoint, GDS goods, PRO production, VIA
viability, STB stability, MIG migration, MER merchants, TRN transport, INT
integration, IND indicators. `FR-`/`NFR-` IDs are the requirement IDs of those
specs.

## What Stage 0 already built, and what is left

Stage 0 is done (commit `023cf97`) and it built, by reading the code on
2026-10-13:

| Concern | Where it already is |
|---------|---------------------|
| The sixteen belief parameters (`BeliefStep` to `PriceWindow`, `CostFloor`, `CostCeiling`, `RefBandLow`, `RefBandHigh`, `BeliefEvidenceShare`) as fields of `World` with doc comments | `config.go` |
| Their defaults (BEL 11), once | `defaults.go` (`DefaultWorld`) |
| Their validation bounds (`belief_step` 1 to 2 500, `price_window` 1 to 1 000, and the rest) and `MaxPrice`, `MaxQuantity`, `MaxBasisPoints` | `bounds.go` |
| The ordering rules (`belief_stock_low` below `belief_stock_high`, `belief_width_min` at most `belief_width_init`, ..., `cost_ceiling` at least `cost_floor`), the resolution floor `ceil(10 000 / BeliefStep)` against the cheapest tradable price, the warm-start seed checks (`ErrWarmStartPrice`) and the warning `cost_floor_at_full_cost` | `validate.go` (`worldRules`, `resolutionFloor`, the warm-start rules) |
| `WarmStart.Prices` (`SeedPrice`), canonicalised | `config.go` |
| The event `BeliefsUpdated` with the fields of BEL 12.1 | `event.go` |
| `warnRun` (the run length of a bounded `Warn`) | `observer.go` |
| The log keys `region`, `commodity`, `agent`, `price`, `reason`, `bound`, `agents` | `internal/logkeys` |
| The draw site `siteOutcome` (D8: tick, agent, sub the commodity) | `random.go` |
| `checked.MulDiv`, `MulDivHalfEven` | `internal/checked` |

What is left is the rule itself and what it needs to be used: the pure
functions (bounds, the update, the draw, the clamp, the reseed), the price
window, the diagnostic run and the per-market tally of the event, all in
`internal/beliefs`, and in the root the exported value types, the conversion of
the parameters, the reason constants and the tally's mapping to the event. No
configuration field, default, bound or validation rule is added by this row.

## Scope

**In scope.**

- `internal/beliefs` (stdlib and `internal/checked` only): `Belief`, `Bounds`,
  `Params`, `Outcome`, `Quote`, `Result`, `Change`; `Limits` (BEL 3.2), `Clamp` (3.3,
  10.3), `Gate` (5.2), `Update` (4, 5), `Draw` (6), `Reseed` (8.2, 8.3), `Window` with its
  push, its statistics and its restore (7), `Position` (7.3), the diagnostic
  run of the `Warn` (12.2) and the per-market `Tally` of the event (12.1).
- `beliefs.go`: the exported `Belief` and `Bounds` value types, the
  construction of `beliefs.Params` from the world's parameters, the mapping
  from the result of an update to its `Debug` reason, and the mapping from a
  tally to a `BeliefsUpdated`.
- `event.go`, appended: the five `Reason` constants of the "belief updated"
  `Debug` line (BEL 12.2, 12.3).
- The glossary terms these files use, in `doc.go`, before they use them
  (architecture.md 3.2): added in Phase 1, the first phase, so that the code
  and the doc comments of Phases 2 to 5 use only terms that are there.
- The spec, note and roadmap edits these files depend on (Phase 1).

**Out of scope**, with where each is tracked:

| Item | Tracked in |
|------|-----------|
| The belief of an agent: its storage in the agent record, its creation at construction and at a later entry, the view's `Belief`, `Bounds` and window fields, `DefaultStrategy.Revise`, the drawing of a limit price in `Trade`, the `OutcomeView` | Roadmap Stage 1, `agent.go` and `strategy.go` row. This plan gives them `Reseed`, `Limits`, `Update`, `Clamp` and `Draw` |
| The market record: the published price and age, the report, the price window beside them, `BoundRun`, the push at P5.1, the seed price before the first trade (FR-BEL-020, 039) | Roadmap Stage 1, `world.go`, `market.go` row (CLR 15 homes it in `market.go`). This plan gives it `Window`, `Position` and the run function |
| The outcome loop at P5.4 (the draw per outcome, the order of regions and commodities, the call of `Revise`, the clamp of a returned belief, the `BeliefsUpdated` emission, the `Debug` and `Warn` lines, with their `Enabled` guards) | Roadmap Stage 1, `world.go` row. This plan gives it `Gate`, `Change`, `Tally`, the reason mapping and the run function (the world builds one `Change` per outcome from the stored belief, the belief `Revise` returned after the clamp, the bounds, the gate and the clamp flag, because it never sees an `Update` result: A13); the filled units per order and the trades come from [plan-phase-1-3.md](plan-phase-1-3.md) (`book.filled()` and the trades `clearMarket` appends, its O5) |
| `UnitCost` (the cost anchor with labour at the observed wage) and the effective reference price per region | Roadmap Stage 1, `internal/production` row (PRO 9) and the reference-price read of plan-phase-1-2.md A10; this row takes both as plain integers |
| The scenarios S01, S04, S07, S09 and S21 and their outcomes (STB 8), the residual offset of N-042 | Roadmap Stage 1 "Done when"; they need a world |
| The encoding of the belief and the window in a checkpoint (BEL 13; FR-BEL-035) | Roadmap Stage 2, `checkpoint.go` row. This plan gives `Window` a lossless export and restore |
| The snapshot fields (`MarketRow` beliefs, mean, low, high, at floor and at ceiling) and the INT query of a belief | Roadmap Stage 1, `snapshot.go`; Stage 2 `integration.go` |
| The metrics `economysim.belief.centre` and `economysim.belief.at_bound` | `otelobserver` (OBS 6.2.1, 6.3.6); fed by `BeliefsUpdated`, which this row only maps |
| A unit-count imbalance signal (BEL open question 6), per-commodity parameters (question 2), the agent's own fill price (question 8) | Not built: not in version 1 (BEL 18) |
| CI | Open item O4 |

## Decisions this plan rests on

Decided by documents already:

- **D40.** The split of homes is BEL 20 and architecture.md 2.9: the pure
  rules over primitive structs are in `internal/beliefs`; the root holds
  `Belief` and `Bounds` as exported values and converts at the call, inline
  (2.9: no converting package, file or type). The belief of an agent and the
  window of a market live in `agent.go` and `market.go` (BEL 20, CLR 15, 16).
- **D41.** The rule is a set of functions, not an interface (BEL 10.4,
  architecture.md 4.2): the update is the strategy's `Revise`, and the default
  `Revise` calls `beliefs.Update` (AGT 7.6, FR-AGT-050).
- **D42.** `internal/beliefs` imports no random source. The caller passes one
  `uint64` to `Update` and one to `Draw`, and each consumes exactly that word
  (BEL 4.5, 5.2, 6; FR-BEL-008, 019). The world takes the outcome's word as the
  first word of the stream `(siteOutcome, tick, agent, commodity)` (WLD 7.3,
  FR-WLD-049, ADR 0008), so the rule needs no site of its own.
- **D43.** No configuration field, default, bound or validation rule is added
  (the table above). The parameter-registry test and its `notBuilt` list do not
  change (architecture.md 9.7), and the pinned hash of the all-off reference
  configuration and `testdata/golden/off/` do not move (D5 of plan 0; N-217).
- **D44.** The `reason` values of the `Debug` line are exported `Reason`
  constants of the root, in `event.go` (BEL 12.3; OBS 4.2.3), declared with
  their first user, the mapping of the result in `beliefs.go`.
- **D45.** The rule's arithmetic is integer, in basis points, with checked
  products (BEL 2.2, FR-BEL-006, architecture.md 3.5); an overflowing ratio is
  the larger side of every comparison it feeds (A9), never a wrap and never a
  panic.
- **D46.** The window is a per-market ring of the market prices of the last
  `PriceWindow` ticks in which units traded, outside what `Clearing` sees
  (BEL 7.1, CLR 5.4). Its statistics are the mean rounded half to even, the
  low and the high (7.2).
- **D47.** `Clamp` is exported by the package and applied by the world to
  every belief a strategy returns (BEL 10.3, FR-BEL-026), and to a stored
  centre when it is read (3.3).
- **D48.** Ordering ties and iteration: nothing in the package sorts or ranges
  over a map; the window is a ring in tick order (architecture.md 7.5, 7.6).
- **D49.** One PR per phase, in phase order (the pattern of plan-phase-1-1.md
  A9).

## Assumptions this plan makes (flagged, not decided)

Sequencing and implementer-level calls. Challenge any of them in review.

- **A1.** The sketch of BEL 20 is the shape, with these changes: `Params`
  gains `WidthInit` (`Reseed` needs it; BEL 8.2); `Bounds` carries a
  `Conflict` flag (BEL 3.2.3); the package gains `Clamp` (D47), `Gate` (the
  adjustment gate of BEL 5.2 as a function of the draw and `AdjustProb`, the
  one definition `Update` and the world both use) and `Change` (what the
  world has for one outcome, A13); `Result` shrinks to what `Update` alone
  knows (`Open`, the gate), because the world never sees an `Update` result
  (A13); `Draw` takes `Params` for the width limits it must re-clamp with
  (BEL 3.3: "re-clamped whenever it is read for a draw"). The types of 4.2 and
  4.3 are as the sketch. The names are the sketch's; the exact signatures are
  the implementer's within BEL 4 and 20 and architecture.md 2.9.
  `PriceWindow` is the capacity of the `Window` and is not a field of
  `Params` (the sketch has none). Phase 1 updates the sketch.
- **A2.** `Belief` and `Bounds` of the root are declared in `beliefs.go`
  (architecture.md 2.6 maps the file to BEL; AGT 6.3 says the type is "exported
  with its two fields"). They are typed with the named value types of
  architecture.md 3.3, which is a MUST for a domain quantity and makes mixing
  units a compile error: `Belief{Centre Price; Width int64}` (`Width` is a
  half-width in basis points of the centre, which has no named type, BEL 3.1)
  and `Bounds{Floor, Ceiling Price}`. The internal structs are plain `int64`
  (architecture.md 2.9), and the conversion at each call is a plain cast
  (`Price(x)` and `int64(p)`), inline and never a function of its own. They
  are not aliases of the internal types, so the root's API names no
  `internal/` type. The exported field types are a compatibility promise
  (architecture.md 2.8, 6.6), which is why they are fixed here and not left to
  the implementer; the built `BeliefsUpdated` already types `MeanCentre`,
  `LowCentre` and `HighCentre` as `Price`, and plan-phase-1-3.md types `Order`
  and `Trade` with `Price`. The sketch of BEL 3.1 and 20 shows `int64` because
  it describes the internal package, where that is right (Phase 1 says so in
  the spec).
- **A3.** The per-market `Tally` of `BeliefsUpdated` is in `internal/beliefs`
  (the half-to-even mean, the low and high, the counts), over primitives; the
  root maps it to the event in `beliefs.go`. This keeps the arithmetic in the
  package that has the 90% coverage rule and its unit tests, and leaves the
  world row one call per outcome and one per market.
- **A4.** `Window` is a value with a slice allocated once at construction (the
  capacity is `PriceWindow`), so a push allocates nothing. It caches its mean,
  low and high at each push and restore, so the view of every agent reads them
  in constant time (NFR-BEL-003). The ring's write position is not part of its
  meaning: two windows with the same logical content, oldest first, behave
  identically (BEL 13: the position is derived).
- **A5.** The `Debug` reason of one update has one value, with this
  precedence: `gate_closed` (nothing was evaluated), then `at_floor`, then
  `at_ceiling` (the centre ended on that bound, whether or not it moved), then
  `moved` (`After.Centre != Before.Centre`), then `held` (an open gate and an
  unchanged centre). It is a function of the `Change` of A13, not of
  `Update`'s result. BEL 12.2 lists the five and gives no precedence. Phase 1
  writes it into BEL 12.2.
- **A6.** When a belief has `Floor` equal to `Ceiling` it is at both bounds:
  `AtFloor` and `AtCeiling` both count it, so their sum can exceed
  `Outcomes`. The rule is about `Floor == Ceiling`, not about the conflict of
  BEL 3.2.3: the cost floor can win the conflict, and the reference band's
  lower end can also meet the cost ceiling with no conflict (reference price
  1 000, unit cost 10: `refLow` 100, `costHigh` 30, so `Floor` and `Ceiling`
  are both 100 and `Conflict` is false). For the `Warn` the bound named
  is the one with more beliefs, and `floor` on a tie. The side effect is kept
  on purpose: the literal test `AtFloor + AtCeiling > Outcomes / 2` is the
  spec's and the counterpart rule (OBS 7.3 row 3 reads the two event fields)
  needs the sum, so one conflicted belief counts twice and fewer than half of
  the outcomes can fire the `Warn`. Phase 1 writes the rule and its side
  effect into BEL 12.1 and 12.2, and O2 records it.
- **A7.** The diagnostic run is advanced once per market per tick, also in a
  tick in which the market had no outcome (that tick does not hold the
  condition and resets the run: OBS 7.3, "a tick in which the condition does
  not hold resets the run"). The function takes the run length `warnRun` as a
  parameter (architecture.md 2.9, 9.6; OBS 7.3).
- **A8.** Tests of the package are external (`beliefs_test`) from tables with
  literals whose arithmetic is in a comment (testing.md 5.5); `beliefs.go` is
  tested from an internal test of the root, as the options were in Stage 0
  (A7 of plan 0) and `ledger.go` is (A8 of plan-phase-1-1.md), until
  `NewWorld` exists.
- **A9.** An input that is outside the configuration bounds (a `Stock` above
  `MaxQuantity`; AGT 6.5.2 treats an intent's `Target` and `Reserve` above
  `MaxQuantity` as 0, and says nothing of a `Stock`, so the package must
  hold on its own) never wraps and never panics in the package: a ratio or
  product that does not fit `int64` is read as larger than every bound it is
  compared with (a stock ratio above `BeliefStockHigh`, a gap above every
  step). The degenerate values are defined here, because the spec's sections
  do not define them and the fuzz test's assertions need them to be:
  a `Quantity` below 1 gives a fill share of 0; a `Filled` below 0 is 0 and a
  `Filled` above `Quantity` is `Quantity` (a fill share of at most 10 000); a
  `Limit` below 1 is read as 1; a negative `Stock` or `Target` is read as 0
  (`Target` 0 is neutral, FR-BEL-014). The inputs of `Limits` are read the
  same way: a `ref` below 1 is read as 1 (a reference price is at least 1 by
  validation, so this is defence only) and a negative `unitCost` is read as 0,
  unknown (BEL 4.4: 0 means unknown, so no cost band applies). A price given
  to `Window.Push` outside `[1, MaxPrice]` is stored as the nearest bound and
  occupies its slot of the ring like any price ("counted" means counted as an
  entry of the ring, so the number of entries stays one per tick in which
  units traded; it is not a separate diagnostic counter). Phase 1 writes the
  first group into BEL 5, the `Limits` reading into BEL 3.2 and the `Push`
  reading into BEL 7 so the spec and the fuzz test agree. Bounds given to `Update`, `Draw`, `Clamp`
  and `Window` are valid (`1 <= Floor <= Ceiling <= MaxPrice`, `WidthMin <=
  WidthMax`, FR-BEL-002 is "for inputs within configuration bounds"): the
  fuzz test and the clamp table vary the centre, the width, the quantities
  and the flows to their extremes, not the bounds. No error result exists:
  BEL 12.2, "an update that cannot be applied does not exist".
- **A10.** The package name `beliefs` and its exported name `Belief` follow the
  sketch of BEL 3.1 and 20; architecture.md 2.5 forbids a package prefix
  (`market.MarketBook`) and `beliefs.Belief` is the package's subject, as
  `ring.Ring` is. If review prefers another name the rename is one PR before
  Phase 3 (O6).
- **A11.** `Adjusted` of `BeliefsUpdated` is the count of open gates, as BEL
  12.1 says ("`Adjusted` (gates open)"). The comment of the field in `event.go`
  says "beliefs that moved"; Phase 5 corrects the comment, and the counts of
  `Raised` and `Lowered` are the moved ones (O1). The comments of `Conflicts`
  ("adjustments that conflicted"), `Clamped` ("adjustments cut by a bound"),
  `AtFloor`, `AtCeiling`, `Raised` and `Lowered` are checked against BEL 12.1
  in the same edit (Phase 5): `Conflicts` counts updates for which the cost
  floor exceeded the band, `Clamped` the beliefs a strategy returned that the
  world had to clamp, `AtFloor` and `AtCeiling` the centres that ended on a
  bound. The comment of the `Reason` type, which lists the rejection, abandon
  and ledger reasons, gains the log-only belief reasons.
- **A12.** The cross-check against the Python model of the rule
  (`doc/specs/models/pb/b_bel.py`) is non-gating and is run in Docker only
  (O5). The hand-computed cases of BEL 15 and the properties of BEL 5.8 are
  the gating tests.
- **A13.** What the tally and the `Debug` reason are built from. The world
  never sees an `Update` result: at P5.4 it calls `Strategy.Revise`, which
  returns only a `Belief`, and it applies that belief inside its bounds
  (WLD P5.4; AGT 6.2, 6.5.3). A consumer's strategy that ignores the default
  rule returns a belief and no `Result`, yet `BeliefsUpdated` and the "belief
  updated" line are emitted for every outcome. So both are built from what the
  world itself holds, one `beliefs.Change` per outcome: `Before` (the stored
  belief before the outcome), `After` (the belief after `Revise` and the clamp
  of 3.3 and 10.3), `Bounds` (from `Limits`, with `Conflict`), `Open` (the
  gate: `beliefs.Gate(u, AdjustProb)` of the outcome's `Draw`, which the world
  holds, so it is the default rule's gate whatever the strategy did with the
  word) and `Clamped` (the flag `Clamp` returned for the strategy's belief).
  Everything else is derived from those, so a custom strategy works:
  - `Raised` when `After.Centre > Before.Centre` and `Lowered` when below,
    the literal reading of BEL 12.1 ("centres that increased or decreased"):
    the stored centre against the centre that ends the outcome. A re-clamp
    of a stored centre to bounds that moved (unit cost rose) under a closed
    gate therefore counts as `Raised` or `Lowered` (the centre did change)
    and its reason is `gate_closed` (A5). Phase 1 writes this into BEL 12.1,
    so the spec and the code agree;
  - `AtFloor` when `After.Centre == Bounds.Floor`, `AtCeiling` when
    `After.Centre == Bounds.Ceiling` (A6), `Conflicts` from `Bounds.Conflict`,
    `Adjusted` from `Open`, `Clamped` from the flag;
  - the reason of A5 from the same `Change`.
  `Update`'s own `Result` is `Update`'s return for the default strategy and
  its tests and carries only what `Update` alone knows (`Open`); the tally
  never reads it.

## Phase 1: settle the owed spec lines

**Why first:** three lines of BEL disagree with the specs around them and with
the built code, and the package's shape is written in a sketch that Phase 2
turns into code. This phase changes specs, notes and the glossary comments of
`doc.go`, so it starts by reading
`doc/notes.md` in full and quoting its tag, it handles the open notes tagged
BEL and `ALL` (the table below), and `doc/notes.md` gets a fresh tag
(`AGENTS.md`).

Deliverables, each in the document that owns it:

1. **The draw under ADR 0008** (N-214, the BEL row). BEL 4.5, 5.1.1, 5.2 and
   FR-BEL-008 say the draws are taken "in order of region, commodity and
   admission". WLD 7.3 and FR-WLD-049 key the outcome's word by
   `(siteOutcome, tick, agent, commodity)`, so the word of an outcome does not
   depend on the order or the count of the others. State that, and that the
   order of the outcomes stays the order of the events and of `Revise` calls
   (CLR 7.3). Keep the IDs (testing.md 4.1).
2. **BEL 12.1 and 12.2** (A5, A6, A7, A13): that every count and the reason
   are functions of what the world holds for an outcome (the stored belief
   before, the belief after `Revise` and the clamp, the bounds with their
   conflict flag, the gate of the outcome's draw, and whether the clamp
   acted), so they hold for a strategy that ignores the default rule and
   returns no `Update` result (the world sees only `Revise`'s `Belief`: WLD
   P5.4, AGT 6.5.3); what `Raised` and `Lowered` mean (the centre after
   against the stored centre before, so a re-clamp under a closed gate counts
   and has the reason `gate_closed`: A13); which `reason` the `Debug` line
   gives (the precedence); the case `Floor` equals `Ceiling`, stated as that
   equality and not as the conflict of 3.2.3, with its side effect on the
   `Warn` (A6); the bound named by the `Warn` and its tie; that the run is
   advanced every tick for every market.
   BEL 12.1 already says `Adjusted` (gates open); only the comment in
   `event.go` is wrong (A11, Phase 5), so no edit of that word is owed.
3. **BEL 20 and its sketch** (A1, A2, A3, A4): the home of `Belief` and
   `Bounds` of the root (`beliefs.go`), of the belief storage (`agent.go`), of
   the window and the run (`market.go`); the added fields and functions; the
   last row ("checked multiply and divide, half-to-even mean") says
   `internal/checked`, not `spec-currency.md` section 7. It also says that
   the sketch is of the internal package (`int64` fields, architecture.md
   2.9) and that the exported root types carry `Price` for `Centre`, `Floor`
   and `Ceiling` (`Width` stays `int64`, basis points), converted by a plain
   cast at each call (architecture.md 3.3, 2.9; A2). The same sentence goes
   into BEL 3.1 (where the struct is first shown) and into the note of AGT
   6.3 ("`Belief` is the value of `spec-beliefs.md` 3.1, exported with its
   two fields"): the fields are typed `Price` and `int64`. Both are one
   clause each, so no ID moves. BEL 20 also says
   that `PriceWindow` is the window's capacity and not a field of `Params`
   (A1), that `Gate` and `Change` exist and that `Result` holds only what
   `Update` alone knows (A1, A13), and that `Draw` takes `Params` (A1).
4. **BEL 8.3** (the one helper): the case of an initial agent and of an agent
   added before the first step is the same call as a later entry, with the
   seed price as the market price and no window mean (FR-BEL-039: the window
   of a market that never traded is empty and its price is the seed price), so
   one function serves every case. **BEL 5** (A9): the reading of a `Quantity`
   below 1, a `Filled` outside `[0, Quantity]`, a `Limit` below 1 and a
   negative `Stock` or `Target`, which no section defines today. **BEL 3.2**
   (A9): a `ref` below 1 is read as 1 and a negative unit cost as 0
   (unknown). **BEL 7** (A9): a price pushed outside `[1, MaxPrice]` is
   stored as the nearest bound and takes its slot in the ring.
5. **The roadmap.** The link from Stage 1 to this plan is already there
   (`roadmap-v1.md`, the row 4 paragraph); only the status is left: set the
   row to **in progress** when Phase 2 starts; do not change another row's status.
6. **Notes** (the table below). Update N-214 (the BEL row applied) and drop
   BEL from its areas. A new note (**N-230**, reserved for this plan by the
   cross-plan review) tagged BEL and WLD: the run of the `Warn`
   is advanced for every market every tick, and what a market with no outcome
   does to it (A7); and a note (**N-231**, reserved) tagged BEL and CODE if the
   exported-name review of O6 asks for it. Any further gap found is a note in the same change.
7. **`doc/documentation_index.md`**: the row of this plan; correct it if the
   phases change.
8. **The glossary in `doc.go`** (architecture.md 3.2: a new domain term
   **MUST** be in the glossary before use). Add price belief, centre, width,
   effective range, belief floor and belief ceiling (bounds of a belief),
   reference band, cost band, adjustment gate, unit of adjustment, evidence
   move, stock ratio, fresh price, price window, window mean, seed price and
   target price, fill share, limit price, re-clamp, window low, window high
   and position in the window (the list of BEL, in bold on first use; the
   later words are the ones the code and the doc comments of Phases 2 to 5
   use), each with its definition, skipping any term already present when
   `doc.go` is re-read (limit price is also in plan-phase-1-3.md's list, with
   market price: `TestGlossary_Vocabulary` fails on a term that appears twice,
   so the second change to land does not add it), in one change that is comments only (no
   identifier, import or package clause of `doc.go` changes). The glossary
   guard of Stage 0 (`TestGlossary_DeclaredTypesNamed`) matches each exported
   type of a listed file against `doc.go` as a case-sensitive whole word, and
   `doc.go` holds neither `Belief` nor `Bounds` today (only
   `BeliefsUpdated`). So the definitions of **price belief** and **belief
   floor and belief ceiling** MUST name the identifiers exactly as the type
   names: `Belief` (fields `Centre` and `Width`) and `Bounds` (fields `Floor`
   and `Ceiling`), so that Phase 5 adds `beliefs.go` to the guard's list
   with nothing left to write. `doc.go` is edited by the other Stage
   1 rows too (the ledger row's terms, N-017): re-read it immediately before
   the edit, use targeted `Edit` calls that add entries and touch no other
   line, and keep the order the file already has. The glossary guard of
   Stage 0 runs on the result (it names each exported type of `value.go`,
   `config.go` and `event.go`; the file list that gains `beliefs.go` is
   Phase 5's, when the exported `Belief` and `Bounds` exist).

### Disposition of the open notes

Every open note tagged BEL or `ALL` in the index of `doc/notes.md` (read in
full on 2026-10-13, tag `6cc73d3abf2fd311fb034a21` when this plan was
written and `5bb1912a893964e99cd253bf` at the audit fix, so it is stale by
construction; N-235 is in the index
since the cross-plan review), each applied, rejected or left open. The tag
moves with every edit of the file, so Phase 1 re-reads `doc/notes.md` in full
when it runs, re-quotes the tag it finds, and adds a row here for any BEL or
`ALL` note that has appeared since. Notes tagged CODE are read for what the
first change that touches the area must do and are in the last paragraph.

| Note | Tag | Disposition |
|------|-----|-------------|
| N-042 The evidence rule may offset the price | BEL, STB | **Left open, with what this row learned.** This row implements the revised rule of 5.4 as written (the limit-based, guarded, shared evidence move; FR-BEL-036, 037) and tests the guard case (BEL 15, case D). Whether the residual offset of 5.7% remains in the library is S01 with FR-STB-018, which needs a world; the model (`models/pb/`) is the only evidence until then |
| N-043 Nothing anchors the price level if flows ignore price | BEL, STB (also AGT, PRO, SPN) | **Left open.** The rule has no anchor of its own by design (5.5: the stock rule fixes the level only where flows respond to price); the flows are the agent, demand and spawner rows' and the scenarios S04 and S16 decide |
| N-214 Keyed random streams (ADR 0008): the edits owed | BEL (and WLD, CHK, EXP, CLB, STB, TEC, PLC, AGT, CLR, EXT, INF, MER, MIG, RSK, WEL, CODE) | **Applied for BEL** (deliverable 1: the draw is keyed by agent and commodity, the text no longer implies an order-dependent stream). The note stays open for the other areas, and BEL leaves its areas |
| N-016 Keep the parameter registry true | ALL | **Applied.** This row adds no field, default or bound (D43); "no new row" is stated in the PR |
| N-017 Words and names | ALL | **Applied.** The terms go to `doc.go` in Phase 1, before the code of Phase 2 uses them; "floor" and "ceiling" are qualified as the bounds of a belief (the note counts seven uses of "floor") |
| N-027 No default macro constant is stable yet | ALL | **Applied (standing).** Every default of BEL 11 stays a placeholder (the preamble of BEL); the row sets none and claims no stability |
| N-103 Running several spec writers in parallel | ALL | **Applied in its lesson.** Shared files (`doc.go`, `notes.md`, the roadmap, the index) are edited with targeted edits after a re-read, and the notes file is read in chunks |
| N-126 A new event or `Warn` adds a row | ALL | **Applied: nothing to add.** `BeliefsUpdated` is a row since Stage 0 and the `Warn` "price beliefs at bound" is row 3 of OBS 7.3; the `Debug` "belief updated" is in the OBS 7.5 list. No event or `Warn` is added |
| N-128 A hosted per-agent field is saved once | ALL, CHK | **Applied: nothing to add.** The centre and width are saved once, in their own Beliefs section (tag 5, per agent and commodity of its tradable set) and the window and `BoundRun` in the Markets section (tag 4), not in the agent record (BEL 13; CHK 4.2); Phase 1 reads CHK 4.2 and confirms both rows are there |
| N-163 The word "treasury" | ALL | **Applied: not used.** No file of the row mentions it |
| N-235 One combined architecture.md edit for the fan-in and admission touch points | ALL | **Left open, not this row's.** It edits `architecture.md` 2.10.4 (a fifth fan-in for tradable items, N-228), the admission-table touch point and the example of 2.9 in one change, with a fresh RULE-TAG, before the first Stage 2 module PR. This row adds no fan-in, no admission and no touch point (`beliefs.go` is a core file, not a module file), so nothing here waits for it or changes it |
| N-217 D5 binds every module's change | ALL | **Applied.** The row appends no event field, no snapshot field and no configuration field; the pinned hash and `testdata/golden/off/` do not move (the gate checks the hash test) |

Read, not tagged BEL: N-039 (a hard cost floor; STB, GDS, PRO, SKL) says
"BEL decides its open question 1": it is answered by BEL 3.2.2 (`CostFloor`
7 000) and what is left is S07, so nothing for this row. N-055, N-040, N-044
and N-006 are closed and are what the spec already applies. N-019 and N-224
(CODE) are read for the guards: the import, determinism and glossary guards
run over the new package and files unchanged (the glossary guard's file list
gains `beliefs.go`, Phase 5), and `TestConstants_Values` gains the five
reasons with `spec-beliefs.md` as their owner (Phase 5; its owner mapping is
by position in the table, so the new rows are a separate slice).

**Tests:** none new; documents and the comments of `doc.go` only. `gofmt -l .`
and the existing tests (the glossary guard above all) run unchanged.

**Observability and rollback:** none, because no behaviour changes. A wrong
edit is reverted like any document change; nothing depends on it until
Phase 2.

**Acceptance:** BEL 4.5, 5.1.1, FR-BEL-008, 12.1, 12.2, 20 and 8.3 say what
deliverables 1 to 4 say; N-214 records the BEL part applied and its index row
no longer lists BEL; the notes carry a fresh tag; the roadmap links this plan;
every term of deliverable 8 is in the `doc.go` glossary, the words `Belief`
and `Bounds` appear in it as whole words (a text search for `\bBelief\b` and
`\bBounds\b` finds them, which is what Phase 5's guard needs), and the glossary
guard passes.

## Phase 2: the package, the bounds, the clamp and the draw

**Why here:** every other function of the rule reads a belief and its bounds,
and the invariant of BEL 3.3 is defined over them. The draw is one function
over the same two values. It needs nothing of the later phases.

- `internal/beliefs` with its package doc: stdlib and `internal/checked`
  only, no root type, no random source, no logger, no map, no float (BEL 2).
- `Belief`, `Bounds`, `Params` (A1) and the integer helpers it needs: the
  basis-point product `v * bp / 10 000` through `checked.MulDiv`, truncated
  toward zero, the rounding stated in each doc comment (architecture.md 3.5).
- `Limits(ref, unitCost, p)` (BEL 3.2): the reference band, the cost band,
  `Floor = min(MaxPrice, max(refLow, costLow))` and `Ceiling =
  max(Floor, min(refHigh, costHigh))`; a unit cost of 0 applies no cost floor
  and no cost ceiling (`MaxPrice`); the `Conflict` flag when the cost floor is
  above the upper end of the reference band (3.2.3). A `UnitCost * CostFloor`
  or `UnitCost * CostCeiling` that overflows `int64` is read as `MaxPrice`
  (`costLow` and `costHigh` both, so the floor is `MaxPrice`, A9).
- `Clamp(b, bd, p)` (3.3, 10.3): the centre into `[Floor, Ceiling]` and the
  width into `[WidthMin, WidthMax]`, and whether anything changed (the world
  counts a changed belief a strategy returned in `Clamped`).
- `Draw(b, bd, p, u)` (BEL 6): it takes `Params` for the width limits, because
  BEL 3.3 re-clamps a belief "whenever it is read for a draw" and the clamp
  of the width needs `WidthMin` and `WidthMax`; without them a negative or
  huge `Width` would give `n <= 0` or an out-of-range limit (`uint64(n)`
  wraps). It uses the effective range (3.3) of the re-clamped belief,
  `Low = max(Floor, Centre - half)`, `High = min(Ceiling, Centre + half)`,
  and `limit = Low + high word of u * n` with `n = High - Low + 1`
  (`math/bits.Mul64`, one multiplication, no loop, no rejection). `n >= 1`
  by the invariant of 3.3 once the re-clamp is applied.

**Tests:** unit, external package, table-driven.

- `Limits`: the case of BEL 15 (ref 1 000, no cost: floor 100, ceiling 10 000;
  unit cost 1 600: floor 1 120, ceiling 4 800, the arithmetic in a comment); a
  unit cost of 0 gives no cost bound (FR-BEL-005); the cost floor above the
  reference band's upper end wins and raises the ceiling to meet it, with
  `Conflict` true (BEL 3.2.3); `MaxPrice` caps the floor; `Ref` 1 gives
  `refLow` 1 (FR-BEL-004); a `ref` of 0 and of a negative number is read as 1
  and a negative unit cost as 0, unknown (A9).
- `Clamp`: a centre below and above the bounds, a width below and above its
  bounds, a belief already inside is unchanged and reported so; the result
  satisfies the invariant for every input of a table that includes the
  extremes of `int64` (FR-BEL-002, 026).
- `Draw`: the range of BEL 15 case A (centre 1 030, width 2 250: half 231, low
  799, high 1 261); `u` 0 gives `Low`, `u` of `MaxUint64` gives `High`; a
  range of one price (`Low` equals `High`) still consumes the word and returns
  that price; a stored belief outside its bounds, a negative `Width` and a
  `Width` above `WidthMax` (and above 9 999) are re-clamped by the call and
  give a price in `[Floor, Ceiling]` with the same word consumed (BEL 3.3;
  FR-BEL-002); the result is in `[Floor, Ceiling]` and `Low >= 1` for a
  property loop over a fixed seed range with the failing seed in the message
  (testing.md 6.2; FR-BEL-003, 019).
- `Guard:` by reflection or by text, the package has no field of type map,
  func or float and imports only `internal/checked` and the standard library
  (FR-BEL-017; the import guard of Stage 0 also checks it).
- Cost, ordinary tests with `testing.AllocsPerRun` (testing.md 8.2): `Limits`,
  `Clamp` and `Draw` allocate nothing (NFR-BEL-001). Phases 3 and 4 add the
  same test for every function on the per-outcome path of the tick loop
  (`Update`, `Gate`, the reads of `Change`, `Tally.Add`, the run function, the
  window reads and `Push`), so that observability-and-logging.md 8.2 is met
  for the whole rule and not for a part of it.

**Observability and rollback:** none, because the functions are pure, log
nothing, emit nothing and hold no state; nothing in the module calls them yet.
Rollback is reverting the PR.

**Acceptance:** `gofmt -l .` prints nothing; `go vet`, `staticcheck` and
`go test -race ./...` pass; the import and determinism guards of Stage 0 pass
over the new package; `internal/beliefs` is at 90% statement coverage or more
(testing.md 9.1); the allocation tests pass at zero.

## Phase 3: the update

**Why here:** it is the rule of the spec, and it is built on Phase 2's bounds
and clamp. It also introduces `Gate` and `Change`, which Phase 4's tally and
run read, so Phase 4 follows this phase (D49: one PR per phase, in order); the
reseed, the window and `Position` of Phase 4 do not read them.

- `Outcome`, `Quote` and `Result` (BEL 4.2, 4.3, 20); `Result` carries only
  `Open`, the gate, which is all `Update` alone knows (A1, A13).
- `Gate(u, adjustProb) bool` (BEL 5.2): `hi, _ = Mul64(u, 10 000)`, open when
  `hi < adjustProb`; the one definition, called by `Update` and by the world
  (which holds the outcome's `Draw` and the parameters and so needs no
  `Update` result).
- `Change` (A13): `Before`, `After`, `Bounds`, `Open`, `Clamped`, with the
  derived reads `Raised`, `Lowered`, `Moved`, `AtFloor`, `AtCeiling` and
  `Conflict` computed from those fields and nothing else. It is the input of
  the tally (Phase 4) and of the reason mapping (Phase 5), so it is a plain
  value the world builds from what it holds, and no field of it comes from
  `Update`'s result.
- `Update(b, bd, o, q, stock, target, p, u)` returning the new belief and the
  result (BEL 5):
  - the re-clamp of the stored belief first (3.3);
  - the gate: `Gate(u, AdjustProb)` (5.2); a closed gate leaves centre and
    width as re-clamped;
  - `Δ = max(1, Centre * Step / 10 000)` (5.3);
  - the target by side and fill (5.4): freshness `Age <= MaxAge`, the
    candidates, the significance test `limit - P > Centre * Significance /
    10 000` for a filled order;
  - the evidence move `sign(gC) * min(|gL|, |gC|, Δe)` with `Δe = max(1, Δ *
    EvidenceShare / 10 000)`, zero when there is no target, a gap is zero or
    the two gaps differ in sign (FR-BEL-036);
  - the stock component from `stock * 10 000 / target` (5.5; `target` 0 is
    neutral; an overflowing ratio is above the upper band: A9);
  - the move, the clamp, and the width step by the fill share (5.6, 5.7);
  - the result: `Open` (the gate). The tally and the reason are not built
    from it but from a `Change` the world assembles (A13), so a strategy
    other than the default is counted the same way.

**Tests:** unit, external package.

- The four worked cases of BEL 15, one case each, every row of the arithmetic
  in a comment: A (stale price, the unmatched ask pulls up: centre 1 030,
  width 2 250), B (an ask that sold below the market: centre 1 010, width
  1 750, the centre reaches the target and does not pass it), C (pinned at the
  cost floor 1 120, `AtFloor`), D (the guard: the gaps have opposite signs and
  the evidence move is 0). FR-BEL-009 to 016, 034, 036.
- One case per branch of 5.4 and 5.5 that breaks only that rule: each of the
  four target selections and "no target"; a stale price is not evidence
  (FR-BEL-012); a significance exactly at and one above the threshold
  (FR-BEL-013); the stock ratio exactly at `StockLow` and at `StockHigh` is
  neutral, one beyond moves; `target` 0 is neutral (FR-BEL-014); an unfilled
  bid and an unfilled ask in a stale market close the distance by `min(Δe, A -
  limit, A - centre)` and neither passes its target (FR-BEL-034).
- `Gate`, called directly and through `Update`: a draw whose high word is
  exactly `AdjustProb - 1` is open and `AdjustProb` is closed, computed from
  `u = ceil(k * 2^64 / 10 000)` for the edge, the arithmetic in a comment;
  `AdjustProb` 10 000 is always open and 0 never (FR-BEL-009). `Update` and
  `Gate` agree for every case of the table.
- `Change` (A13), one case per derived read: a centre above, equal to and
  below the stored one gives `Raised`, neither and `Lowered`; a re-clamped
  stored centre under a closed gate (`Before` below the new floor, `After` at
  it, `Open` false) is `Raised` and `AtFloor`; `Floor == Ceiling` with the
  centre on it is both `AtFloor` and `AtCeiling` and `Conflict` false (the
  reference 1 000, unit cost 10 case of A6); built from a custom strategy's
  belief (a centre no `Update` call produced) the reads are the same
  function of the fields. The reads of the `Update` cases above are checked
  through a `Change` built from the call's input and output (one derivation,
  not a second one in the tests).
- The properties of BEL 5.8, over a fixed seed range with the failing seed in
  the message: (2) the invariant after every update for any inputs
  (FR-BEL-002, 003); (3) `|new centre - old centre| <= Δe + Δ` before the
  clamp, observed through a belief far from both bounds (FR-BEL-015); (5)
  stationary with a closed gate, or with no target and a stock ratio in band;
  (6) scale-free: all prices and the bounds multiplied by `k` where no
  truncation occurs multiply the centre by `k` and leave the width
  (FR-BEL-018); a cohort: quantities, fill and stock multiplied by `w` give
  the same result (FR-BEL-025, rule half).
- Width: the fill share exactly 5 000 narrows, 4 999 widens, both clamp at
  `WidthMin` and `WidthMax` (FR-BEL-016).
- Overflow edges (A9): `stock` and `target` at `MaxQuantity`, a `Centre` at
  `MaxPrice`, `Δ` at its largest, never wrap and the invariant holds
  (FR-BEL-006). The degenerate inputs of A9, each a case: `Quantity` 0 and
  negative, `Filled` above `Quantity` and below 0, `Limit` below 1, a
  negative `Stock` and `Target`; the bounds passed are valid in every case.
- `FuzzUpdate` (not required by testing.md 7.4, which covers parsers and
  validators; it is here because the rule must hold the invariant for every
  input and a table cannot enumerate them): a byte string decoded into a
  belief, bounds, outcome, quote, stock, target and a draw, asserting no
  panic, the invariant, and the bounded step. The bounds are made valid
  (`1 <= Floor <= Ceiling <= MaxPrice`) before the call, because FR-BEL-002 is
  for inputs within configuration bounds (A9): only the centre, the width,
  the quantities and the flows are extreme. Seed corpus with `f.Add`; corpus
  directory `internal/beliefs/testdata/fuzz/`.
- Cost: `Update`, `Gate` and the reads of `Change` (`Raised`, `Lowered`,
  `Moved`, `AtFloor`, `AtCeiling`, `Conflict`) allocate nothing
  (`testing.AllocsPerRun`; NFR-BEL-001, 003). A benchmark with
  `b.ReportAllocs()` and `b.Loop()` (NFR-BEL-003; it reports, it gates
  nothing).

**Observability and rollback:** none emitted: the function is pure and the
world's `BeliefsUpdated`, `Debug` and `Warn` lines are built from its result
(Phases 4 and 5 give the mapping; the lines are the world row's). Rollback is
reverting the PR.

**Acceptance:** the commands of Phase 2 pass; coverage holds at 90%;
`FuzzUpdate` runs with its seed corpus and for five minutes by hand with no
crasher (a crasher is committed with a `Fix:` comment, testing.md 7.5).

## Phase 4: reseed, the window, the run and the tally

**Why here:** each is a small structure that the world and agent rows build
on. `Reseed`, `Window` and `Position` read only Phase 2's types; the run and
the tally read Phase 3's `Change` and `Gate`, so this phase follows Phase 3.

- `Reseed` (BEL 8.2, 8.3): for each commodity of a tradable set, given its
  `Ref`, `UnitCost`, the window mean and whether the window is empty, the
  market price, and, for a role change, the belief kept when the commodity is
  in both sets; returns the beliefs (in the order of the input, which is
  canonical commodity order). The centre is the clamp of the window mean, or
  of the market price while the window is empty; the width is `WidthInit`; a
  kept belief keeps its width and is re-clamped to the new bounds. No draw.
  The seed price of an initial agent is passed as the market price with an
  empty window (FR-BEL-039).
- `Window` (BEL 7): constructed once with its capacity; `Push(price)`
  overwrites the oldest when full and has no error result, so a price outside
  `[1, MaxPrice]` is read as the nearest bound and takes its slot of the ring
  (it is never ignored, so the count of entries stays one per traded tick and
  nothing wraps;
  A9; only `Restore` refuses); `Len`; `Mean`, `Low`, `High` with an
  `ok` result, the mean rounded half to even through
  `checked.MulDivHalfEven`; the values oldest first into a caller slice; a
  restore from such a slice that rejects more values than the capacity, a
  price below 1 and a price above the `MaxPrice` the window is given at
  construction; the statistics cached at each push and restore (A4). The running sum goes
  through `checked.Add` (at most 1 000 prices each at most `MaxPrice`, so it
  cannot overflow for a valid window; a sum that does overflow, from a
  restore that bypassed the check, makes the restore refuse and never wraps),
  and the mean through `checked.MulDivHalfEven` by the count.
- `Position(price, low, high)` (BEL 7.3): `(price - low) * 10 000 / (high -
  low)` clamped to `[0, 10 000]`, 5 000 when `high` equals `low`.
- The diagnostic run (BEL 12.2, A6, A7): from a tick's `Outcomes`, `AtFloor`
  and `AtCeiling` and the run before, the run after, whether to warn, and
  which bound; the run length passed in.
- `Tally` (BEL 12.1, A3): `Add(Change)` accumulates, per market per tick, the
  outcomes, the gates open, the raised and lowered centres, the beliefs at
  floor and at ceiling, the conflicts, the clamps a strategy's return caused,
  and the mean (half to even), low and high of the centres after the outcome;
  it returns its counts as plain values. Every count is read from the
  `Change` of A13 (the stored centre before against the centre after, the
  bounds, the gate and the clamp flag), which the world builds from the
  stored belief, the belief `Revise` returned after the clamp, `Limits`,
  `Gate` and `Clamp`. It reads no `Update` result, so a strategy that ignores
  the default rule is counted the same way. The running sum of the centres
  goes through `checked.Add` and the mean through `checked.MulDivHalfEven`.

**Tests:** unit, external package.

- `Reseed`: an initial agent from the seed price with the arithmetic in a
  comment; a later entrant from a window mean that is clamped by the cost
  floor; the market price while the window is empty; a role change that keeps
  the commodity in both sets (centre and width kept, re-clamped), adds one
  (window rule) and drops one; no randomness is taken (the function has no
  `uint64` parameter) (FR-BEL-022, 023, 024 in part).
- `Window`: empty reports none; pushes up to the capacity and beyond keep the
  last `capacity` values, the mean at a tie rounded to even (a case with a
  half: values 1 and 2 give 2, values 2 and 3 give 2), the low and the high;
  the logical content equal after a restore from the exported values,
  whatever the write position was (`reflect.DeepEqual` of the exported
  values, and the next push behaves the same); a restore that is too long or
  holds a price below 1 or above `MaxPrice` is refused; a capacity of 1 and of 1 000
  (FR-BEL-020; FR-BEL-035 in part). Push allocates nothing and the memory
  does not grow across ten times the capacity of pushes (NFR-BEL-002).
- `Position`: the endpoints, the midpoint, a price outside the window
  clamped, `high` equal to `low` (FR-BEL-021).
- The run: not pinned resets; pinned for 9 ticks does not warn and the tenth
  does and every tenth after; a tick with no outcomes resets; exactly half
  is not pinned and one more is (`Outcomes` 4, sum 2 and 3); the bound is the
  one with more and `floor` on a tie; a belief at both bounds (A6)
  (FR-BEL-031 in part).
- `Window.Push` with a price of 0 and of `MaxPrice + 1` stores 1 and
  `MaxPrice`, each in its own slot of the ring (`Len` goes up by one: A9).
- `Tally`: a scripted set of `Change`s with every count and the mean, low and
  high computed by hand in a comment; the mean at a half rounds to even; a
  market with no outcomes gives an empty tally the caller does not emit; a
  clamped strategy result counts in `Clamped`; a `Change` whose `After` no
  `Update` produced (a custom strategy: a centre moved by an amount the rule
  would never take) is counted as raised or lowered all the same; a
  closed-gate re-clamp counts in `Raised` and not in `Adjusted`
  (FR-BEL-030 in part).
- Reads, `Push`, `Tally.Add`, the tally's reads and the run function allocate
  nothing (`testing.AllocsPerRun`; NFR-BEL-001).

**Observability and rollback:** none emitted, as Phase 3. The run is
diagnostic and affects no result (BEL 13). Rollback is reverting the PR.

**Acceptance:** the commands of Phase 2 pass; coverage holds at 90%.

## Phase 5: `beliefs.go`, the reasons and the glossary guard

**Why here:** it is the root's side of the package and it declares the
exported types the agent row's views and `Revise` name. It needs Phases 2 to
4's types.

- `beliefs.go`:
  - the exported `Belief` (`Centre Price`, `Width int64` in basis points) and
    `Bounds` (`Floor`, `Ceiling Price`), with doc comments that begin with
    their names and state the invariant of BEL 3.3 (A2: the named value
    types of architecture.md 3.3, not `int64`);
  - `beliefParams(World) beliefs.Params`: the fifteen parameters (the
    sixteen belief fields of `World` less `PriceWindow`, which is the
    window's capacity, A1) and `MaxPrice`, built once at world construction and held by the world (as
    `ledger.go` builds its ledger once); it is the one place the root's
    names meet the package's;
  - `beliefReason(beliefs.Change) Reason`, the precedence of A5 over a
    `Change` (A13: never an `Update` result, which the world does not have), and the
    constants it returns;
  - `beliefsEvent(tick, region, commodity, tally)` building a
    `BeliefsUpdated` with the typed IDs and `Price`.
- `event.go`, appended: `ReasonGateClosed`, `ReasonMoved`, `ReasonHeld`,
  `ReasonAtFloor`, `ReasonAtCeiling` (D44), with doc comments; the comment of
  `BeliefsUpdated.Adjusted` is corrected to "gates open", and the comments
  of `Conflicts`, `Clamped`, `AtFloor`, `AtCeiling`, `Raised`, `Lowered` and
  of the `Reason` type are brought to BEL 12.1 (A11). Order and
  collisions (cross-plan check, 2026-10-13): three rows append to `event.go`,
  each at the end of the file's `Reason` constants as it merges, after what is
  there (plan-phase-1-1 Phase 7: `trade`, `estate_money`, `estate_goods`,
  `start_money`, `split`, `merge`, `births`, `starvation`; plan-phase-1-5
  Phase 5: `not_on_role`, `zero_yield`, `no_site`, `no_tool`, `no_input`). The
  Go names and the string values of the three sets and of the Stage 0 constants
  are pairwise distinct; re-read `event.go` and `TestConstants_Values` before
  the append and keep the new rows a slice of their own.
- `glossary_test.go` (N-224): the glossary guard's file list gains
  `beliefs.go`, so that the exported `Belief` and `Bounds` must be named in
  `doc.go` (the guard matches `\bBelief\b` and `\bBounds\b` as whole words,
  case-sensitive). The terms themselves are already in `doc.go` since Phase 1
  (deliverable 8 requires the definitions of price belief and of belief floor
  and belief ceiling to name `Belief` and `Bounds` exactly), so this phase
  adds none and its doc comments use only those; if the guard fails on
  `Belief` or `Bounds`, Phase 1 was incomplete and the words are added to its
  definitions, not worked around here. The file list of the guard and the
  `notYetTerms` list are edited by other rows too (plan-phase-1-3.md edits
  `notYetTerms`): re-read `glossary_test.go` immediately before the edit and
  use a targeted `Edit` that adds `beliefs.go` and touches no other line, as
  `event.go` below is handled.

The conversion between the root's types and the package's is inline where a
call is made (architecture.md 2.9). `beliefs.go` holds no call of the rule: the
callers are `strategy.go` and the world, which are later rows. It is not a
mapper layer (architecture.md 3.1, 2.9): `beliefParams` is a constructor of a
value the world holds, `beliefReason` is the classification of a `Change` into
a closed set (a rule, BEL 12.2) and `beliefsEvent` builds the domain event from
the tally; none converts a type for its own sake. Until the callers exist the
unexported functions are used by the tests of this phase only, which
`staticcheck` accepts (it reads test files); the row's gate runs it to check.

**Tests:** unit, internal test of the root with a small configuration literal
(A8), and one external test.

- `beliefParams` of `DefaultWorld()` equals the fifteen defaults of BEL 11
  that `Params` holds, and `MaxPrice`, field by field; changing each of those
  fifteen `World` fields changes exactly one field of the result (no two
  swapped: a table of one case per field), and changing `PriceWindow` changes
  none (it is the window's capacity, read by the world when it builds the
  `Window`).
- `beliefReason`, over a `Change` (A13): one case per value and the
  precedence of A5 (a moved belief that ends on the floor is `at_floor`; a
  closed gate on the floor is `gate_closed`; a closed gate with a re-clamped
  centre is `gate_closed`; an open gate and an unchanged centre inside the
  bounds is `held`). A `Change` built from a belief no `Update` produced
  gives a reason all the same.
- `beliefsEvent`: a tally of Phase 4's scripted set gives the event with its
  fields, `Tick` first, no map or pointer (the shape test of Stage 0 sees it:
  FR-OBS-001 is unchanged), the IDs and `Price` types (`MeanCentre`,
  `LowCentre`, `HighCentre`).
- The exported types (a `Guard:` test, by `reflect`): `Belief.Centre`,
  `Bounds.Floor` and `Bounds.Ceiling` are of type `Price` and `Belief.Width` is
  `int64`, because architecture.md 3.3 makes the named type a MUST and the
  field types of an exported struct are a compatibility promise (2.8, 6.6); a
  change to `int64` would compile at every internal caller and break only the
  consumers. The conversion is a plain cast: a table of `Belief` and `Bounds`
  built from an internal-package value and back at `1`, `MaxPrice` and the
  extremes of `int64` is equal field by field (so a named type cannot hide a
  scaling or a clamp), and the value `Price(int64)` and `int64(Price)` round
  trip exactly. The cast itself is inline at the call (architecture.md 2.9),
  so the table writes it out and calls no helper of production code.
- `TestConstants_Values` gains the five reasons with owner `spec-beliefs.md`
  (each a backticked token of BEL 12.2), and no two constants of the type
  share a value (FR-OBS-006). The existing test derives the owner by position
  and keeps its `seen` duplicate check inside one closure, and plans 1-1, 1-4
  and 1-5 each add a separate slice: each new slice MUST go through the same
  `check`/`seen` closure and merge its owner into the owner map, or a
  duplicate across two slices goes undetected. Re-read the test before the
  append and extend the closure, not a copy of it.
- The glossary guard passes with `beliefs.go` in its list (N-224).
- Two settings (architecture.md 9.4): `beliefParams` is built from the
  validated packs `one` and `second` and each passes the bounds of BEL 11
  (a loop over the two packs through `config/` in an external test, using no
  unexported symbol: the function is checked through a test-only
  `export_test.go`, so nothing is exported from production code).
- No test asserts a log line or an event emission: the world emits them.

**Observability and rollback:** the checklist of
observability-and-logging.md section 9 is answered in the PR (see the summary
below). No event type, key or metric is added; the five reasons are the
values of the `reason` key of one `Debug` line. Rollback is reverting the PR;
nothing outside `beliefs.go` calls it until the agent and world rows.

**Acceptance:** the commands of Phase 2 pass; the glossary guard, the
constants test and the event shape test pass; root statement coverage does not
fall below Stage 0's (testing.md 9.2).

## Phase 6: the exit gate and the traceability

**Why last:** it checks the row as a whole and closes the documents.

- The exit gate on the whole module: `gofmt -l .`, `go vet ./...`,
  `staticcheck ./...`, `go test -short -race ./...`, the full run with
  `-covermode=atomic -coverpkg=./...`, `go mod tidy`; the per-scope coverage
  read from `go tool cover -func=coverage.out` and recorded; `internal/beliefs`
  at or above 90%, the root at or above its Stage 0 figure.
- The traceability check of testing.md 4.6 by text search: every ID in a
  `Requirement:` line of the new tests exists in BEL, and every requirement
  the table below lists as covered in full is cited. The Stage 0 tests that
  already cover FR-BEL-028 and FR-BEL-038 (the `belief_width_*`,
  `belief_stock_*` and `cost_ceiling` order rules and the warning
  `cost_floor_at_full_cost` in `validate_rules_test.go`) and FR-BEL-027 and
  FR-BEL-037 (the bounds table) cite only FR-CFG-039 or FR-BEL-029: read each,
  and add the BEL ID to its `Requirement:` line where it covers it (a comment
  edit, no behaviour change).
- The proof of D5: the pinned hash test (`TestHashConfig_ReferencePinned`) is
  unchanged and `testdata/golden/off/` (once the world row records it) is
  unchanged.
- The cross-check of O5, if done: the vectors, with their provenance (the
  Docker command, the model's file and the parameters), committed under
  `internal/beliefs/testdata/` and read by one test.
- The roadmap row set to **done** with its commit; `documentation_index.md`
  corrected; N-042 and N-214 updated with what this row applied and what
  stays open.

Not gating, worth doing if time allows: a mutation pass over
`internal/beliefs` (testing.md 9.4, a SHOULD), with `gomu` inside Docker only,
`--cpus=4 --memory=16g --memory-swap=16g`, `GOMAXPROCS` and `--workers` equal
to the CPU cap.

**Tests:** the tests of Phases 2 to 5 run under the full gate. No integration
or end-to-end test: there is no world, so the half of testing.md 2.3 that
exercises a requirement through the public API at the integration level is the
world row's, and the table below says so.

**Observability and rollback:** none added; see the summary. Rollback is
reverting the PR.

**Acceptance:** every command of the gate is clean; the traceability check
finds no ID without a spec row and no "covered in full" requirement without a
test; the roadmap row reads done.

## Cross-plan contract

The coordinator uses these two tables to check consistency across the plans of
Stage 1. A "Consumes" row names the plan or phase that provides the symbol, or
says ASSUMED where no plan fixes it.

### Consumes

| What | Kind | Provided by | Status |
|------|------|-------------|--------|
| `World` belief fields (16), `DefaultWorld`, bounds (`MaxPrice`, `MaxBasisPoints`) and the validation of BEL 11 | config.go, defaults.go, bounds.go, validate.go | plan-phase-0-foundation.md (Stage 0, `023cf97`) | Built; read from the code on 2026-10-13 |
| `WarmStart.Prices` (`SeedPrice`, canonical order) | config.go | plan-phase-0-foundation.md | Built |
| `internal/checked.MulDiv`, `MulDivHalfEven`, `Mul`, `Add` | internal | plan-phase-0-foundation.md | Built |
| `BeliefsUpdated` event with the fields of BEL 12.1 | event.go | plan-phase-0-foundation.md | Built (comment of `Adjusted` corrected here, A11) |
| `Reason` type, `RegionID`, `CommodityID`, `Price` | event.go, value.go | plan-phase-0-foundation.md | Built |
| `warnRun` | observer.go | plan-phase-0-foundation.md | Built; passed in as a parameter |
| `internal/logkeys` keys `region`, `commodity`, `agent`, `price`, `reason`, `bound`, `agents` | internal | plan-phase-0-foundation.md | Built; used by the world row's lines, not by this row |
| `siteOutcome` (D8) and the stream key | random.go, internal/ordering | plan-phase-0-foundation.md | Built; used by the world row, not by this row |
| The effective reference price of a (region, commodity) as an `int64` per call | read from `cfg.Commodities[id].ReferencePrice` (the region override is Stage 2) | plan-phase-1-2.md A10 and its "Cross-plan contract" (read from the configuration by ID, no wrapper) | Provided; this row takes `ref` as a parameter, so it does not depend on how the caller reads it |
| `UnitCost` per (region, commodity) as an `int64`, 0 when unknown | production row | [plan-phase-1-5.md](plan-phase-1-5.md) Phase 7 (the anchor table and the root `UnitCost(region, commodity)`, computed once per tick at P1.2) | Provided in substance; this row takes it as a parameter, so any source fits |
| The market record in `market.go` holding one `beliefs.Window` and one `BoundRun` per market, and the push at P5.1 | market.go | the world row (`world.go`, `market.go`; CLR 15, 16): the state, the push at P5.1 and the `BoundRun`; the report type and its computation are [plan-phase-1-3.md](plan-phase-1-3.md) D35 (`marketReport`, `seedReport`) | the report is provided by plan 1-3; the world row has no plan yet, **ASSUMED** |
| The belief store per agent (two `int64` per commodity of the tradable set, canonical order) | agent.go | the agent row (`agent.go`, `strategy.go`) | ASSUMED |
| The ledger, the tables of plan-phase-1-2.md and the clearing code | - | plans 1-1, 1-2 and the clearing row | Not consumed: the rule reads none of them |

### Provides

| What | Kind | For |
|------|------|-----|
| `beliefs.Belief`, `Bounds` (with `Conflict`), `Params` (with `WidthInit`) | internal types | agent, world, migration, transport rows (inline conversion) |
| `beliefs.Limits(ref, unitCost, p) Bounds` | function | agent row (the view's `Bounds`, the initial belief), world row (the `OutcomeView`'s `Bounds`); optional for `viability.go` diagnostics `SeedBelowCostFloor` and `SeedAboveCostCeiling` (VIA 8.2.3) |
| `beliefs.Update(b, bd, o, q, stock, target, p, u) (Belief, Result)` (`Result` holds `Open` only) | function | agent row: `DefaultStrategy.Revise` (AGT 7.6, FR-AGT-050). The world never reads its `Result` (A13) |
| `beliefs.Gate(u, adjustProb) bool` | function | world row at P5.4: the `Open` field of the outcome's `Change`, from the outcome's `Draw` and `BeliefAdjustProb`; `Update` calls the same function |
| `beliefs.Change` (`Before`, `After`, `Bounds`, `Open`, `Clamped`; derived `Raised`, `Lowered`, `Moved`, `AtFloor`, `AtCeiling`, `Conflict`) | type | world row at P5.4: built inline per outcome from the stored belief, the belief `Revise` returned after `Clamp`, `Limits`, `Gate` and `Clamp`'s flag; input of `Tally.Add` and of `beliefReason` |
| `beliefs.Draw(b, bd, p, u) int64` (re-clamps with `p`, BEL 3.3) | function | agent row: the limit price in `Trade` (BEL 6; FR-BEL-019) |
| `beliefs.Clamp(b, bd, p) (Belief, bool)` | function | world row at P5.4 (FR-BEL-026) and wherever a stored belief is read (BEL 3.3) |
| `beliefs.Reseed(...)` | function | agent row (initial agents, later entrants, external agents), migration, merchants/transport (arrival), integration (`Relocate`), spawner (role change) (BEL 8.3: one helper) |
| `beliefs.Window` (`NewWindow`, `Push`, `Len`, `Mean`, `Low`, `High`, export and restore of the values) and `beliefs.Position` | type and function | world row (`market.go`, P5.1, the view's window fields, the snapshot), `checkpoint.go` (FR-BEL-035) |
| `beliefs` diagnostic run function (takes `warnRun`) | function | world row (the `Warn` "price beliefs at bound", P5.4) |
| `beliefs.Tally` (`Add(Change)`) | type | world row (one `BeliefsUpdated` per market per tick) |
| Root `Belief{Centre Price; Width int64}`, `Bounds{Floor, Ceiling Price}` (exported, A2), `beliefParams`, `beliefReason`, `beliefsEvent` | beliefs.go | agent row (`OutcomeView`, `AgentView`, `Strategy.Revise` signature; converts by a plain cast), world row |
| `ReasonGateClosed`, `ReasonMoved`, `ReasonHeld`, `ReasonAtFloor`, `ReasonAtCeiling` | event.go (appended) | world row ("belief updated" `Debug` line) |
| Spec lines (Phase 1): the outcome draw keyed by (agent, commodity); the tally and the reason as functions of what the world holds (A13); the `Debug` reason precedence; the `Floor` equals `Ceiling` rule; `Adjusted` as gates open; the BEL 20 shapes | spec-beliefs.md | all rows that read BEL |
| No sentinel | - | the package returns no error (A9); no public sentinel is added |
| No configuration field, default, bound, event or snapshot field | - | `testdata/golden/off/` and the pinned hash do not move (D43) |

## Security review

The module is a library with no network, no file access, no credentials and
no persistence. This row adds no trust boundary and no public entry point:
the package is `internal/`, and `beliefs.go` adds two exported value types and
five constants.

- **Who can reach it.** A consumer reaches the rule only through the world
  (later rows): a strategy's returned belief, an intent's `Target` and
  `UnitCost`, a seed table from a configuration. Each is bounded before it
  reaches the package: the configuration by validation, an intent's `Target`
  or `Reserve` above `MaxQuantity` is treated as 0 (AGT 6.5.2), and a returned
  belief is clamped (FR-BEL-026). A `Stock` is the world's own figure and is
  not bounded by AGT 6.5.2; A9 defines the package's reading of it.
- **Integer overflow.** Every product and ratio goes through
  `internal/checked`; the largest quantities (`Stock * 10 000`, `Centre *
  Width`, `Ref * RefBandHigh`) are tested at their edges and an overflowing
  ratio is read as the larger side (A9), so a hostile strategy cannot wrap a
  centre.
- **Division by zero and panics.** `Target` 0, `Quantity` 0, `high` equal to
  `low` and a range of one price are defined: `Target` 0 (BEL 5.5, FR-BEL-014),
  `high` equal to `low` (7.3) and a range of one price (6) by the spec, and
  `Quantity` below 1 by A9, which Phase 1 writes into BEL 5 because no section
  defines it. The package panics on no input: an out-of-range parameter is a configuration
  error caught at validation, and the fuzz test drives arbitrary values.
- **Randomness and determinism.** The package takes no random source and no
  time, keeps no state outside its values, and ranges over no map
  (architecture.md 7.1 to 7.5); the same inputs give the same bytes on any
  architecture.
- **Resource exhaustion.** A window holds at most `PriceWindow` (at most
  1 000, validated) integers, allocated once per market at construction; a
  belief is two `int64`. Neither grows with the length of a run, the number of
  trades or the weights (NFR-BEL-002).
- **Not applicable:** authentication, authorisation, injection, secrets,
  cross-tenant exposure, supply chain (no dependency is added).

## Compatibility and rollout

- **Exported surface added.** `Belief`, `Bounds` and the five `Reason`
  constants are new exported identifiers of the root package, a compatibility
  promise (architecture.md 2.8, 6.6), with the field set fixed by BEL 3.1 and
  AGT 6.3. There is no release and no consumer yet, so a correction before the
  first release is a normal PR. Nothing in `internal/beliefs` is exported from
  the module.
- **No existing contract changes.** No field is added to `Config`, `World`,
  an event or a snapshot row; the hash and `testdata/golden/off/` do not move
  (D5, architecture.md 2.10.5). The comment of `BeliefsUpdated.Adjusted` is
  corrected and its meaning is the spec's, which no code reads yet.
- **Document contracts.** Phase 1 rewords BEL 4.5, 5 (the degenerate inputs
  of A9), 5.1.1, 8.3, 12.1, 12.2, 20 and FR-BEL-008 and renumbers nothing
  (testing.md 4.1); it also adds the glossary terms to `doc.go`. `doc/notes.md` gets a
  fresh tag.
- **What becomes hard to change, and when.** The arithmetic of the rule, the
  rounding of the mean and the window's order enter the golden files of the
  world row and the checkpoint in Stage 2; the draw's use of `u` (high word of
  `u * n`) enters every golden file. BEL's own preamble says every default is
  a placeholder and a change of the rule moves the goldens (testing.md 7.2).
  Until the first golden file is recorded they may be corrected in a normal
  PR.
- **Rollout** is one PR per phase (D49). Each leaves the branch passing every
  command of `AGENTS.md`. A phase is rolled back by reverting its PR; later
  phases depend only on earlier ones, and no other row calls the package yet.
- **Dependency failure.** None: no dependency, no run-time service.
- **The Go toolchain.** `go.mod` says `go 1.25`. No sort is used (the window
  is a ring and the sets are given in canonical order), so no unstable order
  can arise (architecture.md 7.6).

## Testing (summary, testing.md section 11)

The pyramid for this row is all base: table-driven unit tests of the pure
rule, `Guard:` tests for the invariant, the properties of BEL 5.8 over a fixed
seed range, one fuzz test of the update, allocation tests and a benchmark.
There is no integration or end-to-end test because there is no world; the
world row adds them (S01, S04, S07, S09 and S21, the non-interference test,
the cardinality test over `one`, the golden rendering of `BeliefsUpdated`).

| Phase | Covered in full | Covered in part (completed in) |
|-------|-----------------|--------------------------------|
| 2 | FR-BEL-004, 005 | FR-BEL-019 (the draw consumes the one `uint64` it is given and no more; taking exactly one word from the world's stream is the caller's act, `Trade` in the agent row); FR-BEL-002, 003 (the bounds and the clamp here, the update in Phase 3); FR-BEL-006 (checked bounds here, the update in Phase 3); FR-BEL-026 (the clamp function here; the world applies it) |
| 3 | FR-BEL-009 to 018, 034, 036; NFR-BEL-003 (`Gate` and `Update` here; `Change` and its reads are checked with FR-BEL-030 in Phase 4) | FR-BEL-002, 003, 006 (the update); FR-BEL-025 (the rule half: weights scale quantities, not the result; the one draw per outcome is the world's); FR-BEL-037 (`Δe` at least 1; the bound is Stage 0's); NFR-BEL-001 (the pure path) |
| 4 | FR-BEL-021, 023 | FR-BEL-020 (the ring; the push once per tick and the report are the world's); FR-BEL-022 (the helper; the agent row creates the beliefs); FR-BEL-024 (the helper takes the seed price; the market's initial price is the world's); FR-BEL-030, 031 (the tally and the run; the emission and the `Warn` are the world's); FR-BEL-035 (the lossless window; the encoding is Stage 2's); NFR-BEL-001, 002 |
| 5 | none | FR-BEL-030 (the event mapping); FR-OBS-006 (the five constants) |
| 6 | none | the citation of FR-BEL-027, 028, 037, 038 in the Stage 0 tests that cover them |

"Covered in full" means at the lowest level that can verify it (testing.md
2.1). The second half of testing.md 2.3, the same behaviour through the public
API at the integration level, is the world row's for every consumer-visible
requirement above: FR-BEL-007, 008, 030, 031, 032, 033, 039 and the half-ways
listed.

Already built in Stage 0 and not re-tested here: FR-BEL-027, 028, 029, 037 (the
bound), 038 (validation, under FR-CFG-039 and FR-BEL-029).

Deferred with their code: FR-BEL-001 (the belief store and its creation,
`agent.go`); FR-BEL-007, 008 (the outcome loop and its draw, `world.go`);
FR-BEL-032, 033 (the log lines and non-interference, `world.go`); FR-BEL-039
(the seed price before the first trade, the market record); FR-BEL-035 (the
checkpoint, Stage 2); NFR-BEL-001 (the call sites in the tick loop and the
full `Step`, `world.go`).

Boundaries and error paths: the package has no error result, so no sentinel
case; every branch of 5.4 and 5.5 has a case that breaks only that rule, and
the edges (exactly at a threshold, one beyond; `Target` 0; a range of one
price; `Floor` equal to `Ceiling`; the largest inputs) are cases
(testing.md 3.8). Golden files: none exists yet. Fuzz: `FuzzUpdate`. Each PR
states its coverage and that its tests were run against a broken version
(testing.md 9.3).

## Observability (summary, observability-and-logging.md section 9)

None added, because: (1) no failure path exists, since an update that cannot
be applied does not exist (BEL 12.2) and the package returns no error; (2)
the state transition, a belief updated, already has its event, `BeliefsUpdated`
(Stage 0), with no map and fixed-point values, and this row only builds its
counts and maps them; (3) the update is a decision an operator may want
explained, so the `Debug` line "belief updated" with its `reason` is specified
(BEL 12.2), its five values are declared here and the line is the world row's;
(4) the package has no loop that logs, and the world's per-outcome loop guards
its `Debug` line with `Enabled` and constructs no event when no observer was
supplied (the allocation test is the world row's: NFR-BEL-001, 8.2); (5) no
event, metric, unit or attribute changes; (6) no metric attribute is added (the
`economysim.bound` attribute is fed by the existing event); (7) neither
`Error` nor `Info` is used (FR-BEL-032); (8) no error is logged and returned;
(9) the tests of section 8 that apply here are the allocation tests of Phases
2 to 4, and the rest need a world.

## Open items

- **O1.** `BeliefsUpdated.Adjusted`: BEL 12.1 says gates open and the field's
  comment in `event.go` says beliefs that moved. A11 follows the spec and
  Phase 5 corrects the comment; if the owner prefers "moved", the spec and
  the metric text change instead and `Raised` plus `Lowered` already hold it.
- **O2.** Owed to the owner, A5 and A6: the precedence of the `Debug` reason
  and the treatment of a belief with `Floor` equal to `Ceiling` are
  recommendations written into BEL in Phase 1; they change only a log line,
  the event's counts in a rare conflict and the metric `belief.at_bound` in
  the same case. A13's reading of a closed-gate re-clamp (counted in `Raised`
  or `Lowered`, the literal reading of BEL 12.1, with the reason
  `gate_closed`) is the same kind of recommendation. One consequence is stated, not hidden: a belief at both
  bounds counts twice in `AtFloor + AtCeiling`, so fewer than half of a
  market's outcomes can fire the `Warn` (A6).
- **O3.** The unit-count imbalance signal (BEL open question 6) and the
  residual offset of N-042 wait for S01 with AGT's quantity rule; nothing in
  this row blocks or changes them.
- **O4.** No CI exists (O3 of plan 0). The gate of Phase 6 and the timed fuzz
  run are done by hand, on one architecture.
- **O5.** The cross-check against `doc/specs/models/pb/b_bel.py` (its
  `bounds`, `update` and `draw_limit`, with `EV` set to the guarded rule and
  an evidence share of one half): worth doing for an independent second
  reading of the rule, run in Docker only with `PYTHONDONTWRITEBYTECODE=1`
  (notes N-103). It reproduces only the share 5 000 and the model's
  quantity handling, so it complements and does not replace the hand cases.
- **O6.** The exported-name review of A10 (`beliefs.Belief` against
  architecture.md 2.5) and of A2 (root types in `beliefs.go`), for the owner
  or the reviewer of the first PR; the rename, if any, is one PR before
  Phase 3.
- **O6a.** Inline conversion (architecture.md 2.9: "inline in the calling
  code"). `beliefParams` and `beliefsEvent` are named functions, argued in
  Phase 5 to be a constructor and a domain-event builder, not converters. A
  strict reviewer may ask for the conversion of `beliefsEvent` at its first
  caller, the world row, and for `beliefParams` to be a block of
  `NewWorld`. The plan keeps them as functions (they are unit-testable
  without a world and have two or more call sites in tests); the move to the
  caller is one PR and changes no behaviour. The conversions between the root
  `Belief` and `Bounds` and the internal ones are casts at the caller, never
  a function (A2).
- **O7.** Pre-existing, outside this row: the table of BEL 11 gives the range
  of `BeliefWidthMax` as `BeliefWidthInit` to 9 000, while `bounds.go`
  accepts up to 9 999 and FR-BEL-028 says only "10 000 or more" is rejected.
  `Draw` and `Update` are safe at 9 999 (3.3: `Width` below 10 000 keeps the
  half-width below the centre), so nothing here depends on it. Phase 1 may
  align the table to 9 999 in the same edit as BEL 20 if the owner agrees;
  otherwise it is a note at the next BEL edit.
