# Vision and scope

**Status: accepted** by the human on 2026-10-10; the decision is recorded in
[ADR 0003](adr/0003-version-1-scope.md). The scope in section 4 is the starting point for
version 1: the slice may still move specs between stages when it has run. The
conditions listed in section 6 that are still open are tracked in the ADR, not
blocking.

The approach is in [overview.md](overview.md), the plan of the specs in
[spec-roadmap.md](spec-roadmap.md) and the order in which the code is built in
[roadmap-v1.md](roadmap-v1.md). This file says what the first release is for,
what it must contain, and what it leaves out.

## 1. Vision

A Go library that gives a game or simulation a living economy at the level of a
realm: regions with their own markets, specialisation that emerges from supply
chains defined in data, prices that react to events and a player who can disturb
all of it. It is generic, so the same engine runs a fantasy kingdom, a space
colony or a trading-card world by changing configuration only.

## 2. The first consumer

The first consumer is a fantasy role-playing game (`AGENTS.md`). It sets the
needs below, but not the vocabulary: the core stays setting-agnostic, and the
game's content lives in a data pack under `examples/` (`spec-roadmap.md`,
"Deliverables that are not specs").

What the first consumer needs from the library:

- several regions with their own prices, linked by trade: five regions, of
  varying size and resource richness, with different climates, so that a region
  is better at growing some crops than others and specialisation and trade
  follow from it (confirmed by the first consumer, 2026-10-10). Climate is data,
  not a type: it is each region's yield for each gathering or growing recipe, by
  season (RES, CAL), and no code branches on a crop (architecture.md 9.1);
- a player who can trade, and events a game can inject (a failed harvest, a
  closed road, a plague);
- a world that can be saved and resumed exactly;
- prices a game designer can read, query and explain;
- a run cheap enough to tick alongside a game loop.

The last item is a cost budget. `spec-scale.md` (SCL) proposes a provisional
one, which stays a placeholder until the consumer confirms its world size.

## 3. Goals and non-goals

Goals:

1. A plausible, stable economy: bounded prices, no runaway inflation, no
   collapse from a healthy start (STB, VIA, STY).
2. Reproducibility: the same seed, configuration and calls give the same result
   on every platform, and a checkpoint resumes a run bit for bit.
3. Replaceable parts: a consumer tunes parameters, switches modules on or off,
   or replaces a layer through an extension point.
4. Explainable results: every faucet and sink has a reason, and a designer can
   ask why a price moved.
5. Adaptable recipes: a client defines its own goods, recipes and supply chains
   as data, from raw goods that are gathered to advanced goods made from them,
   without changing library code. The recipe format must be expressive enough
   for a client to adapt it to its game: several inputs and outputs,
   byproducts, duration, required tools, and the limit that a regional
   endowment puts on gathering. Whether it also needs conditions (a season, a
   building), variable yield, quality grades or recipes that change over time
   is open, and is decided when PRO and GDS are written. A production hook
   (a further extension point beyond those of `rules/architecture.md` 4.3) is not planned; add one only if a real case shows
   that data cannot express a rule, because client code inside production
   makes determinism and conservation harder to guarantee.

Non-goals (the overview, section 5, has the longer list): forecasting a real
economy; individual psychology, culture, war or politics as endogenous causes;
stock markets and complex finance; a server, transport or storage layer in the
core.

## 4. Proposed scope

Each spec of `spec-roadmap.md`, written or planned, gets one of three stages.

- **Slice**: the vertical slice of the overview, section 8: one region, a short
  chain, clearing, beliefs and stability checks. Its job is to show which of the
  other specs are really needed, so it comes first and may change this table.
- **Version 1**: the first release that meets the first consumer's needs.
- **Later**: valuable, but the first release works without it. The core leaves
  room for it where `spec-roadmap.md` says so.

| Stage | Specs |
|-------|-------|
| Slice | GDS, LED, PRO, CLR, BEL, AGT, SPN, CUR (exists), EST (exists), WEL (exists), CAL (exists), WLD, CFG, STB, VIA, REF, and OBS except its metric adapter (moved here, below) |
| Version 1, in addition | SCL, DEM, REG, RES, SUB, POP, MIG, INF, TRN, CAR (exists), MER, EXT, MON, OBS's metric adapter (`otelobserver/`), CHK, IND, STY, CLB, EXP, and a minimal INT limited to external shocks and a player agent |
| Later | SKL (exists), SET, POL, STO, RSK, LND, ORG, CAP, EMP, SAV, TEC, FIS, PLC, CRD, INS |

Reasons for the less obvious placements:

- **CAL is in the slice** because the year in the wage and welfare formulas
  (`TicksPerYear`) is derived from the calendar. The structure of time is core and
  Earth by default, so a consumer can model another planet; only the seasons are
  optional (`spec-calendar.md`).
- **OBS is in the slice, except its metric adapter** (moved from version 1 when
  `roadmap-v1.md` was written, 2026-10-12). Every module adds an event and a log
  line, and the first golden file renders the events (`spec-observability.md`
  9.5), so the event catalogue, the sealed `Event`, the observer, the options
  `WithLogger` and `WithObserver` and the log keys are built from Stage 0, the
  snapshot with the world in Stage 1. The metric adapter `otelobserver/` is a
  separate module that nothing in the core imports, and stays in version 1
  (Stage 2). The events are built in stages: a module declares its own events
  and fields (`spec-observability.md` 3.3.4).
- **LED is in the slice, with the claims list and the balance sheet** (decided
  2026-10-13, `plan-phase-1-1.md` D8, git history, `2abbe62^`): the first build of `internal/ledger`
  keeps the list of claims, its six operations and the balance sheet of
  `spec-ledger.md` 9.2 and 10, so that the checkpoint and the balance sheets do not
  change shape when credit arrives. The credit and insurance modules that use them
  stay in "Later". The room of a limited account (STO, "Later") and `Relocate`
  with the relocation totals (REG, version 1) are built with their owners
  (`spec-ledger.md` 16.1).
- **SCL and DEM are in version 1** because they change the agent model and are
  costly to retrofit (`spec-roadmap.md`, Open 1).
- **CHK is in version 1** because the consumer needs save and resume, and every
  spec lists the state it contributes, which is cheap to do while writing and
  costly to add afterwards (ADR 0002).
- **MON, EXT, IND, STY, CLB and EXP are in version 1** because the macro layer
  is measured, and tuning needs the measure, compare, adjust and run loop
  (`spec-roadmap.md`, "Macro is measured, never set").
- **Version 1 is built in the order of `roadmap-v1.md` Stage 2 and
  `plans/plan-2-0.md`**: checkpoint, the region graph and scale first, then DEM
  as the first module (money has to return to the non-farm roles, N-314), then
  MON and the pool part of EST, then the rebalance of the reference pack, then
  the other modules.
- **INT is cut down** to shocks and a player agent: STY needs a shock to test
  the response to a harvest failure, and the first consumer needs a player.
  The rest waits.
- **FIS and PLC are later**: the core does not depend on them, and a game can
  inject taxes as shocks until then.
- **SKL is later** although the spec exists: tools and traineeships are a
  module, off unless configured, and the slice shows whether it is needed.
- **LND is later but SUB and RES are in version 1**: the subsistence floor
  needs a minimal endowment, not ownership or rent.
- **STO, RSK and SET are later**: transport in version 1 has capacity and
  delay, but no warehouses, no raids and no size classes. Spoilage in transit
  stays in TRN.

## 5. Version 1 is done when

1. The reference chain runs to a stable state from a warm start (VIA) and passes
   the acceptance bands of STY: bounded prices, no cobweb blow-up, price
   convergence between connected regions within transport cost, a recovery after
   a harvest shock, and a famine in a closed region.
2. The invariant, determinism, golden-file and configuration-only chain tests of
   `rules/testing.md` pass on two architectures (amd64 and arm64: by the CI of
   `roadmap-v1.md` Stage 2 once the owner has decided it, plan 2-1 Phase 5, or by
   hand with an emulated arm64 run).
3. A new commodity or chain needs only a configuration file (architecture.md
   9.4). This holds for a second, unrelated setting's recipes as well as for
   the reference chain, so the recipe format is shown to be adaptable and not
   shaped to one game (goal 5).
4. A checkpoint taken mid-run and resumed gives the same result as an
   uninterrupted run.
5. A consumer example builds a complete server using the public API alone
   (architecture.md 1.3), and the RPG data pack runs.
6. The cost budget of SCL holds for the first consumer's world size.

## 6. What this document needs before it is accepted

- The first consumer confirms the needs in section 2 and the world size for the
  cost budget. Confirmed so far: five regions of varying size, richness and
  climate, about 1 000 households each (5 000 in all) and 30 commodities. Still
  missing: the number of recipes and how often the consumer ticks.
- bazaarBot is run, so that "stable" has a reference (overview, section 8). The Doran and Parberry paper was read on 2026-10-10 and gives stability
  criteria (`spec-beliefs.md` section 19); the run is still missing.
- The slice has run, since it may move specs between stages.
- The answers to the open questions 1 and 3 of `spec-roadmap.md` are recorded.
