# 0008. Random draws come from keyed streams, not from one positional stream

- Status: Accepted
- Date: 2026-10-12
- Supersedes: none
- Narrows: 0002 decision 3

## Context

`doc/rules/architecture.md` 7.3 fixed one `rand.PCG` per world, seeded once
from `WithSeed`, and `spec-world.md` 7 worked it out: every draw of a run takes
words from that one stream in a fixed order (the draw table 7.3, D1 to D17), so
"every draw's value depends on how many were taken before it: one extra draw
early in a run changes every later draw of the run" (7.6.1). The spec registers
this as a hazard, WLD-A5, `spec-stability.md` L120: in its model 98% of the
later draws differ after one extra draw. Three sets of rules exist only to keep
the count aligned between runs: a module that is on and idle takes no word
(7.4.2), a draw that might be skipped is taken anyway (7.4.3, damper D88), and a
creation draw is unconditional (5.7.2). A paired experiment is exact only while
two runs have drawn equal words at every boundary (`spec-experiments.md` 7.3),
which `spec-calibration.md` 4.2.1 turns into a flag `d` on every knob that moves
a count. The checkpoint saves the 20-byte PCG state and a restore installs it
(`spec-checkpoint.md` 4.2 tag 1, 8.4). Per-purpose streams are
`spec-world.md` open question 4, and parallel clearing (question 7) waits on it.

The cost is paid at the worst time. With 24 optional modules, each of which
adds draw sites, every module added and every bug fixed in a draw count moves
every golden file after it, and "module off equals module absent" holds for
the draws only if every count is accounted exactly. The research behind
[ADR 0007](0007-a-module-is-one-file-that-owns-its-touch-points.md) found the
alternative in use for the same reason: streams keyed by the decision and the
entity (the common-random-numbers literature; counter-based generators keyed by
`(seed, entity, step, channel)` in market simulators), so that enabling a
mechanism cannot perturb the draws of another. The library already uses the same
mixer, the SplitMix64 finaliser, for its stateless keys `K(t, id)` and the
ration key (`spec-world.md` 10.1.1, `spec-resources.md` 5.3.3).

## Decision

1. **One seed, no stream state.** A world's randomness is a function of
   `(seed, site, tick, id)`. The seed is the one random datum of a world; no
   position in a sequence is state.

2. **A stream per site call, from the stdlib PCG reseeded by a key.** The
   world holds one `rand.PCG` (`math/rand/v2`) and one `*rand.Rand` built on it
   once, as today. At the start of each draw site it reseeds the PCG with
   `PCG.Seed(hi, lo)`, where `hi` and `lo` are two SplitMix64 outputs over the
   seed, the site constant, the tick and the entity ID, in a form of this
   shape:

   ```
   s  = seed
   for w in (site, tick, id):
       s = finalise(s + w + 0x9E3779B97F4A7C15)
   hi = finalise(s + 0x9E3779B97F4A7C15)
   lo = finalise(s + 2 * 0x9E3779B97F4A7C15)
   ```

   with `finalise` the finaliser of `spec-world.md` 7.1.2. `spec-world.md` 7
   fixes the exact form and publishes the test vectors, as its 7.1.3 does
   today. No allocation: one PCG and one `Rand` for the life of the world.

   *Note (2026-10-12): the exact form is `spec-world.md` 7.1.3. It is the fold above over
   three words, `(site, tick, id)`, with a fourth word `sub` for a row that names a pair
   (the default clearing's market is `(region, commodity)` and a belief outcome is
   `(agent, commodity)`, so neither needs a market index or a packed ID). A signed tick or ID
   enters as its two's-complement `uint64`. The site constants are literals, 64-bit, fixed
   and independent of any row position (7.1.4).*

3. **The draw table is the table of sites.** Each row of `spec-world.md` 7.3
   (D1 to D17 and the between-ticks draws) keeps its count and gains its
   `site` constant and its `id`: the agent ID for a draw about one agent, the
   market index for the default clearing, the edge or line index for a route or
   an outside line, 0 where there is none. The tick is the current tick, the
   next tick for a between-ticks call, 0 at construction. Within one stream the
   draws are sequential as today: the world's draw of 7.2.1 is unchanged, and
   an extension takes what it likes from the `*rand.Rand` it receives
   (architecture.md 7.7 still forbids keeping it).

4. **No draw depends on another site's or another entity's count.** A change
   in the number of draws at `(site, tick, id)` moves only the later draws of
   that same stream. The rules of `spec-world.md` 7.4.2 and 7.4.3 narrow to one
   stream: an owner may keep "draw anyway" inside a stream (an agent's intents
   at D5) and no longer needs it for alignment between runs or between a module
   on and off.

5. **The checkpoint holds the seed and nothing else of randomness.** Tag 1 of
   `spec-checkpoint.md` 4.2 loses the PCG state; the seed is already in the
   header (5.4). A restore installs no random state and takes no word.
   `WithReseed(salt)` on a restore replaces the seed by
   `finalise(seed xor finalise(salt))`, the rule of `spec-technology.md` 4.4.6
   generalised; a module that keys anything from the seed (the technology key,
   D16) derives it from the effective seed and no longer re-derives it at a
   reseed (note N-202). `WithSeed` stays refused on a restore
   (`spec-checkpoint.md` 6.4); the seed of a restored world is the effective
   seed in the header.

6. **Paired runs are aligned by construction.** Two runs of one seed that
   differ in a parameter, a shock or a module draw the same values at every
   `(site, tick, id)` they share; they diverge through state only. The stream
   fingerprint of `spec-experiments.md` 7.3 has nothing to read and goes;
   `Pair` keeps measuring the variance reduction. The flag `d` of
   `spec-calibration.md` 4.2.1 goes with it.

## Consequences

- `doc/rules/architecture.md` 7.3 is changed in this change (the rule must
  change with the ADR, `adr/README.md`), with a fresh read-receipt tag.
- Edits owed, tracked in note N-214: `spec-world.md` 7 (one source becomes one
  seed; the keying rule, constants and vectors; the draw table gains `site` and
  `id`; 7.4.2, 7.4.3 and 7.6 narrowed; 7.4.4's "final state of the PCG" becomes
  the seed), 12 (no PCG state), 13 (the CHK row), 16.5, 17 (WLD-A5), 22
  (questions 4 and 7 answered: a market's stream is its own, so P4.1 may clear
  markets concurrently once architecture.md 12.2 has an owner and a lifecycle);
  `spec-checkpoint.md` 4.2 tag 1, 6.2 step 8, 6.8, 8.4, FR-CHK-051 and the
  worked example of 17; `spec-experiments.md` 7.3 and FR-EXP-062;
  `spec-calibration.md` 4.2.1, 7.2.3, FR-CLB-024; `spec-stability.md` L120,
  L124 and D88; `spec-technology.md` 4.4.1 and 4.4.6; the `id` of each row's
  owner (AGT, BEL, CLR, EXT, INF, MER, MIG, RSK, WEL, PLC); the models
  `models/wld/seed.py` and `stream.py`; [ADR 0002](0002-persistence-by-checkpoint.md)
  decision 3 carries a dated note.
  *Applied so far (2026-10-12), N-214 holds the rest:* `spec-world.md` 7 (and 4.1, 4.4, 5.1,
  5.7, 6.9.1, 6.12, 8.1.1, 12, 14, 19, 22 questions 4 and 7), `spec-checkpoint.md` 6.8 and
  17.5, and the models `wld/seed.py`, `stream.py`, `worked.py` and `chk/worked.py`. The other
  passages of WLD and CHK that still describe a positional stream carry the line "stale
  under ADR 0008, see N-214".
- Each site call costs two finaliser evaluations per word of key and one
  `Seed`; the count per tick is bounded by the agents and sites of the
  configuration. It is measured in the slice against the budget of
  `spec-scale.md` 6; it is expected to be far below the cost of the view it
  precedes.
  *Note (2026-10-12): in the form of `spec-world.md` 7.1.3 a site call is one finaliser
  evaluation per word of key plus two for `hi` and `lo`, five for three words and six for four,
  and one `Seed`; it allocates nothing.*
- Gained: a module added or removed, or a draw count fixed, moves only the
  golden output of its own sites; "module off equals absent" holds for the
  draws by construction; a paired comparison is exact in the draws for every
  knob; a draw can be recomputed from its key without a replay; the checkpoint
  is smaller and the restore simpler; parallel clearing is no longer blocked by
  the stream.
- Lost: nothing of reproducibility (a run is still a function of the seed, the
  configuration and the calls). The independence of two streams rests on the
  mixer, the assumption the stateless keys already make.
- No checkpoint format version changes: there is no code and no fixture yet.

## Alternatives considered

- **Keep one positional stream** (the status quo of rule 7.3). Rejected for the
  reasons of the context: the alignment rules multiply with the modules, every
  module churns every golden, and pairing is exact only by accident.
- **A stateful stream per module**, each seeded from the master seed and saved
  in the checkpoint (the CRN literature's first form). Rejected: positional
  within the module, so a module's own count change still moves its later
  draws across entities, and each stream is checkpoint state.
- **A stream per tick only** (the 2% of the model in `spec-stability.md`
  L120). Rejected: cheaper than the status quo but still positional within a
  tick, where most of the modules' sites are.
- **A fully stateless counter-based draw with no PCG** (one finaliser per
  word). Rejected only on cost of API: an extension receives a `*rand.Rand`,
  which needs a `rand.Source`; reseeding the stdlib PCG per site gives the same
  property with the type the rules already name.

## Open points

1. **Answered (2026-10-12).** The exact keying form, the site constants of the slice
   (D1, D3, D5, D7, D8, D9, D10) and the test vectors are `spec-world.md` 7.1.2 to 7.1.7,
   with `models/wld/seed.py` regenerated and every vector printed again by a Go program on
   Go 1.25 and 1.27; `K(t, id)` is in the form of `spec-resources.md` 5.3.3, with the extra
   increment (7.1.7). The reseed rule `finalise(seed xor finalise(salt))` and its vectors are
   `spec-checkpoint.md` 6.8.2 and 17.5. The rows outside the slice append their constants
   with their modules (7.1.4).
2. **Answered (2026-10-12).** The default clearing's per-order key stays a **word of the
   market's stream**, the stream `(siteClearing, tick, region, commodity)`: one word per
   admitted order in input order, bids then asks (`spec-world.md` 7.3, D7). A stateless
   `K`-style key over the order's input position would gain no alignment, because the input
   position is itself the count that moves when an order is added, and it would give the
   default clearing a different way to draw than a consumer's `Clearing`, which takes what
   it likes from the `Rand` it receives (7.2.2); `AverageClearing.Clear` still takes the
   words in the root and hands plain `uint64` keys to `internal/clearing`.
3. Whether a stream of a site that has no natural entity (D2, a line; D13, a

   route row) is keyed by the row index or by the row's IDs. The row index is
   the proposal; the owner decides when it adds its `id`.

## Related

- `doc/rules/architecture.md` 7.3, 7.4, 7.7, 12.2
- [ADR 0002](0002-persistence-by-checkpoint.md) decision 3,
  [ADR 0007](0007-a-module-is-one-file-that-owns-its-touch-points.md)
- `spec-world.md` 7, 12, 22 (questions 4 and 7); `spec-checkpoint.md` 4.2,
  6.8, 8.4; `spec-experiments.md` 7; `spec-calibration.md` 4.2.1;
  `spec-stability.md` L120; notes N-111, N-124, N-162, N-202, N-214
