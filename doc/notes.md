# Notes for future work

A register of things to take into account in work that has not been done yet:
constraints one spec places on another, decisions to revisit, traps found while
writing, and tasks that wait for a spec or for code. It is read **before writing
or changing a spec** (`AGENTS.md`), and it carries a read-receipt tag at the end
that an agent quotes when it starts that work.

It is not a second home for decisions or requirements. A decision lives in a spec
or an ADR, a requirement in a spec's Requirements section, and a spec's own open
questions stay in that spec. A note points at those and says *who needs to know,
and when*. If a note is only an open question of one spec, leave it there.

## How to use it

**Before writing or changing a spec.** Find the spec's three-letter area code in
the index below (the codes are in [specs/README.md](specs/README.md)). Read every
open note tagged with that code and every note tagged `ALL`. For each one, do one
of three things in your change, and say which in your summary:

- **apply** it (the spec now does what the note asks), then close the note;
- **reject** it, with the reason, then close the note;
- **leave it open**, if it is not yours to settle yet, and add what you learned.

**While working.** If you find something a future spec or a code change must take
into account, add a note in the same change. Do not leave it only in a chat or a
PR description.

**Rules for entries.**

- One note, one topic. Keep it to a few lines and link instead of copying.
- Tag it with the area codes of the specs that must read it. `ALL` is for every
  spec. `CODE` is for the first code change that touches the area. `EVD` is for
  the evidence file.
- Give a stable ID `N-NNN`. IDs are never reused. Never delete a note: close it,
  moving it to "Closed notes" with the resolution and where it was settled.
- Give the date it was raised (the format is `YYYY-MM-DD`) and where, so a reader
  can check it is still true. A note that names a section or a requirement may go
  stale when that changes: fix the pointer when you see it is wrong.
- Update the index in the same change, and replace the read-receipt tag at the
  end of the file with a fresh random value.

## Index by area code

| Area | Open notes |
|------|-----------|
| ALL | N-016, N-017, N-027 |
| AGT | N-043, N-067, N-081, N-087 |
| BEL | N-042, N-043 |
| CAL | N-030 |
| CAR | N-011, N-076 |
| CFG | N-016, N-061, N-075 |
| CHK | N-020, N-051 |
| CLB | N-015 |
| CLR | N-076 |
| CODE | N-019, N-089 |
| CRD | N-032, N-046 |
| DEM | N-080, N-089 |
| EMP | N-046, N-078 |
| EST | N-082 |
| EVD | N-018, N-029 |
| EXP | N-015 |
| EXT | N-021, N-046, N-078, N-084 |
| FIS | N-021, N-046, N-078 |
| GDS | N-031, N-039 |
| IND | N-013 |
| INF | N-046, N-084 |
| INT | N-022, N-031, N-036, N-051, N-079 |
| LED | N-021, N-076, N-083 |
| LND | N-046, N-078 |
| MER | N-011, N-046, N-076, N-084 |
| MIG | N-010, N-038, N-046 |
| MON | N-021, N-046, N-078, N-084 |
| OBS | N-012, N-031, N-035, N-040, N-045, N-065, N-067, N-072, N-081, N-087, N-088 |
| PLC | N-025, N-046 |
| POP | N-008, N-029, N-046, N-083 |
| PRO | N-030, N-031, N-039, N-043 |
| REF | N-027, N-030, N-089 |
| REG | N-010, N-026, N-030, N-084 |
| RES | N-030, N-031, N-036, N-046 |
| RSK | N-046 |
| SAV | N-046 |
| SCL | N-030 |
| SET | N-026 |
| SKL | N-039, N-071 |
| SPN | N-043, N-089 |
| STB | N-028, N-032, N-039, N-042, N-043, N-045, N-067, N-086, N-087 |
| STO | N-011, N-046 |
| STY | N-015, N-027, N-030, N-084, N-085, N-086 |
| SUB | N-046 |
| TRN | N-011, N-046, N-076, N-084 |
| VIA | N-015, N-061, N-071, N-080, N-085 |
| WEL | N-016, N-028, N-081 |
| WLD | N-009, N-031, N-036, N-046, N-051, N-056, N-074, N-082 |

## Open notes

### N-008 Weight changes, cohort failure and the estate

- **Areas:** POP
- **Raised:** 2026-10-09, `spec-scale.md` 4.8 and open question 4, `spec-ledger.md`.
- **Note:** A cohort whose money reaches 0 fails whole. Recommended: reduce its
  weight by the share it cannot support before removing it (`spec-scale.md` 5.1),
  inside a failure ladder (change role, relief, then death). A weight change is a
  ledger operation with reason `split`, `merge`, `births` or `starvation`, scaling
  holdings in proportion with the remainder going to the estate
  (`spec-scale.md` 4.8; `spec-ledger.md` 7.2). Account IDs are never reused, so a
  run with many deaths accumulates closed accounts (`spec-ledger.md` open
  question 6; N-020). The estate pseudo-agent is not an agent: it has no weight and
  is not counted in agents or households (`spec-estates.md` 2.7.4).
- **Update (2026-10-10, AGT):** a cohort does not fail by itself; an agent with no money still exists and goes unfed (`spec-agents.md` 3.6.1). AGT supplies `Unfed` and `Hunger` (5.4, 5.5) and the effect of a weight change on agent state (3.7). The ladder and the partial weight reduction by the unfed share are SPN and POP's.
- **Update (2026-10-10, EST):** weight changes pass money through the estate (`spec-estates.md` 2.8.1); a rise the pool cannot fund is open (N-073). The ladder is SPN and POP's.
- **Update (2026-10-10, SPN and EST):** applied for SPN and EST in `spec-spawner.md` 5 (the ladder, reduction by the unfed share, the estate path) and `spec-estates.md` 2.8. POP owns the demographic rates (`PopDelta` hook, `spec-spawner.md` 8; the ledger reason `deaths`).
### N-009 Decisions the world and tick phases must make

- **Areas:** WLD
- **Raised:** 2026-10-09, from CLR, LED, CFG and SCL.
- **Note:** `spec-world.md` fixes, and several specs already assume:
  - all orders of a tick are placed before any market clears, and markets clear in
    region then commodity order (`spec-clearing.md` 7.2);
  - estate pseudo-agents are created with their regions, before any household
    (`spec-estates.md` 2.7.6);
  - ageing runs at the end of the tick (`spec-ledger.md` 5.3);
  - `Step` surfaces `ErrInvalidTrade` and, with `WithAudit`, `ErrAuditFailed`
    (`spec-clearing.md` open question 5; `spec-ledger.md` 12.4);
  - a checkpoint is taken between ticks, with no reservation open
    (`spec-ledger.md` 6.5);
  - options that change results (such as `WithTargetSpread`) are part of the
    effective configuration and its hash (`spec-config.md` 8.1);
  - decisions run at different rates (roadmap, WLD row).
- **Update (2026-10-10, AGT):** AGT assumes the phase order of `spec-agents.md` 4: start (moment, cost anchors, relief and starting money, directive), production (continue and complete, then `Work`, then starts), `Trade` and orders, clearing, outcomes and `Revise`, consumption, entry and exit, end (ageing). Creation draws in the order target ratio then residue (3.5.2). VIA: the warm start is applied before the agents' creation draws (`spec-viability.md` 8.6 item 5).
- **Update (2026-10-10):** a construction phase precedes tick 0 and ends at the first `Step` (`spec-ledger.md` 5.2.4, N-074).
- **Update (2026-10-10):** AGT phase 6 includes `Privation` and want consumption; phase 7 is detailed in `spec-spawner.md` 10.2 (relief in phase 1 from what the previous tick's entry step left, `spec-estates.md` 2.6; see N-082).
### N-010 Regions

- **Areas:** REG, MIG
- **Raised:** 2026-10-09, from GDS, EST and LED.
- **Note:** `spec-regions.md` owns the per-region override of a commodity's
  reference price (`spec-goods.md` 6.3) and decides whether a region holds one
  settlement or several (roadmap, Open 3). Each region creates its estate
  pseudo-agent and account (`spec-estates.md` 2.7). Changing an account's region
  is an explicit operation (`spec-ledger.md` 3.5) and its interaction with goods in
  transit is open (`spec-ledger.md` open question 8; `spec-migration.md`).

### N-011 Transport, storage and carriage

- **Areas:** TRN, STO, CAR, MER
- **Raised:** 2026-10-09, `spec-goods.md` section 3.
- **Note:** A vehicle's and a store's capacity is a pair, a mass capacity and a
  bulk capacity, and a load fits only if both are within it
  (`spec-goods.md` 3.2). Pricing uses the chargeable size (`BulkFactor`, 3.3, 3.4),
  which over-charges well-mixed cargo (3.5). Measure how much on the reference
  chain, and add a per-mode factor if the error matters. Goods keep aging in
  transit and in transit accounts; cargo lost is a `loss` sink
  (`spec-ledger.md` 7.2). Contract money is held in an Escrow account, not a
  reservation (`spec-ledger.md` 6.6). `spec-carriage.md` has no Requirements
  section yet, and its open items (`specs/README.md`, "Still open" 5 and 6) belong
  to the contract and merchant specs.
- **Update (2026-10-10, CAR):** `spec-carriage.md` now has Requirements (FR-CAR-001 to 013), restates its open items 5 and 6 with the reason they wait, and records a finding in 4.2 (see N-076).
### N-012 Events, metrics and log keys to define

- **Areas:** OBS
- **Raised:** 2026-10-09, collected from LED, SCL, CFG and EST.
- **Note:** `spec-observability.md` must define or accept:
  - `BoundaryFlow`, one aggregated event per region, reason and commodity per tick
    (`spec-ledger.md` 12.1);
  - `Weight` on `AgentSpawned` and `AgentBankrupted`, and agents and households
    reported separately in the snapshot (`spec-scale.md` 9);
  - the configuration hash on `WorldCreated` (`spec-config.md` 10.3);
  - `EstateOpened` and `ReliefPaid` (`spec-estates.md` 2.5, 2.6);
  - a household-count metric, `economysim.household.count`, as a change to the
    metric catalogue (`spec-scale.md` 9.4);
  - the log keys `path`, `rule` and `config_hash`, and the ledger reasons as the
    values of the `reason` key.
  Estate pseudo-agents must not count in `economysim.agent.count`.
- **Update (2026-10-10):** AGT adds `UpkeepShortfall` (one per region and role per tick, `spec-agents.md` 14.1); `AgentSpawned` gains `Role`, `StartMoney`, `External`; the removal event gains `Cause`; log keys `households` and `unfed` (14.4); a metric `economysim.household.unfed` is proposed (14.5). STB asks for counters on dampers that act silently (N-045).
- **Update (2026-10-10):** OBS also decides the name of the Snapshot field for the households held to work by the floor, and whether the construction bucket is a `BoundaryFlow` with `Tick` -1 (N-072).
- **Update (2026-10-10):** the events, keys and metrics of the written specs are now in the rules (`rules/observability-and-logging.md` 3.3.2, 4.3, 5.3). Still open for OBS: the Snapshot fields (N-065, N-081), `Tick` -1 for the construction bucket (N-072), the snapshot-fed gauges, and the remaining conflicts of the rules edit (N-088).
### N-013 What indicators must not get wrong

- **Areas:** IND
- **Raised:** 2026-10-09, from SCL, LED and GDS.
- **Note:** Inequality within a cohort is zero by construction, so an inequality
  measure sees only differences between agents (`spec-scale.md` 5.1). Population
  indicators count households (the sum of weights), never agents, and exclude
  pseudo-agents (FR-SCL-015). Balance sheets value goods at a price the caller
  supplies and are measurement only (`spec-ledger.md` 9.2). Wear units are counted
  by commodity and could back a depreciation measure (`spec-ledger.md` 5.4.2, open
  question 7). Commodity classes are the grouping for price indices and value
  added (`spec-goods.md` 6.1).

### N-015 Placeholders to calibrate once something has run

- **Areas:** VIA, STY, CLB, EXP
- **Raised:** 2026-10-09.
- **Note:** These numbers are guesses, not results, and nothing has been run:
  `MinAgents` of 10 (`spec-scale.md` 5.2); the 10% scale-consistency tolerance
  (7.2); 200 000 agent-ticks a second (6.2); the bounds in `spec-config.md` 4.1;
  the validation time in NFR-CFG-001; the per-field bounds in `spec-goods.md` 8;
  the `TargetSpread` default of 0 (`spec-welfare-and-wages.md` 5.2). The Doran and
  Parberry paper is unread and bazaarBot has not been run, so "stable" has no
  reference yet (`bazaarbot-reference.md` section 12; `vision.md` section 6).
- **Update (2026-10-10):** the Doran and Parberry paper is read (journal version; use `ianparberry.com/pubs/econ.pdf`, the LARC URL 404s), so "stable" has a prior definition: the filter criteria of the paper's section VII, listed in `spec-beliefs.md` section 19. Still true: bazaarBot has not been run and no number has been validated. New placeholders: the 15 belief parameters of `spec-beliefs.md` section 11 (most importantly `CostFloor` 10 000, `BeliefAdjustProb` 5 000, `BeliefStep` 200 and the stock band 5 000 to 15 000), and `TraineeYield` 5 000 and the production bounds of `spec-production.md` 11.
- **Update (2026-10-10):** AGT placeholders (`spec-agents.md` 12): `CoverTicks` 3, `InputJobs` 1, `OutputCoverTicks` 1, `FavourFloor` 2 000, `ReserveTicks` 3, `SpendMaxShare` 5 000, `SpendHalfTicks` 10, `RestBand` 500, `ProduceHaltRatio` 30 000, `BasketTolerance` 1 000, `StartMoney` 10 000, `MaxUpkeepItems` 16, and the worked examples of section 16. VIA placeholders: `StockCoverTicks` 5, `MoneyBufferTicks` 10, `ResidualWarnBP` 1 000, `MaxViabilityPivots` 100 000, `MaxViabilityCells` 4 000 000 and the prototype numbers of `spec-viability.md` 15; the Go speed of the exact solver is not measured. STB adds only placeholders: every threshold of the failure regimes, the example values of its proposed parameters, and every number of its models.
- **Update (2026-10-10, round 3):** new placeholders: `CostFloor` 7 000, `BeliefEvidenceShare` 5 000, `WelfareWindow` 30, `WindowCycles` 2, `MinWorkShare` 4 000, `MaxWelfareWindow` 3 650; scale cost model constants 2 us, 1 us and 8; the rescaled example quantities of `spec-production.md` 12; the skill-barrier tolerance 200 bp and queue discount; and the bounds of `spec-config.md` 4.1 and `spec-estates.md` 3.
- **Update (2026-10-10, round 4 and 5):** new placeholders: every default of `spec-demand.md` 9, `spec-spawner.md` 11 (`EntryShare` 50, `ExitShare` 100, `Renewal` 10 000, `EntryPoolShare`), `spec-reference-chain.md` (every pack quantity, the censuses, seeds and run lengths) and the thresholds of `spec-stability.md` 7 and 11.
### N-016 Keep the parameter registry true

- **Areas:** ALL, CFG, WEL
- **Raised:** 2026-10-09, `spec-config.md` 9.3.
- **Note:** Every new configuration field needs a row in the parameter-owner table
  (architecture.md 9.5, 9.7). Known gaps: the welfare window `T` (default 30,
  `spec-welfare-and-wages.md` 2.3.1) has no row; the tolerance for matching a
  region's `ReferenceBasket` to its reference prices (`spec-currency.md` 5.3) has no
  name or default; `spec-clearing.md` calls `MaxPrice` and `MaxQuantity`
  "configured" although they are bounds in `bounds.go` (`spec-config.md` 4.1).
  A role's default `ExpectedWorkTicks` of 250 becomes invalid if the calendar gives
  fewer ticks a year (`TicksPerYear` is derived, `spec-calendar.md` 3.2;
  `spec-config.md` open question 5). The calendar fields have rows in the
  parameter-owner table since 2026-10-10.
- **Update (2026-10-10):** production added nine rows and beliefs fifteen to the parameter-owner table; no new unnamed parameter. `YieldProfile` has two owners over time (PRO now, RES later); PRO is listed as owner until RES exists.
- **Update (2026-10-10):** AGT adds 17 rows and names `BasketTolerance`, closing one gap of the known list (the unnamed basket tolerance of `spec-currency.md` 5.3); VIA adds `WarmStart`. VIA's solve options (`StockCoverTicks`, `MoneyBufferTicks`, `ResidualWarnBP`, `MaxPivots`, `Lanes`) are not `Config` fields and are NOT in the parameter-owner table, because the registry test of testing.md 6.7 requires a field behind every row; they live in `spec-viability.md` 11. STB adds no field; its four proposals (`MinWorkTicks`, `ProdStopRatio`, `EntryShare`, `RoleHysteresis`) need rows if adopted.
- **Update (2026-10-10, round 3):** `spec-config.md` 3.7 lists every field and 9.2 says a row is a `Config` field; the parameter-owner table now has a row for each (generated from 3.7, plus `WelfareWindow`, `WindowCycles`, `MinWorkShare`, `BeliefEvidenceShare`), and the solve options of `spec-viability.md` 11 are deliberately not rows. Closed: the `BasketTolerance` name, the `MaxPrice` and `MaxQuantity` wording (they are bounds), `ExpectedWorkTicks` against a derived `TicksPerYear` (an error, FR-CFG-048). Remaining: a carriage `m_bp` has no name; a test must verify the table against the code once it exists.
- **Update (2026-10-10, round 4 and 5):** the parameter-owner table gained the demand and spawner fields (`World.Spawner`, `Role.Closed`) and a row for every field of `spec-config.md` 3.7, regenerated after the new modules. No unnamed parameter was found in AGT, WEL, DEM, SPN or REF. A test must still verify the table against the code once it exists.
### N-017 Words and names

- **Areas:** ALL
- **Raised:** 2026-10-09.
- **Note:** Names are setting-neutral (`AGENTS.md`). Traps already hit: `Weight` is
  the households an agent stands for, so a commodity has `Mass`, not weight; a
  "day" in the specs means a tick and a "year" is `TicksPerYear` ticks
  (`spec-currency.md` 2.4, now derived from the calendar); a calendar day is a
  different unit, since a tick may be a part of a day or several days
  (`spec-calendar.md` 2.5); "consumer" means the library consumer (N-001); a "pack"
  is configuration, a "module" is an optional mechanism. Add new terms to the glossary
  list in `specs/README.md`, and to `doc.go` before code uses them.

### N-018 Unverified evidence

- **Areas:** EVD
- **Raised:** 2026-10-09.
- **Note:** Several facts behind defaults are unverified summaries
  (`specs/evidence.md`). The traineeship elasticities disagree and come from unopened
  snippets (`specs/README.md`, "Still open" 8). Do not set a default from such a
  fact without checking the source.

### N-019 Housekeeping for the first code change

- **Areas:** CODE
- **Raised:** 2026-10-09.
- **Note:** When the first code lands, in the same PR:
  - update the layout in architecture.md 2.6 and `AGENTS.md` for `config.go`,
    `defaults.go`, `bounds.go`, `validate.go`, `hash.go`, `commodity.go`, `calendar.go`, the
    optional `denomination/` package (`spec-currency.md` 7), `internal/ledger` and
    `internal/clearing`;
  - add the new terms to the glossary in `doc.go` and the new options
    (`WithAudit`, `WithTargetSpread`) to the options list;
  - add the log keys of N-012 to observability-and-logging.md 3.3.2, and any new
    event to its table 4.3;
  - record an ADR choosing the TOML library, with the justification of
    architecture.md 5.3 (`spec-config.md` 6.1), and check its strict decoding;
  - amend architecture.md 4.3 and 7.7 for the checkpoint and optional extension
    state, as ADR 0002 says, with a fresh read-receipt tag.
- **Update (2026-10-10):** the log keys and events of the written specs are in `rules/observability-and-logging.md`. The rest of this note is unchanged: the architecture.md 4.3 and 7.7 amendment for the optional extension state (ADR 0002) waits for the first code.
### N-020 What each spec contributes to a checkpoint

- **Areas:** CHK
- **Raised:** 2026-10-09.
- **Note:** Every spec lists the state it adds (ADR 0002). Known so far: the ledger
  (`spec-ledger.md` 11) and the estate's listing prices (`spec-estates.md` 2.7.5).
  Not saved: reservations, flow totals and the derived expiry hints. The
  configuration hash includes result-affecting options (`spec-config.md` 8.1). Open:
  compaction of closed accounts (N-008), the extension-state interface, and the
  compatibility policy after 1.0 (ADR 0002, open points).
- **Update (2026-10-10, LED):** `Born` is signed in the saved state; the construction bucket and pending output are not in the ledger state (`spec-ledger.md` 11). The agent state AGT contributes is in `spec-agents.md` 20, the beliefs' in `spec-beliefs.md` 14.
- **Update (2026-10-10):** the spawner's state (`Privation`, `RoleSince`, the renewal carry, two diagnostic runs) is in `spec-spawner.md` 14; the demand module's carries are in `spec-demand.md` 12.
### N-021 Anything that creates or destroys money or goods

- **Areas:** MON, EXT, FIS, LED
- **Raised:** 2026-10-09.
- **Note:** A new faucet or sink is a row in the ledger's reason registry, with its
  kind and owner (`spec-ledger.md` 7.2); there is no other way to change a total.
  The boundary balance per reason is the measure of how much has entered or left.
  Credit will use the ledger's claims list (`spec-ledger.md` 10). A commodity marked
  `Raw` needs a real source once resources, the external sector or faucets exist
  (`spec-goods.md` open question 5).
- **Update (2026-10-10, LED):** the reason registry gained `disposal` and the goods form of `initial`; `initial` is not construction-only (an external agent mid-run, `spec-ledger.md` 7.2.1).
### N-022 The player and other external agents

- **Areas:** INT
- **Raised:** 2026-10-09.
- **Note:** An agent added through the integration interface has weight 1
  (`spec-scale.md` 4.7). It has an ordinary agent account and is bound by the same
  admission and reservation rules as any agent (`spec-clearing.md` 3).
- **Update (2026-10-10, AGT):** `spec-agents.md` 10. An external agent has weight 1, an `External` flag, and a strategy that is either a non-blocking consumer `Strategy` or the library's `Directed`, which reads a `Directive` left in the agent's mailbox between steps (applied to the next tick, discarded unless `Sticky`, saved in a checkpoint). No action happens inside a step; an "act now" operation is not provided, and INT may add one with its own rules. INT owns `AddAgent` and `Direct`. The spawner must not remove or replace an external agent unless the consumer asks.
- **Update (2026-10-10, SPN):** an external agent is refused by the world for removal or replacement unless it is `Mortal` (`spec-spawner.md` 9, FR-SPN-010; `spec-agents.md` 10.6).
### N-025 Do policy actors reuse `Strategy`?

- **Areas:** PLC
- **Raised:** 2026-10-10, `roadmap.md` Open 2.
- **Note:** Open: whether a policy actor (a ruler choosing tax rates, minting,
  granary release, price controls) reuses the `Strategy` extension point or needs
  its own interface. A new interface means a change to the table in
  architecture.md 4.3, which also requires a default implementation and a
  compile-time assertion (4.5), and receives its randomness from the world (7.7).
  Decide when writing `spec-policy.md`.
- **Answer (proposed, 2026-10-10):** a separate interface, written when PLC is.
  ASSUME reuses one `calculate_bids()` interface for generators, demand and custom
  units, which supports one `Strategy` for every actor that *bids*. A policy actor
  does not bid: it reads measures and returns settings (rates, release amounts), so
  the signature and the returned type differ. A ruler that also trades (buying grain
  for a granary) uses `Strategy` for its orders and the policy interface for its
  settings. Do not create the interface before PLC (architecture.md 4.2).
- **Update (2026-10-10):** ADR 0003 decision 3 records the answer (its own interface, written with `spec-policy.md`). Close this note when PLC is written.
### N-026 One settlement per region, or several?

- **Areas:** REG, SET
- **Raised:** 2026-10-10, `roadmap.md` Open 3. See also N-010.
- **Note:** One settlement per region is simpler and makes `spec-settlements.md` a
  thin layer; several lets a region have a large settlement and its smaller ones.
  Decide when writing `spec-regions.md`, before `spec-settlements.md`, because it
  fixes where market, estate and size class live.
- **Answer (decided, 2026-10-10):** one settlement per region, and a region is one
  market. A town and its hamlets are separate regions joined by short edges, which
  costs little because clearing cost grows with regions and commodities, not
  households (`spec-scale.md`). `spec-settlements.md` (later) then only gives a
  region a size class from its households, which is the thin layer its roadmap row
  describes. Victoria 3 treats the market as the unit and varies a state's access to
  it instead of splitting prices inside a state (dev diary 9, snippet only). Apply
  this in `spec-regions.md` and close the note there.
- **Update (2026-10-10):** ADR 0003 decision 4 records the answer. Apply it in `spec-regions.md` and close this note there.
### N-027 No default macro constant is stable yet

- **Areas:** ALL, REF, STY
- **Raised:** 2026-10-10, `roadmap.md` Open 4 (ADR 0001, decision 5).
- **Note:** Stable defaults cannot be claimed until `spec-reference-chain.md` and
  the acceptance bands in `spec-stylised-facts.md` exist. Until then no spec states
  a default macro constant as stable. Related: the guessed numbers in N-015.
- **Answer (2026-10-10):** this is a standing constraint, not a question. Keep the
  note open so every spec reads it; close it when REF and STY exist.

### N-028 Can the target-income rule make a shortage worse?

- **Areas:** WEL, STB
- **Raised:** 2026-10-10, `roadmap.md` Open 5. See also N-014.
- **Note:** The rule in `spec-welfare-and-wages.md` may make labour supply fall as
  the wage rises, so a shortage could worsen itself. It is the first entry for
  `spec-stability.md`. Analyse it numerically when that spec is written, and decide
  whether the rule needs a damper or only a condition on demand. A cohort
  multiplies the lockstep of the rule (N-014).
- **Analysis (2026-10-10, derived by hand from section 5, not yet run):** days
  worked are `D = K/W`, so labour supply `N*K/W` is unit-elastic. A self-employed
  role has output `N*K/p`, the same curve in the goods price. With demand
  `Q = A*p^-e`, excess demand has slope `(N*K/p^2) * (1 - e)` at equilibrium, so a
  shortage corrects itself only if demand elasticity `e` is above 1. Staples have
  `e` below 1, so for them a shortage raises the price, days worked fall and the
  shortage deepens. Only three things stop it: the price bounds, households running
  out of money, and the one-day minimum. A spread above 0 smooths the steps but
  keeps the bend.
- **Answer (proposed):** keep the rule, because the backward bend is evidenced
  (Dyer, via Humphries and Weisdorf) for labourers. Make the damper a condition
  plus a test in `spec-stability.md`: bounded, asynchronous price steps and an
  inventory buffer (BEL), and a numeric run with a staple role that injects a
  shortage and asserts the price stays in bounds and recovers. Decide a work floor
  (a producer also works while its household's needs are short; survival before the
  target) from that run. A search snippet says inflection points in labour supply
  give multiple unstable equilibria; it is unverified (`specs/evidence.md`).
- **Update (2026-10-10, STB):** run in `spec-stability.md` 6.1 (stylised model, integer rule). The slope of excess demand is `(1 - e) Q/p`, exact to four places; stable only if `e > 1`; `e = 1` (a fixed budget) is neutral. For `e = 0.5` every seed ends pinned: at the price ceiling after a rise, or at the price where all households work the full year after a fall. An inventory buffer and `TargetSpread` 3 000 do not remove it. A work floor (minimum working days, D13) replaces the ceiling by a price set by the floor. The survival-floor reading is unit-elastic too and is not a damper by itself. The model has no income feedback, so it does not show that a real pack is unstable. **AGT (2026-10-10):** a staple's demand elasticity is 0 to 1, never above 1 (`spec-agents.md` 11.6), so the stability condition `e > 1` fails; the default strategy applies the rule with a rest band and a proportional work share for a cohort (7.1.2, 9.3). Leave open until WEL decides D13 (`MinWorkTicks`).
- **Update (2026-10-10, WEL and STB):** WEL adopted the floor as `MinWorkShare` (`spec-welfare-and-wages.md` 5.3, default 4 000 bp). `spec-stability.md` 6.1 gives the upper outcome as `P0 * (D0 / DMIN)^(1/e)`: about 1.06 times the reference for D0 = 150 days and the default floor, 2.9 for D0 = 250. The instability remains; scenario S04 records the outcome with the floor at its default and at 0. If a pinned market is still unacceptable, N-052 is the next lever. Leave open until S04 has run.
- **Update (2026-10-10, STB):** DEM answers for a luxury: stable only with schedule elasticity above 10 000 and about 10% budget headroom (`spec-demand.md` 6.3; a model), scenario S23. A staple is unchanged. Open until S04 and S23 have run.
### N-029 How deep a review of other economy libraries?

- **Areas:** POP, EVD
- **Raised:** 2026-10-10, `roadmap.md` Open 6. See also N-015, N-018.
- **Note:** Open before `spec-agents.md` and `spec-population.md`. Only bazaarBot
  has been read in full. abcEconomics, Prehistoric, the Space Project, ASSUME and
  Crusoe are known from search snippets, and two documentation pages failed to
  load. Record what is read in `specs/evidence.md`, and do not set a default from
  an unread source.
- **Answer (proposed, 2026-10-10):** a targeted review, not a survey. Opened today:
  the abcEconomics index (Firm and Household archetypes; the platform moves goods,
  as our ledger does; message passing) and ASSUME's custom-unit example (a unit plus
  a bidding strategy with one `calculate_bids()` reused across actor types; dispatch
  stays in the unit). Both support a single `Strategy` and physical handling outside
  it. Still to read before AGT: the Doran and Parberry bazaarBot paper (N-015), the
  abcEconomics Firm and Household pages, ASSUME's base strategy class, and the rest
  of Lengnick (only pages 3 to 7 are read). Before POP: Clark's own Malthusian
  source, not the VoxEU summary. Skip Prehistoric, the Space Project and Crusoe
  unless a specific question needs them. The evidence rows are added to
  `specs/evidence.md`.
- **Update (2026-10-10):** the Doran and Parberry journal paper is read in full (text only). Lengnick's firm price rule, Table 1 and the price-flexibility experiment are read from extracted text with the Greek symbols lost. ASSUME's custom-unit example re-read; its bidding-agents page (404) and the ZIP paper (403) failed to load. Factorio, Victoria 3 and abcEconomics production pages were opened through a summarising fetch only. Still to read before AGT: the abcEconomics Firm and Household pages and ASSUME's base strategy class.
- **Update (2026-10-10, AGT):** the targeted review for AGT is done. Read: Lengnick's household section (equations 1 to 4, 11, 12, Tables 1 and 2, the search part of 2.2; text extracted from the PDF, symbols partly lost); ASSUME's `base.py` (`BaseStrategy`, `BaseUnit`); abcEconomics `Household`, `Firm` and `Trade` pages through a summarising fetch. ASSUME's bidding-agents page still 404. Still to read before POP: Clark's own source.
### N-030 Five regions with different climates

- **Areas:** RES, CAL, REG, PRO, SCL, STY, REF
- **Raised:** 2026-10-10, the first consumer's world size (`vision.md` section 2).
- **Note:** The first consumer needs five regions of varying size and resource
  richness, with different climates, so a crop grows well in some and poorly or not
  at all in others. Consequences:
  - Climate is not an entity or an enum. It is data: a region's yield for each
    gathering or growing recipe, as a basis-point factor with a bound, optionally
    with a seasonal profile from `spec-calendar.md`. A factor of 0 means the recipe
    cannot run there. Nothing branches on a crop or a climate name (architecture.md
    9.1, `AGENTS.md` setting-agnostic rule). Richness is the same kind of data: the
    capacity and regeneration of an endowment (`spec-resources.md`).
  - So RES owns the yield factor and the endowment limit, CAL owns the seasonal
    profile, and PRO's recipe format must let a region modify a recipe's yield
    without a second recipe per region (N-003). Check this against a second,
    unrelated setting, as `vision.md` section 5 item 3 requires.
  - Differences in yield produce comparative advantage, which is what drives trade,
    price convergence within transport cost and specialisation. STY and REF need a
    scenario with at least three regions whose yields differ, not only the
    two-region arbitrage test (`roadmap.md`, REF row).
  - Varying size means households differ by region, so SCL's weights, `MinAgents`
    floor and cost model are checked at the smallest and largest region (N-015).
  - Size, confirmed 2026-10-10: about 1 000 households per region (5 000 in all)
    and 30 commodities, so 150 markets. At scale 1 that is 5 000 agents, within
    the placeholder budget of `spec-scale.md` 6.3 (about 25 ms a tick). Weights
    above 1 are not needed for this world, which also avoids the cohort lockstep of
    N-014 and the chance-event method of N-005 for the first consumer. SCL stays
    core because other consumers may be larger. Missing: the recipe count and the
    tick rate.
  - Region parameters override commodity reference prices (N-010), so each climate
    needs its own prices in the data pack.
  - Seasons, 2026-10-10: `spec-calendar.md` 5.3 defines one season cycle for the
    world. Regions with different climates share it through a table of yield by
    season (a reverse hemisphere reverses the table, a two-season region repeats
    values), not through a cycle of their own. RES owns that table.
- **Update (2026-10-10):** PRO applied its part in `spec-production.md` 6: the yield profile (`all` and by season, keyed by region and recipe), `Sited` for recipes that need a site, 0 = cannot run, no second recipe per region; two worked climates in 12.1 and 12.2. Remaining for RES (the table values and endowments, the `Draws` seam 6.5), REG, STY and REF (a three-region scenario). Still missing: the recipe count and tick rate of the first consumer.
- **Update (2026-10-10, VIA):** `spec-viability.md` 9 and 13.2 (A3, A5): comparative advantage from yield differences, a three-region check (prototype: autarky surplus 2.980, 2.517 and 3.124; pooled 3.124), scenario A3 handed to REF. REF must choose yield ratios large enough to show specialisation.
- **Update (2026-10-10, SCL):** with AGT, PRO and BEL describing the per-agent work, the cost model of `spec-scale.md` 6 is per tradable commodity (`c_0 + n_T * c_t`, placeholders 2 us, 1 us and a reference of 8 commodities): the first consumer's world of 5 000 agents is about 53 ms a tick (about 19 ticks a second), against the earlier 25 ms. Unmeasured; still missing: the recipe count and the tick rate.
- **Update (2026-10-10, REF):** `five` in `spec-reference-chain.md` 4.4.4: five regions with yields by season and recipe, 5 000 households, 30 commodities, 45 recipes (the missing recipe count), 85 active markets (17 per region, not 150), a mean tradable set of 7.7. Each region is viable alone; pooled 2.039. Still missing: the consumer's tick rate. Trade between regions waits for REG, TRN, MER and INF.
### N-031 What the calendar hands to other specs

- **Areas:** RES, PRO, WLD, OBS, INT, GDS
- **Raised:** 2026-10-10, `spec-calendar.md`.
- **Note:**
  - RES and PRO: a season is the index `Calendar.At(tick).Season`, or -1 for none.
    A yield or a recipe condition by season is a table keyed by season ID; no logic
    names a season (architecture.md 9.1). A recipe whose production time exceeds its
    season cannot complete, and validation does not see it (`spec-calendar.md` 9);
    PRO or VIA should.
  - WLD: compute the moment once per tick, before orders are placed. It is derived,
    not state, and adds nothing to a checkpoint (`spec-calendar.md` 6.5).
  - OBS: decide whether the world emits a season event and whether the snapshot
    carries the moment (`spec-calendar.md` open question 4). Events already carry
    `Tick`.
  - INT: a consumer converts ticks to dates with `Calendar.At`; a shock by season
    (a failed harvest) needs the season index.
  - GDS and the other duration fields (`ShelfLife`, `TrainingTicks`): durations are
    in ticks and are not rescaled when the calendar changes, so a pack for another
    planet restates them (`spec-calendar.md` 6.1).
- **Update (2026-10-10):** PRO reads the season and the yield at **job start** (`spec-production.md` 6.4), so a recipe longer than its season completes; "cannot complete" does not arise for production and needs no validation. A harvest window is a perishable intermediate. INT: the yield-shock slot of 6.8 is where a seasonal failed harvest goes. RES, WLD, OBS and GDS parts unchanged.
- **Update (2026-10-10, VIA):** VIA reads the calendar as periodic (the season of tick `-o` is that of `(-o) mod TicksPerYear`) for the virtual start of initial jobs (`spec-viability.md` 8.5.2) and adds check V4 (supply gap against shelf life) for seasonal storage (4.5). Neither changes CAL; CAL may want to state the periodic extension (`At` rejects negative ticks).
- **Update (2026-10-10, CAL):** `spec-calendar.md` 4.5.1 adds the periodic helper `Calendar.SeasonAt(tick)` (the season of `floorMod(tick, TicksPerYear)`), separate from `At`; `spec-viability.md` 8.5.2 uses it. The "season shorter than a delay" entry (9) does not arise for production.
### N-032 Welfare window against recipe duration

- **Areas:** STB, CRD
- **Raised:** 2026-10-10, `spec-production.md` 5.2.3 and 16.
- **Note:** A job consumes inputs at its start and sells output at its end. The welfare ratio sums income over a window `T` (default 30, `spec-welfare-and-wages.md` 2.3.1), so a role whose recipe lasts `T` ticks or more shows costs and no income and looks like it is starving while healthy. That could trigger relief, migration and bankruptcy. Either the window must exceed the longest job plus selling time (PRO warns, `job_longer_than_window`) or income must be attributed to the job it came from. Credit is the real remedy for long jobs.
- **Answer (proposed, 2026-10-10, by the coordinating agent):** `spec-welfare-and-wages.md` owns the window and the fix (it owns `T`), `spec-stability.md` records the loop. Not settled until WEL is revised.
- **Update (2026-10-10):** STB records it as STB-L19 and STB-L20 (`spec-stability.md` 4.2) with scenario S11 (a healthy producer with a job longer than the window is not flagged as starving). VIA 8.3.1 sets starting money to cover the longest job. The fix is still WEL's.
- **Update (2026-10-10, WEL):** decided in `spec-welfare-and-wages.md` 2.3.3: the window of a role is derived, `T = max(WelfareWindow, WindowCycles * longest Duration)`, `WindowCycles` default 2 (see N-062). Income attribution was rejected: it needs a valuation of work in progress or a rewrite of past records. The cost is a slower response of the ratio for long-job roles. Credit remains the real remedy (CRD); STB keeps scenario S11 (a healthy producer with a long job is not flagged as starving).
### N-035 Working tick with multi-tick jobs

- **Areas:** OBS
- **Raised:** 2026-10-10, `spec-production.md` 5.2.1.
- **Note:** `spec-welfare-and-wages.md` 4.3 defines a working tick by the `RecipeExecuted` event. With jobs, `RecipeExecuted` comes at completion and every tick of a job is a working tick, so the root reads working ticks from the agent's job state. Change the wording of 4.3 to "had a job in progress in the tick (started, continued or completed)". OBS: `RecipeExecuted` gains `StartTick`, `Executions`, outputs, `Disposed`, `EffectiveYield`; `RecipeAbandoned` is new. This is a change to the event table in `rules/observability-and-logging.md` 4.3, which needs a new read-receipt tag when made.
- **Update (2026-10-10, WEL):** applied for WEL in `spec-welfare-and-wages.md` 4.3: a working tick is a tick with a job in progress (started, continued or completed); the root reads it from the agent's job state. OBS: `RecipeExecuted` fields and `RecipeAbandoned` remain.
### N-036 Production's draws, phases and shocks in the world

- **Areas:** WLD, INT, RES
- **Raised:** 2026-10-10, `spec-production.md` 5.5, 6.5, 6.6.2, 6.8.
- **Note:** WLD: production runs before orders; continuations and completions before starts; starts rotate by tick modulo the region's agent count (count taken at the start of the phase); at agent creation one draw in [0, 10 000) for production's residues, after the other creation draws in an order WLD fixes. INT: shocks are set and cleared between ticks through the yield-shock slot (a multiplier by region and recipe, default 10 000). RES: provide `Endowment(region, resource)` and `TakeEndowment(region, resource, n)`, own the resource definitions, depletion and regeneration, and make depletion visible by event since the ledger does not see it.
- **Update (2026-10-10, VIA):** applied for VIA in `spec-viability.md` 8.5, which staggers initial jobs with a fixed rule and answers `spec-production.md` open question 2; WLD must apply the warm start before the agents' creation draws (8.6 item 5).
### N-038 Seed prices, newcomers and migrants

- **Areas:** MIG
- **Raised:** 2026-10-10, `spec-beliefs.md` 8.
- **Note:** A single seed price per region and commodity starts both the beliefs and the market's initial price (`spec-beliefs.md` 8.1, 8.4); VIA may supply a table. CLR: the price before the first trade is that seed price (`spec-clearing.md` 5.4). SPN: replacements and newborns start from the window mean or the market price (8.3). MIG decides whether a migrant keeps or reseeds its beliefs (default reseed).
- **Update (2026-10-10, VIA):** the seed table exists: `Config.WarmStart.Prices` with groups for money, stock and jobs (`spec-viability.md` 8.1, 8.2); newcomers use the window (8.8). It is a start, not a standing anchor.
- **Update (2026-10-10):** the CLR part is applied (`spec-clearing.md` 5.4: the price before the first trade is the seed price). The BEL part is applied (`spec-beliefs.md` 8.1, 8.4).
- **Update (2026-10-10, SPN):** newcomers use `spec-beliefs.md` 8.3, not the warm-start table; role changers keep common beliefs (`spec-spawner.md` 4.2, 7.4). MIG decides whether a migrant keeps or reseeds its beliefs (default reseed).
### N-039 A hard cost floor

- **Areas:** STB, GDS, PRO, SKL
- **Raised:** 2026-10-10, `spec-beliefs.md` 18.1.
- **Note:** `CostFloor` 10 000 (100% of unit cost) stops sales below cost, so perishables can spoil instead of selling cheaply and a market can freeze with buyers below the floor. The stability tests need a perishables-glut scenario. `spec-skill-barriers.md` 2.4 ("cost acts through exit") needs rewording, because cost now also acts directly on a seller's price.
- **Update (2026-10-10, STB):** `spec-stability.md` 6.2 run 5 gives numbers for the perishables-glut scenario (S07): sellers with cost 1 100, buyers capped at 1 000, shelf life 3 ticks. `CostFloor` 10 000: 0% of ticks with a trade, 99% of output spoiled; 9 000: 7% and 98%; 8 000: 97% and 69%; 5 000: 100% and 21%. The market freezes whenever the floor is above what buyers can pay (STB-L09, regime F5). BEL decides its open question 1.
- **Update (2026-10-10, BEL and STB):** `CostFloor` is now 7 000 on a cost that includes labour (`spec-beliefs.md` 3.2.2). `spec-stability.md` 6.2 run 5: 7 000 trades in every tick (45% spoiled with the model's mismatched flows); 10 000 and 9 000 still freeze, and a wage 10% to 25% above the model needs 8 000 or less. `spec-skill-barriers.md` 2.4 is reworded.
### N-040 Belief event, log keys and snapshot

- **Areas:** OBS
- **Raised:** 2026-10-10, `spec-beliefs.md` 12 and 18.9.
- **Note:** `BeliefsUpdated` (one aggregated event per market per tick, 12.1) and the log keys `region`, `bound` and `agents` (12.3) join the list of N-012. Whether `Snapshot` carries beliefs is open.

### N-042 The evidence rule may offset the price

- **Areas:** BEL, STB
- **Raised:** 2026-10-10, `spec-stability.md` 6.2.
- **Note:** In a stylised model of the rules of `spec-beliefs.md` (`spec-stability.md` 6.2), the evidence component of 5.4 settles the price 11% to 13% below the price at which supply equals demand and leaves about 12% of need unmet; with the evidence off, the stock component alone settles at 1 001 of 1 000 with nothing unmet. A trace shows that unfilled bids and asks mostly move their centre down, because the target (the other side's best unmatched quote) can lie on the wrong side of a centre that the limit was drawn away from. Restricting the move to the direction of the failure removes only part of it. This is a finding about a model of the rule as written, with one quantity-sizing rule and no money loop. BEL should reproduce it or revise 5.4 (compare the evidence to the limit, not the centre); FR-STB-018 (clearing residual) is the test.
- **Update (2026-10-10, BEL and STB):** reproduced and revised. `spec-beliefs.md` 5.4 compares the evidence with the order's limit, guarded, scaled by `BeliefEvidenceShare` (FR-BEL-036, 037). In the stylised model the offset falls from 11.3% to 5.7%, with 118 of 2 000 need unmet; `spec-stability.md` 6.2 run 2b reproduces this independently. Reduced, not removed: `spec-beliefs.md` open question 12 owns the rest (candidate remedy: a unit-count imbalance signal). Test: S01 with FR-STB-018.
- **Update (2026-10-10, DEM and STB):** a DEM schedule is a second quantity-sizing rule; run 2b of `spec-stability.md` 6.2 has not been repeated with it (`spec-demand.md` open question 5; register open question 6).
### N-043 Nothing anchors the price level if flows ignore price

- **Areas:** AGT, PRO, SPN, BEL, STB
- **Raised:** 2026-10-10, `spec-stability.md` 6.2 (runs 1 and 2).
- **Note:** With price-inelastic production and consumption, the belief rules have no interior attractor: an early inventory overhang ratchets the price to the floor of the reference band, a shortage to the ceiling, and a small change in a production-stop threshold decides which. A price-responsive flow must exist somewhere (a consumption rule that depends on price or money, a production choice by margin, an inventory capacity or production stop, exit through the spawner). The register proposes a production stop D24 and records the elasticity condition `es < ed` for cobwebs (6.3).
- **Answer (2026-10-10, AGT):** `spec-agents.md` 11.6 states that a staple's demand elasticity is 0 to 1, never above 1, so `e > 1` and `es < ed` fail for a staple; the responsive flows are buffer buying, recipe choice by margin, the production halt D24 (now specified with `ProduceHaltRatio`) and exit via the spawner. An elastic subsistence demand is not invented; if S04 or S16 shows a pinned market that the halt and exit do not cure, see N-052. Leave open for SPN (exit) and the scenarios.
- **Update (2026-10-10, STB):** `spec-stability.md` 6.2 run 1b: with AGT's target the default halt removes the floor pin and leaves a shortage cycle (price 1.5 to 3.1 times the reference, 12% of need unmet), so the halt bounds the glut but is not an anchor. Open for SPN (exit) and scenarios S04 and S16; see N-067.
- **Update (2026-10-10, DEM and SPN):** DEM supplies a price-responsive flow for non-staples (D30). SPN's entry and exit are slow, capped flows (`EntryShare` 50 bp, `ExitShare` 100) and are not an anchor for a pinned staple (`spec-spawner.md` 16). A pure staple is unchanged: halt, buffer buying, recipe choice, work floor. Scenarios S04 and S16 decide whether this is enough.
### N-045 Dampers that act silently

- **Areas:** OBS, STB
- **Raised:** 2026-10-10, `spec-stability.md` (FR-STB-012, FR-STB-013).
- **Note:** FR-STB-013 requires every clamp, refusal, floor or limit to be counted in an event or a bounded `Warn`. Counters exist for belief clamps and order rejections (`BeliefsUpdated`, `OrderRejected`). Missing: the cap or floor on days worked, entry limits, the production stop. FR-STB-012: the regime measures must be computable from events and the snapshot alone: price, volume and unmatched per market, at-floor and at-ceiling per market, weights on spawn and bankruptcy, boundary flows, pool balance, pending output.
- **Update (2026-10-10):** counters now exist for the work floor (Snapshot and bounded `Warn`, `spec-welfare-and-wages.md` 5.3, 11.3; `spec-agents.md` 7.1.3, FR-AGT-072) and for the traineeship queue and places (`spec-skill-barriers.md` 8.3). Still missing: the production halt (a per-agent `Debug` line only, N-067) and a spawner entry limit.
- **Update (2026-10-10):** counters now exist for the demand schedule and ladder (`WantShortfall`) and for the spawner's limits (`EntryLimited`). Still missing: the production halt (N-067).
### N-046 Register entries owed by unwritten specs

- **Areas:** POP, MIG, INF, EMP, MON, EXT, SAV, CRD, TRN, MER, STO, RSK, FIS, PLC, RES, LND, SUB, WLD
- **Raised:** 2026-10-10, `spec-stability.md` 4.7 and 9.
- **Note:** Each adds its entries by the checklist of `spec-stability.md` 9 and states its dampers by `STB-Dnn`. The expected entries are in section 4.7 (SPN: the failure ladder, entry limits, replacement by the most profitable role; POP: the Malthusian fixed point; and so on). A spec that adds a loop and no entry is not done (FR-STB-003). Undamped now: see `spec-stability.md` section 10.
- **Update (2026-10-10):** the DEM and SPN entries are merged into the register (L35, L36, L45, L46, L47, L50, L54 and L57 to L65).
### N-051 What the Strategy contract needs from other specs

- **Areas:** INT, CHK, WLD
- **Raised:** 2026-10-10, `spec-agents.md` 6, 10.
- **Note:** INT: `AddAgent`, `Direct` and the mailbox. CHK: the strategy names are hashed, the code is not (`spec-agents.md` open question 4), and the directive is state. WLD: the phase order of `spec-agents.md` 4, `Work` between continuations and starts, `Trade` after production, `Revise` per outcome with the world's draw.
- **Update (2026-10-10, SPN):** SPN needs from WLD the phase-7 position, regions in order and the creation draws after all `Decide` calls (`spec-spawner.md` 10.2); from INT the `Mortal` flag of `AddAgent` (N-079); from CHK the spawner name hashed and its state pair.
### N-056 What the ledger and world need for a warm start

- **Areas:** WLD
- **Raised:** 2026-10-10, `spec-viability.md` 8.6.
- **Note:** (1) `initial` for goods into any account at construction, including estate accounts; (2) lots with a negative `Born` and durable items with remaining durability below full at construction; (3) a construction bucket in the flow totals before tick 0, included in the audit; (4) a job created at construction with `Remaining < Duration` and pending output fixed from a virtual past start; (5) the warm start is applied before the agent creation draws (N-009, N-036).
- **Update (2026-10-10, LED):** items 1 to 3 are applied in `spec-ledger.md` 5.2.4, 7.4.1 and 9.1 (stated lot key at construction, signed `Born`, construction bucket in the audit). Items 4 and 5 stay with WLD (a job created at construction; the warm start before the creation draws); see N-074.
### N-061 The load-time size gate is a guess

- **Areas:** VIA, CFG
- **Raised:** 2026-10-10, `spec-config.md` 4.1.
- **Note:** `MaxViabilityCells` and `MaxViabilityPivots` are placeholders and the speed of the exact solver in Go is unmeasured. Measure in the slice so that NFR-CFG-001 holds with the programmes on.
- **Update (2026-10-10, REF):** `five` runs 202 pivots in the Python prototype; still unmeasured in Go.
### N-065 Silent dampers need counters

- **Areas:** OBS
- **Raised:** 2026-10-10, `spec-agents.md` 7.1.3, `spec-welfare-and-wages.md` 5.3.
- **Note:** The work floor is counted by the world (a pure function of the welfare record, so a consumer's strategy cannot hide it): a Snapshot field and a bounded `Warn` "work floor binding" (`spec-agents.md` FR-AGT-072). The traineeship queue has its own `Warn`. OBS names the Snapshot fields. The production halt and a spawner entry limit are still uncounted (N-067).

### N-067 The production halt is not an anchor and has no counter

- **Areas:** AGT, OBS, STB
- **Raised:** 2026-10-10, `spec-stability.md` 6.2 run 1b.
- **Note:** With inelastic flows the default halt (30 000 bp of a one-tick target) gives a shortage cycle, not a stable price, and it is counted only by a per-agent `Debug` line. AGT and OBS: an aggregate per region and role per tick in the style of `UpkeepShortfall`. Scenario S16 must set the threshold.
- **Update (2026-10-10):** `spec-reference-chain.md` 9 repeats the request for an aggregate counter; `spec-demand.md`'s handoff proposes the name `ProductionHalted` (Region, Role, Households, Recipes). OBS decides.
### N-071 One wage per region and skilled recipes

- **Areas:** VIA, SKL
- **Raised:** 2026-10-10, `spec-viability.md` open question 12.
- **Note:** One wage per region costs a skilled recipe at the common wage. Decide whether the solver reports a wage per recipe class.

### N-072 The construction bucket as an event

- **Areas:** OBS
- **Raised:** 2026-10-10, `spec-ledger.md` 7.4.1 and open question 9.
- **Note:** Decide whether the construction bucket is emitted as a `BoundaryFlow` with `Tick` -1 (recommended in the ledger).

### N-074 The construction phase ends at the first Step

- **Areas:** WLD
- **Raised:** 2026-10-10, `spec-ledger.md` 5.2.4.
- **Note:** A stated lot key (negative `Born`, a worn durable) is allowed only during construction; WLD ends the phase at the first `Step`. With N-056 items 4 and 5: a job created at construction, and the warm start before the creation draws.

### N-075 Mapping the ledger's sentinels

- **Areas:** CFG
- **Raised:** 2026-10-10, `spec-ledger.md` 5.2.4 and 8.2.1.
- **Note:** `ErrInvalidLot`, `ErrNotConstruction` and `ErrEmptyTransform` are ledger-internal; the root needs a mapping to public sentinels. Bounds for the estate fields are in `spec-estates.md` 3.
- **Update (2026-10-10):** the sentinels `ErrUnknownSpawner`, `ErrUnknownWant`, `ErrWantItems` and `ErrWantDurable` also need a root mapping.
### N-076 A carriage market is not a goods market

- **Areas:** CAR, MER, TRN, CLR, LED
- **Raised:** 2026-10-10, `spec-carriage.md` 4.2.
- **Note:** `spec-clearing.md` admits asks only for stock and settles by moving goods; hired carriage moves none and uses escrow. The contract, transport and merchant specs must pick: the carriage unit as a produced good (a trip is a job), or a contract-aware admission and settlement that keeps the guarantees of `spec-clearing.md` 2.3. The carriage unit's price may fall below the resolution floor (an ask of 24 against a floor of 50), so choose a coarser unit.

### N-078 What the demand module needs from the money and income specs

- **Areas:** MON, EXT, EMP, LND, FIS
- **Raised:** 2026-10-10, `spec-demand.md` 7.3.
- **Note:** A non-producing class needs a recurring named transfer of money and a paired sink or return flow. In the model a stipend of 300 a tick to 50 households with no sink left household money at 47 700 against 2 700 at the start.

### N-079 External agents are mortal only when the consumer says so

- **Areas:** INT
- **Raised:** 2026-10-10, `spec-agents.md` 10.6, `spec-spawner.md` 9.
- **Note:** `AddAgent` carries a `Mortal` flag (default false); the world refuses decisions on an external agent unless it is mortal.

### N-080 Wants in the steady-state programme use equal-price shares

- **Areas:** VIA, DEM
- **Raised:** 2026-10-10, `spec-viability.md` 3.2.
- **Note:** Wants enter the programme at the reference weights, fixed and unscaled by the surplus factor. Revisit with scenario S25 (a substitution flip changes the shares).

### N-081 Snapshot figures the scenarios need

- **Areas:** OBS, WEL, AGT
- **Raised:** 2026-10-10, `spec-reference-chain.md` 9.
- **Note:** (1) the mean welfare ratio per role and region (`spec-welfare-and-wages.md` 11.2, FR-WEL-036: specified); (2) unsold stock per commodity; (3) a production-halt counter (N-067).

### N-082 Relief and entry share the pool in time

- **Areas:** WLD, EST
- **Raised:** 2026-10-10, `spec-spawner.md` 7.3, `spec-estates.md` 2.6.
- **Note:** Relief is paid in phase 1 from what the previous tick's phase-7 entry step left, so the pool is shared in time. WLD must keep the order: phase 1 relief, phase 7 entry.

### N-083 Population adds the `deaths` reason and decides renewal against births

- **Areas:** POP, LED
- **Raised:** 2026-10-10, `spec-spawner.md` 4.1 and 8.2.
- **Note:** POP adds the ledger reason `deaths` and decides `Renewal` against births (warning `renewal_with_population`).

### N-084 Scenarios that wait for trade, money and the coarse tick

- **Areas:** MON, EXT, REG, TRN, MER, INF, STY
- **Raised:** 2026-10-10, `spec-reference-chain.md` 7.6.
- **Note:** S13 (the faucet overlay), the dynamic parts of X1 and X2 (the trade overlay) and the coarse-tick leg of S12 wait for these specs.

### N-085 Nominal table against the full-labour table

- **Areas:** VIA, STY
- **Raised:** 2026-10-10, `spec-viability.md` 5.3.4.
- **Note:** When the wage dual is 0 the warm start uses the nominal programme. Run both tables on `famine` and compare.

### N-086 Re-run the demand and spawner models under the register

- **Areas:** STB, STY
- **Raised:** 2026-10-10, `spec-stability.md` 6.5 and open question 6.
- **Note:** The numbers quoted in 6.5 are from the owners' models. STY or EXP should re-run the luxury model with DEM's schedule and the occupation model with SPN's discrete rules, and set the `entry_share_window` threshold from the run.

### N-087 Wants against the production halt

- **Areas:** AGT, OBS, STB
- **Raised:** 2026-10-10, `spec-stability.md` open question 8.
- **Note:** Does the halt cut the supply of want goods in a downturn and deepen the wage and demand loop (L57)? Run S24 with the halt on and off.

### N-088 Conflicts left by the rules edits

- **Areas:** OBS
- **Raised:** 2026-10-10, `rules2` report.
- **Note:** (1) `household.unfed` is an up-down counter fed by an event that is absent when everyone is fed: the adapter keeps per-region and role state and resets at the end of the tick; (2) the construction bucket as a `BoundaryFlow` with `Tick` -1 is undecided (N-072); (3) the snapshot-fed gauges (production work in progress, the beliefs' mean centre) have no name and conflict with 5.3.1 ("derivable from events"); (4) `agent.bankruptcies` counts every cause of `AgentBankrupted`, including `consumer` and `population`, though the name says bankruptcies; (5) whether the estate's `ReliefPaid` and `EstateOpened` belong to a module that can be off.

### N-089 Model scripts of the demand, spawner and reference-chain specs

- **Areas:** DEM, SPN, REF, CODE
- **Raised:** 2026-10-10, `spec-demand.md` 14, `spec-spawner.md` 16, `spec-reference-chain.md` 14.
- **Note:** The scripts behind these specs are still in `/tmp/economy-boss/` (`dem/`, `spn/`, `ref/`). Archive them under `doc/specs/models/` like the others, and update the specs' paths.

## Closed notes

When a note is closed, move it here and add **Closed:** with the date, the
resolution (applied or rejected, and why) and where it was settled.

### N-023 Calendar

- **Areas:** CAL, WEL
- **Raised:** 2026-10-09.
- **Note:** `spec-calendar.md` builds on `TicksPerYear` and does not redefine it
  (`spec-currency.md` 2.4). Per-tick quantities (upkeep, the reference basket) are
  not rescaled when the tick length changes. `ExpectedWorkTicks` is an expectation
  for calibration and the skill entry rule, and must be at most `TicksPerYear`
  (`spec-welfare-and-wages.md` 3.3, 4.2).
- **Closed:** 2026-10-10. Applied in part, rejected in part. Applied: per-tick
  quantities and durations are not rescaled (`spec-calendar.md` 6.1, 6.2) and
  `ExpectedWorkTicks` is at most `TicksPerYear` (6.3). Rejected: the first sentence.
  To make time configurable, the calendar defines `TicksPerYear` as a derived
  quantity (`spec-calendar.md` 3.2) and `spec-currency.md` 2.4 now says so, so it is
  no longer a separate world parameter.

### N-003 Non-tradable goods, disposal and the recipe format

- **Areas:** PRO, GDS, LED
- **Raised:** 2026-10-09, `spec-goods.md` open question 4 and `vision.md` goal 5.
- **Note:** Open for the production spec: a byproduct with no buyer, a waste
  product, whether a `Tradable` flag and a disposal sink are needed. Quality
  grades are separate commodities (`spec-goods.md` 7.1), so the recipe format must
  not assume a grade. Still undecided: variable yield, and conditions on a recipe
  (a season, a building), which touch `spec-calendar.md` and
  `spec-infrastructure.md`. Decide the recipe format against a second, unrelated
  setting, as `vision.md` section 5, item 3 requires.
- **Closed:** 2026-10-10. Applied in `spec-production.md` 7 and 3.2: no `Tradable` flag (rejected for version 1); an output may be `Discard` (created and disposed in one step, measured in flow totals); a `disposal` goods sink is needed in the ledger registry (N-033); byproducts are outputs with `CostShare` 0. Variable yield and conditions: deterministic yield factor by region and season, season read at job start; building conditions and random yield deferred with costs in 6.7. The recipe format assumes no grade and was checked against two unrelated settings (section 12). `spec-goods.md` open question 4 is answered by `spec-production.md` 7.5.

### N-004 What production takes on from other specs

- **Areas:** PRO
- **Raised:** 2026-10-09, collected while writing GDS, SCL and LED.
- **Note:** The production spec inherits these, already stated elsewhere:
  - it owns `Requires` on a recipe and `wear_per_use`; GDS owns `Durability`
    (`spec-goods.md` 5.3), and wear falls on the most worn item first
    (`spec-ledger.md` 5.4);
  - a recipe execution is one all-or-nothing ledger transform; a recipe with no
    inputs is a source (`spec-ledger.md` 8.1, 8.2; `spec-goods.md` 6.2);
  - an agent of weight `w` runs up to `w` executions, and `k` if inputs cover
    only `k` (`spec-scale.md` 4.2, FR-SCL-007, FR-SCL-008);
  - a durable no recipe requires must warn (FR-GDS-023);
  - which recipes are trainee-eligible and how large their output is sets
    whether training pays (`specs/README.md`, "Still open" 1).
- **Closed:** 2026-10-10. `Requires` and `WearPerUse` are owned in `spec-production.md` 4.4 (wear on distinct items, most worn first; it asks the ledger for `WearItems`, N-033); one all-or-nothing transform and source recipes in 4.3 and 3.1.2; weight `w` runs up to `w`, `k` if inputs cover `k`, in 4.2; the durable-not-required warning is FR-PRO-040; the trainee-eligible lever is in section 8. The chain's actual values stay open (`specs/README.md`, "Still open" 1).

### N-005 How chance events resolve for a cohort

- **Areas:** PRO, SKL
- **Raised:** 2026-10-09, `spec-scale.md` open question 3.
- **Note:** A per-household chance event for an agent of weight `w` must have the
  right expected count with integer arithmetic and the world's random source
  (`spec-scale.md` 4.6, FR-SCL-012). The recommended method is a carried
  remainder with a random starting offset. Tool breakage is no longer random (it
  is a durability counter, `spec-skill-barriers.md` 2.1), so this may only matter
  for other events. Decide whether the method is used at every weight or only
  above 1.
- **Closed:** 2026-10-10. Settled for production and skill barriers in `spec-production.md` 6.6: yield fractions use a deterministic carried remainder at every weight (including 1), started from one seeded draw per agent at creation; production draws no chance trials; tool breakage is a counter (`spec-skill-barriers.md` 2.1). The belief gate is a decision with one draw per outcome whatever the weight, not a per-household chance event (`spec-beliefs.md` 9.3). A future spec with a true per-household chance uses the helper of `spec-scale.md` 4.6. This also settles `spec-scale.md` open question 3 for production.

### N-006 What price beliefs must do with clearing's outputs

- **Areas:** BEL
- **Raised:** 2026-10-09, `spec-clearing.md` sections 5.4, 5.5, 7.3 and 13.
- **Note:** `spec-beliefs.md` must:
  - use the **price age** and the **best unmatched quotes** to move off a stale
    price, since clearing publishes a carried-forward price when nothing trades;
  - own the **belief step** that sets the resolution floor
    (`spec-currency.md` 3.1; named in `spec-config.md` 7.4), bound its steps, and
    adjust asynchronously;
  - fix the bazaarBot defects assigned to it (`bazaarbot-reference.md` section 11):
    7 (the lookback window), 9 (a threshold that depends on the price scale; use
    basis points of the mean), 10 (the range can invert; define the invariant) and
    11 (update per order outcome, once, not per match; `spec-clearing.md` 7.3).
  - the Doran and Parberry paper has not been read (N-015).
- **Closed:** 2026-10-10. Applied in `spec-beliefs.md`. Price age and the best unmatched quotes move a belief off a stale price (5.4, FR-BEL-011, FR-BEL-012, FR-BEL-034). `BeliefStep` is the belief step and sets the resolution floor `ceil(10 000 / BeliefStep)` (5.3, 11, FR-BEL-029); steps are bounded to `2 * BeliefStep` and asynchronous (5.2, 5.6). Defect 7: a per-market window, no per-agent history (7). Defect 9: `BeliefSignificance` in basis points of the centre (5.4). Defect 10: centre and half-width, invariant in 3.3. Defect 11: one update per admitted order (5.1.2). The Doran and Parberry paper is now read (journal version, `specs/evidence.md`); it uses no window for an agent's own range and its pseudocode has no crossing check.

### N-002 Does an agent have an inventory capacity?

- **Areas:** AGT, LED
- **Raised:** 2026-10-09, writing `spec-goods.md` (open question 3).
- **Note:** bazaarBot bounded an agent's whole inventory (`max_size`) and made it
  discard goods when full (`bazaarbot-reference.md` sections 3 and 10). GDS gives
  `Mass` and `Bulk`; whether an agent has a capacity in version 1 is undecided.
  If it does, a discard needs a goods sink reason in the ledger registry
  (`spec-ledger.md` 7.2) and an event, and the choice of what to discard is a
  strategy decision, not a rule.
- **Learned (2026-10-10, `spec-production.md` 5.3.2):** production never refuses an output for lack of room; overflow beyond `MaxQuantity` is disposed with reason `disposal`. If AGT adds an inventory capacity, a full inventory is the strategy's problem on the next tick, not a reason to lose output at completion. `disposal` is available to AGT for discards.
- **Closed:** 2026-10-10. Applied in `spec-agents.md` 8: no inventory capacity in version 1 (bazaarBot's `max_size` is the source of defects 6 and 8, storage is later, production never refuses output, overstock is limited by spoilage, the buffer cap, the production halt and the belief stock rule). A discard is a strategy decision executed as a move to the `disposal` boundary, so it is counted in `BoundaryFlow`; the default never discards. `spec-ledger.md` needs nothing beyond `disposal` (N-033).

### N-007 What the strategy takes on

- **Areas:** AGT
- **Raised:** 2026-10-09, `spec-clearing.md` section 3 and `spec-scale.md` section 4.
- **Note:** Order sizing is the strategy's job: the market rejects an unaffordable
  order whole and never clamps it (`spec-clearing.md` 3.2). One order per
  commodity per tick (3.1.1). A hard liquidity constraint, and a consumption rule
  that saves when wealth is low (roadmap, AGT row). Define the tradable set
  explicitly and a missing ideal stock as 0 (`bazaarbot-reference.md` item 5).
  The default strategy receives the world's random source and may not keep it
  (architecture.md 7.7). Quantities of an agent of weight `w` are `w` times the
  household amounts, and a limit price is never scaled (`spec-scale.md` 4.4).
- **Closed:** 2026-10-10. Applied in `spec-agents.md`: order sizing by the strategy with a budget that never gets an order rejected (7.4, FR-AGT-044); one order per commodity per tick (7.3.3); the hard liquidity constraint and the saving rule (7.4); the tradable set explicit (3.4) and a missing ideal as target 0 (7.3.1); the strategy gets the world's random source and may not keep it (6.1.2); quantities scale by the weight and a limit price never (9.1, 9.2).

### N-014 Stability register entries to collect

- **Areas:** STB
- **Raised:** 2026-10-09; each spec left a section for it.
- **Note:** Entries are waiting in `spec-clearing.md` section 13,
  `spec-scale.md` section 10, `spec-config.md` section 12, `spec-goods.md`
  section 10 and `spec-ledger.md` section 14. The roadmap's first entries are the
  target-income labour loop, the occupation-choice cobweb, and the delays in
  production, travel and traineeship (roadmap, STB row). Also record that a cohort
  multiplies the lockstep of the target-income rule (`spec-scale.md` 5.1) and that
  whole-lot spoilage makes a sawtooth (`spec-goods.md` 4.5).
- **Entries ready (2026-10-10):** `spec-beliefs.md` section 17 has nine (price/belief/limit loop, stale price loop, cobweb, herding and lockstep including the cohort gate, cost-push ratchet, shortage with no supply, width collapse, hard floor with perishables, reference band). `spec-production.md` section 16 has its own. The register itself is not written.
- **Closed:** 2026-10-10. Applied in `spec-stability.md`: all entries waiting in `spec-clearing.md` 13, `spec-scale.md` 10, `spec-config.md` 12, `spec-goods.md` 10, `spec-ledger.md` 14, `spec-beliefs.md` 17 (nine), `spec-production.md` 16 and `spec-calendar.md` 9 are in section 4, merged where two specs named one loop; loops the other specs leave unlabelled (WEL, SKL, EST, CAR) are added; the cohort lockstep of the target-income rule (L34) and the whole-lot spoilage sawtooth (L25) are recorded. The roadmap's first entries are L14 (target-income), L36 (occupation cobweb) and L21 (production, travel and traineeship delays). Future specs add entries by the checklist of section 9 (FR-STB-002, FR-STB-003). The thirteen entries of `spec-agents.md` 18 (AGT-A1 to A13) still have to be merged into `spec-stability.md` 4.7 (see N-059).

### N-037 What the strategy must supply to beliefs

- **Areas:** AGT, PRO
- **Raised:** 2026-10-10, `spec-beliefs.md` 4.4 and 10.
- **Note:** Beliefs need three inputs per order from the strategy or production: `Stock`, `Target` (units the agent wants to hold, from its recent demand over a cover period; 0 means none) and `UnitCost` (money per unit, 0 unknown). AGT must define `Target` and the cover period. PRO supplies `UnitCost` per (region, commodity) through `internal/production.UnitCost` (`spec-production.md` 9.4: input cost of one unit, rounded up, the minimum over recipes that can run there, byproducts and labour excluded) and the root hands beliefs a plain `int64`; the two specs agree on this. AGT creates a belief for every commodity in the tradable set at creation (`spec-beliefs.md` 3.1), draws limit prices with `beliefs.Draw` or its own rule, and uses the market price window (7) for quantity favourability. Watch: "a missing ideal stock as 0" (N-007) now has two readings, a zero ideal quantity for sizing and `Target = 0` meaning no stock signal for beliefs.
- **Closed:** 2026-10-10. Applied in `spec-agents.md` 7.3: `Target` is the units to hold after this tick's upkeep, `U * CoverTicks` plus the next job's inputs for a commodity used, `ceil(Make * OutputCoverTicks)` for one only made, 0 otherwise; cover periods `CoverTicks`, `InputJobs`, `OutputCoverTicks`; the `Stock` passed to `Revise` is stock after the trades less this tick's upkeep reserve (6.4.4); `UnitCost` is passed only for a commodity the role can make (7.3.6); beliefs for the whole tradable set at creation (3.4.1). The two readings of a missing ideal are one rule (7.3.1).

### N-033 Ledger changes production needs

- **Areas:** LED
- **Raised:** 2026-10-10, `spec-production.md` 4.4.2, 7.3, 15.
- **Note:** (1) add reason `disposal` (goods sink, agent to boundary, owner PRO); (2) allow `Transform` with an empty input list or an empty output list, not both (start step = inputs only, completion = outputs only); (3) add `WearItems(account, commodity, n, units)` that wears the `n` most worn distinct items by `units` each, destroying those that reach 0, with the same semantics as `Wear` otherwise (needed for a cohort of `k` executions); (4) the invariant "goods conserved except by recipes" holds with work in progress: pending output is in no account, so the snapshot reports it.
- **Answer (proposed, 2026-10-10, by the coordinating agent):** accept all four. Not yet applied to `spec-ledger.md`; do it in the next change that touches that spec.
- **Closed:** 2026-10-10. Applied in `spec-ledger.md`: the `disposal` reason and the goods form of `initial` (7.2), a `Transform` with an empty side (8.2.1), `WearItems` (5.4.3) and work in progress (8.2.2); FR-LED-037 to 049.

### N-053 Ledger query for expiring lots

- **Areas:** LED
- **Raised:** 2026-10-10, `spec-agents.md` 7.3.5.
- **Note:** The default `Trade` offers units that spoil at the end of the tick. It needs a read-only query `Expiring(account, commodity, tick)`; `spec-ledger.md` 5.3.2 keeps an earliest-expiry hint per account but no per-commodity query.
- **Closed:** 2026-10-10. Applied in `spec-ledger.md` 5.3.3: the read-only `Expiring(account, commodity, tick)`.

### N-050 Spending classification for the welfare income

- **Areas:** WEL, AGT
- **Raised:** 2026-10-10, `spec-agents.md` FR-AGT-057.
- **Note:** `spec-welfare-and-wages.md` 2.1 nets inputs out of income. AGT's `Intent.Purpose` (`Upkeep`, `Input`, `Tool`, `Other`) classifies each bid; the world totals money received and money spent by purpose per agent per tick. A commodity that is both an input and upkeep is classed as an input. WEL decides what is netted (and whether tools are capex or recipe cost) and amends its 4.3 per N-035.
- **Closed:** 2026-10-10. Applied in `spec-welfare-and-wages.md` 2.1: `income_t` nets money spent on bids with the purpose `Input` and the wear cost of tools; upkeep, tool purchases and other spending are not netted; a tool is capital whose use is netted as wear (2.1.3); relief and in-kind count as income, a starting-money grant does not; a commodity that is both input and upkeep is classed as input and the ratio is understated by at most the upkeep bought of it (2.1.5, conservative).

### N-055 The belief cost ceiling against the equilibrium price

- **Areas:** BEL, PRO, STB
- **Raised:** 2026-10-10, `spec-viability.md` 8.2.3 and 15.
- **Note:** `UnitCost` (`spec-production.md` 9.1) excludes labour. At zero-profit prices the seed is 3.7, 9.7, 8.4 and 19 times unit cost for flour, dough, loaf and a tool in the prototype, against `CostCeiling` of 3 times (`spec-beliefs.md` 3.2.2). BEL clamps a belief to the ceiling, so sellers could not price at the wage and the seed would be clamped at creation. Options: count recipe labour at the wage in `UnitCost`; raise `CostCeiling`; drop the ceiling where the labour share is high. The solver reports `SeedAboveCostCeiling` meanwhile.
- **Answer (proposed, 2026-10-10, by the coordinating agent):** count labour in `UnitCost` at the observed wage (`spec-welfare-and-wages.md` already defines an observed wage), measured from the previous period so there is no circularity. Reason: Lengnick's price bounds are multiples of marginal cost, and his marginal cost includes wages; the ceiling then stays near the equilibrium price instead of far below it. Cost: with labour counted, `CostFloor` 10 000 binds at the zero-profit price, which makes the freeze of N-039 more likely, so the floor must be re-examined together. Not applied: it changes `spec-production.md` 9 and `spec-beliefs.md` 3.2.2, and needs a revision pass of both specs.
- **Closed:** 2026-10-10. Resolved in design. `spec-production.md` 9 counts labour at the pooled observed wage of the previous tick (FR-PRO-035 to 037, 047 to 049); `spec-beliefs.md` 3.2.2 sets `CostFloor` 7 000 and `CostCeiling` 30 000 on the full cost, because a 100% floor freezes markets with perishables (`spec-stability.md` 6.2 run 5); `spec-viability.md` 8.2.3 and 15 re-ran the seed check against the full unit cost: ratio 1.000, minimum 0.991 (whole-unit rounding). A starting wage per region closes the first-ticks gap (`WarmStart.Wages`, `spec-viability.md` 8.2.5). Residual: the wage loop has gain `CostFloor / 10 000` times the labour share (STB-L10).

### N-057 The worked examples of `spec-production.md` 12 are not viable as written

- **Areas:** PRO, REF
- **Raised:** 2026-10-10, `spec-viability.md` 15.
- **Note:** Example A (6 grain per 120-tick job) needs 73 times the available labour; example B needs 12 times. They are packs with placeholder numbers, but FR-PRO-046 says they must "run to a stable state". Rescale them before REF or any test uses them. A set that passes (with invented completing recipes) is in `spec-viability.md` 15. Rule of thumb from the runs: the output per labour tick of each step of the chain must exceed the household's need by the depth of the chain.
- **Closed:** 2026-10-10. Applied in `spec-production.md` 12: both examples rescaled to the viable set of `spec-viability.md` 15 (invented completing recipes labelled as such); `spec-viability.md` confirms both pass check V3.

### N-059 Merge the AGT entries into the stability register

- **Areas:** STB, AGT
- **Raised:** 2026-10-10, `spec-agents.md` 18 and `spec-stability.md` 4.7.
- **Note:** The thirteen entries `AGT-A1` to `AGT-A13` of `spec-agents.md` 18 were written against the checklist of `spec-stability.md` 9 while that spec was being written. Merge them into `spec-stability.md` 4.7 and remove the AGT hook row. D24 and D09 become specified (`spec-agents.md` 7.1.5 and 7.1.2); D26 (survival rule, not a damper of L14), D27 (budget stack and saving rule) and D28 (proportional work share) are new; scenarios S19 and S20 are new. D13 (`MinWorkTicks`) is not adopted; WEL decides.
- **Closed:** 2026-10-10. Applied in `spec-stability.md` 4.7: the thirteen AGT entries merged (L44 to L51; five merged into existing loops), D24 and D09 recorded as specified, D26 to D28 added, S19 and S20 added; D13 is adopted by WEL as `MinWorkShare` and D29 is the derived window.

### N-044 The stock signal alone is an integrator

- **Areas:** BEL, STB
- **Raised:** 2026-10-10, `spec-stability.md` 6.1.
- **Note:** A price rule driven only by the stock ratio gives a sustained oscillation of 6% to 29% of the price in the model; the amplitude scales with `BeliefStep`. The evidence rule is the proportional term that stops this, and the combination is stable in the model, so keep both and re-test when the evidence rule is revised (N-042). Test: S01, S04.
- **Closed:** 2026-10-10. Applied in `spec-beliefs.md` 5.5: both signals are needed. Re-tested with the revised evidence rule in `spec-stability.md` 6.2 run 2b (stock-only recovers in 98 ticks, combined in 30).

### N-049 Order reasons the strategy contract adds

- **Areas:** CLR, PRO
- **Raised:** 2026-10-10, `spec-agents.md` 6.5.
- **Note:** `not_tradable` (an intent outside the agent's tradable set) joins the table of `spec-clearing.md` 3.2; `not_on_role` (a job request off the role's list) joins the refusal reasons of `spec-production.md` 4.2, with a `Debug` log. Both are constants in `internal/logkeys`.
- **Closed:** 2026-10-10. Applied: `not_tradable` in `spec-clearing.md` 3.2 (FR-CLR-030, checked after `unknown_commodity`, pseudo-agents exempt) and `not_on_role` in `spec-production.md` 4.2 and 14.2 (FR-PRO-008, FR-PRO-048, checked first).

### N-048 The tradable set gives "tradable" a meaning

- **Areas:** GDS, CUR, CFG
- **Raised:** 2026-10-10, `spec-agents.md` 3.4.
- **Note:** `spec-goods.md` 6.4 and `spec-currency.md` 3.1 speak of the cheapest "tradable" commodity without defining it. `spec-agents.md` 3.4 defines it: a commodity in the tradable set of at least one role. Edit both specs' wording and FR-GDS-002.
- **Closed:** 2026-10-10. Applied: "tradable" is defined by `spec-agents.md` 3.4 and used in `spec-goods.md` 6.4 and FR-GDS-002, `spec-currency.md` 3.1 and `spec-config.md` 7.4.1.

### N-041 Edits to other specs owed by PRO and BEL

- **Areas:** LED, CFG, CLR, CUR, GDS, SKL, WEL, SCL, CAL, AGT, WLD, EST, PRO, BEL, STB, VIA
- **Raised:** 2026-10-10, handoffs of the workers that wrote `spec-production.md` and `spec-beliefs.md`.
- **Note:** The two new specs need changes in specs that already exist. None is made yet; make each in the next change that touches that spec, and close this note when all are done.
  - `spec-ledger.md`: 7.2 add the `disposal` reason; 8.1 allow an empty side on `Transform`; 5.4 add `WearItems` (N-033).
  - `spec-welfare-and-wages.md`: 4.3 working tick = job in progress (N-035); a note on the window against job length (N-032).
  - `spec-skill-barriers.md`: 2.1 and 4.2 refer to `spec-production.md` 4.4 and 8 (trainee-eligible recipes have no tool; trainee output = `TraineeYield`; breakage is not random); 2.4 reword (N-039).
  - `spec-config.md`: 4.1 bounds `MaxRecipeTicks`, `MaxYieldBP`, `MaxRequires`, `MaxDraws` (provisional); 4.2 the recipe checks of `spec-production.md` 10.3; 3.1 a `Yields` section; 7.3 and 7.4 the sentinels `ErrCostShares`, `ErrToolNotDurable`, `ErrEligibleHasTool`, `ErrUnknownSeason` and the PRO rules, and the 15 BEL parameters and rules (FR-BEL-027 to 029); 7.5 a `Sited` source with `Draws` counts as sourced; 6.2 the example keys; 7.4 name `BeliefStep`.
  - `spec-clearing.md`: 5.4 the seed price; 7.3 and FR-CLR-025 the outcome goes to beliefs once per admitted order; 5.4 or 7.1 who keeps the price window; open question 1 is also true of the paper's pseudocode; 9.2 cross-reference to the BEL `Warn`.
  - `spec-currency.md`: 3.1 name `BeliefStep` (floor `ceil(10 000 / BeliefStep)`; the default 200 gives 50).
  - `spec-goods.md`: 6.3 the effective reference price is also the default seed price and the centre of the reference band; open question 4 is answered.
  - `spec-scale.md`: open question 3 is settled for production.
  - `spec-calendar.md`: 9 the "season shorter than a delay" entry does not arise for production.
  - Future specs: AGT (N-034, N-037), WLD (N-009, N-036), OBS (N-012, N-035, N-040), STB (N-014), VIA (N-036, N-038).
  - `rules/observability-and-logging.md` 4.3: `RecipeExecuted` at completion with new fields, `RecipeAbandoned`, and later `BeliefsUpdated`. A rules change: it needs a new read-receipt tag and is not made here.
- **Added by round 2 (2026-10-10).** Further edits owed, none yet made:
  - `spec-clearing.md` 3.2, 9: the rejection reason `not_tradable` (N-049); 7.3 and FR-CLR-025: the outcome goes to `Revise` with the market report once per admitted order.
  - `spec-production.md` 3.3, 4.1, 4.2, 14.2: the strategy names the recipe through `JobRequest`; refusal reason `not_on_role`; `Role.Recipes` is owned by `spec-agents.md` 3.3; 12 and 18: rescale the worked examples (N-057), question 2 is answered by `spec-viability.md` 8.5; 9: the decision on labour in `UnitCost` (N-055).
  - `spec-beliefs.md` 3.2.2 (cost ceiling, N-055), 4.4, 10.1, 10.5 (`Target` and `Stock` are defined in `spec-agents.md` 7.3 and 6.4.4; the method that receives the outcome is `Revise`), 5.4 (the evidence move, N-042), 5.5 and 17 (the stock rule alone is an integrator, N-044; section 17 can point to STB-L01, L03, L21, L13, L10, L05, L07, L09, L08), 8.4 (the table is `Config.WarmStart.Prices`).
  - `spec-ledger.md` 7.2, 5.2.3, 7.4, 8.3: the goods form of `initial` at construction, lots with negative `Born`, a construction bucket in the flow totals (N-056); the query `Expiring` (N-053).
  - `spec-estates.md` 2.3, 3: `StartMoney` is `Role.StartMoney`; a later agent receives no goods; removal mechanics are `spec-agents.md` 3.6; pool seeding by `WarmStart.Pools` (`spec-viability.md` 8.3.3).
  - `spec-scale.md` 4.1, 4.4, 4.8, 7.2: cross-references to `spec-agents.md` 3.7, 5, 9; the scale-consistency check can use a warm start.
  - `spec-welfare-and-wages.md` 2.1, 4.3, 5, 5.1, 7: `cost_t` is the effective upkeep of `spec-agents.md` 5.2; income netting by `Purpose` (N-050); 5.1's "left to the strategy" is answered by `spec-agents.md` 7.1.2; WEL decides D13; the role economics check fails in a closed upkeep-only economy (N-054); note that labour supply is unit-elastic (`spec-stability.md` 6.1).
  - `spec-skill-barriers.md` 4.2: a trainee's upkeep is consumed from the trainer's account (`spec-agents.md` 5.7).
  - `spec-currency.md` 5.1, 5.3: the effective upkeep and `BasketTolerance`. `spec-goods.md` 6.4, FR-GDS-002, FR-GDS-003: "tradable" is defined by `spec-agents.md` 3.4 (N-048).
  - `spec-config.md` 3.1, 4.1, 4.2, 6.2, 7.3 to 7.6, 8, 9.2: the `WarmStart` section and its hash, bounds `MaxViabilityPivots`, `MaxViabilityCells`, `MaxLanes`, `MaxUpkeepItems`, the sentinels `ErrNotViable`, `ErrWarmStartPrice`, `ErrWarmStartJob`, `ErrUnknownStrategy`, `ErrBasketMismatch`, the viability rules and warnings, FR-CFG-033 satisfied by FR-VIA-004 to 008 and 011, the cycle-gain check (FR-STB-017).
  - `spec-calendar.md` 4.5, 9: state the periodic extension used by `spec-viability.md` 8.5.2 (optional).
  - The sources' own register sections (`spec-clearing.md` 13, `spec-scale.md` 10, `spec-config.md` 12, `spec-goods.md` 10, `spec-ledger.md` 14, `spec-calendar.md` 9, `spec-production.md` 16): a line "merged into `spec-stability.md` section 4" at the next touch.
  - `rules/architecture.md` 3.5 (allow `math/big` in `internal/steady`, N-058) and 4.3 (the `Strategy` row could say "Work, Trade and Revise") and `rules/observability-and-logging.md` 4.3 (`UpkeepShortfall`, `AgentSpawned` fields): rules changes, each needs a fresh read-receipt tag; not made.
- **Closed:** 2026-10-10. All owed edits were made in round 3: ledger, estates, scale and calendar; clearing, goods, currency and carriage; welfare and wages and skill barriers; config; production and beliefs; stability; and the agents and viability sweeps. Rules changes remain and are N-077.

### N-060 A starting wage for the cost anchor

- **Areas:** VIA, PRO, AGT, WEL
- **Raised:** 2026-10-10, `spec-production.md` 9.3 and 18.11, `spec-agents.md` 19.
- **Note:** The cost anchor is 0 while a region has no observed wage (`spec-production.md` 9.1.1), so beliefs have only the reference band in the first ticks.
- **Closed:** 2026-10-10. `spec-viability.md` 8.2.5 adds `WarmStart.Wages` (a starting wage per region from the labour-row dual, used only until the first working household-tick is observed; FR-VIA-049 to 052); `spec-welfare-and-wages.md` 4.3 exposes the observed wage per working household-tick.

### N-062 The derived welfare window

- **Areas:** WEL, CFG, PRO, STB
- **Raised:** 2026-10-10, `spec-welfare-and-wages.md` 2.3.3.
- **Note:** The effective window is `T = max(WelfareWindow, WindowCycles * longest Duration)`. CFG needs `WindowCycles`, `MaxWelfareWindow` and the effective window in the overflow check; PRO's `job_longer_than_window` becomes obsolete; STB's L19 damper is the window.
- **Closed:** 2026-10-10. Applied in `spec-config.md` (3.7, 4.1, 4.2, 7.4; FR-CFG-050 to 054), `spec-production.md` 10.4 and FR-PRO-040 (the warning is replaced by the informational `window_extended`) and `spec-stability.md` (D29).

### N-063 Estate ask anchor

- **Areas:** EST, SKL
- **Raised:** 2026-10-10, `spec-skill-barriers.md` 2.7.
- **Note:** `spec-skill-barriers.md` 2.7 anchored the estate's ask to the region's recent average price while `spec-estates.md` 2.7.3 anchors it to the effective reference price.
- **Closed:** 2026-10-10. Decided by the coordinating agent: the effective reference price, times remaining over full durability for a durable, less the per-tick markdown. It is deterministic and does not couple the estate to belief state. A market-anchored rule (the price-window mean, then the cost floor, then the reference price) is an experiment for later. Recorded in `specs/README.md` decisions.

### N-066 The cycle-gain warning formula

- **Areas:** CFG, STB
- **Raised:** 2026-10-10, `spec-stability.md` 4.1 (L10).
- **Note:** `spec-config.md` 7.6 item 3 computed the cost-loop gain from inputs only, but labour is now in the unit cost.
- **Closed:** 2026-10-10. Restated in `spec-config.md` 7.6 (FR-CFG-044): the material loop gain is `CostFloor / 10 000` times the full-cost share of the cycle's own inputs, which at the solver's prices is `q_in / q_out`; the wage loop is flagged by `cost_floor_at_full_cost`. `spec-stability.md` FR-STB-017 follows.

### N-069 Warning rule names

- **Areas:** CFG, CAL, SKL
- **Raised:** 2026-10-10, `spec-config.md` 7.4.2.
- **Note:** Three warnings were stated by other specs without a name: `inexact_hour`, `season_shorter_than_tick` (`spec-calendar.md` 7.3) and `tool_unaffordable` (`spec-skill-barriers.md` 2.3).
- **Closed:** 2026-10-10. Adopted in `spec-calendar.md` 7.3 and `spec-skill-barriers.md` 2.3 under the names given by `spec-config.md` 7.4.2.

### N-001 Households are the consumers; avoid the word

- **Areas:** DEM
- **Raised:** 2026-10-09, in a question about simulating consumers.
- **Note:** There is no separate consumer population. Every agent is a household
  that earns, pays upkeep and buys wants (`spec-currency.md` 5.5; overview).
  `spec-agents.md` owns upkeep orders and the consumption rule. `spec-demand.md`
  (version 1) must decide whether it needs households that do not produce
  (people on unearned income) or whether classes are only roles. The wage and
  demand loop (roadmap, "Macro is measured, never set") depends on this.
- **Watch for:** "consumer" in these docs means the *library consumer* (a game or
  server). Call the household a household or the demand side.
- **Update (2026-10-10, AGT):** a class of non-producing households is a role with an empty `Recipes` list (`spec-agents.md` 3.3.1). Upkeep, the consumption rule and the saving rule are in `spec-agents.md` 5 and 7. Spending beyond upkeep does not exist in version 1 (11.7); DEM must supply it (see N-054).
- **Closed:** 2026-10-10. Applied: a class of non-producing households is a role with an empty `Recipes` list (`spec-agents.md` 3.3.1, `spec-demand.md` 3.3); its income is a transfer named by another spec (`spec-demand.md` 7.3, warning `class_without_income`).

### N-052 A price-responsive flow for a staple

- **Areas:** STB, DEM, STY
- **Raised:** 2026-10-10, `spec-agents.md` 11.6.
- **Note:** A staple's demand elasticity is 0 to 1. If S04 or S16 shows a pinned market that the halt and exit do not cure, add the rationing of tier 1 above a dear price (11.6.1) or an elastic demand in DEM.
- **Closed:** 2026-10-10. Rejected as framed (`spec-demand.md` 17): tier-1 rationing makes hunger a choice and is an unsupported guess. DEM supplies elastic demand for discretionary goods, including an elastic increment on a staple listed as a want item. Reopen condition: S04 or S16 shows a pinned staple market that the halt, exit and the work floor do not cure; then raise a new note for AGT (upkeep alternatives).

### N-054 The closed economy fixes the mean welfare ratio at 1

- **Areas:** WEL, DEM, STB, MON
- **Raised:** 2026-10-10, `spec-viability.md` 4.7.
- **Note:** With no money faucet or sink and upkeep as the only purchase, household money is conserved, so total income equals total upkeep spending and the household-weighted mean welfare ratio is 1 whatever `TargetRatio` is. The default 15 000 cannot be met by all roles; the target-income rule never rests; the surplus of the steady state is produced and unsold. The role economics check of `spec-welfare-and-wages.md` 7 fails for every role in such a world. DEM, MON or EXT must supply the spending or money that lets a ratio above 1 exist. Same loop as N-028 from the other side. STB: record it, with a test that runs the default target with no DEM and asserts a bounded glut.
- **Answer (2026-10-10, AGT):** version 1 has no spending beyond upkeep; the glut is accepted and bounded by the production halt (`spec-agents.md` 11.7, FR-AGT-070). Open for WEL: whether the default `TargetRatio` stays 15 000 in a world without DEM, MON or EXT.
- **Update (2026-10-10):** WEL made the role-economics check a report, never an error (`spec-welfare-and-wages.md` 7.1). Decided by the coordinating agent: the default `TargetRatio` stays 15 000. In a world whose only demand is upkeep the mean ratio is 1, agents work at the floor-bounded maximum and the glut is bounded by the production halt; lowering the default would hide the missing demand, money-supply and external-sector specs. Revisit when DEM, MON or EXT exist. STB records it as STB-L46 with scenario S19.
- **Closed:** 2026-10-10. Answered by `spec-demand.md` 7: with wants the closed economy's mean welfare ratio is `1 + W/U` (model: 1.53 at target 15 000, against exactly 1.000 with upkeep only), up to `1 + Wmax/U`; no money faucet is needed; the money needed is the reserve plus about `e*` per household. The default `TargetRatio` stays 15 000 with the module on or off (`spec-welfare-and-wages.md` 7.2); the warning `target_above_demand` compares a pack's target with its own ceiling. STB-L46 stays for a pack without the demand module.

### N-047 What the reference chain must contain

- **Areas:** REF, STY, VIA, STB
- **Raised:** 2026-10-10, `spec-stability.md` 8.
- **Note:** To build scenarios S04 to S20 the chain needs a self-employed staple role with demand elasticity below 1, a perishable, a recipe cycle (for the cost-push gain), a long recipe or trip (the cobweb), a durable with a replacement cycle, a role with a job longer than the welfare window, and a region that can lose its income (pool, relief, failure ladder). STY sets the bands and the regime thresholds; VIA staggers the initial jobs (S18).
- **Update (2026-10-10):** scenarios now run to S22 (S19 closed economy, S20 reserve, S21 seeds against the cost band, S22 warm against cold start). The chain also needs a world with upkeep as the only demand, a role with a job longer than the base window, and a warm start. The three-region run of VIA used wood yields by region that `spec-production.md` 12.1 does not list (N-070).
- **Closed:** 2026-10-10. Applied in `spec-reference-chain.md` 3 and 4: features C01 to C17 (a self-employed staple role, a perishable, a seed cycle, a 60-tick job, durables, a job longer than the window, a famine variant with estates, a warm start, an upkeep-only world, trainee recipes above and below break-even); the scenarios are assigned to packs in `spec-stability.md` 8. Bands are STY's.

### N-070 Regional yields of the three-region run

- **Areas:** VIA, REF
- **Raised:** 2026-10-10, `spec-viability.md` 15.
- **Note:** The three-region run used wood yields by region that `spec-production.md` 12.1 does not list. REF adds them when it builds scenario A3.
- **Closed:** 2026-10-10. The yields of the three-region run are listed in `spec-reference-chain.md` 4.4.3.

### N-034 Role-to-recipe membership and the choice of recipe

- **Areas:** SPN
- **Raised:** 2026-10-10, `spec-production.md` 3.3, 4.1, 5.1.2.
- **Note:** `Role.Recipes` (sorted list of recipe IDs) is a field AGT owns. An agent has one job at a time, starts only when free, and the `Strategy` names the recipe and an optional maximum number of executions. A role change abandons the job (reason `role_changed`). The default rule for choosing among executable recipes (for example by expected net value using `UnitCost` and the expected price) is AGT's. The spawner must expect a job to be lost when it moves an agent to another role.
- **Update (2026-10-10, AGT):** `Role.Recipes` is in `spec-agents.md` 3.3; the default recipe choice is the eligible recipe with the greatest net value per tick at the agent's belief centres, ties to the lower ID, net value positive or a source (7.1.4); the world refuses a request off the role's list with `not_on_role` (6.5.1). The spawner must expect a job to be lost when it moves an agent to another role.
- **Closed:** 2026-10-10. Applied: `Role.Recipes` in `spec-agents.md` 3.3; the default recipe choice in 7.1.4; a role change abandons the job with `role_changed` (`spec-spawner.md` 4.2).

### N-064 Traineeship is for weight 1

- **Areas:** SPN, SKL, SCL
- **Raised:** 2026-10-10, `spec-skill-barriers.md` 4.1.1.
- **Note:** A trainer and a trainee are agents of weight 1; a role with `TrainingTicks` above 0 needs groups of scale 1 (validation `traineeship_needs_weight_one`). SPN: the entry rule must not offer a traineeship to a cohort.
- **Closed:** 2026-10-10. Applied in `spec-spawner.md` 4.4 and FR-SPN-009 (the world refuses `Apply` for weight above 1, `needs_weight_one`) and `spec-skill-barriers.md` 4.1.1.

### N-073 A weight rise the pool cannot fund

- **Areas:** SPN, POP, EST
- **Raised:** 2026-10-10, `spec-estates.md` 2.8.1 and open question 2.
- **Note:** A weight rise draws money from the estate pool; refuse it or apply it partially when the pool is short. Goods drawn for a rise are unspecified.
- **Closed:** 2026-10-10. Applied for entry in `spec-spawner.md` 4.3 and 7.3 and `spec-estates.md` 2.8.1: partial, in whole households, money only, within the whole-pool budget `EntryPoolShare`; entrants get no goods. POP confirms for births.

### N-058 Exact big integers in the solver

- **Areas:** ALL, CODE
- **Raised:** 2026-10-10, `spec-viability.md` 6 and open question 7.
- **Note:** `internal/steady` uses `math/big` (standard library). architecture.md 3.5 bans floats and requires checked arithmetic where `int64` can overflow; unbounded integers cannot overflow. A sentence in 3.5 would remove doubt. That is a rules change and needs a fresh read-receipt tag; not made.
- **Closed:** 2026-10-10. Applied in `rules/architecture.md` 3.5: `math/big` is allowed in `internal/steady` only, with the conversion rules and no `math/big` type in anything exported.

### N-077 Rules changes owed by the specs

- **Areas:** ALL, CODE
- **Raised:** 2026-10-10, handoffs of rounds 2 and 3.
- **Note:** None is made; each needs a fresh read-receipt tag. (1) `rules/architecture.md` 3.5: allow `math/big` in `internal/steady` (N-058). (2) 4.3: the `Strategy` row could say "`Work`, `Trade` and `Revise`". (3) `rules/observability-and-logging.md` 4.3: `RecipeExecuted` at completion with new fields, `RecipeAbandoned`, `UpkeepShortfall`, `AgentSpawned` fields and the removal event's `Cause`, and later `BeliefsUpdated`; 3.3.2: the log keys `path`, `rule`, `config_hash`, `region`, `bound`, `agents`, `households`, `unfed`.
- **Closed:** 2026-10-10. Applied in two edits: `rules/architecture.md` 3.5 and 4.3, and `rules/observability-and-logging.md` 3.3.2 (log keys), 4.3 (events including `WeightChanged`, `RoleChanged`, `EntryLimited`, `WantShortfall`) and 5.3 (metrics including `household.unfed`, `role.changes`, `want.unfunded`), with fresh read-receipt tags. Remaining conflicts of the rules edit are N-088.

### N-068 Model scripts are outside the repository

- **Areas:** STB, VIA, CODE
- **Raised:** 2026-10-10, `spec-stability.md` 15.
- **Note:** The numbers of `spec-stability.md` 6, `spec-viability.md` 15 and the belief revision come from small Docker Python models in `/tmp/economy-boss/` (`stb/`, `via/`, `pb/`), which will vanish with that directory. Decision for the human: archive them under `doc/specs/` (with a documentation index row; the project would then hold Python) or accept that the specs' descriptions are the record.
- **Closed:** 2026-10-10. Decided by the human: archive. The scripts are in `doc/specs/models/` (`stb/`, `via/`, `pb/`, `agt/`) with a README and a documentation index row; the specs cite the new paths. The scripts of the demand, spawner and reference-chain specs are still in `/tmp/economy-boss/` (see N-089).

### N-024 Which of FIS, INT and PLC are in the first release

- **Areas:** FIS, INT, PLC
- **Raised:** 2026-10-10, `roadmap.md` Open 1.
- **Note:** `spec-fiscal.md`, `spec-integration.md` and `spec-policy.md` are
  candidates to defer past the first release, and the decision is open. CHK is
  core (ADR 0002). SCL and DEM change how agents work and are costly to retrofit,
  so they stay in the core. Settle it against `vision.md` section 6 (version 1
  scope). Note that INT carries the player agent (N-022) and external shocks, so
  deferring it also defers those.
- **Answer (proposed, 2026-10-10):** follow `vision.md` section 4. FIS and PLC are
  later. INT is in version 1 only as shocks and a player agent, because STY needs a
  shock (a failed harvest) and the first consumer needs a player. A game can inject
  taxes as shocks until FIS exists. It binds only when the vision is accepted and
  recorded in an ADR; close this note then.
- **Closed:** 2026-10-10. Decided by the human (ADR 0003): FIS and PLC are later; INT is in version 1 only as shocks and a player agent.

## Read receipt

If you are an agent about to write or change a spec, you quote the tag below in
your first message about that work, as required by
[AGENTS.md](../AGENTS.md). Reaching it means you have read this file in full. Do
not guess it or reuse one from memory.

When you edit this file (add, update or close a note), replace the tag with a
fresh random value in the same change (for example
`od -An -N12 -tx1 /dev/urandom | tr -d ' \n'`). A stale tag in an agent's
message means it read an older version, and may have missed a note.

NOTES-TAG: 99961e725dede5ae4e730591
