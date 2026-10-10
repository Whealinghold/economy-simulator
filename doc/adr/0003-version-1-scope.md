# 0003. Version 1 scope: the slice, version 1 and later

- Status: Accepted
- Date: 2026-10-10
- Supersedes: none

## Context

The roadmap lists more than fifty specs, and `vision.md` proposed which of them
the first release contains. The proposal stayed a draft until the human
accepted it, and three roadmap questions depended on it: whether `spec-fiscal.md`
(FIS), `spec-integration.md` (INT) and `spec-policy.md` (PLC) are in the first
release (roadmap Open 1), whether a policy actor reuses `Strategy` (Open 2), and
whether a region holds one settlement or several (Open 3).

Since the draft, the specs of the slice and of much of version 1 have been
written (currency, goods, ledger, production, clearing, beliefs, agents, scale,
config, calendar, stability, viability, welfare and wages, skill barriers,
estates, carriage), and numeric prototypes showed that some assumptions of the
draft need care (a closed economy has a mean welfare ratio of 1, the cost floor
freezes perishable markets at 100%). None of that changes the staging; it
changes what the remaining specs must supply.

## Decision

1. **The staging of `vision.md` section 4 is the scope.** Three stages: the
   **slice** (one region, a short chain, clearing, beliefs and the stability
   checks, which shows which other specs are really needed and may change the
   table), **version 1** (the first release that meets the first consumer's
   needs) and **later**. `vision.md` section 4 holds the table; a change to it
   is a new ADR or an amendment recorded in the vision and the roadmap in the same
   change.
2. **FIS and PLC are later; INT is in version 1 only as external shocks and a
   player agent.** The state-of-the-world shocks STY needs (a failed harvest) and
   the player the first consumer needs are in; taxes are injected by a game as
   shocks until FIS exists. `spec-agents.md` 10 already defines the player as a
   directed external agent that acts through a mailbox between steps.
3. **A policy actor gets its own interface, written with `spec-policy.md`.** It
   reads measures and returns settings (rates, release amounts); it does not
   bid, so the `Strategy` signature does not fit. A ruler that also trades uses
   `Strategy` for its orders and the policy interface for its settings. The
   interface is not created before PLC (architecture.md 4.2).
4. **One settlement per region; a region is one market.** A town and its hamlets
   are separate regions joined by short edges. `spec-settlements.md` (later) only
   gives a region a size class from its households. Apply in `spec-regions.md`.
5. **The done criteria of `vision.md` section 5 stand.** In particular a new
   commodity or chain needs only a configuration file, shown by a second,
   unrelated setting's recipes (done in `spec-production.md` 12), and a checkpoint
   mid-run resumes bit for bit.
6. **No default macro constant is called stable** until `spec-reference-chain.md`
   and the acceptance bands of `spec-stylised-facts.md` exist (ADR 0001
   decision 5; note N-027). Every number in the written specs is a placeholder.

## Consequences

- The next specs are the remaining slice and version-1 specs in dependency order:
  SPN, DEM and REF (tier 1), then WLD, REG, RES, SUB, POP, MIG, INF, TRN, MER, MON,
  EXT, OBS, CHK, IND, STY, CLB, EXP and the minimal INT. The "later" specs are not
  written, but the core leaves room for them as the roadmap says.
- Specs that serve version 1 do not wait for the later ones. A spec that needs a
  later one leaves a named hook and says what it assumes (for example, `spec-agents.md`
  assumes no spending beyond upkeep until DEM, MON or EXT exist).
- Moving a spec between stages after the slice runs is allowed and cheap; this ADR
  fixes the starting point, not the end state.
- The first consumer's world (five regions, about 1 000 households each, 30
  commodities) is the sizing target of the cost budget of `spec-scale.md`.

## Alternatives considered

- **Everything in version 1.** Rejected: more than fifty specs and the macro layer
  (credit, tax, policy, technology) cannot be validated before the slice runs.
- **A smaller slice only (no regions).** Rejected: a realm of regions is the
  point of the library, and retrofitting regions into per-market specs is costly
  (roadmap, "The per-region core").
- **Policy actors reuse `Strategy`.** Rejected: the signature differs (settings,
  not orders); ASSUME reuses one interface for actors that all bid, which supports
  one `Strategy` for those and a separate interface for the rest.
- **Several settlements per region.** Rejected: it splits a market, complicates
  clearing and the estate, and a town with hamlets is already expressible as
  regions with short edges.

## Open points

Conditions the vision listed that are still open and are tracked, not blocking:

- The number of recipes and the tick rate of the first consumer are not confirmed
  (note N-030; the cost budget in `spec-scale.md` 6.3 is a placeholder).
- bazaarBot has not been run, so "stable" has no reference run (the Doran and
  Parberry paper has been read; its stability criteria are in `spec-beliefs.md` 19).
- The slice has not run, so the staging table may still move specs between stages.

## Related

- [vision.md](../vision.md), [roadmap.md](../roadmap.md) (Open 1 to 3 and Decisions)
- [ADR 0001](0001-macro-economy-as-replaceable-layers.md), [ADR 0002](0002-persistence-by-checkpoint.md)
- notes N-024, N-025, N-026, N-027, N-030
