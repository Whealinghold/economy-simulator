moved
# Spec: Calibration: the registry of tuning knobs with their classes, ranges and expected effects, and the procedure that calibrates a setting against the stylised facts

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-CLB-NNN` and
`NFR-CLB-NNN`, which tests cite (testing.md section 4). They are in section 21.

Layer: core ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md)
decisions 5 and 7), version 1 (`vision.md` section 4). It is a **fixture and a
procedure in nature**: it adds no API, no `Config` field, no default, no event and
no row to the parameter-owner table. It is the "adjust a knob" step of the loop of
the roadmap's "Macro is measured, never set": measure (IND), compare with the
expected properties (STY), adjust a knob (CLB), run again (EXP). It needs
`spec-stylised-facts.md` (STY: the facts, the bands, the regimes, the seed sets and
the neighbour rule of 7.3.1), `spec-experiments.md` (EXP: the harness, the designs,
the reductions and the four steps of 12.4.2), `spec-indicators.md` (IND: the
measures, by preset name and field), `spec-stability.md` (STB: the loops and the
dampers each knob sits in), `spec-viability.md` (VIA: the solver whose table a
calibration must regenerate), `spec-config.md` (CFG: the catalogue, the bounds, the
layering and the hash), `spec-reference-chain.md` (REF: the packs a calibration runs
on) and `spec-world.md` (WLD: the draw table that says which knobs move a count). It
asks the owner specs for small edits (section 23).

**Nothing has run.** The library does not exist, bazaarBot has not been run and no
band has been compared with a run (notes N-015, N-027). **Every number in this spec
that is not a derivation is a labelled placeholder**: the screening rule, the
seed blocks, the sizes of the designs, the tolerance of the alias test, the
run counts and hours. The registry's rows carry **no value**: no default, no bound,
no calibrated number. They carry what is *known*: which knobs there are, what kind
each is, which fact each is expected to move and in which direction, which loops it
sits in, which pairs must move together, and, in nearly every row, that its search
range is **unknown, to be found**. The scripts of section 18 are stylised Docker
Python models of the *rules of this spec* (a grid rule, an acceptance rate, a cost
count, a registry parse), run on synthetic values; they are not runs of the library
and show no property of it. The settings of the worked examples (19) are invented
and flagged; the core and its vocabulary are setting-agnostic (N-017) and nothing
here branches on a commodity, a recipe or a role (architecture.md 9.1): a pack's
names are arguments, from its data.

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game, a server, a command-line tool), never a
household (N-001, N-017).

## 1. Purpose and scope

The specs fix what each parameter means; STB says how the parts can go wrong; IND
says what to measure; STY says what a healthy run looks like and EXP gives the
harness. None of them says **which numbers a person may move, which they may not,
how far, in what order, against what, with which seeds, and how the result is
written down**. This spec does. It covers:

- **the registry** of tuning knobs: one row for each tunable of the configuration
  catalogue (`spec-config.md` 3.7) and for each solve option of
  `spec-viability.md` 11, classified as a knob, a damper strength, a structural
  choice or a presentation-only field, with its owner, unit, kind of range, grid,
  stage, flags, expected effects, loops and links to the pairs that move together
  (sections 4 to 6);
- **the honesty of a range**: what a validation bound, an evidence range, a derived
  condition and "unknown" each are, and the rule that gives a knob its grid and its
  two neighbours (section 5);
- **the procedure** that calibrates a setting, and the library's own defaults,
  against the facts of STY: the seeds, the four steps, the stages in order, the
  stop rules, what to do when two facts conflict, and what must never be done
  (section 7);
- **the record** of a calibration as plain data (section 8) and **the packs**: where
  a calibrated number lives, how a calibration is versioned and compared, what
  transfers between settings and what does not (section 9);
- **sensitivity and identifiability**: how to find which knob moves which fact, and
  which knobs the facts cannot tell apart (section 10);
- **the tests** (section 11) and **what CLB hands to the first code change and what
  the specs owe** (section 12).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| The meaning, the default and the validation bound of a parameter | the owner spec (`specs/README.md`, "Parameter owners"); a registry row points at it and restates neither |
| A fact, a band, a regime threshold, the seed sets and the stability record | `spec-stylised-facts.md` (STY); this spec uses them and sets none |
| The harness: a design, a run, a branch, a reduction, a comparison | `spec-experiments.md` (EXP); this spec says which form each step takes |
| A measure and its arithmetic | `spec-indicators.md` (IND) |
| The solver and the warm start | `spec-viability.md` (VIA); this spec says when the table must be regenerated |
| The packs, the overlays and the shock scripts | `spec-reference-chain.md` (REF) |
| A calibrated value | none: nothing has run (N-027); a value is a result of a record (section 8), never of a spec |
| The setting's content (its names, its currency, its region list) | a pack under `examples/` (AGENTS.md); this spec names none |
| An in-run controller that moves a knob by a measure | none in version 1; a policy actor is PLC's, later (ADR 0003 decision 3) |

## 2. Positions taken

| # | Position | Section |
|---|----------|---------|
| 1 | The registry's **one home is this spec**: tables of plain text in section 6, parsed by a test as `testing.md` 6.7 parses the parameter-owner table. There is no second copy and no Go table; the library has no use for one (no rule reads a knob's class) | 4.1 |
| 2 | A row **classifies, it does not define**. It carries no default, no validation bound and no value. The owner's table is the one definition (architecture.md 9.8); a test that needs a default reads it there | 4.7 |
| 3 | Four **classes**: **K** knob, **D** damper strength, **S** structural choice, **P** presentation only. A **D** knob is never calibrated to its **off value**, and a pack that sets it there records the loops it leaves undamped | 4.3 |
| 4 | A **range** is one of four kinds: **V** a validation bound (the legal range, never a search range), **E** evidence, **D** derived from a stated condition, **U** unknown. Almost every row is **U**, and says so | 5.1 |
| 5 | Every searched knob has a **grid**: a rule that gives ordered levels containing its value and so its two **neighbours**, the points STY 7.3.1 asks a default to hold at | 5.2 |
| 6 | A calibration is a search for a **feasible region** and the point in it **closest to the default** (minimal change), not an optimisation. There is **no scalar objective**: facts are never weighted into a sum | 3.4, 7.3 |
| 7 | **Seed roles**: **search blocks** (101 and up, 100 seeds each, one choice per block, the only seeds a choice is made on), **confirm seeds** (STY's setting seeds 1 to 50, on which a chosen point is confirmed), the **gate** (1 to 10, inside the confirm range, never an input to a choice) and the **hold-out** (51 to 60, the out-of-sample check of a stability record) | 7.1 |
| 8 | Four **steps**, the forms of EXP 12.4.2: screen, explore, choose, confirm. The neighbour rule of STY 7.3.1 closes a default's **stability record** | 7.2 |
| 9 | **Stages C0 to C9** in the order structure, viability and warm start, scale, then the facts of STY by group; a stage **fixes** its knobs, and a later stage may not move one without re-running the confirm of the stages it touches (the re-entry rule) | 7.3 |
| 10 | The product of a calibration is an **overlay document** that holds only K and D keys, layered after the pack, and a plain-text **record** with the hash, the experiment, the results and the bands met | 8, 9 |
| 11 | What transfers between settings is the procedure, the classes, the conditions of the specs and the regime thresholds. Defaults and the priors of STY transfer as **starting points**. A band value, a stability record, a warm start and a calibration overlay **do not** | 9.3 |
| 12 | Sensitivity is **measured** by one-at-a-time elasticities against the signs of the registry; a mismatch is a **finding**. Aliases and gauges are declared; a knob whose damper never acted is **dormant** and is left alone | 10 |
| 13 | CLB adds **no public package** in version 1: the pure helpers live in `internal/calibration`; the loop is the consumer's, with `experiment` and `indicator`. Promoting `internal/verdict` and `internal/calibration` is open question 1 | 12, 22 |

## 3. What calibration is and is not

### 3.1 The loop and who closes it

The loop of the roadmap is measure, compare, adjust, run again. IND measures and
STY compares; EXP runs; **CLB adjusts**. The adjustment is a change of the value of a
configuration field (a **knob**) **between runs**, made by a person or by a
consumer's own program that reads a result and builds the next experiment (EXP 3.7:
no adaptive design in the library). The library never reads a measure and moves a
knob: no rule of a run reads an indicator (IND 3.1), the world holds no aggregate
that agents react to (ADR 0001 decision 7, WLD 9.4), and the harness changes no
parameter inside a run (EXP 12.4.4). A calibration is therefore a **sequence of
experiments**, each a value of plain data, and its output is a document and a
record (section 8).

### 3.2 Two objects

A calibration has one of two objects, and the procedure is the same for both:

| | The **defaults** of the library | A **setting's** pack |
|--|----------------------------------|----------------------|
| What is changed | the value a `Default*` constructor gives a K or D field, in the owner's spec | the pack's data, and, where a default fails on that pack, an overlay of K and D keys |
| Who | the maintainers of the library | a consumer, on its own pack |
| Packs it runs on | every shipped pack of REF that sets the knob | the consumer's pack (and, as a control, the reference packs) |
| Where the result lives | the owner's parameter table and `defaults.go`; the record under `testdata/calibration/` | the pack's overlay documents and the consumer's records |
| What it costs | a behaviour change: the golden files and the verdict records move and are explained (architecture.md 9.8, testing.md 7.2) | the pack's own hash and golden file |
| The word "stable" | allowed only under STY 7.3.1: the three checks STABLE at the default and its two neighbours on the gate seeds, and, where REF defines one, on its hold-out | a pack may say "no regime was detected in the runs of record R", and nothing stronger (STY 7.3.2) |

### 3.3 What a calibration is against

A calibration is judged by the **acceptance set** of STY 8.1 and by nothing the
code under test has defined for itself. In the order of strength:

1. **X and O facts**: exact identities and ordinal relations. No band, asserted from
   the first run. A calibration never starts while one fails (stage C0).
2. **The regime detectors** R1 to R9 (STY 6): `none` over the healthy runs. A
   detector with an unset threshold returns `unknown` (STY 6.5.1); a search may use
   the priors of STY 6.2 as **working thresholds**, which are gross-failure screens
   (a regime threshold is looser than the fact bands it backs, STY 6.5.2). A record
   that used them says `working` beside each; it is never an acceptance record.
3. **B facts whose keys are set**: SR2 on the confirm seeds (1 to 50) against the set band.
4. **B facts whose keys are unset**: reported with the prior beside the observed
   values (STY 3.4). A prior whose **size** has confidence H, M or E (STY 3.3) is a
   **working target** of the search; a prior whose size is P (a placeholder, a guess)
   is **not**: the observed values are recorded, are the evidence for the setting of
   the band (STY 8.4), and a knob is **not moved to meet a guess**.
5. **R facts**: recorded, never asserted, never a target.

This is the rule that keeps the loop from closing on itself (STY-A1): a band is set
**after** a calibration, from runs of the frozen configuration on seeds the
calibration did not choose on, and is then a guard. A calibration that fitted a
knob to a band that had been set from the same code would pass by construction and
say nothing.

### 3.4 Feasibility, not optimisation

The facts of STY are statements of health: a price that is not pinned, a recovery
that happens, an identity that holds. They are not a score to maximise, and no
published figure in them is a target to hit (STY 3.3: level E is a sanity range for
another place and time). A calibration therefore asks two questions and no more:

- **Which points of the searched knobs are feasible**: every X and O fact passes on
  every seed, every detector is `none`, every B fact with a target is inside its
  band by SR2? This is the **feasible region** (7.3).
- **Which feasible point is closest to the default**, counted in grid steps
  (5.2)? That is the **minimal change**: a calibration changes as few knobs as it
  can, by as little as it can, and **if the default is feasible it does not move**.

Two consequences are stated because they surprise. First, a calibration that finds
the default feasible on every stage has **nothing to report but a record**: that is
the common and the good outcome. Second, there is no weighting of facts: when no
point is feasible the answer is the **Pareto set** of pass shares (7.5), a
reviewer's decision, never a weighted sum that hides which fact was sacrificed.

### 3.5 What a calibration never does

Each of these is a defect of the calibration (and a requirement, section 21):

1. **An in-run correction.** No knob changes inside a run because of a measure of
   that run. The slots INT exposes (yield shocks, edge statuses, flow scales) are set
   **between** steps by a **script that is data**, as a shock to be studied, never by
   a rule that reads a measure of the same run (that is a policy actor, PLC).
2. **A knob set from an aggregate of the run it governs.** A value is chosen once for
   a configuration, before the run, from the *recorded results of earlier runs*.
   Setting a per-region or per-role knob from that same run's regional mean is
   "macro set, not measured" (ADR 0001 decision 7). A pack value computed by the
   **solver** from the configuration (a warm-start price, a reference price) is not
   an aggregate of a run and is the way a number is derived (REF position 4).
3. **Disabling a damper to pass a fact.** A D knob is never moved to its off value
   (4.3) as a calibration move. A pack may configure a world with no occupational
   mobility, no relief or no halt as a **structural** choice, recorded as such with
   the loops it leaves undamped (STB 5.1.8); it never does so to turn a failing
   fact green.
4. **Choosing on a seed of STY's sets, or on a search block twice.** A choice is made
   on search blocks only; a block serves one choice; a confirm's per-seed values are
   not read into the next search, and the gate (1 to 10) is never an input to a
   choice (7.1.1).
5. **Fitting a guess.** A knob is not moved to meet a B fact whose prior has size P
   (3.3).
6. **Changing the measure, the window or the transient** to make a fact pass, or a
   regime threshold, a band, a validation bound or a rule constant (STY 8.4.6;
   architecture.md 9.6).
7. **Editing structure to pass a fact in the middle of a stage.** A change of an S
   row is a change of the pack (a new lineage, 9.2), made in stage C1 or C2 and
   followed by every later stage again.
8. **A seed-conditional configuration.** No value depends on the seed or on which
   seed failed; a failing seed is reproduced and read, never patched (testing.md
   6.2).
9. **A stale warm start.** A knob the solver reads (flag `in`, 4.2) is swept with the
   table regenerated per point, or the experiment says it is a perturbation
   (EXP-A5, EXP 3.3.3).
10. **A claim of stability without its record** (STY 7.3.2, 7.3.3).

## 4. The registry: form

### 4.1 Home and form

The registry is **section 6 of this spec**. A **row** is one line of one of its
tables whose first cell is an ID of the form `K-<CODE><nn>` (a **knob row**: a class
K or D tunable, or an S or P field that needs a row of its own) or `S-<CODE><nn>` (a
**structure group**: the S and P fields of one scope, listed together). `<CODE>` is
the area code of the owner (`specs/README.md`), `<nn>` a two-digit counter. An ID
is stable and **never reused**; a row is removed only in the same change that
removes its field, and its ID is then listed as retired in the register's own
last line (section 6.6), so no test and no record names a number twice.

Why in a spec and not in a file or in Go: (1) nothing in the library reads a knob's
class, so a Go table would have no reader and would be a second registry beside the
parameter-owner table (architecture.md 9.8); (2) `testing.md` 6.7 already reads a
table of `specs/README.md` as text, and STY 12 checks its own data the same way; the
test of 11.1 reads this one by the same plain text parse and needs no dependency;
(3) a calibration's *results* are records (section 8), which are data in files; the
registry is knowledge about the fields, which is documentation that a test keeps true.
A consumer's tool that wants the rows as data parses the tables the same way.

### 4.2 The row

Knob rows have these cells, in this order. The closed sets are rule constants of the
checks in 11.1.

| Cell | Content |
|------|---------|
| **ID** | `K-<CODE><nn>` |
| **Key** | `<Scope>.<Field>`: the scope as the parameter-owner table writes it (4.7), the field as the Go name. Never a file path: the paths are `spec-config.md` 6.2's, and N-102 has not yet unified the module tables, so the registry does not depend on them. The experiment's path of a knob is the file spelling of its key (EXP 3.2.1) |
| **Class** | `K`, `D`, `S` or `P` (4.3) |
| **Owner** | the owner's code and the section of its parameter table or definition, as `BEL 11`. The test resolves it to the owner's table |
| **Unit** | `bp`, `ticks`, `units`, `minor`, `count`, `years`, `share`, `name`, `flag` or `list` |
| **Search** | the kind of the search range (5.1): `U`, `E<n>` (an anchor of 6.1), `I<nn>` (a derived condition of 6.4), or `-` for S and P rows (an S field may carry an `E<n>` anchor) |
| **Grid** | the rule of 5.2: `geo`, `lin<step>`, `set:<a/b/c>` or `-` |
| **Stage** | the stage that first searches it (7.3): `C1` to `C9`, or `-` |
| **Flags** | a space-separated subset of `in`, `chk`, `out`, `tbl`, `d`, `z`, `v1x`, `off=<v>` (4.2.1) |
| **Moves** | the expected effects: `<fact>/<Measure>:<sign><src>` tokens, separated by `;` (4.4) |
| **Loops** | the register entries it sits in: `L<nn>` and `D<nn>`, with `D<nn>=<Counter>` where the damper has a counter |
| **Links** | `I<nn>` (a pair that must move together or a condition, 6.4) and `A<nn>` (an alias group or a gauge, 6.5) |

Structure groups have the cells **ID**, **Scope**, **Fields** (a comma-separated
list of the owner's field names), **Class** (`S` or `P`; for the elements of a list that has a knob row, the
class of that list), **Owner**, **Stage** and **Flags**. A group's Owner cell may name several sections of one owner (`TRN 3.1; 11`) or several owners (`INF 8; RES 3.3`), separated by `;`: the test checks each against the README owner column.

4.2.1. **Flags.**

| Flag | Meaning | Source |
|------|---------|--------|
| `in` | an **input of the steady-state programme** (VIA 3): a change makes the `WarmStart` table stale, so the experiment regenerates it per point (`Prepare`, EXP 3.3.3) or says it is a perturbation | VIA 3.1 to 3.3, 8 |
| `chk` | read only by a **check or a diagnostic** of VIA (a warning or a report line): the table does not change, a warning may | VIA 10 |
| `out` | **written by the solver**: a reference price, a warm-start price, a starting wage: regenerated by `SolveSteadyState`, never edited by hand (REF position 4) | VIA 8, REF 2 |
| `d` | can change **the number of words drawn** at the same state, so a paired comparison across it is not exact beyond the first divergence (WLD 7.3.1, 7.6.1) | WLD 7.3 |
| `tbl` | a **solver option** (VIA 11): it changes the table `SolveSteadyState` makes and no rule of a run of the world; a change regenerates the table as `in` does and is never an input to a rule of the world | VIA 11 |
| `z` | the owner's default is **0 or none**, and the pack's base may be non-zero: a `geo` grid is built on the pack's non-zero base (5.2); where the pack leaves it at 0 the knob is at its off value and has one neighbour (`edge:off`) | the owner, 5.2 |
| `v1x` | **no reader in version 1**: a field with no mechanism that reads it yet; its effect is nil and it is not calibrated | the owner |
| `off=<v>` | class D only: the value at which the damper is disabled: a number, or `off=min` or `off=max` (the owner's lowest or highest legal value, read from the owner's bound and never copied into the row), `off=flat` (no curve: a list that carries no variation), or `off=-` when no value does (the damper can only be weakened) | the owner's mechanism |

The `in` and `chk` flags **offer** EXP open question 7 and VIA's part of EXP-A5 a
candidate list of the paths the solver reads, for VIA to confirm and publish, so
that `stale_warm_start` can be on by default (EXP 14.4 asks VIA for the list).

### 4.3 Classes

| Class | Name | Test | What a calibration may do |
|-------|------|------|---------------------------|
| **K** | knob | none of the three below | search it, inside its legal range, for the reason that a fact needs it |
| **D** | damper strength | it is the parameter of a damper of the register (`Dnn` of `spec-stability.md` 5, or a limit that one counts) | search it **within the range where the damper acts**; never to its `off` value; the three checks hold at it and at both neighbours (STY 7.3.1) |
| **S** | structural choice | it defines what the setting *is*: an identity, a topology, a list, a physical or economic quantity (a duration, a yield, a quantity of upkeep, a population, a distance, a capacity), a unit (the money, the time, the distance), a policy shape (a flow of money) or a module switch | change it only in stage C1 (identity and topology) or C2 (quantities, against the solver); never as a knob; a change is a new variant of the pack (9.2) |
| **P** | presentation only | it changes no result and is not hashed, or it is a diagnostic threshold that changes no result | nothing |

The class is a statement about the *calibration*, not about importance: an S row is
as consequential as any. The test is applied in this order, and a field has exactly
one class: (1) no effect on any result or on the hash is **P**; (2) it defines the
identity, the topology, a quantity of the setting, a unit or a policy shape is **S**;
(3) it is a parameter named in a damper entry (the Home cell of `spec-stability.md` 5, or a limit that one counts: a limit one counts wins over rule 4) is **D**, even where the registry's loop cell also lists it for another loop; (4) otherwise **K**. Two exceptions are stated: a **K row may carry a `D<nn>` in its Loops cell** where the damper entry names another parameter and the knob only sits in its loop or is ignored by it (K-SPN10 in D48), and a **K or D row whose Grid is `-`** (a list or a schedule that is not searched) MUST have an `A<nn>` link naming what is searched instead (FR-CLB-004). **S rows at C3 and C8** are *choices* of the pack made at their stage (the scale; the open layer's design), not knobs: a stage C3 or C8 S row is searched as a **choice among listed variants** of the pack and each is a new variant (9.2), never a point of a grid. A spec that
adds a field adds its row in the same change (FR-CLB-004; N-016).

4.3.1. **The off value of a D knob.** A damper is *disabled* when its value is such
that the loop it damps runs as if the damper did not exist. The row's `off=` gives
that value: `RestBand` at 0 (a rest as soon as the target is met: the chatter of
L15), `EntryShare` at its highest legal value, `off=max` (no cap on role change; **0 is a different
thing**: it turns role change off, which removes the loop and not the damper, and is
a structural choice), `BeliefAdjustProb` at `off=max` (every agent adjusts at every
outcome: the lockstep of L13). A row writes `off=min` or `off=max` and the check reads the owner's bound, so no row restates it (4.5).

A default is **at its off value** for several D knobs, and the rule is stated once:
`TargetSpread` 0 turns D08 off by default ("0 until experiments say otherwise",
`specs/README.md` decisions), `HouseholdCap` 0 is "no cap" (D50), `Refuge` 0 is a
stock that never regrows from nothing (D56), a flat `Abundance` is no curve (D54), a
`Limit` or an `Exempt` of 0 is none (D41, D43). That means the damper is **not in
force by default**, and it is the honest state of the library: no run has shown the
loop it would damp. The rule is that a calibration may move such a knob **away from
off** when a fact shows the loop (its `Loops` cell), and **never back to it**; it is
not forbidden to leave a default where it is. `TargetSpread` is the first knob a
calibration is *for* (`N-015`; EXP names "`TargetSpread` experiments"): it is class
D, row K-WEL03.

### 4.4 Expected effects: signs and sources

A **move** token is `<fact>/<Measure>:<sign><src>`:

- `<fact>` is the STY fact ID without the prefix (`P02`, `S01`, `W05`); a scenario
  of STB is named by `S<nn>` only in a Links cell, never here;
- `<Measure>` is an IND preset and field, the closed list of the registry's tokens
  (`Pinned.Share`, `Turnover.Volume`, `PriceRange`, `Offset.abs`, `Dispersion`,
  `RoleShares.Swing`, `Households`, `Unfed`, `HouseholdIncome`, `UpkeepShortfall`,
  `WelfareDistribution.Mean`, `PriceGap.GapAfterFreight`, `StockCover`, `Sectors.M`,
  `PriceIndex.Fisher`), or the **band key** of a scenario of STY that is no preset field
  (`s01.recover_ticks`, `s07.spoiled_bp`, `s14.recover_ticks`, `s17.spike_factor`,
  `s20.ticks`, `s23.band_bp`, `a1.price_band_bp`, `h02.money_bp`). The field names are
  **STY's spelling** of an IND preset's noun (IND publishes no field list: owed, 23); `.abs`
  reads the magnitude of a signed measure and is CLB's suffix;
- `<sign>` is `+` (the knob up raises the measure), `-` (lowers it), `0` (no effect
  expected, and the owner says so), `~` (non-monotone, stated) or `?` (unknown, and
  then the token ends there: a `?` has no source);
- `<src>` follows a `+`, `-`, `0` or `~` and says what the sign rests on: **S** a
  sentence of the owner's spec or of a register entry (the `Loops` cell names it),
  **M** a stylised model of a spec (never a run of the library), **E** a source read
  directly (`evidence.md`), **H** a hand derivation from the rule text, one step away
  from it.

A row holds **at most four** tokens, the ones a spec supports; a knob that moves
many facts says so by its loops. **A `?` is the commonest sign in this registry and
is not a gap to fill by guessing**: it is what screening (10.1) is for. A sign with
`H` is the weakest of the three that name one: a mismatch with a measured sign is a
**finding** in every case (FR-STY-092), and with `H` the first suspect is the
derivation.

### 4.5 What a row does not hold

A row holds **no default**, **no validation bound** and **no value**. The default is
in the owner's table (architecture.md 9.8: a default is defined once); the bound is
in the owner's table and in `bounds.go`; a value is a result of a record. A cell that
restates one is a defect that the check of 11.1 (FR-CLB-005) finds as a cell of the
form of a number in the Search, Grid or Moves columns, except in the closed positions
(a `lin` step, a `set:` level, an `off=` value). The `lin` step and an `off=` number
are the two numbers a row holds, because neither is in the owner's table; an off value that is
the owner's bound is written `off=min` or `off=max` and not as the number.

### 4.6 How the rows are kept true

The rows are checked **in both directions** against the parameter-owner table
(4.8, 11.1): every owner row has a registry row or a group that lists it, and every
registry row names an owner row, except the fields the owner table does not hold: the
solve options of VIA 11, the presentation-only fields and the proposed `World.Clearing`. The class of a row
is not derivable by a test, and a human decides it by the order of 4.3; what the test
checks is that the decision was made.

### 4.7 The owner table's scope names

The parameter-owner table writes a **scope** per row, sometimes with a remark
(`Group; world default 1`, `Commodity (region override: ...)`, `Modules (module;
nil = off)`). The **scope token** of a row is the text up to the first `;` or
` (`, trimmed. A registry key and a group's scope use the token. The tokens in use
are listed in 6.3 with the groups that cover them, so that a new scope in the owner
table is found by the coverage check and not by a reader.

### 4.8 The test's reading of the owner table

An owner-table row is `| \`Parameter\` | Scope | Owner |`. The check forms the pair
(scope token, parameter) and finds exactly one registry key `<scope token>.<parameter>`
or exactly one structure group whose scope is the token and whose fields list the
parameter. A parameter listed under two scopes (`Seasons` under Calendar and under
Yield) is two pairs, and two owner rows of one pair are one pair. The owner table
also writes a remark after the parameter (`MinAgents (K)`, `ReferenceBasket (B_r)`):
the parameter is the first backticked token, without the remark.

## 5. Ranges and their honesty

### 5.1 Four kinds

"Range" is two different things, and a registry that does not say which misleads. The
words of this spec are fixed:

| Kind | Name | What it is | Where it is written | May it be a search range? |
|------|------|------------|---------------------|---------------------------|
| **V** | validation bound | the **legal** range: the values `Validate` accepts. It exists to keep arithmetic from overflowing and work bounded (`spec-config.md` 4) and contains configurations that are absurd (a belief step of 25% of the centre is legal) | the owner's table, `bounds.go` | **No.** It clamps a grid and decides whether a level is legal |
| **E** | evidence range | a range or an anchor read directly in a source of `evidence.md` (status "read directly" or "read in full"), for another place and time. A snippet, a summary or an unverified row is **not** E (STY 3.3, N-018) | 6.1, with the evidence row | **Only as a sanity range**: it bounds a search from outside and calibrates nothing |
| **D** | derived condition | an inequality or a closed form that a spec derives, from other quantities of the same configuration (`EnterTicks < ExitTicks`, `ExitTicks` at most what purchases can reach, a refill rate at least 0.16 a year for a monotone approach). As strong as its derivation (H or M) | 6.4, one interaction row | **Yes**, as a constraint on the others |
| **U** | unknown | nothing supports a range. The search starts at the default and its neighbours and widens by the rule of 5.3 | the row's Search cell | By the widening rule only |

Nearly every knob is **U**. That is a result of reading the specs, not a gap to be
filled by invention: of the 105 K and D rows (75 D, 30 K) of section 6, **96 are U**, 7
have a derived condition (I07, I11, I12, I18) and 2 have an evidence anchor (E5, 6.1; E1 and
E4 sit on S rows).
A **placeholder** default (N-015) is a starting point and not a range; "calibrated
once something has run" means a record (section 8) names a value and the runs it
rests on.

### 5.2 The grid and the neighbours

STY 7.3.1 asks a default to be STABLE "at the two neighbouring points of the
default's knob on the grid CLB gives it". The grid rule is therefore a **rule
constant of this spec** (FR-CLB-010, FR-CLB-011, FR-CLB-012, FR-CLB-013, FR-CLB-014, FR-CLB-108) and not a choice per run:

Let `b` be the **base value**: the owner's default if it has one, otherwise the value
the pack gives (the knob has no default: `InfoSpeed`, `MoveSpeed`), recorded in the
record. Let `[lo, hi]` be the validation bound of the owner, evaluated with the
other fields as configured. All division is half to even (IND 5.2.2).

- **`geo`**: the levels are `b * 2^k` for `k >= 0` and `rdiv(b, 2^-k)` for `k < 0`,
  kept in `[lo, hi]` and at least 1 (a `geo` grid has no level below 1: **0 is never a
  geo level**, because 0 is usually an off value, 4.3.1), duplicates removed, ascending.
  `b` is always a level. Defined for `b >= 1`; a knob whose base can be 0 is `lin`. The factor is 2: a **placeholder**
  (P). Ratios in basis points and ticks are scale-free enough that a halving and a
  doubling are the unit step of a first look; they are not a claim that the
  response is geometric.
- A `geo` row with the flag `z` (owner default 0 or none): `b` is the pack's non-zero value; where the pack gives none, the knob is *at its off value* (4.3.1) and the grid is the single level 0 plus the first level the pack names, with **`edge:off`** below; a record says so, and the knob is searched only away from off.
- **`lin<s>`**: the levels are `b + k * s` for integer `k`, kept in `[lo, hi]`. The
  step `s` is in the unit and is given in the row.
- **`set:<a/b/c>`**: the levels are the listed values, in the order given (a name, a
  flag, a short enumeration).
- A level is **invalid** when `Validate` rejects the configuration with it (a rule
  that involves another field: 6.4), found by `Check` (EXP 14.3) without a run. An
  invalid level is **skipped**: the neighbour on that side is the next valid level.
- A level equal to the row's **off value** (`off=<v>`, or the owner's `lo` or `hi` for `off=min` or `off=max`) is **not a level of a calibration grid** of a D row: it is cut, and the side is an edge, `edge:off` (the damper is never moved to it, 4.3.1). A default that is itself at its off value is the flag `z` case above; the check of 14 reads the owner's bound, not the row (4.5).
- The **neighbours** of `b` are the nearest valid level below it and the nearest
  above. A side with none is an **edge**, recorded as `edge:bound` (the bound `lo` or
  `hi` was reached), `edge:rule` with the rule that cut it, or `edge:off` (the side below a knob whose base is at its off value, flag `z`; the check reads the owner's bound, not the row). A default with an edge
  side is held to the **one** neighbour it has, and its stability record says so;
  that is not a pass for the missing side and not a fail for it either, and the default is **not called stable** on it (7.4.3).
- A **grid is not a range**: it says where the first look is, not how far the search
  may go (5.3).

### 5.3 The widening rule

The search of a knob starts with its two neighbours (the screen, 7.2). The **explore**
step covers the interval from the second level below `b` to the second level above
(`k` from -2 to +2 of a `geo` grid: a factor of four each side; two steps of a `lin`
grid), clamped to `[lo, hi]` and to the valid levels. If the feasible points of the
choice **touch an end of the interval** that is not a bound, the interval is widened
by two levels on that side and the choice is **made again on a new search block**
(7.1). The search never leaves `[lo, hi]`. Both numbers (two levels, two more) are
placeholders (P).

### 5.4 Ranges that depend on other fields

Some bounds are not constants: `BeliefStockLow` is at most `BeliefStockHigh - 1`;
`MinWorkShare` is at most `ExpectedWorkTicks * 10 000 / TicksPerYear`; `MoveCooldown`
is at most `MaxWelfareWindow`. The row's Links cell names the condition (`I03`,
`I05`), the grid is cut by it as in 5.2, and **a factor with a condition on another
factor is a composite factor** (EXP 3.2.4) or is searched in two steps (the one that
bounds the other first). The test of 11.1 checks that every `I<nn>` a row names
exists.

## 6. The registry: rows

The rows are data. The **fixed cells** are the cells of 4.2. A `-` is an empty
cell. The classes were decided by the order of 4.3 and the `Moves` by the rules of
4.4; **every sign marked `?` is unknown and is what the screen is for**. No cell
holds a default, a bound or a value (4.5). The reading is: *this knob is of this
kind, an owner defines it, a calibration searches it at this stage, and the specs
expect it to move these facts like this*.

### 6.1 Evidence anchors

The five places where a source gives a range or an anchor for a knob. Each is for
another place and time and **calibrates nothing**: it is a sanity range, never a target
(STY 3.3). **E1, E4 and E5 are read directly; E2 and E3 are leads**, not anchors in the
sense of 5.1 (E2's row is read from extracted text with inferred bounds, E3's cited row
holds no such number): no row has `E2` or `E3` as its Search, and both knobs are `U`. A
knob with no anchor here is `U`.

| Anchor | Knob | What the source gives | Evidence row |
|--------|------|-----------------------|--------------|
| E1 | `Role.ExpectedWorkTicks` | annual income as day wage times 250 days is the usual reading (Allen; Humphries and Weisdorf), and medieval labourers worked **under** 150 days a year (Humphries and Weisdorf): two figures, an upper bound below 150 and a usual reading of 250, in **days**, which are ticks only where a tick is a day (`spec-calendar.md` 3.2; the Earth default) | `evidence.md` rows "Annual income is day wage times 250 days" and "Medieval labourers worked under 150 days" |
| E2 | `World.BeliefStep` | an adjusted price changes by an amount uniform in `[0, 2%]` (Lengnick: `vartheta` 0.02, explicit in the extracted text of equation 10 and Table 1; the other bounds of the row are inferred): a **lead**, not an anchor (the row is read from extracted text, not "read directly"), an upper bound of the per-adjustment move of that model (the mean is half), not a range | `evidence.md` row "Firm price rule" (`vartheta` = 0.02 explicit; the other bounds of the row are inferred and unverified) |
| E3 | `World.PriceWindow` | bazaarBot's `lookBack` default of 15 rounds and the paper's profitability window of 8 to 15 rounds (`spec-beliefs.md` 11, and `bazaarbot-reference.md`): a **lead** for the order of the window and no anchor (the row cited lists window statistics and not these numbers), not a range that transfers (the paper's rounds are not ticks) | `evidence.md` row "bazaarBot's algorithm as published" |
| E4 | `Role.TrainingTicks` | apprenticeship usually lasted seven years under an indenture (Justman and van der Beek): a point, to be read in the pack's years (`TicksPerYear`) | `evidence.md` row "Apprenticeship: usually seven years" |
| E5 | `Population.BirthSchedule`, `Population.DeathSchedule` | the **shape**: births flat or rising with income, deaths falling with it, so a stable population sits where they cross, and a higher death schedule raises equilibrium living standards (Clark, chapter 2, read directly). **No rate is given**: surviving children per woman 1.93 to 2.10 is recorded and sets no default | `evidence.md` rows of Clark (2007) chapter 2 |

The rows of `evidence.md` that are a snippet, a summary or marked unverified are
**not** anchors: the capital return of 4% to 10% (`Region.CapitalReturn`), the
Statute of Artificers' three apprentices (`Role.MaxTrainees`), the tool prices and
durabilities, and the carter rates. A knob that rests on one of them is `U`
(N-018).

### 6.2 The knob rows

The tables are grouped by owner so that a reader finds a knob beside the spec that
defines it. Within a table the rows are in the order of the owner's parameter table.

#### 6.2.1 Scale, goods and welfare (SCL, GDS, WEL)

| ID | Key | Class | Owner | Unit | Search | Grid | Stage | Flags | Moves | Loops | Links |
|----|-----|-------|-------|------|--------|------|-------|-------|-------|-------|-------|
| K-SCL01 | Group.Scale | S | SCL 3.1 | count | - | - | C3 | d | P09/Dispersion:? | L13 L33 L34 | I20 |
| K-SCL02 | World.MinAgents | D | SCL 5.2 | count | U | geo | C3 | d off=1 | P09/Dispersion:? | L33 L34 D10 | I20 |
| K-GDS01 | World.BulkFactor | K | GDS 8 | bp | U | geo | - | v1x | - | - | - |
| K-WEL01 | World.WelfareWindow | D | WEL 2.3.1 | ticks | U | geo | C4 | off=min | D03/RoleShares.Swing:+M | L14 L19 L36 D29 | I04 A02 |
| K-WEL02 | World.WindowCycles | D | WEL 2.3.3 | count | U | lin1 | C4 | off=- | D03/RoleShares.Swing:+H | L19 L36 D29 | I04 A02 |
| K-WEL03 | Role.TargetSpread | D | WEL 5.1 | bp | U | lin500 | C4 | off=0 | P03/PriceRange:-S | L14 L15 D08 | I06 A01 |
| K-WEL04 | Role.TargetRatio | K | WEL 5 | bp | U | lin1000 | C4 | chk | M03/HouseholdIncome:0S; M05/StockCover:+S | L14 L46 | I06 |
| K-WEL05 | Role.DefaultRatio | K | WEL 2.2 | bp | U | lin1000 | C4 | - | H01/a1.price_band_bp:? | L53 | - |
| K-WEL06 | Role.ExpectedWorkTicks | S | WEL 4.2 | ticks | E1 | - | C2 | in | D01/Unfed:? | L14 | I05 |
| K-WEL07 | Role.MinWorkShare | D | WEL 5.3 | bp | U | lin1000 | C4 | off=0 | P04/Offset.abs:-M | L14 L15 L51 D13=HeldByFloor | I05 |
| K-WEL08 | World.WithTargetSpread | D | WEL 5.2 | bp | U | lin500 | C4 | off=0 | P03/PriceRange:-S | L14 L15 D08 | I06 A01 |

Notes. **K-SCL01**: the scale is a structural choice of resolution (STY 4.4: facts are stated at weight 1); STY-P09 is its consistency check. **K-WEL03 and K-WEL08**
are one knob in two forms (A01): the role field sets one role's spread, the option
overrides every role's and is in the hash (`spec-world.md` 4.1). Both are D with the
default **at the off value**: a calibration moves it away from 0 and never back
(4.3.1). **K-WEL04**: in a closed economy with upkeep as the only demand the mean
ratio is exactly 10 000 whatever the target says (`spec-viability.md` 4.7, N-054), so
`TargetRatio` is **not** a lever on `M03`; the `0S` says so, and a measured move is a
finding. **K-WEL06** is read by the nominal programme (VIA 3.2, phi) and carries E1.

#### 6.2.2 Agents (AGT)

| ID | Key | Class | Owner | Unit | Search | Grid | Stage | Flags | Moves | Loops | Links |
|----|-----|-------|-------|------|--------|------|-------|-------|-------|-------|-------|
| K-AGT01 | World.CoverTicks | K | AGT 12 | ticks | U | geo | C4 | - | M05/StockCover:+H | L04 L49 | I07 I13 |
| K-AGT02 | World.InputJobs | K | AGT 12 | count | U | lin1 | C4 | - | M05/StockCover:+H | L49 | - |
| K-AGT03 | World.OutputCoverTicks | K | AGT 12 | ticks | U | geo | C4 | - | M05/StockCover:+S | L02 L04 | I10 |
| K-AGT04 | World.FavourFloor | K | AGT 12 | bp | U | lin1000 | C6 | - | M04/HouseholdIncome:? | L49 | - |
| K-AGT05 | World.ReserveTicks | K | AGT 12 | ticks | U | lin1 | C4 | - | M06/s20.ticks:+H | L11 L45 | I09 I18 |
| K-AGT06 | World.SpendMaxShare | D | AGT 12 | bp | U | lin1000 | C6 | off=- | M04/HouseholdIncome:+H | L45 L57 D27 | I18 A04 |
| K-AGT07 | World.SpendHalfTicks | D | AGT 12 | ticks | U | geo | C6 | off=- | M04/HouseholdIncome:-H | L45 L57 D27 | I18 A04 |
| K-AGT08 | World.RestBand | D | AGT 12 | bp | U | lin250 | C4 | off=0 | P03/PriceRange:? | L14 L15 D09 | - |
| K-AGT09 | World.ProduceHaltRatio | D | AGT 12 | bp | U | geo | C4 | off=max | M05/StockCover:+S; P02/Pinned.Share:? | L05 L06 L44 D24=ProductionHalted | I10 |
| K-AGT10 | Role.StartMoney | S | AGT 3.3 | minor | - | - | C2 | - | M06/s20.ticks:-H | L11 L45 | I09 A03 A08 |

Notes. **K-AGT09**: the halt's threshold decides which bound a pinned price reaches
(STB 10, L44) and its counter is `ProductionHalted` (OBS 4.4.6, N-067). The
sign on `StockCover` is STY 10.2's ("`ProduceHaltRatio` down lowers the unsold
stock"). **K-AGT10** is read by a cold start only: a warm-start table that names the
group overrides it (`spec-viability.md` 3.3 item 3).

#### 6.2.3 Beliefs (BEL)

| ID | Key | Class | Owner | Unit | Search | Grid | Stage | Flags | Moves | Loops | Links |
|----|-----|-------|-------|------|--------|------|-------|-------|-------|-------|-------|
| K-BEL01 | World.BeliefStep | D | BEL 11 | bp | U | geo | C4 | in off=- | P03/PriceRange:+S; P04/Offset.abs:? | L01 L04 D02 D21 | I01 A08 |
| K-BEL02 | World.BeliefAdjustProb | D | BEL 11 | bp | U | lin1000 | C4 | off=max | P09/Dispersion:-H | L01 L13 D03 | I03 |
| K-BEL03 | World.BeliefSignificance | K | BEL 11 | bp | U | lin250 | C4 | - | P04/Offset.abs:? | L02 | - |
| K-BEL04 | World.BeliefStockLow | D | BEL 11 | bp | U | lin1000 | C4 | off=- | M05/StockCover:? | L04 L05 D07 | I03 A06 |
| K-BEL05 | World.BeliefStockHigh | D | BEL 11 | bp | U | lin1000 | C4 | off=- | M05/StockCover:? | L04 L06 D07 | I03 I10 A06 |
| K-BEL06 | World.BeliefMaxAge | D | BEL 11 | ticks | U | lin1 | C4 | off=- | P02/Pinned.Share:? | L03 D05 | - |
| K-BEL07 | World.BeliefEvidenceShare | D | BEL 11 | bp | U | lin1000 | C4 | off=- | P04/Offset.abs:+M; S01/s01.recover_ticks:-M | L02 D05 | - |
| K-BEL08 | World.BeliefWidthInit | K | BEL 11 | bp | U | lin500 | C4 | - | H01/a1.price_band_bp:? | L07 | I03 |
| K-BEL09 | World.BeliefWidthMin | D | BEL 11 | bp | U | lin100 | C4 | off=min | P02/Pinned.Share:? | L07 D17 | I03 |
| K-BEL10 | World.BeliefWidthMax | D | BEL 11 | bp | U | lin500 | C4 | off=max | P03/PriceRange:? | L07 D17 | I03 |
| K-BEL11 | World.BeliefWidthStep | K | BEL 11 | bp | U | lin50 | C4 | - | P09/Dispersion:? | L07 | - |
| K-BEL12 | World.CostFloor | D | BEL 11 | bp | U | geo | C4 | chk off=min | P02/Pinned.Share:+S; P08/Turnover.Volume:-S; P02/s07.spoiled_bp:+M | L09 L10 L52 D04=BeliefsUpdated | I02 A13 |
| K-BEL13 | World.CostCeiling | D | BEL 11 | bp | U | geo | C4 | chk off=max | P04/Offset.abs:-S | L52 D04=BeliefsUpdated | I02 A13 |
| K-BEL14 | World.RefBandLow | D | BEL 11 | bp | U | geo | C4 | chk off=min | P02/Pinned.Share:? | L05 L08 D04=BeliefsUpdated | I03 A13 |
| K-BEL15 | World.RefBandHigh | D | BEL 11 | bp | U | geo | C4 | chk off=max | P02/Pinned.Share:? | L05 L08 D04=BeliefsUpdated | I03 A13 |
| K-BEL16 | World.PriceWindow | K | BEL 11 | ticks | U | geo | C4 | - | P07/PriceIndex.Fisher:? | - | - |

Notes. **K-BEL01** sets the **resolution floor** `ceil(10 000 / BeliefStep)` below
which no price is legal (I01) and which the solver rounds its table to (flag `in`):
moving it moves the legal region of the money unit (A08). STY 10.2 states the sign on
the oscillation of a price (STB-L04). **K-BEL07**: the table of `spec-beliefs.md`
5.4 shows that a smaller share removes more of the offset and slows the recovery
(a model), so a larger share leaves more of the offset and recovers faster.
**K-BEL12**: STY 10.2 and `spec-stability.md` 6.2 run 5 give the three signs
(placeholder numbers: 7 000 trades in every tick with 45% spoiled; 10 000 and 9 000
freeze). It is "the most consequential default" of BEL (open question 1 there).
**K-BEL09 and K-BEL10** are D17 with the off values at the bounds of the owner.

#### 6.2.4 Spawner, estates, skills and production (SPN, EST, SKL, PRO)

| ID | Key | Class | Owner | Unit | Search | Grid | Stage | Flags | Moves | Loops | Links |
|----|-----|-------|-------|------|--------|------|-------|-------|-------|-------|-------|
| K-SPN01 | World.ReviewTicks | D | SPN 11 | ticks | U | geo | C4 | off=1 | D03/RoleShares.Swing:? | L36 L61 D33 | A12 |
| K-SPN02 | World.EntryShare | D | SPN 11 | bp | U | geo | C4 | off=max | D03/RoleShares.Swing:+S | L36 L54 L61 D14=EntryLimited | I04 A02 A12 |
| K-SPN03 | World.RoleHysteresis | D | SPN 11 | bp | U | lin500 | C4 | off=0 | D03/RoleShares.Swing:? | L36 L54 L61 D09 | A12 |
| K-SPN04 | World.RoleCooldown | D | SPN 11 | ticks | U | geo | C4 | off=0 | D03/RoleShares.Swing:-M | L36 D34=EntryLimited | I04 A12 |
| K-SPN05 | World.EntryRatio | D | SPN 11 | bp | U | lin1000 | C4 | off=0 | S02/Households:? | L61 L64 D36=EntryLimited | - |
| K-SPN06 | World.ChangePrivation | D | SPN 11 | count | U | lin1 | C5 | off=- | D01/Unfed:+H | L35 L50 L62 D35 | I08 |
| K-SPN07 | World.RemovePrivation | D | SPN 11 | ticks | U | geo | C5 | off=- | D01/Unfed:+S; S02/Households:? | L35 L50 L62 D35 | I08 |
| K-SPN08 | World.DeathStep | D | SPN 11 | bp | U | geo | C5 | off=max | S02/Households:? | L35 D35=EntryLimited | - |
| K-SPN09 | World.ExitShare | D | SPN 11 | bp | U | geo | C5 | off=max | S02/s14.recover_ticks:? | L35 L83 D35=EntryLimited | - |
| K-SPN10 | World.Renewal | K | SPN 11 | bp | U | lin2500 | C5 | - | S02/Households:+H | L61 L79 D48 | - |
| K-SPN11 | World.EntryPoolShare | D | SPN 11 | bp | U | lin1000 | C5 | off=max | D01/Unfed:+H | L39 L63 D37=EntryLimited | I21 |
| K-EST01 | Region.EstateMarkdown | K | EST 3 | bp | U | lin250 | C5 | - | S02/Households:? | L38 | - |
| K-EST02 | Region.PoolRelease | D | EST 2.6 | bp | U | geo | C5 | off=0 | D01/Unfed:-H | L39 L63 D11 D12 | I21 A14 |
| K-EST03 | Region.ReliefBelow | D | EST 2.6 | bp | U | lin1000 | C5 | off=0 | D01/Unfed:-H | L39 L40 D11 | A14 |
| K-SKL01 | Role.TrainingTicks | S | SKL 4.1 | ticks | E4 | - | C1 | - | D03/RoleShares.Swing:? | L36 L37 L41 | I20 I23 |
| K-SKL02 | Role.MaxTrainees | D | SKL 4.4 | count | U | lin1 | C6 | off=max | D03/RoleShares.Swing:? | L37 L41 D20 | I23 |
| K-SKL03 | Role.TeachingLoad | K | SKL 4.2 | bp | U | lin500 | C6 | - | D03/RoleShares.Swing:? | L37 | I23 |
| K-SKL04 | Region.CapitalReturn | K | SKL 2.5 | bp | U | lin500 | C6 | - | D03/RoleShares.Swing:? | L36 L37 | - |
| K-PRO01 | Recipe.TraineeYield | S | PRO 8.2 | bp | - | - | C2 | - | D03/RoleShares.Swing:? | L37 L41 | I23 |

Notes. **K-SPN02**: `EntryShare` at 0 turns role change **off** (a structural choice);
at 10 000 it removes the cap (its off value). The condition is on the product with the
effective window (A02, `spec-stability.md` 10: "`EntryShare * T` below about 2 000");
the sign on the swing is STY 10.2's. **K-SPN03**: SPN 16.1 measures hysteresis alone at the
same amplitude as with a 10% share, so its sign is `?` until a screen reads it. **K-SPN10** is ignored while the population
module is on (`spec-population.md` 4.6; D48). **K-SKL01** is class S: the length of a
traineeship defines a barrier; E4 is its anchor.

#### 6.2.5 Demand, money, population and resources (DEM, MON, POP, RES)

| ID | Key | Class | Owner | Unit | Search | Grid | Stage | Flags | Moves | Loops | Links |
|----|-----|-------|-------|------|--------|------|-------|-------|-------|-------|-------|
| K-DEM01 | Want.Elasticity | D | DEM 9 | bp | I18 | lin5000 | C6 | off=0 | S23/s23.band_bp:-M | L14 L57 D30=WantShortfall | I18 |
| K-DEM02 | Want.Ceiling | D | DEM 9 | bp | U | lin5000 | C6 | off=min | S23/s23.band_bp:-M | L14 L57 D30=WantShortfall | I18 |
| K-DEM03 | Want.Substitution | K | DEM 9 | count | U | lin1 | C6 | - | M04/HouseholdIncome:? | L58 | - |
| K-MON01 | MoneyFlow.Limit | D | MON 4.4.2 | minor | U | geo | C8 | z off=0 | S05/Sectors.M:? | L30 L69 L72 D41=MoneyFlowEnded | - |
| K-MON02 | MoneyFlow.Exempt | D | MON 5.3 | minor | U | geo | C8 | z off=0 | D01/Unfed:-S | L71 D43 | - |
| K-MON03 | MoneyFlow.RateBP | D | MON 4.2 | bp | U | geo | C8 | z off=0 | S05/Sectors.M:-H | L70 D42 | - |
| K-POP01 | Population.BirthSchedule | D | POP 8 | list | E5 | - | C9 | off=- | W02/Households:? | L75 L76 D46 | I11 A15 |
| K-POP02 | Population.DeathSchedule | D | POP 8 | list | E5 | - | C9 | off=- | W02/Households:-E | L75 L77 D46 | I11 A15 |
| K-POP03 | Population.MinHouseholds | D | POP 8 | count | U | lin1 | C9 | off=0 | W06/Households:? | L82 D49=PopulationChanged | I11 |
| K-POP04 | Population.BirthStake | D | POP 8 | bp | U | lin2500 | C9 | off=0 | W06/Households:-S | L76 L77 D51=PopulationChanged | I09 A03 |
| K-POP05 | RegionPopulation.BirthScale | K | POP 8 | bp | I11 | geo | C9 | - | W02/WelfareDistribution.Mean:-H; W02/Households:+H | L75 | I11 A15 |
| K-POP06 | RegionPopulation.DeathScale | K | POP 8 | bp | I11 | geo | C9 | - | W02/WelfareDistribution.Mean:+E; W02/Households:-E | L75 L77 | I11 A15 |
| K-POP07 | RegionPopulation.HouseholdCap | D | POP 8 | count | U | geo | C9 | z off=0 | W06/Households:+H | L73 D50=PopulationChanged | I11 |
| K-RES01 | Resource.RateBP | K | RES 12 | bp | I12 | geo | C9 | in z | W02/Households:+H | L87 L88 | I12 |
| K-RES02 | Resource.Refuge | D | RES 12 | units | U | geo | C9 | z off=0 | W03/Households:? | L88 L89 D56 | I12 |
| K-RES03 | Resource.PlanShareBP | D | RES 12 | bp | I12 | lin1000 | C9 | in off=- | W02/Households:? | L87 L88 D58 | I12 |
| K-RES04 | Resource.Abundance | D | RES 4.7 | list | U | - | C9 | in off=flat | W02/WelfareDistribution.Mean:? | L87 L91 D54 | I12 A16 |
| K-RES05 | Resources.HorizonYears | D | RES 12 | years | U | geo | C2 | in off=- | - | L89 D58 | I12 |

Notes. **K-DEM01 and K-DEM02**: a luxury is stable only with `Elasticity` above
10 000 and about 10% budget headroom (`spec-demand.md` 6.3), a **derived condition**
on `Elasticity` (I18, Search of K-DEM01); the headroom is a property of the money
held above the reserve (`SpendMaxShare`, `SpendHalfTicks`, `ReserveTicks`), not of
`Ceiling`, whose Search is `U`. **K-POP01 and K-POP02**:
the *shape* is E5 and no rate is given; a calibration moves a schedule only through
its two region multipliers K-POP05 and K-POP06 (A15), as a composite of its knots is
not a grid. The sign on `DeathScale` is Clark's ("what raises the death rate raises
living standards", read directly), on `BirthScale` the same logic, hand-derived
(`H`). **K-POP04**: STY 10.2. **K-RES04**: a flat curve is the default and so the
damper's off value (4.3.1).

#### 6.2.6 Information, transport, merchants and the open layer (INF, TRN, MER, EXT)

| ID | Key | Class | Owner | Unit | Search | Grid | Stage | Flags | Moves | Loops | Links |
|----|-----|-------|-------|------|--------|------|-------|-------|-------|-------|-------|
| K-INF01 | Information.InfoSpeed | K | INF 8 | units | U | geo | C7 | - | P06/PriceGap.GapAfterFreight:-M | L84 L93 L94 | I13 A07 |
| K-INF02 | Information.RumourErrorBP | D | INF 8 | bp | U | geo | C7 | d off=0 | P06/PriceGap.GapAfterFreight:+S | L93 L108 D60 | - |
| K-INF03 | Information.RumourMaxAge | D | INF 8 | ticks | U | geo | C7 | off=max | P06/PriceGap.GapAfterFreight:? | L93 L108 D61=RumoursServed | I13 |
| K-MER01 | Merchants.MaxOpen | D | MER 13 | count | U | geo | C7 | off=max | P06/PriceGap.GapAfterFreight:? | L113 L114 D84=ContractRound | - |
| K-MER02 | Merchant.LotUnits | D | MER 3.2 | count | U | lin1 | C7 | off=max | P03/PriceRange:? | L116 D80=MerchantsPlanned | - |
| K-MER03 | Merchant.FeeShareSpreadBP | D | MER 3.2 | bp | U | lin500 | C7 | off=0 | P06/PriceGap.GapAfterFreight:-S | L84 L93 D81 | I14 A10 |
| K-MER04 | Merchant.MaxSourceAge | D | MER 3.2 | ticks | U | lin1 | C7 | off=max | P06/PriceGap.GapAfterFreight:+S | L94 D82=MerchantsPlanned | - |
| K-MER05 | Merchant.TimeCostBP | K | MER 3.2 | bp | U | lin2500 | C7 | - | P06/PriceGap.GapAfterFreight:? | - | - |
| K-MER06 | Merchant.ConsigneeCutBP | K | MER 3.2 | bp | U | lin250 | C7 | - | P06/PriceGap.GapAfterFreight:? | - | - |
| K-MER07 | Merchant.TakeMarginBP | K | MER 3.2 | bp | U | lin500 | C7 | - | P06/PriceGap.GapAfterFreight:? | L113 | - |
| K-MER08 | Role.MerchantFeeShare | D | CAR 2.2 | bp | U | lin1000 | C7 | off=max | P06/PriceGap.GapAfterFreight:? | L16 L84 L116 D86=MerchantsPlanned | I14 A10 |
| K-MER09 | Merchant.TakeUnits | K | MER 3.2 | units | U | geo | C7 | - | P06/PriceGap.GapAfterFreight:? | L113 | - |
| K-EXT01 | OutsideGate.ScaleBP | K | EXT 3.2.4 | bp | U | lin1000 | C8 | in | S05/Sectors.M:? | L96 L99 | - |
| K-EXT02 | OutsideLine.FloorBP | D | EXT 6.3 | bp | U | geo | C8 | off=min | P03/PriceRange:? | L100 D66=OutsideTraded | I17 |
| K-EXT03 | OutsideLine.CeilingBP | D | EXT 6.3 | bp | U | geo | C8 | off=max | P03/PriceRange:? | L100 D66=OutsideTraded | I17 |
| K-EXT04 | OutsideLine.WalkRevertBP | K | EXT 6.4 | bp | U | lin500 | C8 | - | P03/PriceRange:? | L100 | - |
| K-EXT05 | OutsideLine.WalkBoundBP | D | EXT 6.4 | bp | U | geo | C8 | z off=max | P03/PriceRange:? | L100 D66=OutsideTraded | I17 |
| K-EXT06 | OutsideSide.MarginBP | D | EXT 6.6 | bp | U | geo | C8 | in z off=- | P03/PriceRange:+M | L96 L99 D62 | I17 |
| K-EXT07 | OutsideSide.Capacity | D | EXT 7.1 | units | U | geo | C8 | in z off=max | P03/PriceRange:-M | L97 D63=OutsideTraded | - |
| K-EXT08 | OutsideSide.TotalUnits | D | EXT 7.3 | units | U | geo | C8 | z off=0 | S04/UpkeepShortfall:? | L101 D63=OutsideEnded | - |
| K-EXT09 | OutsideSide.TotalValue | D | EXT 7.3 | minor | U | geo | C8 | z off=0 | S04/UpkeepShortfall:? | L98 D63=OutsideEnded | - |

Notes. **K-INF02**: STY 10.2 and `spec-information.md` 11.2 state that a larger
independent error widens the converged gap, a *model* of an invented shipper rule
(placeholder numbers), and that a spread of the shippers' own thresholds
(K-MER03, A10) settles the gap where the error alone does not. **K-INF01** is the
speed of news in the pack's distance unit (A07). **K-MER08** is `MerchantFeeShare`,
owned by CAR 2.2 and read by MER; with the default spread MER 3.2 rejects both
extremes (`ErrOutOfBounds`), so they are reachable only with a spread of 0. **K-EXT06**: the corridor widens with the margins (EXT 6.6, 19.2); a pair of
margins that sum below `MinQuoteSpreadBP` is `ErrCrossedQuotes`, so no side is moved to
0 alone. The model of EXT 19.2 shows a corridor confining a cobweb *provided the capacity covers the swing*, which
is K-EXT07's sign.

#### 6.2.7 Own work and movement (SUB, MIG)

| ID | Key | Class | Owner | Unit | Search | Grid | Stage | Flags | Moves | Loops | Links |
|----|-----|-------|-------|------|--------|------|-------|-------|-------|-------|-------|
| K-SUB01 | Subsistence.EnterTicks | D | SUB 12 | ticks | I07 | lin1 | C5 | off=- | D01/Unfed:? | L102 L103 L104 L106 D67=OwnUseChanged | I07 |
| K-SUB02 | Subsistence.ExitTicks | D | SUB 12 | ticks | I07 | lin1 | C5 | off=- | D01/Unfed:? | L102 L105 L106 D67=OwnUseChanged | I07 |
| K-SUB03 | Subsistence.RetryJobs | D | SUB 12 | count | U | geo | C5 | off=0 | D01/Unfed:? | L102 L103 L106 D69=OwnUseChanged | I15 |
| K-MIG01 | Migration.MoveReviewTicks | D | MIG 11 | ticks | U | geo | C7 | d off=1 | G01/Households:? | L107 D72 | I16 |
| K-MIG02 | Migration.MoveBelow | K | MIG 11 | bp | U | lin1000 | C7 | d | G01/Households:? | L107 | - |
| K-MIG03 | Migration.MoveGain | D | MIG 11 | bp | U | geo | C7 | off=- | G01/Households:? | L107 L108 D74=MigrationReviewed | A05 I16 |
| K-MIG04 | Migration.MoveWidth | D | MIG 11 | bp | U | geo | C7 | off=- | G01/Households:? | L107 L108 D74=MigrationReviewed | A05 |
| K-MIG05 | Migration.MoveCooldown | D | MIG 11 | ticks | U | geo | C7 | d off=- | G01/Households:? | L107 L111 D73=MigrationReviewed | I04 |
| K-MIG06 | Migration.MoveHorizon | D | MIG 11 | ticks | U | geo | C7 | off=max | G01/Households:? | L107 D77=MigrationReviewed | I13 |
| K-MIG07 | Migration.MoveOutShare | D | MIG 11 | bp | U | geo | C7 | off=max | G01/Households:? | L107 L111 D75=MigrationReviewed | I16 |
| K-MIG08 | Migration.MoveInShare | D | MIG 11 | bp | U | geo | C7 | off=max | G01/Households:? | L107 L108 L109 D76=MigrationReviewed | I16 |
| K-MIG09 | RegionMove.OutShare | D | MIG 11.2 | bp | U | geo | C7 | off=max | G01/Households:? | L107 L111 D75=MigrationReviewed | I16 |
| K-MIG10 | RegionMove.InShare | D | MIG 11.2 | bp | U | geo | C7 | off=max | G01/Households:? | L108 L109 D76=MigrationReviewed | I16 |

Notes. The module has **no STY fact of its own**: its scenarios S61 to S66 are
exact checks and bands of `spec-stylised-facts.md` 5.11, so its knobs' `Moves` name
the umbrella fact **G01** (a healthy start stays healthy), which a migration cycle
breaks by emptying a region (L111). **K-SUB01 and K-SUB02** are a **pair** (I07): the
mode ends only when stock covers `ExitTicks` above `EnterTicks`, and `ExitTicks` is
bounded above by what purchases can reach, `CoverTicks`.

#### 6.2.8 Solve options (VIA)

The solve options of `spec-viability.md` 11 are **not `Config` fields** and are not
rows of the parameter-owner table (`spec-config.md` 9.2, FR-CFG-047): the registry
test of `testing.md` 6.7 requires a field behind every owner row. They are rows of
this registry, with the scope `Solve option`, and the coverage check of 4.8 does not
look for them in the owner table. This resolves a disagreement between
`spec-viability.md` 11 (which says they "are rows in the parameter-owner table") and
`spec-config.md` 9.2 (which says they are not), in favour of CFG; section 22 lists it.

| ID | Key | Class | Owner | Unit | Search | Grid | Stage | Flags | Moves | Loops | Links |
|----|-----|-------|-------|------|--------|------|-------|-------|-------|-------|-------|
| K-VIA01 | Solve option.StockCoverTicks | K | VIA 11 | ticks | U | lin1 | C2 | tbl | H04/s17.spike_factor:? | L22 L26 L53 | I19 |
| K-VIA02 | Solve option.MoneyBufferTicks | K | VIA 11 | ticks | U | lin5 | C2 | tbl | H02/h02.money_bp:? | L11 L53 | I19 |
| K-VIA03 | Solve option.ResidualWarnBP | P | VIA 11 | bp | - | - | C2 | - | - | - | - |
| K-VIA04 | Solve option.MaxPivots | P | VIA 11 | count | - | - | - | - | - | - | - |
| K-VIA05 | Solve option.Lanes | S | VIA 11 | list | - | - | C1 | in | - | - | - |

Notes. `StockCoverTicks` and `MoneyBufferTicks` do not change the solved flows; they
shape **the table the solver writes** (VIA 8.3, 8.4), so a change regenerates the
table and the table is data that is hashed (`spec-viability.md` 7.4). They carry the
flag `tbl` (4.2.1): they are inputs of the solver, so not `out`, and the effect is on the table, not on a rule. **K-VIA03** changes only what
the report lists, and **K-VIA04** is a work cap that changes no result (class P by rule (1) of 4.3). **K-VIA05** (`Lanes`) is read by the programme (VIA 9.4) and is structure.

### 6.3 Structure groups

The S and P fields of a scope that need no row of their own. A group row names the
**scope token** (4.7) and the **fields** it covers. Its class is S or P, or, for the
elements of a list that has a knob row, the class of that list (the knots of a
schedule, the knots of an abundance curve). The groups are the registry's answer to
"which fields of the owner table are not knobs": **identity and topology** (stage C1),
**quantities of the setting** (stage C2: the solver checks them) and **units**.

| ID | Scope | Fields | Class | Owner | Stage | Flags |
|----|-------|--------|-------|-------|-------|-------|
| S-CAL01 | Calendar | HoursPerDay, DaysPerWeek, Months, TicksPerDay, DaysPerTick, StartDay, Seasons, SeasonStart | S | CAL 3.1 | C1 | - |
| S-GDS01 | Commodity | ID, Classes, Raw | S | GDS 8 | C1 | - |
| S-GDS02 | Commodity | Mass, Bulk, ShelfLife, Durability | S | GDS 8 | C2 | in |
| S-GDS03 | Commodity | ReferencePrice | S | GDS 8 | C2 | out |
| S-CUR01 | Region | ID | S | CUR 5 | C1 | - |
| S-CUR02 | Region | ReferenceBasket | S | CUR 5 | C2 | - |
| S-AGT01 | World | Upkeep | S | AGT 5.2 | C2 | in |
| S-AGT02 | Region | ExtraUpkeep | S | AGT 5.2 | C2 | in |
| S-AGT03 | World | BasketTolerance | S | AGT 12 | C1 | - |
| S-AGT04 | Role | ID, Recipes, Trades, Strategy | S | AGT 3.3 6.6 | C1 | in |
| S-AGT05 | Role | StartStock | S | AGT 3.3 | C2 | in |
| S-SPN01 | World | Spawner | S | SPN 3.1.3 | C1 | - |
| S-SPN02 | Role | Closed | S | SPN 11 | C1 | - |
| S-CLR01 | World | Clearing | S | CLR 8 | C1 | - |
| S-WEL01 | Role | Unskilled, InKind | S | WEL 4.2 6 | C1 | - |
| S-PRO01 | Recipe | ID, Inputs, Outputs, Requires, Draws, Sited, TraineeEligible | S | PRO 3.1 6.2 8 | C1 | in |
| S-PRO02 | Recipe | Duration | S | PRO 5 | C2 | in |
| S-PRO03 | Recipe output | CostShare, Discard | S | PRO 3.2 7.2 | C1 | in |
| S-PRO04 | Recipe tool | WearPerUse | S | PRO 4.4 | C2 | in |
| S-PRO05 | Region and recipe | YieldProfile | S | RES 7 | C2 | in |
| S-RES01 | Yield | Region, Recipe, All, Seasons | S | RES 7 | C2 | in |
| S-SCL02 | Group | Households, Region, Role | S | SCL 3.1 | C2 | in d |
| S-REG01 | Config section | Edges, RegionPrices | S | REG 4.1 5.2 | C1 | - |
| S-REG02 | Edge | From, To | S | REG 4.1 | C1 | - |
| S-REG03 | Edge | Distance | S | REG 4.1 | C2 | in |
| S-REG04 | RegionPrice | Region, Commodity | S | REG 5.2 | C1 | - |
| S-REG05 | RegionPrice | Price | S | REG 5.2 | C2 | out |
| S-VIA01 | Config section | WarmStart | S | VIA 8.1 | C2 | out |
| S-VIA02 | WarmStart | Prices, Wages, Groups, Pools | S | VIA 8.1 | C2 | out |
| S-DEM01 | Demand | Wants | S | DEM 9 | C1 | - |
| S-DEM02 | Want | Items, Quantity, Period, Rank | S | DEM 9 | C2 | in |
| S-DEM03 | Want item | Commodity, Weight | S | DEM 9 | C2 | in |
| S-DEM04 | Taste | Want, Scale | S | DEM 9 | C2 | in |
| S-MON01 | Modules.MoneySupply | Flows | S | MON 4.1 | C1 | - |
| S-MON02 | MoneyFlow | ID, Kind, Target, Amount, Level, Per, Start, Until, NotIncome | S | MON 4 7.2 | C8 | - |
| S-POP01 | Knot | Ratio, Rate | K | POP 8 | C9 | - |
| S-POP02 | RegionPopulation | Region | S | POP 8 | C1 | - |
| S-TRN01 | Transport | Modes | S | TRN 3.1 | C1 | - |
| S-TRN02 | Mode | ID | S | TRN 3.1 | C1 | - |
| S-TRN04 | Mode | Crew, Vehicle, WearPerTick, Feed | S | TRN 3.1 | C2 | - |
| S-TRN03 | Mode | Speed, UnitMass, UnitBulk, Haul, HaulSpan | S | TRN 3.1 | C2 | in |
| S-RES02 | Resources | Resources, Endowments | S | RES 3.3 | C1 | - |
| S-RES03 | Resource | ID, Shape, RatePerYear, Capacity, InitialBP, Seasons | S | RES 12 4.5 | C2 | in |
| S-RES04 | AbundanceKnot | Share, Factor | D | RES 4.7 | C9 | in |
| S-RES05 | Endowment | Region, Resource, Capacity, Initial, RegenScaleBP | S | RES 3.3.1 | C2 | in |
| S-RES06 | SeasonRegen | Season, BP | S | RES 4.5 | C2 | in |
| S-CFG01 | Modules | Information, Resources, Transport, External, Merchants, Migration, Population, Subsistence | S | INF 8; RES 3.3; TRN 3.1; EXT 3.1; MER 13; MIG 11; POP 8; SUB 3.2 | C1 | - |
| S-INF01 | Role | Informed | S | INF 6.1 | C1 | d |
| S-EXT01 | External | Nodes | S | EXT 3.1 | C1 | - |
| S-EXT02 | OutsideNode | Region, Lines, Gates | S | EXT 3.1 3.3 3.2.4 | C1 | - |
| S-EXT03 | OutsideGate | Region | S | EXT 3.2.4 | C1 | - |
| S-EXT04 | OutsideLine | Commodity, Buy, Sell | S | EXT 3.3 | C1 | - |
| S-EXT05 | OutsideLine | Base, PriceBySeason, TrendBP | S | EXT 6.1 6.2 6.3 | C8 | in |
| S-EXT06 | OutsideLine | WalkStepBP | S | EXT 6.4 | C8 | d |
| S-EXT07 | OutsideSide | Window | S | EXT 7.2 | C8 | in |
| S-EXT08 | SeasonFactor | Season, BP | S | EXT 6.2 | C8 | in |
| S-SUB01 | Subsistence | Recipes, Roles | S | SUB 3.2 | C1 | - |
| S-SUB02 | RoleOwnUse | Role, Recipes | S | SUB 3.2 | C1 | - |
| S-MIG01 | Migration | MoveSpeed, Regions | S | MIG 6.4 11.2 | C2 | - |
| S-MIG02 | RegionMove | Region | S | MIG 11.2 | C1 | - |
| S-MIG03 | Role | Fixed | S | MIG 3.1 | C1 | d |
| S-MER01 | Merchant | Modes, Consigns | S | MER 3.2 | C1 | - |
| S-MER02 | Role | Merchant | S | MER 3.2 | C1 | - |
| S-PRS01 | Presentation | Label, UnitLabel, DistanceLabel | P | CFG 8.3 | - | - |

Notes. **S-GDS03, S-CUR02, S-REG05, S-VIA01, S-VIA02** are `out`: the table of a
pack is **solver output** and is regenerated, never written by hand (REF position 4,
FR-REF-007). **S-PRS01** is the class P group: the presentation-only fields
(`spec-config.md` 8.3), which are not rows of the owner table and are not checked
against it (the coverage check is one-directional for them). **S-SCL02** carries `d`
because the households of a group fix the number of agents and so the number of
creation draws (WLD 7.3, D1 and D10). **S-MIG01**: `MoveSpeed` is a *gauge* (A07). **Fields that a damper entry also names** are S by rule (2) of 4.3, which is applied before rule (3): `Role.Closed` (D36), a want's `Rank` (D31) and `OutsideSide.Window` (D65) define identity, order and the memory of a line, and a calibration does not move them (C1 or C2). **S-AGT03** (`BasketTolerance`) only decides whether a configuration is accepted (`ErrBasketMismatch`) and changes no result; it is kept S at C1 with the reference basket, not P, because it is in the hash. **S-TRN04** is a quantity staged at C2 with the other quantities of the mode.

### 6.4 Interactions: the pairs that must move together

A row is a statement the specs already make, with where it is enforced. The
registry only **collects** them so that a calibration reads them in one place and a
knob row can link to them; none is invented here. The *Kind* is `rule` (validation
rejects a violation: an error), `warn` (a warning), `cond` (a condition the owner
derives that no rule enforces) or `gauge` (a pair of units that must be read
together).

| ID | Statement | Kind | Where | Knobs |
|----|-----------|------|-------|-------|
| I01 | The cheapest tradable commodity's effective reference price in every region is at least the resolution floor `ceil(10 000 / BeliefStep)`: a lower `BeliefStep` raises the floor, and the prices (or the money unit, A08) must follow | rule | `spec-config.md` 7.4.1; FR-BEL-029 | BeliefStep, ReferencePrice, RegionPrice.Price |
| I02 | `CostFloor` is at most `CostCeiling`; `CostFloor` of 10 000 or more draws `cost_floor_at_full_cost`; a seed outside the cost band is reported (`SeedBelowCostFloor`, `SeedAboveCostCeiling`) | rule, warn | `spec-beliefs.md` 11; `spec-viability.md` 8.2.3 | CostFloor, CostCeiling |
| I03 | `BeliefWidthMin` is at most `BeliefWidthInit` is at most `BeliefWidthMax`, which is within its bound in the owner's table; `BeliefStockLow` is below `BeliefStockHigh`; `RefBandLow` is at most 10 000 and `RefBandHigh` at least 10 000; `BeliefAdjustProb` is at least 1 | rule | `spec-config.md` 7.4.1; FR-BEL-027, FR-BEL-028 | the BEL widths, stock band and bands |
| I04 | The effective welfare window of a role is `max(WelfareWindow, WindowCycles * longest Duration)` over the role's own-use and market jobs and, for a merchant, the trip terms (`spec-welfare-and-wages.md` 2.3.3), and is at most `MaxWelfareWindow`; `RoleCooldown` and `MoveCooldown` are at most `MaxWelfareWindow`; migration's settling uses the longer of `MoveCooldown` and the agent's window; the occupation condition is on `EntryShare` times that window (`entry_share_window` above 2 500) | rule, warn, cond | `spec-welfare-and-wages.md` 2.3.3; `spec-spawner.md` 12; `spec-migration.md` 5.2; `spec-stability.md` 10 | WelfareWindow, WindowCycles, Duration, EntryShare, RoleCooldown, MoveCooldown |
| I05 | `ExpectedWorkTicks` is between 1 and `TicksPerYear` (an error, never scaled); `MinWorkShare` is at most `ExpectedWorkTicks * 10 000 / TicksPerYear` | rule | `spec-config.md` 7.4.1; FR-WEL-028, FR-CFG-048 | ExpectedWorkTicks, MinWorkShare, the calendar |
| I06 | The lowest per-agent target `TargetRatio * (10 000 - TargetSpread) / 10 000` is at least 10 000; `target_above_surplus` (VIA) and `target_above_demand` (DEM) compare a target with the pack's own ceiling | rule, warn | `spec-welfare-and-wages.md` 5.1; `spec-viability.md` 10.2; `spec-demand.md` 7 | TargetRatio, TargetSpread, WithTargetSpread |
| I07 | `EnterTicks` is below `ExitTicks` (`ErrOwnUseBand`); `ExitTicks` of 3 is the highest purchases alone reach at `CoverTicks` 3, so `ExitTicks` and `CoverTicks` move together | rule, cond | `spec-subsistence.md` 5.3, 5.3.3, 12 | EnterTicks, ExitTicks, CoverTicks |
| I08 | `ChangePrivation` is below `RemovePrivation` (`ladder_order`) | rule | `spec-config.md` 7.4.1; FR-SPN-037, FR-SPN-038 | ChangePrivation, RemovePrivation |
| I09 | A start below `ReserveTicks` of upkeep cost cannot reach the reserve in a closed economy and draws `reserve_exceeds_money`; a born household brings `StartMoney * BirthStake / 10 000` | warn, cond | `spec-agents.md` FR-AGT-060; `spec-population.md` 4.3 | StartMoney, ReserveTicks, BirthStake |
| I10 | `ProduceHaltRatio` is at least 10 000; it and `BeliefStockHigh` are shares of the same output target, and which of the two is reached first decides which bound a pinned price reaches | rule, cond | `spec-agents.md` 7.1.5, 12; `spec-stability.md` 10 (L44, S16) | ProduceHaltRatio, BeliefStockHigh, OutputCoverTicks |
| I11 | The schedules cross at the replacement ratio `r*`, which must be at most the closed ceiling of the pack (`replacement_above_closed_ceiling`); gain times delay is bounded (`population_gain`); `MinHouseholds` and `HouseholdCap` bound the group | warn, cond | `spec-population.md` 5.3, 5.4, 9.2 | BirthSchedule, DeathSchedule, BirthScale, DeathScale, MinHouseholds, HouseholdCap |
| I12 | A `logistic` plan share is above the maximum-yield point (5 000); a `refill` stock closing at least 0.16 of its gap a year approaches monotonically; `resource_overshoot` and `resource_unstable_plan` warn | cond, warn | `spec-resources.md` 8.3, 11 | PlanShareBP, RateBP, Refuge, Abundance, HorizonYears |
| I13 | News and journeys are read in one distance unit: `RumourMaxAge` is at least the longest lag (`rumour_horizon_short`); a journey as long as `MoveHorizon` is no destination; the shortest journey is 1 or 2 ticks and the longest under `CoverTicks` in the migration overlay | warn, cond | `spec-information.md` 5.2, 5.3; `spec-migration.md` 6.4, 15.3 | InfoSpeed, RumourMaxAge, MoveSpeed, MoveHorizon, Distance, Speed |
| I14 | `FeeShareSpreadBP` is at most `MerchantFeeShare`, and the share plus the spread is at most 10 000 (`ErrOutOfBounds`, `spec-merchants.md` 14.1); the bars are a separate warning (`merchant_spread_narrow`, 13). `MaxSourceAge` is a price age and `RumourMaxAge` a rumour age: the two are not linked by a rule | rule, warn | `spec-merchants.md` 3.2, 13, 14.1 | FeeShareSpreadBP, MerchantFeeShare |
| I15 | A household with 1-tick own-use jobs tests the market every `RetryJobs` ticks, one with 30-tick jobs every thirty times that: `RetryJobs` is read against the longest own-use `Duration` | cond | `spec-subsistence.md` 11, 12 | RetryJobs, Duration |
| I16 | `MoveOutShare` times the longest effective welfare window above 3 times `MoveGain` draws `migration_flow_window`; a flow that no cap bounds draws `migration_uncapped` (`MoveReviewTicks` and the caps) | warn | `spec-migration.md` 7.3, 7.5, 12.3 | MoveOutShare, MoveInShare, MoveGain, MoveReviewTicks |
| I17 | An outside line has `FloorBP` at most 10 000 at most `CeilingBP`, `WalkStepBP` at most `WalkBoundBP`, and margins that sum to at least `MinQuoteSpreadBP`; a walk faster than beliefs draws `walk_faster_than_beliefs` | rule, warn | `spec-external-sector.md` 6.3, 6.4, 6.6, 13 | FloorBP, CeilingBP, WalkStepBP, WalkBoundBP, MarginBP |
| I18 | A luxury's price loop is stable only with `Elasticity` above 10 000 and about 10% budget headroom: the money held above the reserve, which is `SpendMaxShare`, `SpendHalfTicks` and `ReserveTicks`, not `Ceiling` (a model) | cond | `spec-demand.md` 5.1, 6.3 | Elasticity, SpendMaxShare, SpendHalfTicks, ReserveTicks |
| I19 | A change of any `in` knob makes the `WarmStart` table stale: the experiment regenerates it per point (`Prepare`) or says it is a perturbation | gauge | `spec-experiments.md` 3.3.3, EXP-A5 | every `in` row |
| I20 | A group's weight is at most `MaxWeight`; a role with `TrainingTicks` above 0 needs groups of scale 1; a group with a weight above 1 whose agents fall below `MinAgents` draws `group_below_floor` | rule, warn | `spec-scale.md` 3, 5.2; `spec-skill-barriers.md` 4.1.1 | Scale, MinAgents, Households, TrainingTicks |
| I21 | Relief and entry draw on one pool in time: entrants take at most `EntryPoolShare` of it, relief is `PoolRelease` of it, and the default keeps at least half for relief | cond | `spec-spawner.md` 7.3; `spec-estates.md` 2.6; N-082 | EntryPoolShare, PoolRelease |
| I22 | A region's `ReferenceBasket` equals the cost of the baseline `Upkeep` plus its `ExtraUpkeep` at its effective reference prices within `BasketTolerance` (`ErrBasketMismatch`) | rule | `spec-config.md` 7.4.1; `spec-currency.md` 5.3 | Upkeep, ExtraUpkeep, ReferencePrice, ReferenceBasket |
| I23 | The traineeship throttle is `trainers * MaxTrainees / TrainingTicks`; a trainer's gain per trainee-tick depends on `TraineeYield`, `TeachingLoad` and the wage; training pays only above a break-even `TraineeYield` | cond | `spec-skill-barriers.md` 4.3, 4.4, 4.9; `spec-production.md` 8.3 | TrainingTicks, MaxTrainees, TeachingLoad, TraineeYield |

### 6.5 Aliases and gauges: what the facts cannot tell apart

An **alias group** is a set of knobs that enter a run only through one combination:
the facts see the combination, never the members, so a calibration searches the
**combination** (as a composite factor, EXP 3.2.4) and fixes the members by
structure. A **gauge** is a unit whose choice changes no result up to rounding and
the legal region: a pack fixes it once, in stage C1, and no calibration moves it.
Each row is a hand derivation from the rule text (`H`), not a measurement; section 10
gives the integer test that confirms or refutes an alias on a screen.

| ID | Kind | Knobs | What the facts see | What a calibration does |
|----|------|-------|--------------------|-------------------------|
| A01 | alias | `Role.TargetSpread` (per role) and `World.WithTargetSpread` (every role) | one spread per role; the option overrides the field | sweeps the option for a world-wide value and the role field only for a role-specific one; never both in one point |
| A02 | alias | `EntryShare` and the effective window `T` (`WelfareWindow`, `WindowCycles`, the longest `Duration`) | the product `EntryShare * T` (the condition of `entry_share_window`) | fixes `T` (it is derived from the recipes, D29) and searches `EntryShare`, or searches the product as `entry_window` |
| A03 | alias | `Population.BirthStake` and `Role.StartMoney` | the stake `StartMoney * BirthStake / 10 000`; the money cap `M / stake` | fixes `StartMoney` in stage C2 (it also funds entrants) and searches `BirthStake` |
| A04 | alias | `SpendMaxShare` and `SpendHalfTicks` | one concave cap on spending above the reserve | searches them as a pair at two or three combinations, never one at a time to a conclusion |
| A05 | alias | `MoveGain` and `MoveWidth` | one distribution of margins `[MoveGain, MoveGain + MoveWidth]`: its centre and spread | searches the centre and the width as two composite factors |
| A06 | alias | `BeliefStockLow` and `BeliefStockHigh` | one stock band: its centre and width | searches the centre and the width as two composite factors (I03 holds) |
| A07 | gauge | the distance unit: `Edge.Distance`, `Mode.Speed`, `Mode.HaulSpan`, `InfoSpeed`, `MoveSpeed` (only `InfoSpeed` is a K row; `Mode.Speed` and the others are the pack's structure) | travel time `max(1, ceil(Distance * 10 000 / (Speed * F)))` and lag `1 + ceil(D / InfoSpeed)`: only ratios and the ceilings | fixes the unit once (C1) and searches `InfoSpeed` against it; scaling all of them together is not a move |
| A08 | gauge | the money unit: reference prices, money amounts, `StartMoney`, `ReferenceBasket`, flow amounts | relative prices; the resolution floor and integer rounding break the invariance | fixes the unit so the cheapest tradable price is above the resolution floor with room (I01); a finer unit is not a calibration |
| A09 | gauge | the time unit: the calendar's tick, durations in ticks, `TicksPerYear` | durations are in ticks and are **not** rescaled when the calendar changes (`spec-calendar.md` 6.1) | fixes the tick in C1; a pack for another tick restates every duration and re-enters C2 |
| A10 | alias | `FeeShareSpreadBP` and `MerchantFeeShare` | one fee-share interval `[share - spread, share + spread]` (the bars are `10 000 / phi`, `spec-merchants.md` 5.7.3, a separate quantity) | searches the centre and the spread as two composite factors (I14 holds) |
| A12 | alias | `ReviewTicks`, `RoleCooldown`, `EntryShare`, `RoleHysteresis` | the rate of occupational change; the model finds the entry share the one that works and the others not enough alone (`spec-spawner.md` 16.1) | searches `EntryShare` first with the others at default, then the others one at a time |
| A13 | alias | `RefBandLow`, `RefBandHigh` with `CostFloor`, `CostCeiling` | where a unit cost is known the band that binds is the tighter; the other is dormant there | reads the counters (`BeliefsUpdated`) to say which binds before moving either |
| A14 | alias | `PoolRelease` and `ReliefBelow` | relief paid on the gap of each household (`spec-estates.md` 2.6, `need_i`), while the pool lasts | searches `PoolRelease` first (it bounds the drain), then `ReliefBelow` |
| A15 | alias | the knots of a schedule and its region multiplier | a schedule's level; its slopes set the gain (D46) | searches the multiplier (a scalar) and leaves the knots to the pack's demography |
| A16 | alias | the knots of `Resource.Abundance` and the curve they draw | one yield curve per resource; a flat curve is the default (4.3.1) | searches no knot: the curve is the pack's, and a calibration moves the yield only through `RateBP` and `PlanShareBP` (I12) |

### 6.6 Retired rows

None. A row retired with its field is listed here, with the change that removed it.

## 7. The procedure

The procedure is the same for the library's defaults and for a setting (3.2). It is
written as a **consumer's loop** over EXP's experiments: nothing in it is a library
call, and every number in it is a placeholder (P) that the first reviewed run may
move (changing one is a change to this spec, explained as `testing.md` 7.2 asks).

### 7.1 Seed sets: search, confirm, gate, hold-out

STY fixes the sets of a run (8.2): the **gate** 1 to 10 (REF 8.2: the band
scenarios), the **setting** seeds 1 to 50 (the nightly range of REF 8.2; tier L: 1
to 10) and the **nightly** 1 to 50. Its text is plain that the gate lies **inside**
the setting range, so a band set from 1 to 50 is not asserted out of sample by the
gate, and it recommends a **hold-out** (51 to 60) that REF does not yet provide. A
calibration adds one role of its own, the **search blocks**, and uses the others for
their purposes. **The validation step of the procedure is the confirm**: a point chosen
on seeds outside every set of REF (101 and up) is run on REF's own nightly seeds 1
to 50 (tier L 1 to 10). Nothing here depends on seeds outside REF's sets for a confirm, or on a hold-out that REF does not define.


| Role | Seeds | Used for | Never used for |
|------|-------|----------|----------------|
| **search block** `b` (`b` at least 1) | `100 b + 1` to `100 b + 100`: 101 to 200, 201 to 300, and so on | the screen, the explore and the choose of **one** choice | a second choice; the confirm; a band |
| **confirm** | 1 to 50 (tier L: 1 to 10): STY's setting seeds, which are not seeds a choice was made on | the confirm of the chosen point; the stage stop; the runs STY 8.4 sets the stage's bands from | a choice |
| **gate** | 1 to 10, inside the confirm range | STY's merge-gate assertion of a set band, and the stability record of a default (STY 7.3) | an input to a choice: its verdict is a **finding** |
| **hold-out** | 51 to 60, **only if REF defines it** (STY 8.2 recommends it; nothing here relies on it) | the out-of-sample check of a stability record (7.4.2), when it exists | a choice; a band |

7.1.1. **Where a choice is made.** A choice is made on **search blocks only**, which
are disjoint from every set STY names (1 to 60). The confirm then evaluates the
chosen point on seeds it was **not chosen on** (the phrase "the calibration's own hold-out" is STY 10.2's; EXP 4.6.4's example of another `Seed0` is 1 001). The confirm's per-seed values are **not read into the
next search**: a failed confirm returns to the next block **from the original base**,
and an accepted one is frozen. That is what keeps the gate, which is among the
confirm seeds, from being an input to a choice.

7.1.2. **Why a block is used once.** A point chosen as the best of `N` noisy pass
shares has its noise selected for (the winner's curse, EXP-A2). A second choice on
the same block would fit the second choice to the first's noise. A block therefore
serves **one** choice, the block is named in the record (section 8), and a failed
confirm sends the calibration to the **next** block. The arithmetic is in 18 and
19.4: of 64 points that are each really inside their band in 85% of seeds, the best
passes a 10-seed SR2 search with certainty (10 000 bp), and passes a 50-seed
confirm in 22% of cases (2 194 bp). A search that stopped at its own seeds would
call an 85% point stable.

7.1.3. **Seeds per point.** The search uses `R` replicas a point (placeholder 10: a
band is never set from fewer than 10 seeds, FR-STY-078, and a search that sees fewer
cannot tell SR2's one in ten from none); the confirm uses 50 (tier L: 10), STY's own
setting range. Both use `SameSeeds` (EXP 4.6.3) so that two points of one block are
seed-matched, except for a knob with the flag `d`, which is compared as
distributions (7.2.3).

7.1.4. **Escalation.** After **three** failed confirms at one stage (placeholder),
the stage is not searched again: the facts, the priors or the structure are
reviewed (7.5), because three independent blocks that all missed say that the region
is empty or the target is wrong, not that the search was unlucky.

### 7.2 The four steps

The forms are those of EXP 12.4.2, restated here with the rules a stage needs.

7.2.1. **Screen.** An experiment of EXP's `OneAtATime` design, `SameSeeds`, one
factor per K or D knob of the stage whose levels are the knob's **base value and its
two neighbours** (5.2), over the pack and overlays of the stage, `R` replicas on the
search block, `Ticks` and `Transient` of the stage, the stage's facts as `Measure`s
(a detector's verdict is mapped by the consumer to an integer, `none` being 0 as CLB's convention: neither STY nor EXP fixes the code values; the criterion is `AtMost 0`, EXP 12.2.3) and the
`Criterion`s of 3.3. Its points are `1 + 2 k` for `k` knobs, fewer where a side is an
edge. It yields, for each knob and each measure: the **elasticity** at each
neighbour and the **effect range** and its share (EXP 8.5). Three results leave it:

- the **screened-in** set: the knobs whose effect-range **share** of some measure of
  the stage is at least `screen_share_bp` (placeholder 500), and every D knob whose
  counter **fired** in some run (it acted);
- the **dormant** set: the D knobs whose counter never fired at the base or at a
  neighbour (10.4). They are left at the default and recorded; a dormant knob is not
  fitted;
- the **findings**: every measured sign that differs from the row's expected sign
  (FR-STY-092), and every point that failed to resolve (`unresolved`,
  `invalid_config`), which is a record of an invalid neighbour and an edge.

An experiment has at most `MaxFactors` factors, 16 (EXP 13.2), so **a stage whose
knobs are more than 16 is split by owner into sub-stages**, each with its own block
and confirm: C4 is (C4a has exactly 16) and so is C7 (21 knobs).

7.2.2. **Explore.** An experiment of `Spread` over the screened-in knobs as range
factors, **`d` knobs** with `N = max(16, 10 d)` points (a placeholder rule), on the
**same block** as the screen, `SameSeeds`, the same `Ticks` and measures. The range
of each knob is the interval of 5.3 (levels -2 to +2). With one screened-in knob the
design is a `Grid` of its five levels. A knob in an alias group (6.5) enters as its
**composite** (EXP 3.2.4): `entry_window` for A02, the centre and the width for A05,
A06 and A10. Where a knob has a condition on another (I03, I05), the explored
points the condition rejects are recorded `invalid_config` and are not points.
The reductions are the pass share of each criterion, `Mean`, `Std` and the quantiles
of each measure, and the main-effect bins (EXP 8.5.1). **No model is fitted**: a
response surface is not offered and not needed, because the choice is a set
operation (7.3).

7.2.3. **Choose.** A **pure function** of the screen and explore results (FR-CLB-030
to FR-CLB-034):

1. the **feasible set**: the points on which every X and O criterion and every
   detector criterion passes on **all** `R` seeds, and every B criterion with a target
   (3.3) passes on at least `ceil(9 R / 10)`;
2. if the base is in it, the choice is **the base**: nothing moves;
3. otherwise the feasible point with the least **grid distance** to the base, the sum
   over knobs of the absolute difference of the levels' indices, a Spread value
   mapped to the nearest grid level and a tie to the lower; then the fewest knobs
   changed; then the lowest point index;
4. if the feasible set is empty, the result is **no choice** and the **Pareto set**
   (7.5): a point that passes every exact criterion and whose vector of B pass counts
   no other point of the same set dominates.

A knob with the flag `d` is compared across points by **distributions** (EXP 8.4 `Dist`
and `PerPointSeeds`), because a change that moves a draw count leaves the runs
unpaired (WLD 7.6.1); the feasible set is the same set operation on those
distributions' pass counts. A paired comparison is used only for a knob that moves
no count, and, when CHK offers the read of the stream (EXP 7.3.2, without which
`Pair` is refused), the alignment horizon is recorded (EXP 7.3).

7.2.4. **Confirm.** One experiment: the chosen configuration (the base with the
overlay of the choice) and, as a **control**, the base, each over the confirm seeds
(`Seed0` 1, `Replicas` 50, tier L: 10), the same `Ticks`, `Transient` and measures,
on the stage's packs. The verdict is the stage stop (7.4.1). The confirm's runs are
**also** the reviewed run from which STY 8.4 sets the stage's band keys (once the
choice is frozen); the gate, which is inside that range, then asserts them at the merge
gate (STY 3.4, state "held"), and STY 8.2 says plainly that this is not out of sample.

7.2.5. **Between the steps.** No step reads a result of the **gate**. No step
changes a pack document except the overlay of the choice. A step that fails to
resolve a point (`unresolved`, `invalid_config`) records it and goes on (EXP 9); a world that fails at run time is a seed outside the band over the planned `n` (STY 8.3.3), not a skipped seed.

### 7.3 The stages

Stages C0 to C9, with C4 and C7 in sub-stages. **A stage fixes its knobs**: all the
K and D knobs of earlier stages, and all the S rows, are as the previous stage left
them. The order is the brief's:
structure, viability and warm start, scale, then the facts of STY by group, in the
order in which STY sets bands (STY 10.3), so that each stage's runs are the evidence
the next one stands on. The search order and the order of setting bands differ in the places listed after the
table and are not in conflict: for one, the **scale** is chosen at C3, before the
dynamics knobs, because it fixes noise and cohort effects for everything after it,
while STY's P09 *band* is set at its step 5, after the closed-region bands. (STY 5.9 also names its conflicts C1 to C8; they are
always written with "STY 5.9" or "conflict".)

| Stage | Name | What is searched (the registry's **Stage** cell) | Facts and detectors | Packs and design | Placeholder cost |
|-------|------|--------------------------------------------------|---------------------|------------------|------------------|
| **C0** | Exact | nothing | every X and O fact; STY 10.3 steps 0 to 2 (baseline, provoked runs, regime thresholds) | the packs of REF 7.5; no search | the suite |
| **C1** | Structure | S rows at C1: identity, topology, lists, units (A07 to A09) | `Validate`; VIA V1 to V4; the census warning | every variant; EXP `Check` over `Docs` levels, **no run** | minutes |
| **C2** | Viability and warm start | S rows at C2 (quantities); K rows at C2 (`ExpectedWorkTicks`, `StartMoney`, `HorizonYears`, the two solve options) | the solver report; H01 to H04; D01 | a `Grid` over a pack quantity with `Prepare` = the solver; then `R` replicas | seconds of solver, then a stage run |
| **C3** | Scale | `Scale`, `MinAgents` | P09 (X parts; B part by its working prior); scenario S10 | `one` + `overlay/h200` at scales 1, 4 and 10 (`Grid`); not applicable at weight 1 | 30 runs, 0.0 core-hours (in the totals) |
| **C4a** | Price formation | the 16 BEL rows | P01 to P04, P07, P08; detectors R1, R4, R5a, R5b, R6, R7, R8 | `one`, `second`; screen, explore, choose, confirm | about 1.5 core-hours |
| **C4b** | Work and welfare | the 6 AGT and 7 WEL rows of C4 | the same, and D01, M03, M05, M06, H01 to H04 | `one`, `second` | about 1.3 |
| **C4c** | Occupation | the 5 SPN rows of C4 | D03, R6 (role shares) | `one` | about 0.6 |
| **C5** | Shocks and the ladder | the 6 SPN, 3 EST and 3 SUB rows of C5 | P05, S01, S02, S06 | `one`, `famine` + `overlay/estates`; **branch** designs (EXP 6) | about 3.1 |
| **C6** | A larger world, barriers | the DEM, AGT, SKL and PRO rows of C6 | W01, M04, W04, D03 with the skill overlay | `one-demand` searched; `three` and `five` run once per choice on their REF invariant seeds (4 and 2) as a confirm only (the "once" of 7.3.7) | about 1.0 |
| **C7a** | Trade | the INF and MER rows (12 knobs) | P06, S03, T02 | `three` searched, confirmed on `two` and `three` with the trade overlay; `Dist` for `d` knobs | about 3.0 |
| **C7b** | Movement | the MIG rows (10 knobs), the C7a knobs fixed | G01 (umbrella), P06 held | `three` with the trade and migration overlays | about 2.7 |
| **C8** | The open layer | the EXT and MON rows | S04, S05, T01 (X), T03 (R) | `one` + `overlay/external.toml`, `overlay/faucet.toml` | about 0.8 |
| **C9** | The long facts | the POP and RES rows | W02, W03, W06 (tier L) | `one-demand` + resources and population overlays; 100 years | about 24.7 |

**Departures from STY 10.3.** Bands are *set* in STY's order (FR-STY-091) whatever the order of the search: (1) the scale (STY step 5) is *searched* at C3, before C4, and its B part is read against its working prior until step 5 sets the band; (2) STY's step 3 is searched in three sub-stages C4a to C4c, each confirmed before the next; (3) P05 (STY step 4) is searched at C5, not C4a, as it is a response to a shock; (4) the long facts of step 9 are searched at C9 on tier L. A stage's result is a *candidate* point; the bands it meets are written in STY's order when the stage chain is confirmed.

The cost column is **a count times the placeholders** of `spec-scale.md` 6, STY
4.5.1, EXP 10 and CHK 10.3 (`cost.py`, 18): one core, no parallelism, a screened-in
set of a few knobs, 50 confirm seeds and a control. The whole sweep of one pass is
about 7 010 runs and **39 core-hours**, C3 included (`cost.py`); the length of a run is the
scenario's own (REF 8.2: 1 095 ticks, 1 800 for a shock, 400 of burn-in), so a band set at
a confirm is asserted by the gate at the length it was set at, and `one-demand` is 600 households
of 6.5 us (REF 8.2). A stability record of the 49 defaults of the claim set
(7.4.2), taken on all seven packs of REF 4.4 (an upper bound: a pack records only the knobs it
sets), on the gate and, where REF defines one, the hold-out, is 13 860 runs and **57 core-hours**
more (half that on the gate alone). `RunUnit`
is the unit a consumer fans out (EXP 5.3), so these are an afternoon on a few cores.
They are placeholders and nothing here is measured (N-015).

7.3.1. **C0, Exact.** No knob moves while an X or O fact fails: such a failure is a
defect of the library or of the fixture, or a wrong derivation, "never a band to
widen" (STY 3.2). C0 includes STY's steps 1 and 2: the healthy baseline of every
detector and the provoked runs from which the regime thresholds are set (STY 6.5.3).
Until a threshold is set its detector returns `unknown`; **the later stages then use
the priors of STY 6.2 as working thresholds** (3.3 item 2) and say so in the record.

7.3.2. **C1, Structure.** The identity of the world: the commodities, recipes, roles
and regions, the topology of the graph and of the modes, the module switches, the
calendar and the three **units**: the money unit (A08), the distance unit (A07) and
the tick (A09). The check is `Validate` and the four layers of VIA (sourced and
runnable; productive cycles; labour sufficient; storable) through EXP's `Check`,
which resolves every variant without a run (EXP 14.3). A warning is **accepted with a
reason in the record or removed**; none is left unread. Everything C1 decides is
**structure**: after it, changing it is a new lineage of the pack (9.2) and every
later stage is run again.

7.3.3. **C2, Viability and the warm start.** The quantities of the setting
(durations, yields, upkeep, populations, distances, capacities) are chosen against
the **solver**: the experiment is a `Grid` or the author's loop over a quantity with
`Prepare` regenerating `WarmStart` for each point (EXP 3.3.3), and the response is the
solver's report (`SurplusBP`, `LabourShareBP`, the errors and warnings of VIA 10),
which needs no seed. The warm start, the reference prices and the starting wages are
`out`: **written from the solver, never by hand** (REF position 4). Then the stage's
runs (H01 to H04, D01, G01) from the warm start, on a search block. Stop: V1 to V4
pass, no `labour_short` or `storage_gap`, and `LabourShareBP` at most
`ExpectedWorkTicks * 10 000 / TicksPerYear` unless the pack is a famine variant by
design, where D01 is not asserted (conflict C4 of STY 5.9). A pack whose labour share to subsist
exceeds what the role is expected to work is **not viable at its expected effort**
and no later knob repairs it.

7.3.4. **C3, Scale.** The scale a consumer will run: weight 1 for the first
consumer's world (N-030), in which case the stage is **not applicable** and `MinAgents`
is irrelevant (every household is an agent). Where a scale above 1 is used, the
stage is STY's P09 over 1, 4 and 10, and `MinAgents` is the least `K` at which the
X parts hold and the B part meets its working prior (`spec-scale.md` 5.2 says exactly
that). After C3 neither is moved.

7.3.5. **C4, Price formation, work and occupation.** Three sub-stages in this order,
each with its own block and its own confirm, because each fixes what the next one
stands on: **C4a** the belief machinery (every agent sees its output as the price),
**C4b** the work and welfare rules (they set the supply and the effective window
`T`), **C4c** occupational change (the slowest loop, rate-limited by design, whose
condition `EntryShare * T` needs `T` from C4b, A02). A later sub-stage may not move an
earlier one's knob without the re-entry rule (7.3.11). The packs are `one` and, for the
configuration-only property, `second` (STY P08); the length is the scenario's own, REF 8.2's 1 095 ticks (three years), the detectors read whole years from tick 0, the first included (STY 4.1.2), and a band fact excludes the first year of a warm run as its own transient (STY 4.1.2).

7.3.6. **C5, Shocks and the ladder.** The facts here are responses to a **shock at a
tick**, so the experiment is a **branch** (EXP 6): a burn-in to the tick, a checkpoint,
the variants (none, the shock) restored from it. The branch cannot change a
configuration value (the hash), so **a knob sweep at this stage is a sweep of units**,
each point with its own burn-in shared by its variants (EXP 6.2.1). The shock is a
script of INT calls (REF 7.1.3). The `famine` pack asserts S02 and S06 (conflict C4 of STY 5.9); its
depth is a factor.

7.3.7. **C6, A larger world and the barriers.** The same loops at more commodities
and roles (`three`, `five`) and the demand and skill modules. `five` is thirteen
times the cost of `one` per tick (`cost.py`), so it is run **once per choice** as a
confirm, not as a search; the search stays on the small pack (`one-demand`).

7.3.8. **C7, Trade (C7a) and movement (C7b).** The loops with a delay: shippers, rumours,
contracts and migration (L84 to L95, L107 to L116). The packs are `two` and `three`
with `overlay/trade.toml` (and `overlay/migration.toml`). The **stability limit of a
loop with a delay is a product** (`spec-information.md` 11: the gain against the lag
plus the transit time), so the knobs are searched with the gauges of A07 fixed.
Several knobs here have the flag `d`.

7.3.9. **C8, The open layer.** Most facts here are exact (S05's accounts, T01) or
**report-only** (S05's drift, T03): the calibration is that the accounts close and
the response is recorded; the knobs are largely design (class S) and the D rows are
searched only for the counters to be seen acting.

7.3.10. **C9, The long facts.** The Malthusian approach (W02, W03, W06) is a
hundred years or more (STY 4.5.1: a time constant of about 33 years at the default
gain), so it is tier L with ten seeds (`stylised_long`), never in the merge gate. The
knobs are the two region multipliers and the caps, not the schedules' knots (A15).
A pack with the population module on and no demand module or faucet has **no healthy
start** at the default schedules (conflict C1 of STY 5.9): the calibration of such a pack is
refused at C2 with the warning `replacement_above_closed_ceiling` as an error.

7.3.11. **The re-entry rule.** A later stage that finds a fact of an earlier stage
failing may move that stage's knob **only** by re-running that stage's confirm with
the new value on a fresh block and then every stage between them, recorded as
`reentry C<n>`. A change to a knob with the flag `in` (or to any S row) re-runs C2's
H facts, because the warm table is stale (I19). A change at C1 is a new lineage and
starts again.

### 7.4 The stop rules

7.4.1. **The stage stop.** A stage is done when, on the confirm seeds, the chosen
point (or the base, if the base was feasible) satisfies **all** of:

1. every X and O fact of the stage on all seeds (SR1);
2. every detector the stage names is `none` on the healthy runs on all seeds (SR1,
   STY-G01), or `unknown` **only** because its threshold is unset, in which case it
   is judged at its **working** threshold: a reading that is CLB's own (STY 6.5.1 would
   call it `unknown`) and that the record says is `working`;
3. every B fact with a **set** key inside its band on at least `ceil(9 n / 10)` of the
   `n` seeds (SR2), the seeds outside listed (STY 8.3.2);
4. every B fact with an unset key and a prior of **size** H, M or E inside its prior by
   SR2 (a working target); a prior of size P is recorded, not asserted (3.3);
5. no **finding** open: a sign that differed from the registry was reviewed, and the
   row or the derivation was corrected or the difference was explained;
6. every D knob that moved **acted**: its counter fired in at least one confirm seed
   (10.4); a D knob that moved and stayed dormant is **reverted**, because the move
   did nothing the facts can justify;
7. the control (the base) was run on the same seeds and is in the record.

7.4.2. **The program stop: a default is called stable.** Only under STY 7.3.1, and
only for a default the calibration **claims** (the claim set is the K and D rows whose knobs the sweep table of STY 10.1
names, by CLB's count of the registry: 11 for beliefs, 8 for the work and occupation loops, 6
for the trade loops, 5 for demand and spending, 10 for the ladder, the estates and the
floor, 5 for population and resources (`BirthStake`, `MinHouseholds`,
`HouseholdCap`, `RateBP`, `PlanShareBP`) and 4 for the warm start, 49 in all (the sweep table
names 58 items; the S rows `Duration`, a mode's `Speed` and `HaulSpan`, `Shape`, `Capacity` of a
resource and `Rank`, the schedule lists, which are searched only through their two multipliers
(A15), and `MaxDispatchPerTick`, a bound, are not claims); some of these rows back facts that
the version 1 criteria of STY 8.1.1 do not list, and the claim is the sweep table's, not 8.1.1's; the rest carry no claim and are labelled placeholders). For each claimed knob and for every
shipped pack that sets it: the three checks of STY 7.1 STABLE **at the default and at
each neighbour** of its grid, on the **gate** seeds, with every key those checks read
set by review (STY 8.4), and the **stability record** of STY 7.3.3 is the output.
The gate is inside the setting range (STY 8.2), so a verdict on REF's seeds alone is
**in sample**. Where REF defines a hold-out, the default and each neighbour are also
run on it and the three checks must hold there; where it does not, the record says
`in sample` and the claim is only that no regime was detected on those seeds.
Nothing else is a stable default.

7.4.3. **The neighbour rule with an edge.** STY 7.3.1 allows "stable" only for a default that
is STABLE at **both** neighbours, and "a default at the edge of the stable region is not
stable". A default whose grid has an edge on one side (5.2) therefore has **one
neighbour examined**, and its `claim` line says `one_neighbour`: "no regime detected at the default
and its one neighbour, the region beyond the edge not examined". The word *stable* is **not**
used of it (FR-CLB-039); it is not a fail for the missing side either. A rule edge (for example
`BeliefStep` at the resolution floor, 19.1) is a **finding about the pack**, because the
default sits at the edge of the legal region of its own money unit. Whether STY wants an
edge clause is an owed row to STY (23).

7.4.4. **A failed gate or hold-out.** A failure on the gate or on the hold-out is a
**finding**: the stage returns to its screen on a **new block**, with the attempt
counted; no knob is moved "because seed 7 failed", and the seeds that failed are
listed in the record (STY-A4), and a failed world counts as a seed outside over the planned `n` (STY 8.3.3). The review of FR-STY-079 decides whether the band, the code or the fact was wrong (7.5 item 4). The hold-out is the only out-of-sample check of a set
band, and a loop that reads its verdict and adjusts has spent it.

### 7.5 When two facts conflict

The order of 3.3 settles most conflicts, and the next steps are in this order:

1. **Look in STY 5.9 first.** If the pair is one of its eight conflicts, named C1
   to C8 there and not to be confused with the stages (they cannot both hold in one
   pack), it is not a conflict of the calibration: the pack asserts one of
   them, and the pack's record says which.
2. **A lower class never overrides a higher one**: X, then O, then the detectors, then
   B facts with a set key, then B facts with a working target, then R. A miss of a
   lower class beside a pass of a higher one is a **known miss**, listed.
3. **There is no weighted sum.** A weight would say how much one fact is worth against
   another, which no spec says, and would hide which was sacrificed. The result when
   no point is feasible is the **Pareto set** of the B pass counts among the points
   that pass every exact criterion (`choose.py`, 19.3), each with its knob settings
   and its grid distance.
4. **The decision is a review**, with exactly three outcomes, written in the pull
   request (`testing.md` 7.2): (a) **change the structure** (C1 or C2), a new lineage;
   (b) **the prior is wrong**, an edit to STY by its review (a prior of size P is a
   guess, STY 3.3) and never an edit to the code to fit it; (c) **accept a known
   miss**, recorded as `unmet <fact> <measure> <values>` with the loop it leaves
   undamped.
5. **No fact is dropped to pass.** A fact that stays unmet is in the record with its
   observed values; it is not removed from the acceptance set, and a band is not
   loosened (STY 8.4.5, FR-STY-079).

### 7.6 Where the procedure meets the rest

| The procedure uses | From | What it takes |
|--------------------|------|---------------|
| the facts, the classes, the seed sets, SR1 and SR2, the band-setting rule, the neighbour rule, the stability record | STY 3, 5, 7.3, 8 | as written; none is set here |
| the experiment, the designs, the reductions, `Compare`, `MainEffects`, `Elasticity`, branch, `Pair`, `Check` | EXP 3 to 8, 12.4 | as written |
| the measures and the blind-spot fields | IND 9, STY 4.3 | a band is read with its blind spots; a knob is not turned until an index looks flat when `Stale` has risen (STY 10.2) |
| the solver and the table | VIA 7, 8 | `SolveSteadyState` as `Prepare`; the table is output |
| the layering and the hash | CFG 6.4, 8 | the overlay of a choice is a document; the record names the hash |
| the draw table | WLD 7.3 | the `d` flag |

## 8. Recording a calibration

### 8.1 What a record is

A **calibration record** is a text file of plain integers and closed keywords, one
for each (pack, stage, attempt). It is the **only** durable output of a calibration
besides the overlay (9.1): a value of a knob is in a record or it is a placeholder.
It carries no wall-clock time, no floating-point number and no map order, so that two
runs of the same experiment give the same bytes (STY 8.5, EXP 8.2.2, NFR-CLB-002).
It is **not** a copy of the series: it holds digests, statistics and the settings, and
a result is reproduced from the documents, the hash, the experiment and the seeds.

### 8.2 The grammar

Each line is `<keyword> <fields...>`. The keywords are a **closed set** (a rule
constant of the check of 11.2). `note` is the one line with free text, to the end of
the line, and is never parsed. Lines come in the order shown; a repeated keyword
repeats in the order of its subject (documents in layering order, factors in the
experiment's order, facts by ID, knobs by registry ID).

```
calibration 1
pack <name>  lineage <16 hex>  stage <C-id>  attempt <n>  block <b>
commit <id>  parent <16 hex or ->
config base <16 hex>  chosen <16 hex or same>
document <name> <16 hex>                     # one per document, in layering order
experiment <step> design <oat|spread|grid|replicate> points <n> replicas <R>
           seed0 <s> seeding <same|per_point> ticks <T> transient <t> every <e>
           salt <s> render <16 hex>          # EXP Render digest of the Result
factor <name> key <Scope.Field> grid <geo|linN|set> base <v> lower <v or -> upper <v or ->
           edge <none|below:bound|below:rule|above:bound|above:rule>
measure <name> fact <ID> class <X|O|B|R|P> kind <fact|detector|counter>
criterion <name> measure <name> op <AtMost|AtLeast|Between> a <a> b <b> target <set|working|report>
screen <knob> measure <name> el_lower <bp or -> el_upper <bp or -> range <v> share_bp <v>
screened_in <knob ...>
dormant <knob ...>
finding <knob> <fact>/<measure> expected <sign> measured <sign> status <open|explained|row_corrected>
feasible <n> of <N>
chosen <point index or none> distance <d>
pareto <point index ...>
set <Scope.Field> <v>                       # one per knob the overlay moves
confirm seeds <a>..<b> control <yes|no>
fact <ID> class <X|O|B|R|P> key <band key or -> target <set|working|report|none>
     values min <v> median <v> max <v> outside <seed,seed or -> result <pass|fail|report|na>
detector <R-id> <none|detected|unknown> working <yes|no> onset <t or ->
stability <knob> pack <name> point <default|lower|upper> value <v> set <gate|holdout> seeds <a>..<b>
          in_sample <yes|no> c_bp <STABLE|NOT_STABLE|UNDETERMINED> c_nc <v> c_ws <v> edge <none|below:bound|below:rule|below:off|above:bound|above:rule>
claim <knob> pack <name> <stable|not_stable|one_neighbour|undetermined>
unmet <fact> <measure> <values>
decision <stop|escalate|reentry C<n>|review|revert>
note <free text>
```

A `knob` is a registry ID (`K-BEL12`), so a record names a knob by the same token as
the registry and the check of 11.2 resolves it. `lineage` is the first 8 bytes of the
hash of the effective configuration at the end of stage C1 (9.2), so two records of
one lineage are comparable and two of different lineages are not.

### 8.3 What the record must say

1. **The configuration**: the hash of the base of the attempt and of the chosen
   point, each the `ConfigHash` of **a world a run built** from the documents (EXP
   3.3.2: no function hashes a `Config` and `Check` computes none), and the documents
   (with their digests) the hashes came from.
2. **The experiment**: every step, as the plain-data part of an EXP `Experiment`
   (design, points, replicas, `Seed0`, seeding, ticks, transient, windows, salt, the
   factors with their keys and grid, the measures and the criteria) and the digest of
   its rendered `Result`. The function values (the measures, the steps) are named by
   the labels the experiment gave them (EXP 3.1.2).
3. **The results**: for the screen the elasticities and the effect ranges, the
   screened-in and dormant sets and every finding; for the explore the number
   feasible; for the choose the index, its distance and the Pareto set if there is no
   choice; for the confirm each fact's class, band state, `min`, `median`, `max`, the
   seeds outside and the verdict.
4. **The bands met and not met**: a `fact` line for every fact of the stage, with
   `result` `pass`, `fail`, `report` (an R fact or an unset key without a working
   target) or `na` (its modules are absent, STY 4.6), and an `unmet` line for every
   known miss (7.5).
5. **The detectors**: each with its verdict and `working` `yes` if its threshold was a
   prior (3.3).
6. **The attempt and the block**, so that a reader can see how many tries a choice
   took and which seeds each used (7.1.2).
7. **The decision**: `stop`, `escalate`, `reentry`, `review` or `revert`.

8. **The stability lines** (STY 7.3.3): for each claimed default and shipped pack that sets
   it, one `stability` line per point (the default and each neighbour that exists) with
   the three verdicts of STY 7.2, the seed set and `in_sample`, and a `claim` line
   (`stable`, `not_stable`, `one_neighbour` or `undetermined`).

### 8.4 Where records live

For the library's defaults, under `testdata/calibration/<pack>/<stage>-<n>.txt`
beside the golden files (`testing.md` 10.4), changed only by a reviewed change that
says what moved (`testing.md` 7.2). For a setting, wherever the consumer keeps its
pack: the grammar is the library's and the location is not. A record is **never** a
golden file of the economy: it is data about a decision, and the golden file of a
pack moves only when the pack does.

### 8.5 Reproducing and comparing

A record is **reproduced** by building the world of each document layered in the
order given, checking the hash, running the experiment of each step with its seeds and
comparing the digest of its rendered result (FR-CLB-040). Two records of one lineage
and one stage are **compared** line by line on `set`, `config chosen` and the `fact`
lines: which knobs moved, by how much (in grid steps), which facts changed result and
which seeds went outside. A comparison of the **series** of two records needs the
runs again, on the same seeds, as `Compare` with `Dist` (EXP 8.4); the record keeps
digests, not series.

## 9. Packs

### 9.1 Where a calibrated number lives

In an **overlay document** of the pack, layered **after** the variant's documents and
before the warm-start document, one document for each stage that moved a knob:
`calibrated/<stage>-<n>.toml`, in the form of EXP 3.3.4 (a table header and a key
for each knob, integers only). The layering is the configuration's own
(`spec-config.md` 6.4): later wins, tables merge by key. Three rules:

- **Purity.** A calibration overlay holds **only K and D keys**: a key whose
  registry class is S or P in a calibration overlay is a defect (a structural change
  is a new variant of the pack, 9.2). A check reads each overlay's keys against the
  registry (FR-CLB-050), mapping a path to a registry key by **the table of EXP
  11.3.1** (the table prefix to the scope token, then the field by the snake-case
  rule that CFG 6.2 owes): never by the last segment alone, which is unsound because
  a field name repeats across scopes.
- **A header.** The file begins with a comment that names the record of its stage and
  attempt (testing.md 5.6, REF 4.1), so that a number can be traced to its evidence.
- **The warm start last.** The warm-start document is **generated from the effective
  configuration of every other document**, calibration overlays included, last
  (REF position 4; I19). Its keys are disjoint from every other document's, so its
  position in the list does not matter to the merge, only to its generation.

A default of the library is the exception: its number is in the owner's table and in
`defaults.go` (architecture.md 9.8), and **no overlay stands for it**.

### 9.2 Versions and lineages

A pack has a **lineage**: the line of its structure. A change to an S row (an
identity, a topology, a quantity, a unit) starts a **new lineage**, with the
previous one named as `parent` in its first record; every stage is then run again
(7.3.11). Within a lineage, a calibration overlay is versioned by its attempt number
(`calibrated/c4a-1.toml`, `c4a-2.toml`); the overlay **in force** for a stage is
the highest accepted one, and the hash of the effective configuration identifies the
state. Nothing is overwritten: a rejected attempt's overlay stays with its record,
marked `revert`.

### 9.3 What transfers between settings and what does not

| | Transfers to another setting | Why |
|-|------------------------------|-----|
| **The procedure** (this section 7), the classes, the registry, the grid rule, the seed roles | yes | none of it names a commodity, a region or a number of a pack |
| **The conditions of the specs** (I01 to I23, the alias groups, the gauges) | yes, **as conditions**: evaluated on the new pack's own quantities | an inequality such as `EntryShare * T` below a bound is about the pack's `T`, not about the number 50 |
| **The regime thresholds** (`regimes.toml`) and the X and O facts | yes | a regime means the same in every pack (STY 3.5, position 3) |
| **The library's defaults** and **STY's priors** | **as starting points only** | a default is a placeholder (N-015); a prior's basis (STY 3.3) is for the pack and the place it was derived on |
| **A band value** | no | set from the seeds of one pack by review (STY 8.4); another pack sets its own, from its own setting seeds |
| **A stability record** | no | it is about a pack set, a module set and a version (STY 7.4: "they do not cover combinations of modules") |
| **A warm-start table, a reference price, a calibration overlay** | no | solver output and fitted values of one lineage |
| **An evidence range** (E1 to E5) | as a sanity range | for another place and time (STY 3.3) |

The **second setting** of REF (`second`) is the standing test of what transfers: it
runs with the engine defaults and **no Go specific to it** (architecture.md 9.4,
STY-P08), and its bands are its own. A claim that a default is "stable" is a claim
about a set of packs (7.4.2); a new pack is not covered by it.

### 9.4 The first consumer's data pack

The first consumer's pack is a **data pack** under `examples/<name>/`, its own
module, with its own documents, its own band files and its own records (AGENTS.md:
setting content belongs there). Everything setting-specific (names, currency ladder,
the list of roles and regions, the content of the world) is in it, and **this spec
names none**. The pack is calibrated by the procedure of 7 with these properties:

- its C1 is its author's design, checked by `Validate` and VIA (7.3.2); its **units**
  are chosen once (A07 to A09), and the denominations of a display ladder are
  presentation only (class P) and take no part;
- its C2 regenerates its warm start from the solver and **refuses** a pack whose
  labour share to subsist exceeds its expected effort (7.3.3);
- C3 is **not applicable**: the first consumer's world is at weight 1 (N-030);
- it runs C4 to C9 with its own bands: the **regime thresholds** are the library's,
  the **X and O facts** are the library's, and each **B band** is set from its own
  setting seeds by STY 8.4 or stays report-only;
- "the RPG data pack runs" (`vision.md` 5, item 5) is **not** a stability claim: it is
  the statement that the pack builds, validates and steps. The claim "no regime was
  detected in the runs of record R" is what its records support (STY 7.3.2).

In version 1 the consumer is given the **procedure as a document and a tested reference
implementation**, not a library call: `experiment` and `indicator` are public, a measure
of a fact is a `Measure` (EXP 12.2.3), a band is a `Criterion` and the loop is its own,
but the grid, the neighbours, the seed blocks, the choose step, the alias test, the
record and the overlay renderer are `internal/calibration` and the registry is this
spec, which a consumer reads (it cannot import either). What it does not have in
version 1 is those helpers and the regime detectors (open question 1).

## 10. Sensitivity and identifiability

### 10.1 Which knob moves which fact

The tool is the screen (7.2.1): for each knob of the stage and each measure of its
facts, the **elasticity** at each neighbour, `rdiv(10 000 * (y_l - y_0) * x_0, y_0 *
(x_l - x_0))` (EXP 8.5.3: `OK` is false where `y_0` or `x_0` is 0 or `Y_0` is at or below 0, where `x_l` equals `x_0`, and where the value is `Clamped`), and the **effect
range and share** (EXP 8.5.1). They are the **sensitivity matrix** of the stage, kept
in the record. Three things are read from it:

- **The sign against the registry.** Each measured sign (the sign of the elasticity at
  the upper neighbour, and the lower, if they agree) is compared with the row's
  expected sign. A difference is a **finding** (FR-STY-092): if the row's source is `S`
  or `M` the owner's spec or the model is what is in question; if it is `H` the
  derivation is; if it is `?` nothing is contradicted and **the row is updated**
  (`?` becomes `+` or `-` with the source `M` of the record) in the same change.
- **The screened-in set** (7.2.1).
- **The non-monotone** knobs: a pair of neighbours whose elasticities have opposite
  signs is `~`, and is the case for which the explore step, and not the screen, finds
  the response.

### 10.2 The alias test

Two knobs are **aliased in a stage** when their columns of the sensitivity matrix, as
vectors of integers over the stage's measures, are collinear to a tolerance. With
`e_i`, `e_j` the vectors of signed elasticities at the upper neighbours over the
measures where either is defined and `OK`:

```
aliased(i, j)  iff  10 000 * (e_i . e_j)^2  >=  (10 000 - alias_tol_bp) * (e_i . e_i) * (e_j . e_j)
```

with `alias_tol_bp` 500 (a placeholder: `cos^2` at least 0.95; 9 500 bp). Both sides are
128-bit integers **under a bound**: an elasticity must be at most `2^26` in absolute value for the test (EXP 8.5.3 clamps at `2^31 - 1`, a wider range, so a clamped reading is an overflow error here) and the number of measures `m` is at most 16, so that `10 000 * m^2 * 2^104` stays below `2^127`; a vector above either bound is an **overflow error** (checked arithmetic, architecture.md 3.5), never a wrapped value; a zero vector is never aliased (it is **invisible**, 10.5). The
test is **exact** (`choose.py`: collinear vectors give 10 000 bp, an independent
pair 495).

Two cautions follow from the arithmetic and are part of the rule. **With one measure
every non-zero pair is aliased** (`choose.py`: `[120]` and `[-35]` give 10 000): a
stage whose facts offer fewer measures than it has screened-in knobs **cannot
identify them**, and the knobs beyond the number of measures are aliased by
construction, so they are fixed at their default or searched as a composite. And an
alias found on a screen is **a statement about this pack, these measures and these
levels**, not about the rule: it is recorded and not promoted to 6.5.

### 10.3 Declared aliases and gauges

Section 6.5 lists the aliases and gauges the specs imply, derived by hand. The
screen **confirms or refutes** them: the members of a declared alias group must show
collinear columns (or one of them must be dormant), and if they do not, the group row
is wrong and is **corrected as a finding**. A gauge (A07 to A09) is not tested by the
alias test; it is tested by a **scaling check** at C1 (every member scaled together
leaves the run unchanged up to the rounding the row names), a hand test the author
makes once.

### 10.4 Dormant knobs

A D knob is **dormant** at a point when the counter its row names (`D<nn>=<Counter>`)
did not fire in any run of the point. A dormant knob's effect on every fact is zero
by construction (a clamp that never acts does nothing), so no fact can calibrate it,
and a move would be a move "for nothing" (7.4.1 item 6). A knob with no counter (a rule constant of STB 11, or a damper whose catalogue entry names none) cannot be shown dormant and **is a gap
the calibration records**: its screen's effect range is the only evidence of acting.

### 10.5 Invisible knobs

A knob whose effect range is 0 on every measure of every stage that searches it is
**invisible** to the facts: either its mechanism does not touch the measures, or the
module is off, or it has no reader (flag `v1x`). It is not calibrated, it is left at
its default, and the record says `invisible`. Whether a knob's configuration change
moved the hash at all is the `no_effect` check that EXP names as a footnote and does
not specify until CFG offers a hash function (EXP 14.4 specifies only
`ranged_narrower_than_design`, `duplicate_point` and `stale_warm_start`); it would be a stronger
statement: the hash did not move.

### 10.6 What sensitivity is not

It is **not a variance decomposition** (EXP 8.5.2): two knobs whose effects cancel in
a bin look small. It is **not an interaction measure**: one at a time sees none, and a
pair that interacts is found by the explore step only through its main-effect bins.
A pair the specs name as moving together (6.4) is therefore **searched together**
(a composite) whatever the screen says, and the screen's silence about an interaction
is not evidence that there is none.

## 11. The tests

The pyramid of `testing.md` 2, applied. A calibration's **economy** numbers are never
asserted by these tests (N-027): they assert that the registry is true, that the pure
rules are the rules, that a record means what it says, and that the procedure, run on
the reference chain with a tiny experiment, behaves as this spec says.

### 11.1 The registry test (tier S, unit, `-short`)

A test in the root's external test package reads this spec and the
parameter-owner table of `specs/README.md` as **text**, with no dependency, as the
parameter registry test of `testing.md` 6.7 does (it is a test that reads documents, the one-sentence row owed to `testing.md` in 23 places it among the integration tests). It asserts (the model of 18, `registry.py`, implements the checks marked in the last column of its output today and reads the IND preset names from `spec-indicators.md` 9.1; it does not read a field list, because IND has none):

| Check | Fails when | Requirement |
|-------|------------|-------------|
| rows | a knob row has not 12 cells, a group row not 7, or an ID repeats | FR-CLB-001 |
| coverage, owner to registry | a pair (scope token, parameter) of the owner table is neither a knob row's key nor a field of a group of that scope (4.7, 4.8) | FR-CLB-002 |
| coverage, registry to owner | a key or a group field of a scope the owner table has names no owner row, outside the scopes `Solve option`, `Presentation` and the field `World.Clearing` (owed by WLD) | FR-CLB-002 |
| class | a class is not K, D, S or P; a D row has no `off=`; a non-D row has one | FR-CLB-003 |
| no restated number | a column named Default or Bound exists, or a cell holds a number outside a `lin` step, an `off=` value or a `set:` level | FR-CLB-005 |
| search | a Search cell is not `U`, `E<n>`, `I<nn>` or `-`; an `E<n>` names no anchor of 6.1; an `I<nn>` names no interaction; a K or D row has `-` | FR-CLB-006 |
| moves | a token has not the form of 4.4, names a fact that is not in STY, a measure whose preset is not named in IND 9.1, whose field is not in the closed list of 4.4 (STY's spelling; IND publishes no field column: owed, 23) and is no band key of STY, or a row has more than four | FR-CLB-007 |
| references | a `L<nn>` or `D<nn>` is not in `spec-stability.md`, an `I<nn>` or `A<nn>` is not in 6.4 or 6.5 | FR-CLB-008 |
| owner | an Owner cell does not resolve to a section of the owner's spec, or is not the section of the README's owner column for that pair (a group names each of its sections) | FR-CLB-001 |
| flags | a flag is outside the closed set of 4.2.1; `z` is on a row whose owner default is not 0 or none; a `geo` row has an owner default below 1 and no `z` | FR-CLB-003 |
| off value | an `off=<number>` is a level of the row's grid, or `off=min`/`off=max` is reached by a grid level (read from the owner's bound, never from the row) | FR-CLB-071 |
| solver flags | a field the programme reads (VIA 3.1 to 3.3) lacks `in`, or a row has `in` that VIA does not read | FR-CLB-009 |

The last check is **a review item until VIA lists the paths it reads** (EXP open
question 7): the test then asserts the list; until then the flags are this spec's
reading of VIA 3, confirmed by VIA's owner (section 23).

### 11.2 The record check (tier S)

A **parser** of the record of 8.2 is a function that reads external bytes, so it
has a **fuzz test** (`testing.md` 7.4: never panics, `Parse(Render(r))` is `r`, an
unknown keyword is an error). The checks on a record: every keyword is of the closed
set; every `knob` token resolves to a registry ID; every `fact` is a fact of STY;
every `set` key names a K or D row (FR-CLB-050's twin); `feasible n of N` has `n` at
most `N`; a `chosen` index is below the number of points; `screened_in` and `dormant`
are disjoint; a `set` names no knob of `dormant` unless the decision is `revert` (FR-CLB-062); a screen whose elasticity sign differs from the registry's sign of the row has a `finding` line (FR-CLB-060); the number of `screened_in` knobs of a stage is at most its `measure` lines unless a `factor` is a composite (FR-CLB-063); and, for a stage that ends `stop`, every `fact` line of the stage has a
`result` of `pass`, `report` or `na`; a `claim` of `stable` has three `stability` lines (default, lower, upper), every verdict STABLE and no edge, and a `claim` of `one_neighbour` has two with one edge; a `stability` line with `set holdout` names seeds that REF defines (FR-CLB-039, FR-CLB-100). Requirements: FR-CLB-040, FR-CLB-041, FR-CLB-042, FR-CLB-043, FR-CLB-039, FR-CLB-100.

### 11.3 The pure rules (tier S, `internal/calibration`)

Table-driven (`testing.md` 3.1), each with the arithmetic of its expectation
(`testing.md` 5.5):

| Test | Asserts | Requirement |
|------|---------|-------------|
| the grid | `geo`, `lin` and `set` levels for a table of (base, bound) including a base at a bound, a base of 1, a base of 0 (`lin` only) and a half-to-even tie; the base is always a level; levels are ascending, unique and in bounds | FR-CLB-010, FR-CLB-011, FR-CLB-013, FR-CLB-014, FR-CLB-108 |
| the neighbours | the nearest valid level on each side; an invalid level is skipped; a side with none is an edge of the right kind | FR-CLB-012 |
| the plan | the screen is the base and two neighbours of each knob with at least 10 replicas by default; the explore has `max(16, 10 d)` points on the block of the screen | FR-CLB-022, FR-CLB-023 |
| the seed blocks | block `b` is `100 b + 1` to `100 b + 100`, disjoint from 1 to 50 for `b` at least 1; a block is not reused by a second choice in a record | FR-CLB-020, FR-CLB-021 |
| the choice | the feasible set; the base chosen when feasible; the least grid distance; the tie order; no choice when empty; the Pareto set; **no weighted sum is computed** | FR-CLB-030, FR-CLB-031, FR-CLB-032, FR-CLB-033, FR-CLB-034, FR-CLB-106 |
| the alias test | collinear vectors, independent vectors, one measure, a zero vector, the 128-bit comparison at its largest inputs | FR-CLB-061 |
| the record | render, parse, render again; the order of lines; no float and no map order; the `stability` and `claim` lines | FR-CLB-041, FR-CLB-043, FR-CLB-039, FR-CLB-100 |
| the process rules | a pure check over a sequence of records: a stage order with the re-entry rule, three failed confirms, a stage with an open X failure, a seven-condition stop | FR-CLB-027, FR-CLB-035, FR-CLB-036, FR-CLB-038 |
| the overlay | a choice renders to the document of EXP 3.3.4 byte for byte; a key of class S or P is refused | FR-CLB-050 |

### 11.4 The dry run (integration, `-short` skip)

The procedure, run end to end on the reference chain with a **tiny** experiment,
to show it is a procedure and not a prose (`testing.md` 6.4 is its model). It is an
integration test, begins with the `testing.Short()` skip, and **asserts no economy
number**:

- **Setup.** Pack `one`, 400 ticks, transient 100, windows of 100; a screen of
  `OneAtATime` over two knobs (`World.CostFloor`, `geo`; `World.BeliefEvidenceShare`,
  `lin1000`) at their base and neighbours, so five points; three replicas on **search
  block 1** (seeds 101 to 103); a confirm on seeds 1 to 3 (a prefix of the confirm range) with a control; criteria
  that every point meets (a run's status `ok`), and a second set that no point
  meets.
- **Asserts.** (1) the plan is the base and the two neighbours of each knob, and
  `Check` resolves them with no tick run (FR-CLB-012; the dry run sets three replicas as a declared test constant, and the default of 10 is asserted by the plan test of 11.3, FR-CLB-022); (2) the choice is
  a function of the stored result: running `choose` again on it gives the same index
  (FR-CLB-034); (3) with criteria every point meets, the **base is chosen** and the
  overlay is **empty** (FR-CLB-031); (4) with criteria none meets, the result is
  `chosen none` and a record whose `decision` is not `stop` (FR-CLB-033, FR-CLB-038);
  (5) the seeds used by each step are the ones of 7.1: the screen on block 1, the
  confirm on the confirm range, and no search seed lies in 1 to 60 (FR-CLB-021); (6) the overlay of a choice parses and validates, and the world built from it has
  the `ConfigHash` the record names as `config chosen` (FR-CLB-050); (7) the whole procedure run twice gives the same
  record bytes (FR-CLB-041); (8) a neighbour that fails validation is **recorded and
  the run goes on** (FR-CLB-012, FR-EXP-050).
- **Leg B (the rest of the procedure).** Same pack and length, three further runs of the
  driver: (a) a knob with the flag `d` (`Information.RumourErrorBP`, `two` with the trade
  overlay) is compared by `Dist` and `PerPointSeeds`, not paired (FR-CLB-024); (b) a
  knob with the flag `in` (`Role.ExpectedWorkTicks`, K-WEL06) regenerates the table
  per point with `Prepare`, the record names the digest of the table (FR-CLB-037,
  CLB-A4); (c) a knob with an invalid level (`World.BeliefStep` with the money unit of
  19.1) is skipped and the neighbour is an `edge:rule` (FR-CLB-012, FR-CLB-108);
  a third criteria set on which the base fails and one neighbour passes gives that neighbour,
  with the overlay of one key that parses (FR-CLB-032, FR-CLB-050, FR-CLB-051); an explore
  whose feasible points touch its end is widened by two levels on a new block
  (FR-CLB-023, FR-CLB-026); three failed confirms in a row stop the stage with no fourth
  attempt before a `review` (FR-CLB-025, FR-CLB-027); a two-stage order in which the
  second stage moves the first one's knob needs the `reentry` (FR-CLB-035, FR-CLB-036);
  a stage with a failing X fact searches nothing (FR-CLB-036); the stop needs all seven
  conditions (FR-CLB-038).
- **Cost.** 5 points of three seeds of 400 ticks and a confirm of two points of
  three: at the placeholder of 4 ms a tick about 35 seconds. A placeholder.

### 11.5 The overlay purity test (tier G)

A test over every `calibrated/*.toml` of `testdata/chains/` (when any exists)
asserts that each key of each document is a K or D row of the registry (FR-CLB-050)
and that each document begins with the header comment of 9.1 (FR-CLB-051).

### 11.6 What the tests do not do

They do not assert a band: no number of the economy is asserted by this spec's tests
(N-027). The scenarios that assert bands are STY's. They do not run the stages: a
stage is a calibration of a consumer's pack and its output is a record, not a test
result. A record of the library's own defaults is **checked** by 11.2 and **read** by
a reviewer; it is not a golden file.

### 11.7 Which test covers which requirement

Every requirement of 21 is in one row; a requirement with no test says so and is a
**review item**, with its owner.

| Requirement | Test |
|-------------|------|
| FR-CLB-001, 002, 003, 005, 006, 007, 008, 101, 102, 103, 104, 105, 107, 108 | the registry test (11.1) |
| FR-CLB-004 | review item: a change that adds a field adds its row; the registry test fails the owner table coverage (FR-CLB-002) the same change |
| FR-CLB-009 | review item until VIA lists the paths the programme reads (11.1); then the registry test |
| FR-CLB-010, 011, 013, 014 | the grid test (11.3) |
| FR-CLB-012 | the neighbours test (11.3); the dry run, leg A assert 8 and leg B (c) |
| FR-CLB-020, 021 | the seed blocks test (11.3); the record check "blocks" (14) |
| FR-CLB-022, 023 | the plan test (11.3) |
| FR-CLB-024 | the dry run, leg B (a) |
| FR-CLB-025, 026 | the dry run, leg B |
| FR-CLB-027, 035, 036, 038 | the process rules test (11.3); the dry run, leg B |
| FR-CLB-030, 031, 032, 033, 034, 106 | the choice test (11.3); the dry run, leg A asserts 2 to 4 |
| FR-CLB-037 | the dry run, leg B (b) |
| FR-CLB-039, 100 | the record test (11.3) and the record check (11.2) |
| FR-CLB-040 | the dry run, leg A assert 6 (the reproduction of the digest) |
| FR-CLB-041, 042, 043 | the record test (11.3), the fuzz test of the parser (11.2), the dry run assert 7 |
| FR-CLB-050, 051 | the overlay test (11.3), the purity test (11.5) |
| FR-CLB-052 | the dry run, leg B (b): the table is generated last |
| FR-CLB-053, 054 | review item: a pack lineage is a document (9.2); the record check names `lineage` and fails a comparison across lineages (8.5) |
| FR-CLB-060 | the record check: a `finding` line for a sign that differs (11.2) |
| FR-CLB-061 | the alias test (11.3) |
| FR-CLB-062, 063 | the record check: a `set` line of a dormant knob, a stage with fewer measures than knobs (11.2) |
| FR-CLB-070, 072, 073, 074, 080, 081 | review items (process rules of 3.5 and 7.5); the record check lists the `decision` and `unmet` lines |
| FR-CLB-071 | the record check "no off value" (14) |
| FR-CLB-090 | the `-short` marker of the tests of 11.1 to 11.3 (review) |
| FR-CLB-091 | the dry run is skipped with `-short` (11.4) |
| NFR-CLB-001 | the registry test imports only the standard library (lint) |
| NFR-CLB-002 | the record test on two platforms in the nightly (11.3) |
| NFR-CLB-003 | `go vet` and the import check of architecture.md 2.9 (lint) |
| NFR-CLB-004 | the fuzz tests of the parser, the grid and the choose function (11.2, 11.3) |
| NFR-CLB-005 | the coverage run of `testing.md` 9.1 |

## 12. What CLB hands to the first code change, and what the specs owe

### 12.1 What the first code change must create

The first change that adds the code of the experiment harness and the stylised
tests creates, in the same change:

1. **`internal/calibration/`**: the pure rules of 11.3 over primitive slices
   (standard library only, no import of the root, `config/` or `strategy/`,
   architecture.md 2.9): the grid and neighbours, the seed blocks, the choose and
   Pareto functions, the alias test, the record render and parse and the overlay
   renderer. It holds **no state, no float, no clock and no goroutine**
   (NFR-CLB-003).
2. **The registry test** of 11.1 and the record checks of 11.2, reading this spec
   and `specs/README.md`.
3. **`testdata/calibration/`**, with a `README.md` of the record's grammar and the
   fixture of the dry run (11.4).
4. **The dry run** (11.4) and the overlay purity test (11.5).
5. **The layout lines**: `internal/calibration/` in architecture.md 2.6 and in
   `AGENTS.md`, and the terms of the glossary of section 23 in `doc.go`.

It creates **no public package** and **no exported identifier**. The coverage
threshold is the ordinary 90% of each package under `internal/` (`testing.md` 9.1).

### 12.2 What the other specs owe

| To | What | Why |
|----|------|-----|
| `specs/README.md` | a sentence under "Parameter owners" that the **class, the kind of range, the expected effects and the interactions** of each row are in `spec-calibration.md` section 6, found by (scope token, parameter) | the "ranges" of the roadmap row; a column would restate the owners' bounds and drift (section 4.5) |
| `spec-config.md` | 9.2: the parameter registry has a second reader, the registry test of CLB 11.1; 9.3 gap list: "the solve options are rows of CLB's registry" | FR-CFG-047 |
| `spec-viability.md` | 11: the sentence "rows in the parameter-owner table" becomes "rows of the registry of `spec-calibration.md`" (6.2.8); and **the list of the configuration paths the programme reads** (flag `in`), so that EXP's `stale_warm_start` is on by default | EXP open question 7 |
| `spec-stylised-facts.md` | 8.2: add the **search blocks** (7.1) to the seed sets, with the rule that the gate is never an input to a choice; open question 3 (where the detectors live) | the third seed role was not defined |
| `spec-experiments.md` | 12.4: the steps carry the rules of 7.2 and the blocks of 7.1; open question 7 is answered by the `in` flag | |
| `spec-stability.md` | 9 (the checklist): a step "**classify each new tunable and add its registry row** (`spec-calibration.md` 4.3)"; 11: no change | FR-CLB-004 |
| `roadmap.md`, `documentation_index.md` | CLB `exists`; rows for the spec and `specs/models/clb/` | the roadmap is updated in the change that closes a spec |
| `notes.md` | N-015's text; the new notes of 20 | |
| `doc.go`, `doc/rules/testing.md` 6.7, `AGENTS.md` | the glossary terms of 23, the one-sentence registry-test row and the layout line of `internal/calibration/` | the rows of 23 that FR-CLB-004 and the registry test need |
| `spec-reference-chain.md` | 4.1, 8.2 | the overlay directory `calibrated/`, the records `testdata/calibration/<pack>/`, and the hold-out and confirm lengths above |

### 12.3 What the first consumer does

It adds its pack under `examples/<name>/` and calls the procedure of section 7 with
`experiment` and `indicator`. It needs the **regime detectors** to close a stage and
does not have them in version 1 (open question 1): until they are public it
classifies by `Criterion`s on the measures the detectors read (`Pinned.Share`,
`PriceRange`, `Households`, `Turnover.Volume`), which is a coarser instrument and is
the reason the question is open.

## 13. Parameters

This spec adds **no `Config` field, no default and no row** to the parameter-owner
table. A calibration changes no result of a run by itself (it chooses a configuration
before a run), so nothing in it is a tunable (architecture.md 9.5), and the registry
test of `testing.md` 6.7 needs nothing from it. The numbers it fixes are of two
kinds:

- **procedure constants**, rule constants of `internal/calibration` and of the checks
  (architecture.md 9.6), all **placeholders (P)** that the first reviewed run may move;
  changing one is a change to this spec and is explained as `testing.md` 7.2 asks:

| Constant | Value | Where | Why it is what it is |
|----------|-------|-------|----------------------|
| grid factor | 2 | 5.2 | a halving and a doubling are the unit step of a first look; no claim that the response is geometric |
| `lin` steps | per row | 6.2 | a step of the unit that the owner's default is written in (250 bp, 1 000 bp, one tick); no claim |
| explore interval | levels -2 to +2 | 5.3 | a factor of four each side of a `geo` base |
| widening | two levels, on a new block | 5.3 | |
| `screen_share_bp` | 500 | 7.2.1 | an effect range that is under 5% of the stage's total is not screened in |
| explore size | `max(16, 10 d)` | 7.2.2 | a Latin hypercube of ten points a factor is the usual first design; 16 is the floor |
| replicas, search | 10 | 7.1.3 | the least FR-STY-078 allows a band to be set from |
| replicas, confirm | 50 (tier L 10) | 7.1.3 | STY's setting seeds 1 to 50 (1 to 10) |
| block width, first block | 100, `Seed0` 101 | 7.1 | above STY's seeds 1 to 50 |
| attempts before review | 3 | 7.1.4 | three independent blocks that all miss |
| `alias_tol_bp` | 500 | 10.2 | `cos^2` of 0.95 |
| claim set | 49 K and D rows | 7.4.2 | the K and D rows of the sweep table of STY 10.1, CLB's count |

- **no value of a knob** is in this spec: a value is a result of a record (8).

## 14. Validation

No rule is added to `spec-config.md`. The checks of this spec are checks of its **own
data** and of the records and overlays a calibration produces, run as unit tests
(tier S) or, for the overlay, as a check of files:

| Check | Fails when | Requirement |
|-------|------------|-------------|
| registry | any failure of 11.1 | FR-CLB-001, FR-CLB-002, FR-CLB-003, FR-CLB-005, FR-CLB-006, FR-CLB-007, FR-CLB-008, FR-CLB-101, FR-CLB-102, FR-CLB-103, FR-CLB-104, FR-CLB-105, FR-CLB-107, FR-CLB-108 |
| record | any failure of 11.2 | FR-CLB-040, FR-CLB-041, FR-CLB-042, FR-CLB-043, FR-CLB-039, FR-CLB-100 |
| overlay | a key of a `calibrated/` document is not a K or D row, or the header comment is missing | FR-CLB-050, FR-CLB-051 |
| blocks | a record names a search block used by an earlier record's choice, or a seed of 1 to 60 in a search line, or a search seed in a confirm line | FR-CLB-020, FR-CLB-021 |
| attempts | a stage has a fourth attempt with no `review` decision before it | FR-CLB-027 |
| no off value | a record's `set` line moves a D knob to its `off=` value without a `note` that says the loops left undamped | FR-CLB-071 |

## 15. Observability

The Observability entry (observability-and-logging.md 9): **None, because** this
spec adds no failure path that is survived, no state transition of a world, no
decision point in a strategy, clearing or spawner, no loop in a tick and no metric.
A calibration reads the events and the snapshot that OBS defines through IND and EXP,
and writes a record; it never writes a log line (architecture.md 11.1). Two things so
that a reader does not look for them:

15.1. **No logger and no event.** `internal/calibration` has no `*slog.Logger`; a
failure is a returned error and a decision is a line of the record (observability-and-logging.md
2.3: anything a caller needs belongs in the return value).

15.2. **The counters that make a damper's action visible are the owners'.** The
dormant rule (10.4) reads the counters STB's catalogue names (`EntryLimited`,
`BeliefsUpdated`, `MoneyFlowEnded`, ...). A damper with no counter (the production
halt, N-067) cannot be shown dormant, and the record says so; the ask for its counter
is N-067's, not this spec's.

## 16. State contributed to a checkpoint

**None.** A calibration is a sequence of experiments and a record; a checkpoint is
the state of one world, and no calibration state is in it (EXP 16). A branch's
checkpoint (7.3.6) is internal to a unit of EXP. The overlay and the record are
files of the consumer and are not part of any checkpoint; a restore checks the
configuration hash, which a calibration overlay is part of like any document.

## 17. Entries for the stability register

Provisional IDs (N-103); the supervisor assigns the final numbers. The checklist of
`spec-stability.md` 9 was followed. This spec adds **no feedback loop of the library**:
no rule of a run reads a measure and moves a knob (FR-STB-003, IND 3.1), and the
calibration loop is the consumer's (3.1). It adds six **hazards of method**, in the
manner of IND-A2 and STY-A1, five dampers (all of method) and three scenarios.

- **CLB-A1. A knob fitted to a band the same code set.** *Kind:* hazard of method (a
  neighbour of STY-A1). *Loop:* a knob (+) a measure (-) the distance to a band (+) the
  knob, with the band itself set from the code under test, so the loop closes
  without the library moving. *Owner:* CLB 3.3, 7.1. *Damper:* CLB-D1 and CLB-D2: the
  targets are the acceptance set and the priors of size H, M or E; a prior of size P is
  never a target; bands are set **after** a calibration from seeds it did not choose
  on. *Undamped:* a reviewer's judgement of a prior's size. *Test:* CLB-S1.
- **CLB-A2. The winner's curse.** *Kind:* hazard (EXP-A2 restated for the procedure).
  *Loop:* none: the best of `N` noisy pass shares has its noise selected for. *Owner:*
  CLB 7.1.2. *Damper:* CLB-D1 (a block serves one choice) and CLB-D2 (a confirm on
  other seeds). *Undamped:* a confirm so small it has the same noise. *Condition:*
  the confirm share, not the search share, is claimed. *Test:* CLB-S1; the arithmetic
  is `stop.py` (an 85% point passes a 10-seed search among 16 points with certainty and
  a 50-seed confirm 22% of the time).
- **CLB-A3. A damper disabled to pass a fact.** *Kind:* hazard of method. *Loop:* a
  failing fact (+) a damper moved to off (-) the failing fact, which hides the loop and
  leaves it undamped. *Owner:* CLB 4.3.1, 7.4.1. *Damper:* CLB-D3 (the off value is
  never a calibration move) and the dormant rule (a damper that never acted is not
  moved). *Undamped:* a pack that chooses the off value as structure and says so.
  *Test:* the check "no off value" of 14.
- **CLB-A4. A stale warm start across a calibration.** *Kind:* hazard (EXP-A5 for the
  procedure). *Loop:* none: a calibration moves an `in` knob and the table was solved
  for the old value, so the points start away from their own equilibrium. *Owner:* CLB
  4.2.1, 7.3.11, 9.1. *Damper:* the flag `in`, the re-entry rule and the warm start
  generated last. *Test:* the dry run's check that a point's `config chosen` hash is
  that of a document set whose table was regenerated.
- **CLB-A5. An alias read as two knobs.** *Kind:* hazard. *Loop:* none: two knobs
  that the facts see only through a product are both "calibrated", and the values
  found are one equation in two unknowns. *Owner:* CLB 6.5, 10. *Damper:* the alias
  groups, the alias test, the composite factor. *Undamped:* an alias no spec has
  named, found only by the test. *Test:* CLB-S3.
- **CLB-A6. The gate as a training set.** *Kind:* hazard of method. *Loop:* a gate
  verdict (+) a knob moved because of it (+) the next gate verdict, which makes the
  held-out check a fitted one. *Owner:* CLB 7.1, 7.4.4. *Damper:* CLB-D4 (a gate
  failure is a finding that returns the stage to a new block) and CLB-D5 (attempts
  counted, review after three). *Test:* the check "blocks" of 14.

Dampers of method: **CLB-D1** a search block serves one choice; **CLB-D2** the
confirm is on other seeds, with a control; **CLB-D3** a D knob is never moved to its
off value; **CLB-D4** the gate is never an input to a choice; **CLB-D5** attempts are
counted and a fourth needs a review. A sixth, **minimal change** (7.2.3), is not a
damper of a library loop but is what keeps a calibration from drifting: the default
is feasible or it is changed by as little as it can be.

Scenarios (tests of the procedure, against the model of its own rules): **CLB-S1**
*the dry run* (11.4); **CLB-S2** *coverage*: the registry test against the owner
table; **CLB-S3** *the alias test*: a synthetic sensitivity matrix with a known
proportional column and a known independent one.

## 18. What was run

Seven scripts, standard library only, archived in `doc/specs/models/clb/`, run only in
Docker (`python:3.12-slim`, `PYTHONDONTWRITEBYTECODE=1`, never the host interpreter;
the one slip is in the handoff). They are models of the **rules of this spec**, run on
synthetic or invented values; nothing in them is a run of the library, a default or a
band (N-027).

| Script | Output | What it does |
|--------|--------|--------------|
| `registry.py` | `registry.out` | parses the registry of section 6 as plain text and runs the checks of 11.1 against the owner table of `specs/README.md`, the facts of STY and the register of STB: cells, closed sets, references, owners, **coverage of all 255 owner pairs both ways** |
| `grid.py` | `grid.out` | the grid and neighbour rule of 5.2 in integers: half to even, `geo`, `lin`, the skip of an invalid level and the edge record; 20 000 random (base, bound) cases check the levels are ascending, unique, in bounds and contain the base; the table of 19.1 |
| `stop.py` | `stop.out` | the exact binomial arithmetic of the stop rules: SR2 at 10 and 50 seeds, the joint pass of 9, 27 and 45 conjuncts, the winner's curse of a search among 16 and 64 points, attempts |
| `cost.py` | `cost.out` | the run counts and core-hours of the stages (7.3) and of a stability record, from the placeholders of `spec-scale.md` 6, EXP 10 and CHK 10.3 |
| `choose.py` | `choose.out` | the feasible set, the minimal change, the Pareto set and the integer alias test of 7.2.3, 7.5 and 10.2 |
| `worked.py` | `worked.out` | every number of the examples of 19, asserted (imports the others) |

What they do **not** show: that any default is stable, that any prior is right,
that a knob has the sign a row expects, that the placeholders of 13 are good ones, or
anything about the library, which has no code. They show that the rules are well
formed, that the registry is complete against the owner table as it is today, and
what the seed arithmetic says about the stop rules.

## 19. Worked examples

An **invented** setting (N-017): the prices and measures are from the reference
chain's placeholders or are made up for the arithmetic, and no setting word is used.
`worked.py` asserts every figure.

### 19.1 Grids and neighbours

`World.CostFloor`, base 7 000, legal range 1 to 1 000 000: the levels near the base
are 1 750, 3 500, **7 000**, 14 000 and 28 000; the neighbours are **3 500 and 14 000**
(a `CostFloor` of 14 000 draws `cost_floor_at_full_cost`, a warning and not an error,
so it is a neighbour).

`World.BeliefStep`, base 200, legal range 1 to 2 500: the levels are 50, 100, **200**,
400, 800. The resolution floor `ceil(10 000 / BeliefStep)` is 200, 100, **50**, 25
and the cheapest tradable commodity of `one` has the reference price 96 (REF 4.4, a
placeholder). A `BeliefStep` of 100 gives a floor of 100, **above 96**: the level is
invalid (I01), and so is 50. The default has **no lower neighbour**: `edge:rule`. Its
claim is `one_neighbour`: "no regime detected at the default and at 400, with the region below an edge of the
pack's own money unit not examined" (7.4.3), and the lower edge is a **finding about
the pack**: raise the money unit (A08) or accept it.

`World.EntryShare`, base 50, legal range 0 to 10 000: 12, 25, **50**, 100, 200
(`rdiv(50, 4)` is 12.5, which half to even makes 12); neighbours 25 and 100. The
level 0 (role change off) is **never** a `geo` level.

`Role.MinWorkShare`, base 4 000, legal range 0 to `ExpectedWorkTicks * 10 000 /
TicksPerYear` = 250 * 10 000 / 365 = 6 849 (I05): `lin1000` gives 2 000, 3 000,
**4 000**, 5 000, 6 000; neighbours 3 000 and 5 000.

`Subsistence.ExitTicks`, base 3, `lin1`, with `EnterTicks` 2: the level 2 is invalid
(`ErrOwnUseBand`), so the lower side is an edge; the upper neighbour 4 is **legal**
but, at `CoverTicks` 3, is higher than purchases alone reach (I07): the stage moves
`CoverTicks` with it, as a composite.

### 19.2 A screen

On a search block of ten seeds, `World.CostFloor` at 3 500, 7 000 and 14 000 gives a
mean `Pinned.Share` of 300, 600 and 1 500 bp. The elasticity at the upper neighbour is
`rdiv(10 000 * (1 500 - 600) * 7 000, 600 * (14 000 - 7 000))` = **15 000** and at the
lower `rdiv(10 000 * (300 - 600) * 7 000, 600 * (3 500 - 7 000))` = **10 000**: both
positive, so the row's `P02/Pinned.Share:+S` holds. `World.BeliefEvidenceShare` at 4 000,
5 000 and 6 000 gives `Offset.abs` 380, 470 and 560 (elasticity **9 574** both ways,
`+M` holds) and `s01.recover_ticks` 70, 60 and 55 (elasticity **-4 167** up and
**-8 333** down, `-M` holds). Their effect ranges on `Offset.abs`, with `CostFloor`'s
bins at 520, 470 and 410, are 110 and 180 bp, so their **shares** are
`rdiv(10 000 * 180, 290)` = **6 207** for the evidence share and **3 793** for the
floor; both are above `screen_share_bp` 500 and are screened in.

A **finding**: `World.ProduceHaltRatio` at 30 000 and 60 000 gives `StockCover` 800 and
700 bp. The elasticity is `rdiv(10 000 * (700 - 800) * 30 000, 800 * 30 000)` =
**-1 250**, while the row says `M05/StockCover:+S` (STY 10.2). The row's source is `S`,
so it is the owner's spec that is in question: the finding is `open`, and the stage
does not stop until it is `explained` or the row is corrected (7.4.1 item 5).

### 19.3 A choice

Two knobs (`CostFloor` and `BeliefEvidenceShare`), seven points, ten seeds, three
criteria that need every seed (the exact facts and the detector) and two B criteria
that need nine (`ceil(9 * 10 / 10)`):

| Point | CostFloor | Evidence share | P02 passes | P04 passes | feasible | grid distance |
|-------|-----------|----------------|-----------:|-----------:|----------|--------------:|
| 0 (the base) | 7 000 | 5 000 | 7 | 10 | no | 0 |
| 1 | 3 500 | 5 000 | 9 | 10 | **yes** | 1 |
| 2 | 14 000 | 5 000 | 4 | 10 | no | 1 |
| 3 | 7 000 | 4 000 | 8 | 10 | no | 1 |
| 4 | 7 000 | 6 000 | 6 | 9 | no | 1 |
| 5 | 3 500 | 6 000 | 10 | 10 | yes | 2 |
| 6 | 3 500 | 4 000 | 10 | 10 | yes | 2 |

The base is not feasible (P02 passes 7 of 10, below 9), so the feasible points are 1,
5 and 6; the least distance is **point 1**: `CostFloor` 3 500 only. Points 5 and 6
pass more seeds on P02 and are not chosen: a calibration does not take more than it
needs. The Pareto case: four points that pass the exact facts with (P02, P04) passes
(10, 5), (7, 9), (6, 6) and (9, 4) have no feasible point, and the non-dominated ones
are the first two: the review decides (7.5).

### 19.4 The seed arithmetic

A point that is inside its band on 85% of seeds passes a ten-seed SR2 (need 9) with
probability 5 443 bp, so the best of 16 such points passes it with probability 9 999.97
bp, 10 000 to the nearest basis point (1 - (1 - 0.5443)^16, which is not certainty): **a search among 16 mediocre points certainly finds one that
looks good**. On 50 confirm seeds (need 45) such a point passes with probability
**2 194 bp**: a calibration that stopped at its search would call it stable, one that
confirmed rejects it four times in five. A point at 95% passes the ten-seed gate
9 139 bp of the time and the confirm 9 622. The **joint** pass of a stability record
is the harsh number: at 99% a conjunct passes a ten-seed gate 9 957 bp of the time and
63 independent conjuncts (3 checks, 7 packs, 3 points) pass together 7 639 bp; at 95%,
34 bp. That is why the bands are set with headroom (STY 8.4.3: a ten-seed hold-out
all inside in 98.5% to 99.95% of trials at 50 setting seeds) and why a claim is made for a **set** of knobs
only where the checks are needed (7.4.2).

### 19.5 A record

(Invented values; the format is 8.2. The `claim` is `undetermined`: the third check waits for a key that is unset, STY 7.2, so no default is called stable; the line of the upper neighbour is left out for brevity.)

```
calibration 1
pack one  lineage 5a1c0e4477d3b210  stage C4a  attempt 1  block 1
commit 4f2a9c1  parent -
config base 91d3a7c2e05b4a18  chosen 0be4417f6a2d93c5
document base.toml 3b9c11aa5e07f4d2
document one.toml 77e0d2c9b3a1608e
document warm-one.toml 0be4c0319d7a52f6
experiment screen design oat points 33 replicas 10 seed0 101 seeding same ticks 1095 transient 365 every 365 salt 0 render c7f20b19a4e83d60
experiment explore design spread points 60 replicas 10 seed0 101 seeding same ticks 1095 transient 365 every 365 salt 0 render 4e9d02b6f18c7a35
experiment confirm design replicate points 2 replicas 50 seed0 1 seeding same ticks 1095 transient 365 every 365 salt 0 render a07715fe62bd9c04
factor cost_floor key World.CostFloor grid geo base 7000 lower 3500 upper 14000 edge none
screen K-BEL12 measure pinned_share_bp el_lower 10000 el_upper 15000 range 1200 share_bp 7500
screened_in K-BEL07 K-BEL12
dormant K-BEL14 K-BEL15
feasible 3 of 60
chosen 1 distance 1
set World.CostFloor 3500
confirm seeds 1..50 control yes
fact STY-P02 class B key s16.pinned_share_bp target report values min 40 median 130 max 410 outside 23 result report
fact STY-P04 class B key s01.offset_bp target working values min 310 median 780 max 1210 outside - result pass
detector R5a none working yes onset -
stability K-BEL12 pack one point default value 7000 set gate seeds 1..10 in_sample yes c_bp STABLE c_nc STABLE c_ws UNDETERMINED edge none
stability K-BEL12 pack one point lower value 3500 set gate seeds 1..10 in_sample yes c_bp STABLE c_nc STABLE c_ws UNDETERMINED edge none
claim K-BEL12 pack one undetermined
decision stop
note CostFloor halved; the base missed P02 on 3 of 10 search seeds
```

### 19.6 The overlay of the choice

The overlay of the record of 19.5 is the document of EXP 3.3.4, one table header and
one key, with the header comment of 9.1:

```
# calibration C4a attempt 1 of lineage 5a1c0e44: see testdata/calibration/one/c4a-1.txt
[world]
cost_floor = 3500
```

It is layered after `one.toml` and before the regenerated `warm-one.toml`; the
`CostFloor` is read by the solver only as a check (`chk`), so the table is unchanged
and its hash is the same.

### 19.7 An alias

Over three measures (`Pinned.Share`, `Turnover.Volume`, `PriceRange`) the elasticities
at the upper neighbour of `EntryShare` are (300, -450, 0) and of a second knob
(600, -900, 0): their `cos^2` is exactly 10 000 bp, above the 9 500 of the test, so
they are **aliased** in this stage: the facts see one lever. A third knob with
(300, 100, -200) has `cos^2` against the first of **495**: independent. With one
measure, (120) and (-35) have `cos^2` 10 000: they cannot be told apart.

### 19.8 What a stage costs

C4a: 16 knobs give a screen of 33 points; six of them screened in give an explore of
`max(16, 60)` = 60; at ten seeds that is 930 search runs of 1 095 ticks of `one`
(6.1 s each at the placeholders), plus 200 confirm runs (50 seeds, the chosen point
and the base, on `one` and `second`): **about 1.5 core-hours**. The whole of the
stages in one pass is 7 012 runs, 39 core-hours; the stability record of 49 defaults
on the gate and a hold-out, if REF defines one, is 13 860 runs, 57 core-hours (half on the gate alone).

## 20. Notes settled

Read: every open note tagged CLB (N-015) and every `ALL` note (N-016, N-017, N-027,
N-103), and the notes that name what a calibration relies on (N-018, N-028, N-061,
N-084, N-085, N-086, N-092, N-102, N-113).

| Note | Disposition | Where |
|------|-------------|-------|
| N-015 placeholders to calibrate once something has run | **Left open**, updated: the registry gives every placeholder a class and says that nearly every search range is unknown (5.1); "calibrated" now means "a record names the value and the runs it rests on" (8); the new placeholders are the procedure constants of 13 | 5, 8, 13 |
| N-016 keep the parameter registry true | **Applied in part**: the registry classifies each owner row, the solve options of VIA 11 are rows of **this** registry (not of the owner table, which resolves a disagreement between VIA 11 and CFG 9.2 in CFG's favour), and a spec that adds a field adds its row (FR-CLB-004). The registry tests of `testing.md` 6.7 and CLB 11.1 are still to be written | 4, 6.2.8, 11.1 |
| N-017 words and names | **Applied**: setting-agnostic names; the invented examples are flagged; the terms of section 23 go to the glossary | 19, 23 |
| N-027 no default macro constant is stable yet | **Applied**: no row holds a value; "stable" is the output of STY 7.3.1 and a record (7.4.2); the facts for which nothing is known say `?` | 4.5, 7.4 |
| N-103 running spec writers in parallel | **Applied**: only this spec, `models/clb/` and `.supervisor/clb.*` were created; handoff; models in Docker. **Slip:** the host `python3 --version` was run once, by reflex, after the instruction not to; nothing else ran on the host | 18 |
| N-084 scenarios that wait for trade, money and the coarse tick | **Read, left open**: the trade, faucet and external overlays are the packs of C7 and C8 (7.3.8, 7.3.9) and exist as REF and the owners say; the dynamic parts of X1 and X2 and the coarse-tick leg of S12 are STY's and no CLB stage waits on them | 7.3.8, 7.3.9 |
| N-086 re-run the demand and spawner models under the register | **Left open**, with this added: the re-runs are the experiments of C4c (the occupation model: `entry_share_window` set from the screen of `EntryShare`) and C6 (the luxury model: the elasticity and headroom of I18), whose runs are STY's and EXP's and whose steps are CLB's | 7.3 |
| N-018 unverified evidence | **Applied**: only rows "read directly" or "read in full" are E anchors; the capital return, the three apprentices and the tool prices are not (6.1) | 6.1 |
| N-028 the target-income loop | **Read**: `MinWorkShare` and `TargetSpread` are D rows (K-WEL07, K-WEL03) with the loops L14, L15, L51; whether the pinned staple needs another lever is STB's open question and S04's | 6.2.1 |
| N-092 overlays and scenarios handed to REF | **Applied in part**: a calibration overlay directory (`calibrated/`) and the records are REF's to host (section 23) | 9.1 |
| N-102 unify the module table names | **Read**: the registry's key is `<Scope>.<Field>` and does not depend on a table's spelling; EXP's path of a knob is `spec-config.md` 6.2's, settled when N-102 is | 4.2 |
| N-113 a quote is a limit | **Read**: STY-T03 is a report fact at C8; CLR adds the flag only if its share falls below `t03.pull_bp` | 7.3.9 |
| N-061 the load-time size gate is a guess | **Read**: a C2 loop over many points makes the solver cost (NFR-CFG-001) the stage's, which is a reason EXP's `Check` is the structural tool | 7.3.3 |
| N-085 nominal against full-labour table | **Read**: C2 may run both tables of `famine` side by side as STY-H02's report | 7.3.3 |

**New notes to add** (IDs are the supervisor's):

1. *ALL, CODE*: every new `Config` field needs a **registry row** of
   `spec-calibration.md` in the same change, with a class decided by the order of 4.3
   (extends N-016).
2. *VIA, EXP*: the list of configuration paths the steady-state programme reads is the
   `in` and `chk` flags of the registry; VIA confirms it and EXP's `stale_warm_start`
   is on by default (EXP open question 7).
3. *STY, EXP, IND*: the regime detectors are `internal/verdict`; a consumer's
   calibration needs them (open question 1 below), and a promotion to a public package
   is to be decided with the first consumer's own calibration.
4. *STY, REF*: the **search blocks** (101 and up, one choice a block) are a role of seeds
   that STY 8.2 does not list, and the **hold-out** 51 to 60 that STY 8.2 recommends
   is **not** relied on by CLB: a claim without one says `in sample` (7.4.2); REF
   8.2 may add the row. The working use of a prior as a threshold (3.3 item 2) is
   outside STY's `unknown` rule (6.5.1) and needs STY's agreement.
5. *CFG, CODE*: `internal/calibration` and `testdata/calibration/` are new homes
   (section 12.1) for architecture.md 2.6 and `AGENTS.md` at the first code.
6. *WEL, STB*: the D knobs whose default **is** the off value (`TargetSpread`,
   `HouseholdCap`, `Refuge`, a flat `Abundance`, `Limit`, `Exempt`) mean that the
   damper is not in force by default: a calibration moves them away from off when a
   fact shows the loop (4.3.1); the register's D08 should say so.
7. *EXP*: `MaxFactors` 16 forces the split of C4 and C7 into sub-stages (7.2.1); if a
   stage ever needs more, EXP 13.2 is the bound to revisit.

## 21. Requirements

Each states one thing that a test or a review can check; a test cites its IDs (`testing.md` 4.3). A requirement with no test is listed in 11.7 as a review item.

### Functional

| ID | Requirement |
|----|-------------|
| FR-CLB-001 | The registry MUST be the tables of section 6 of this spec; a knob row MUST have the 12 cells of 4.2 and a structure group the 7 cells of 4.2. |
| FR-CLB-101 | A registry ID MUST be unique and MUST NOT be reused. |
| FR-CLB-102 | An Owner cell MUST resolve to a section of the owner's spec that is a section of the README owner column for that pair. |
| FR-CLB-002 | Every (scope token, parameter) pair of the parameter-owner table MUST be the key of exactly one knob row or a field of exactly one structure group of that scope, and every key of the registry MUST name an owner row, except the scopes `Solve option` and `Presentation` and the field `World.Clearing`. |
| FR-CLB-003 | Every row MUST have one class of K, D, S or P, decided by the order of 4.3. |
| FR-CLB-103 | A D row MUST give its `off=` value and no other row MAY. |
| FR-CLB-104 | A flag MUST be one of the closed set of 4.2.1. |
| FR-CLB-004 | A change that adds a `Config` field, or a solve option, MUST add its registry row in the same change. |
| FR-CLB-005 | A registry row MUST NOT hold a default, a validation bound or a calibrated value, and the registry MUST NOT have a column named Default or Bound. |
| FR-CLB-006 | A Search cell MUST be `U`, `E<n>`, `I<nn>` or `-`; an `E<n>` MUST name an anchor of 6.1 whose source is read directly; a K or D row MUST NOT be `-`. |
| FR-CLB-007 | A move token MUST have the form of 4.4, name a fact of STY, a measure of an IND preset named in IND 9.1 with a field of the closed list of 4.4 or a band key of STY, a sign and, for a sign other than `?`, a source. |
| FR-CLB-105 | A row MUST hold at most four move tokens. |
| FR-CLB-008 | Every `L<nn>` and `D<nn>` of a row MUST be an entry of `spec-stability.md`, and every `I<nn>` and `A<nn>` MUST be a row of 6.4 or 6.5. |
| FR-CLB-009 | The flag `in` MUST be set on exactly the fields the steady-state programme reads, and `out` on exactly the fields the solver writes. |
| FR-CLB-010 | The levels of a `geo` grid MUST be `b * 2^k` for `k` at least 0 and `rdiv(b, 2^-k)` for `k` below 0, kept in `[max(lo, 1), hi]`, with `b` always a level, sorted and without duplicates. |
| FR-CLB-011 | The levels of a `lin<s>` grid MUST be `b + k * s` in `[lo, hi]`, and those of a `set` grid the listed values in order. |
| FR-CLB-012 | A level that `Validate` rejects MUST be skipped; the neighbours of the base MUST be the nearest valid level on each side; a side with none MUST be an edge, recorded as `edge:bound` or `edge:rule`. |
| FR-CLB-013 | Every division of the grid, the choice and the alias test MUST be integer, rounded half to even, with no floating point (architecture.md 3.5), with two stated exceptions: the seed threshold `ceil(9 R / 10)` is exact integer arithmetic `R - floor(R / 10)` (for R of 25 it is 23, and half to even would give 22), and the mapping of a Spread value to a grid level breaks a tie to the **lower** level (a rule of its own, not a division). The checked multiply of the grid reports overflow (an error, never a wrapped level). |
| FR-CLB-014 | The base MUST be the owner's default, or the pack's value where the field has none, and MUST be recorded. |
| FR-CLB-020 | A search block `b` MUST be the seeds `100 b + 1` to `100 b + 100` for `b` at least 1, MUST be disjoint from the seeds 1 to 60, and MUST serve one choice. |
| FR-CLB-021 | A choice MUST be made on search blocks only; the confirm MUST be run on STY's setting seeds (1 to 50; tier L: 1 to 10); the per-seed values of a confirm, and the verdict of the gate, MUST NOT be read into a later choice. |
| FR-CLB-022 | The screen MUST be a one-at-a-time experiment over the base and the two neighbours of each knob, with at least 10 replicas on a search block. |
| FR-CLB-023 | The explore MUST be a spread over the screened-in knobs, with `max(16, 10 d)` points on the block of the screen, an aliased knob entering as a composite. |
| FR-CLB-024 | A knob with the flag `d` MUST be compared across points as distributions, and a paired comparison MUST be used only for a knob that moves no count. |
| FR-CLB-025 | The confirm MUST run the chosen configuration and the base as a control, on the confirm seeds, with the measures and windows of the search. |
| FR-CLB-026 | If the feasible points touch an end of the explore interval that is not a bound, the interval MUST be widened by two levels and the choice made again on a new block; the search MUST NOT leave `[lo, hi]`. |
| FR-CLB-027 | After three failed confirms at one stage the stage MUST NOT be searched again before a review. |
| FR-CLB-030 | The feasible set MUST be the points on which every X, O and detector criterion passes on all seeds. |
| FR-CLB-106 | A feasible point MUST also pass every B criterion with a target on at least `R - floor(R / 10)` of the `R` seeds. |
| FR-CLB-107 | A K or D row whose Grid is `-` MUST carry an `A<nn>` link that names what is searched instead. |
| FR-CLB-108 | A level equal to the `off=` value of a D row MUST NOT be a level of its grid (`edge:off`), and a `geo` row MUST have an owner default of at least 1 or the flag `z`. |
| FR-CLB-031 | If the base is feasible the choice MUST be the base and the overlay MUST be empty. |
| FR-CLB-032 | Otherwise the choice MUST be the feasible point with the least grid distance to the base, then the fewest knobs changed, then the lowest index. |
| FR-CLB-033 | If no point is feasible there MUST be no choice and the Pareto set of the B pass counts among the points that pass every exact criterion MUST be reported; no weighted sum of facts MUST be computed. |
| FR-CLB-034 | The choice MUST be a pure function of the stored results of the screen and the explore. |
| FR-CLB-035 | The stages MUST be run in the order C0 to C9, a stage MUST NOT move a knob of an earlier one without the re-entry rule of 7.3.11, and a stage of more than `MaxFactors` knobs MUST be split by owner. |
| FR-CLB-036 | No knob MUST be searched while an X or O fact fails on the pack. |
| FR-CLB-037 | A point that changes a knob with the flag `in` MUST regenerate the warm-start table from the solver, or the experiment MUST say that it is a perturbation. |
| FR-CLB-038 | A stage MUST stop only when the seven conditions of 7.4.1 hold on the confirm seeds. |
| FR-CLB-039 | A default MUST be called stable only under STY 7.3.1: STABLE at the default and at **both** neighbours of its grid on the gate seeds, with a `stability` line for each point and a `claim` line `stable` in the record; a default with an edge on one side MUST be recorded `one_neighbour` and MUST NOT be called stable. |
| FR-CLB-100 | A `stability` line MUST say `in_sample no` only for a `holdout` set that REF defines and that no choice has read; every other line MUST say `in_sample yes`. |
| FR-CLB-040 | A record MUST have the lines of 8.2 in order, and it MUST be reproducible: the documents, the hash, the experiment and the seeds MUST give the digest of each rendered result. |
| FR-CLB-041 | A record MUST contain no floating point, no map order and no wall-clock time, and two runs of one calibration MUST give the same bytes. |
| FR-CLB-042 | A record MUST name the configuration hashes and documents, every experiment, the results, the bands met and unmet, the detectors with `working`, the attempt and block, and the decision. |
| FR-CLB-043 | Parsing the rendering of a record MUST give the record, and an unknown keyword MUST be an error. |
| FR-CLB-050 | A calibration overlay MUST contain only keys whose registry class is K or D. |
| FR-CLB-051 | A calibration overlay MUST begin with a comment that names its record. |
| FR-CLB-052 | The warm-start document MUST be generated from the effective configuration of every other document, calibration overlays included, last. |
| FR-CLB-053 | A change to a row of class S MUST start a new lineage and every stage MUST be run again. |
| FR-CLB-054 | A band value, a stability record, a warm-start table and a calibration overlay of one lineage MUST NOT be applied to another. |
| FR-CLB-060 | The screen's elasticities and effect ranges MUST be kept in the record, and a measured sign that differs from the registry's MUST be a finding. |
| FR-CLB-061 | Two knobs MUST be aliased when `10 000 (e_i . e_j)^2` is at least `(10 000 - alias_tol_bp) (e_i . e_i)(e_j . e_j)`, in 128-bit integers, and a zero vector MUST NOT be aliased. |
| FR-CLB-062 | A D knob whose counter never fired MUST NOT be moved, and a D knob that moved and stayed dormant MUST be reverted. |
| FR-CLB-063 | A stage with fewer measures than screened-in knobs MUST fix or compose the knobs beyond the number of measures. |
| FR-CLB-070 | No knob MUST change inside a run because of a measure of that run. |
| FR-CLB-071 | A D knob MUST NOT be moved to its off value as a calibration move; a pack that sets it there MUST record the loops it leaves undamped. |
| FR-CLB-072 | A knob MUST NOT be moved to meet a B fact whose prior has size P. |
| FR-CLB-073 | A band, a regime threshold, a validation bound, a measure or a window MUST NOT be changed to make a fact pass. |
| FR-CLB-074 | No configuration value MUST depend on a seed, or on which seed failed. |
| FR-CLB-080 | When two facts conflict the pair MUST first be looked up in STY 5.9, then ordered by class, and an unmet fact MUST be recorded and MUST NOT be dropped from the acceptance set. |
| FR-CLB-081 | The decision on an empty feasible set MUST be one of changing the structure, correcting a prior in STY, or accepting a recorded known miss, and MUST be written in the change. |
| FR-CLB-090 | The registry test, the record checks and the unit tests of the pure rules MUST run under `-short`. |
| FR-CLB-091 | The dry run of 11.4 MUST be an integration test, skipped with `-short`, that asserts no number of the economy. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-CLB-001 | The registry test MUST need no dependency outside the standard library and MUST read the registry and the owner table as plain text. |
| NFR-CLB-002 | A record and a rendering MUST be identical on every platform (architecture.md 7.4). |
| NFR-CLB-003 | `internal/calibration` MUST import only the standard library, MUST NOT import the root package, `config/` or `strategy/`, and MUST NOT use floating point, a clock, a random source, package-level mutable state or a goroutine. |
| NFR-CLB-004 | The record parser and the grid and choose functions MUST have fuzz tests that assert no panic and the stated identities (testing.md 7.4). |
| NFR-CLB-005 | `internal/calibration` MUST meet the coverage threshold of `testing.md` 9.1 (90%). |

## 22. Open questions

1. **Where the regime detectors and the calibration helpers live for a consumer.** They are `internal/verdict`,
   which a consumer cannot import (STY open question 3). A consumer's calibration
   needs them to close a stage, and without them it uses `Criterion`s on the measures
   they read, a coarser instrument. *Recommendation:* promote `internal/verdict` to a
   public `verdict/` package, and `internal/calibration` (the grid, the choose step and the
   record) to a public `calibration/` package, when the first consumer runs its own
   calibration (architecture.md 2.8: export when a consumer needs it, and then it is a
   promise); not in this spec's first code change. *Next step:* the supervisor, with STY.
2. **The numbers of the search.** Every placeholder of 13 and every `U` of the
   registry is waiting for a first reviewed run. *Next step:* C0 to C4a on `one`;
   the record of that run is where the first `Search` range with a number can be
   written.
3. **A working threshold in STY's terms.** STY 6.5.1 returns `unknown` for an
   unset threshold; CLB 3.3 uses the prior of STY 6.2 as a **working** threshold in a
   search and labels it. *Next step:* STY agrees or gives another guide.
4. **The size of a prior.** 3.3 makes a prior a target only if its **size** has
   confidence H, M or E. STY states shape and size separately for some facts and not
   for all (`G01`: "shape I, size P"). *Next step:* STY 3.3 states the size
   confidence of every B key in one column.
5. **A gauge's scaling check.** 10.3 leaves the check of a unit (A07 to A09) to the
   pack's author, by hand. *Next step:* the first pack that changes a unit; a helper in
   `experiment` that scales a list of paths would make it a test.
6. **The grid factor.** A factor of 2 is a placeholder; a knob whose response is
   sharply non-linear (`CostFloor` near 10 000, 4.3.1) may want a finer grid near its
   cliff. *Next step:* the first screen that shows it.
7. **Interactions.** One at a time sees none (10.6); the explore sees them only
   through bins. *Next step:* a second-order screen of the pairs of 6.4, when a stage
   shows a feasible region that is not a box.
8. **The claim set.** 49 rows is CLB's count of what the sweep table of STY 10.1 names; a smaller claim is cheaper and
   says less (7.4.2). *Next step:* the consumer decides which defaults it needs to call
   stable.
9. **Region-keyed knobs.** A knob that is a row of a region table (`BirthScale`,
   `OutShare`) is searched as one scale for all regions in 6.2; a per-region search
   is `d` rows each. *Next step:* the first pack whose regions differ in the facts.
10. **The review of a prior's size (7.5 item 4b).** Whether STY's priors are edited by
    the supervisor, by the reviewer of the run or by the author is a process question
    this spec does not settle.

### Conflicts found between specs

| # | Between | The conflict | Recommendation |
|---|---------|--------------|----------------|
| X1 | `spec-viability.md` 11 and `spec-config.md` 9.2 (N-016) | VIA says the solve options "are rows in the parameter-owner table"; CFG says they are NOT rows, because the registry test requires a field behind every owner row | CFG wins; the options are rows of CLB's registry (6.2.8); edit VIA 11 |
| X2 | `roadmap.md` (CLB row: "it grows the parameter-owner table") and the registry's rule that a default is defined once | a ranges column in the owner table would restate the owners' bounds and the registry's search ranges and drift | a pointer sentence under the owner table and the registry in this spec (12.2) |
| X3 | STY 8.2 (the gate 1 to 10 lies inside the setting range 1 to 50, so a set band is not asserted out of sample) against STY 7.3.1 ("on the gate seeds") and 10.2 ("another `Seed0`", the calibration's own hold-out) | the gate cannot be the held-out check of a stability claim, and STY 8.2 defines neither search seeds nor a hold-out it relies on | search blocks of 101 and up for a choice, REF's nightly seeds 1 to 50 for the confirm, and a hold-out only if REF defines one (7.1, 7.4.2) |
| X4 | STY 6.5.1 (an unset threshold gives `unknown`) against a search that needs a guide | a calibration cannot steer by a detector that says "I do not know" | a labelled **working** threshold (3.3), outside acceptance; STY to agree |
| X5 | EXP 13.2 `MaxFactors` 16 against stages of 21 knobs | one experiment cannot screen C7 as written | split by owner (7.2.1); no change to EXP |
| X6 | `spec-config.md` 3.7 (`World.Scale`) against the owner table (`Scale`, scope `Group; world default 1`) | the same field under two scopes | the registry follows the owner table (`Group.Scale`); noted in 6.2.1 |
| X7 | the owner table (no row for `World.Clearing`, `Label`s) against WLD 14 and `spec-config.md` 8.3 | fields that exist and are not owner rows | registry-only rows (6.3); the owner row of `Clearing` is WLD's owed edit |

## 23. Edits owed to other files

One consolidated table: file, section, exact change. None alters a result that a test
could depend on (nothing has run); the rule files need a fresh read-receipt tag each.
The **asks** (A) are requests to the owner, with this spec's recommendation.

| File | Section | Change |
|------|---------|--------|
| `doc/specs/README.md` | Index and reading order | add `spec-calibration.md` (core): "The registry of tuning knobs: one row for each tunable and solve option, classified as a knob, a damper strength, a structural choice or presentation, with its owner, its kind of range (a validation bound is never a search range; nearly every search range is unknown), its grid and neighbours, its expected effects with sign and source, its loops and the pairs that must move together; the procedure that calibrates a setting against the stylised facts (search blocks, screen, explore, choose, confirm; stages C0 to C9; stop rules), the record, the packs, sensitivity and identifiability, `FR-CLB`/`NFR-CLB`"; reading order after `spec-experiments.md` |
| | Requirement IDs table | the `CLB` row: "exists; Requirements section written (FR-CLB-001 to 091 and 100 to 108, NFR-CLB-001 to 005); core; no `Config` field; all procedure constants are placeholders" |
| | Parameter owners | one sentence under the header: "The class, the kind of range, the expected effects and the interactions of each row are in `spec-calibration.md` section 6, found by (scope token, parameter)." |
| | Decisions taken | rows: *Where is the registry of knobs?* In `spec-calibration.md` 6; the owner table is not given a column (4.1). *What is a calibration for?* Feasibility and minimal change, never an optimum (3.4). *Which seeds may a calibration choose on?* Search blocks of 100 from 101, one choice each; the confirm on STY's setting seeds 1 to 50 (REF 8.2); never the gate (7.1). *What transfers between settings?* The procedure, the conditions and the regime thresholds; defaults and priors as starting points; no band, record or table (9.3) |
| | Glossary | knob, damper strength, structural choice, registry (of knobs), search range, evidence anchor, derived condition, base value, grid (of a knob), neighbour, edge (of a grid), screen, screened in, dormant, invisible, alias group, gauge, composite (knob), search block, confirm seeds, feasible region, minimal change, grid distance, Pareto set, known miss, working target, working threshold, calibration overlay, calibration record, lineage, re-entry, stage (of a calibration), attempt, off value, structure group, scope token, overlay document, claim set, sensitivity matrix, scaling check, stage stop, program stop; the terms **gate**, **hold-out**, **finding**, **stability record**, **elasticity** and **effect range** are used as STY, EXP and IND define them and are not redefined |
| `doc/roadmap.md` | tier 3 table, existing specs table, "Suggested next specs" | CLB `planned` to `exists`; a row in the existing-specs table; remove CLB from the planned list |
| `doc/documentation_index.md` | rows | `specs/spec-calibration.md` and `specs/models/clb/` |
| `doc/specs/models/README.md` | contents | a `clb/` row: `registry.py`, `refs.py`, `grid.py`, `stop.py`, `cost.py`, `choose.py`, `worked.py` with outputs |
| `doc/notes.md` | open notes, index | N-015 updated (20); the new notes of 20; the index; a fresh tag |
| `doc/rules/architecture.md` | 2.6 | add `internal/calibration/` (the grid, the choice, the alias test, the record and the overlay over primitive values, standard library only), created at the first code that needs it (2.1); a fresh read-receipt tag |
| `AGENTS.md` | Layout | the same line |
| `doc.go` (glossary) | | the terms above |
| `doc/specs/spec-config.md` | 9.2, 9.3 | 9.2: the registry has a second reader (CLB 11.1); the solve options are rows of CLB's registry (FR-CFG-047 stays); 9.3: a line "the solve options are CLB registry rows" |
| `doc/specs/spec-viability.md` (A) | 11 | "rows in the parameter-owner table" becomes "rows of the registry of `spec-calibration.md` 6.2.8"; and a list of the configuration paths the programme reads, matching the `in` and `chk` flags |
| `doc/specs/spec-stylised-facts.md` (A) | 7.3.1, 8.2, 10.2, 20 | 8.2: the search blocks (7.1) as a role of seeds outside REF's sets; 7.3.1 and 10.2: a claim says `in sample` unless REF provides a hold-out (7.4.2); question 3 answered by CLB open question 1; the working use of a prior (CLB 3.3) |
| `doc/specs/spec-reference-chain.md` (A) | 8.2 | the search blocks as seeds a calibration uses outside REF's sets, and, if wanted, a hold-out 51 to 60 (no band set from it); FR-REF-024 and 025 stand |
| `doc/specs/spec-experiments.md` (A) | 12.4, 22 | the steps of 12.4.2 cite CLB 7.2 and the blocks of 7.1; open question 7 has the `in` and `chk` flags of CLB 4.2.1 as a candidate list, for VIA to confirm |
| `doc/specs/spec-stability.md` | 9 | a step 9.10: "classify each new tunable and add its registry row (`spec-calibration.md` 4.3)"; 14 FR-STB-003 cites CLB 3.5; the entries CLB-A1 to A6, D1 to D5 and S1 to S3 of 17 merged with their final numbers |
| `doc/specs/spec-reference-chain.md` (A) | 4.1 | the layout gains `calibrated/` (overlays) and the location of the library's records `testdata/calibration/<pack>/` |
| `doc/rules/testing.md` | 6.7 | one sentence: the registry test also reads the classification of `spec-calibration.md` 6 (optional; a fresh read-receipt tag) |
| `doc/specs/spec-indicators.md` (A) | 9.1 | a **field column** (a closed list of the result fields of each preset, with the spelling STY and CLB use: `Pinned.Share`, `RoleShares.Swing`, `HouseholdIncome`, `UpkeepShortfall`), so that the registry test of CLB 11.1 can parse it instead of a list in the test |
| `doc/specs/spec-scale.md` (A) | 5.2 and the owner table | a validation bound for `MinAgents` (it has none in SCL or CFG 4.1, so the grid of K-SCL02 is unbounded above) |
| `doc/specs/spec-stylised-facts.md` (A) | 7.3.1 | an **edge clause**: whether a default at the edge of a grid may be called stable on one neighbour with the beyond not examined; CLB 7.4.3 says it may not and records `one_neighbour` |
| `doc/specs/spec-reference-chain.md` (A) | 8.2 | the confirm lengths: a band set at a confirm is asserted by the gate at the scenario's REF length (1 095, up to 1 800), which CLB 7.3 takes; a hold-out 51 to 60 only if wanted |
| `doc/specs/spec-config.md` | 6.2 | the experiment path of a key is the snake-case file spelling of `<Scope>.<Field>` (CLB 4.2; N-102) |
| `doc/specs/spec-observability.md` | 4.4.6 | `ProductionHalted` is the counter of D24 (K-AGT09), cited in CLB 10.4 |

## 24. Where this lives in the code

| Concern | Home |
|---------|------|
| the grid and neighbours, the seed blocks, the choice, the Pareto set, the alias test, the record's render and parse, the overlay renderer: pure functions over primitive values | `internal/calibration/` (`grid.go`, `choose.go`, `alias.go`, `record.go`), standard library only; unit tests in the package |
| the registry test and the record checks | the root's external test package, a file named `calibration_registry_test.go`, reading `doc/specs/spec-calibration.md` and `doc/specs/README.md` as text |
| the dry run | the root's external test package, `calibration_dry_test.go`, skipped with `-short` |
| the overlay purity test | the same package, reading `testdata/chains/calibrated/` |
| the library's records | `testdata/calibration/<pack>/<stage>-<n>.txt` |
| the models | `doc/specs/models/clb/` |
