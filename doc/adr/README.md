# Architecture decision records

An ADR records one decision that shapes the design, why it was taken, and what
it costs. Specs describe how a thing works; an ADR explains why the design is
shaped that way and is where a later reader looks when they want to reverse it.

## Convention

- File name: `NNNN-short-title.md`, where `NNNN` is a four-digit sequence number
  starting at `0001`, and the title is lowercase words joined by hyphens.
- Numbers are assigned in order of creation and are never reused or renumbered,
  even when an ADR is rejected or superseded.
- One decision per ADR. A decision that changes an earlier one is a new ADR that
  says `Supersedes: NNNN`; the old ADR keeps its text and gets the status
  `Superseded by NNNN`.
- A decision that changes only one clause of an earlier one, and leaves the rest
  standing, is a new ADR with `Supersedes: none` and `Narrows: NNNN decision N`.
  The old ADR keeps its text and its status; a dated note under the decision
  points to the new one, and the new ADR is listed in the index.
- Status is one of: `Proposed`, `Accepted`, `Rejected`, `Deprecated`,
  `Superseded by NNNN`.
- A decision that conflicts with a rule in `doc/rules/` is not valid until the
  rule is changed in the same change (rules win; `AGENTS.md`).
- Every ADR uses the template below and is listed in the table in this file.

## Template

```markdown
# NNNN. Title

- Status: Proposed | Accepted | ...
- Date: YYYY-MM-DD
- Supersedes: none (or NNNN)
- Narrows: (optional) NNNN decision N

## Context
## Decision
## Consequences
## Alternatives considered
## Open points
## Related
```

## Index

| ADR | Title | Status |
|-----|-------|--------|
| [0001](0001-macro-economy-as-replaceable-layers.md) | The macro economy is a replaceable layer with stable defaults | Accepted |
| [0002](0002-persistence-by-checkpoint.md) | State is persisted by the consumer, from a checkpoint the library produces | Accepted |
| [0003](0003-version-1-scope.md) | Version 1 scope: the slice, version 1 and later | Accepted |
| [0004](0004-outside-trades-through-pass-through-counterparties.md) | The outside trades through pass-through counterparties | Accepted (human review pending) |
| [0005](0005-delivery-contracts-are-matched-by-a-first-fit-round.md) | Delivery contracts are matched by a first-fit round | Accepted (human review pending) |
| [0006](0006-structural-rank-may-gate-entry-rights-and-places.md) | A derived structural rank may gate an entry, a right or a place; no agent reacts to an aggregate | Accepted (human review pending) |
| [0007](0007-a-module-is-one-file-that-owns-its-touch-points.md) | A module is one root file that owns every one of its touch points | Accepted |
| [0008](0008-random-draws-are-keyed-streams.md) | Random draws come from keyed streams, not from one positional stream | Accepted |
| [0009](0009-the-toml-library-is-pelletier-go-toml-v2.md) | The TOML library is pelletier/go-toml/v2, behind a guard pass | Accepted |
| [0010](0010-a-module-has-named-hooks-outside-its-file.md) | A module has named hooks outside its file (narrows 0007 decisions 1, 2, 3) | Accepted |

## Updating this file

Add a row in the same change that adds an ADR. Keep the rows in number order.
When a status changes, change it here and in the ADR.
