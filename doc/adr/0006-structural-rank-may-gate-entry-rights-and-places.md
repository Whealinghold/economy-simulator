# 0006. A derived structural rank may gate an entry, a right or a place; no agent reacts to an aggregate

- Status: Accepted (human review pending)
- Date: 2026-10-13
- Supersedes: none
- Narrows: 0001 decision 7

## Context

[ADR 0001](0001-macro-economy-as-replaceable-layers.md) decision 7 says: "The
engine holds no aggregate state that agents react to." Three more texts say the
same rule in their own words: `spec-world.md` 9.4.1 (no cadence, period, window,
gate probability or share is a function of a measured aggregate, "a count of
households or agents" among them), `spec-stability.md` 5.1.3 with FR-STB-020 (a
damper does not read a macro aggregate and feed it back), and the
section "Macro is measured, never set" of `spec-roadmap.md`.

Three later specs each need one thing the letter of that rule forbids, and each
named it as a conflict rather than hiding it:

- **Settlements** (`spec-settlements.md` 7). A region's settlement class is a
  step function of its household count, quantised, with a band and a dwell. It
  gates entry to a role and a count of places, and nothing else. A count that
  bounds a number is the one place where a count gates.
- **Institutions** (`spec-institutions.md` 3.4.1). A body's `Places` bounds how
  many households of its roles may be members in a region, so the count of
  households of those roles is read against it: a second reader of a count.
- **Tenure** (`spec-land.md`). The units unserved or idle at one endowment, and
  a title's share of it, are quantities local to the entity a rent belongs to,
  like the unmatched units at one market in `spec-beliefs.md` 5.4. They are not
  an aggregate of the world, but the letter of 9.4.1 does not say so.

None of the three lets an agent read an aggregate, and none sets a rate, price,
share, period or window from one. The question is whether the rule has these
exceptions, written once and in one place, or whether each spec argues its case
against four texts.

## Decision

Decision 7 of ADR 0001 stays and gains three named exceptions. Nothing else in
ADR 0001 changes.

1. **A derived structural rank may gate an entry, a right or a place.** A rank
   is a quantised step function of a count of households of one region, held by
   the world with a band and a dwell, and read only by the world's own entry
   rules (never by an agent's view). It gates a permission or a number of
   places. It never sets a rate, price, wage, share, probability, period or
   window.
2. **A count of households of one region may gate a permission, a place or a
   structure** (a body's places against the households of its roles). It is
   never an input to a rate, price, share, period or window.
3. **A quantity local to the entity a price belongs to is not an aggregate.**
   The units unserved or idle at one endowment, and a title's share, are read by
   the rule that prices or apportions that one endowment, as a market's
   unmatched units are read by its own belief update.
4. **The rule otherwise stands.** No cadence, period, window length, gate
   probability or share of the library is a function of a measured aggregate;
   a decision reads what a household can know; macro is measured, never set.
5. **Each exception has an owner.** The reader of an exception is named in the
   spec that owns it, with its position and its entry in the register of
   `spec-stability.md`. A new reader, or a new kind of exception, is a new ADR.

`spec-world.md` 9.4.1, `spec-stability.md` 5.1.3 and FR-STB-020, and `spec-roadmap.md`
each take one sentence that names the exceptions and cites this ADR.

## Consequences

- The four texts agree, and a later spec that needs a gate on a count cites this
  ADR instead of arguing the point.
- The loop remains that the count moves the rank and the rank (through the role
  it opens or closes) moves the count. It is registered in the stability
  register, bounded by the `K - 1` classes of the rank, and a pack must not
  rely on it for stability.
- ADR 0001 keeps its text and its status. Its decision 7 carries a dated pointer
  to this ADR, so a reader of 0001 finds the narrowing.
- The README of the ADRs gains the `Narrows` line, so a later narrowing is
  written the same way.

## Alternatives considered

- **Amend decision 7 in place.** Rejected: an accepted ADR keeps its text
  (`doc/adr/README.md`), and the history of why the rule was once absolute is
  worth keeping.
- **Supersede ADR 0001.** Rejected: it would mark all eight decisions
  superseded for the sake of one clause of one of them.
- **Refuse the exceptions and use flags only.** A settlement class as a flag a
  consumer sets between ticks needs no exception, but the consumer then has to
  compute what the library already knows, and a body's places cannot be
  enforced at the world's own entry step. Rejected for settlements and
  institutions; kept as the fallback if a human refuses the rank
  (`spec-settlements.md` open question 1).
- **One sentence in `spec-world.md` only.** Rejected: four owners say the rule,
  and a carve-out in one of them leaves three that contradict it.

## Open points

1. If the human refuses the settlement class (`spec-settlements.md` open
   question 1), exception 1 is withdrawn, and exceptions 2 and 3 stand or fall
   with the institutions and tenure specs.
2. Whether a consumer's own `Strategy` may read a rank. It does not today: no
   view carries one.

## Related

- [ADR 0001](0001-macro-economy-as-replaceable-layers.md), decision 7
- `doc/specs/spec-world.md` 9.4.1; `doc/specs/spec-stability.md` 5.1.3 and
  FR-STB-020; `doc/spec-roadmap.md`, "Macro is measured, never set"
- `doc/specs/spec-settlements.md` 7; `doc/specs/spec-institutions.md` 3.4.1;
  `doc/specs/spec-land.md`
