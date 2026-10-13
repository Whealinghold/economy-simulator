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
| 7 | **Seed roles**: **search blocks** (101 and up, 100 seeds each, one choice per block, the only seeds a choice is made on), **confirm seeds** (STY's setting seeds 1 to 50, on which a chosen point is confirmed), the **gate** (1 to 10, inside the confirm range, never an input to a choice) and the **hold-out** (51 to 60: a SHOULD of REF 8.2, the out-of-sample check of a stability record; no band and no choice is made from it) | 7.1 |
| 8 | Four **steps**, the forms of EXP 12.4.2: screen, explore, choose, confirm. The neighbour rule of STY 7.3.1 closes a default's **stability record** | 7.2 |
| 9 | **Stages C0 to C9** in the order structure, viability and warm start, scale, then the facts of STY by group; a stage **fixes** its knobs, and a later stage may not move one without re-running the confirm of the stages it touches (the re-entry rule) | 7.3 |
| 10 | The product of a calibration is an **overlay document** that holds only K and D keys, layered after the pack, and a plain-text **record** with the hash, the experiment, the results and the bands met | 8, 9 |
| 11 | What transfers between settings is the procedure, the classes, the conditions of the specs and the regime thresholds. Defaults and the priors of STY transfer as **starting points**. A band value, a stability record, a warm start and a calibration overlay **do not** | 9.3 |
| 12 | Sensitivity is **measured** by one-at-a-time elasticities against the signs of the registry; a mismatch is a **finding**. Aliases and gauges are declared; a knob whose damper never acted is **dormant** and is left alone | 10 |
| 13 | CLB adds **no public package** in version 1: the pure helpers live in `internal/calibration`, which imports the standard library only (and `internal/design` for the overlay text); the loop is the consumer's, with `experiment` and `indicator`. Promoting `internal/verdict` and `internal/calibration` is open question 1 | 12, 22 |

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
| The word "stable" | allowed only under STY 7.3.1: the three checks STABLE at the default and its two neighbours on the gate seeds, and, where the hold-out of REF 8.2 was run, on it | a pack may say "no regime was detected in the runs of record R", and nothing stronger (STY 7.3.2) |

### 3.3 What a calibration is against

A calibration is judged by the **acceptance set** of STY 8.1 and by nothing the
code under test has defined for itself. In the order of strength:

1. **X and O facts**: exact identities and ordinal relations. No band; asserted where
   REF lists the check as exact (FR-REF-023, STY 3.2), and otherwise **reported** with
   its observed values until REF's list is extended: a reported O fact that fails is a
   finding that is read, and an asserted one is a gate. A calibration never starts
   while an asserted one fails (stage C0). M03 and M04 are class R until
   `HouseholdIncome` is `OK` (STY-M03, M04; IND 9.1 row 21: `OK` is false in version 1):
   in version 1 every `HouseholdIncome` token is unmeasurable, so no mismatch of it is a
   finding and 7.4.1 neither asserts nor targets those two facts.
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
5. **R facts**: recorded, never asserted, never a target. **Class P** (a fact asserted
   only for the pack that names it, STY 3.2: P08 on `second`) is treated as its B or
   O part for that pack; the letter **P** in a record's `class` is STY's, and the
   registry's class P of 4.3 is a different thing (the `class` of a `fact` line is
   one of `X O B R P`, the registry's class is in the registry only).

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
   row (an identity, a topology, a quantity, a unit) is a change of the pack's documents and so
   of its lineage (9.2), made in stage C1 or C2 and followed by **every** stage again from C1.
8. **A seed-conditional configuration.** No value depends on the seed or on which
   seed failed; a failing seed is reproduced and read, never patched (testing.md
   6.2).
9. **A stale warm start.** A knob the solver reads (flag `in` or `tbl`, 4.2.1) is swept with the
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
| **Stage** | the stage that first searches it (7.3): `C1` to `C9`, or `-` for a row that no stage searches (a P row, a `v1x` row, and the K and D rows of the fourteen later modules, 7.3.12) |
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
| `in` | an **input of the steady-state programme**: a data field that a path of the first table of VIA 11 (the configuration paths the programme reads) marks as moving the table, a `.*` there naming every field of its table: a change makes the `WarmStart` table stale, so the experiment regenerates it per point (`Prepare`, EXP 3.3.3) or says it is a perturbation. A **key** (`ID`, `Region`, `Role`, `Recipe`, `Resource`, `Season`, `Want`, `From`, `To`, and the `Commodity` of a table keyed by it) and a **container** (a list or a nested table that only holds rows) are not data fields the programme reads: they are not checked, and are listed with the fields they key or alone with `-` | VIA 11 (the first table), 3 |
| `chk` | read only by a **check or a report** of VIA: a path of the same table that VIA 11 marks `no (a check)` or `no (a report)`, and a solve option that only shapes the report: the table does not change, a warning or a report line may | VIA 11, 10 |
| `out` | **written by the solver**: a reference price, a warm-start price, a starting wage: regenerated by `SolveSteadyState`, never edited by hand (REF position 4) | VIA 8, REF 2 |
| `d` | the knob **selects how many items** a term of the draw count of WLD 7.3.1 counts, at the same state: a walking line (D2), a candidate of an informed view (D4), an intent considered (D5), an offer or a take admitted (D6), an order admitted (D7, D8), an agent created in phase 7 (D10), a pushed household evaluated (D11), **as the admission rule of that term**: its own cap, share, trigger or budget. A **budget** counts, because WLD 7.3.1 reads the term as "orders the agents' budgets admit" (`ReserveTicks`, `SpendMaxShare` and `SpendHalfTicks` decide whether an order of zero units is placed), and so does the **trigger of an offer** (the fee-share test of MER 5.5.1, "passes if and only if": `MerchantFeeShare`, `FeeShareSpreadBP`, `TimeCostBP`, `ConsigneeCutBP`, `MaxSourceAge`). A knob that changes only **what an item does** (a price, a size or a target, a limit, a timing that selects no item: `LotUnits`, `TakeUnits`, `TakeMarginBP`, `CoverTicks`) is not `d`, however much it moves later states; nor is a cap that acts **after** the draws of its term (`MaxOpen` fails a formation, MER 9.5 and FR-MER-046, after D6); and a crossing of 0 that no grid level reaches is not `d` of a gridded knob (`RumourErrorBP` moves D4 and D11 only between 0 and above 0, and 0 is its off value, never a level). On an **S or P row** `d` says that a change of the pack's own value moves a count (`WalkStepBP` of 0 against above 0, the households of a group) and the row is never gridded. A paired comparison across a `d` knob is not exact beyond the first divergence (WLD 7.6.1); one across any knob is exact only until the first divergence | WLD 7.3, 7.3.1 |
| `tbl` | a **solver option** (VIA 11): it changes the table `SolveSteadyState` makes and no rule of a run of the world; a change regenerates the table as `in` does and is never an input to a rule of the world | VIA 11 |
| `z` | the **base may be 0**: the owner's default is 0 or none, or is another field's value that may be 0 (`the world's`). A `geo` grid is built on the pack's **non-zero** value (5.2); where the pack gives none the knob is at 0 (its off value, or an absent thing such as a side with `Capacity` 0), has **no grid**, is **not searched** and the record says `unsearched at_off` (8.2). Not set on a row whose owner default is `required` (the base is the pack's value, whose lowest legal value is at least 1) | the owner, 5.2 |
| `v1x` | **no reader in version 1**: a field with no mechanism that reads it yet; its effect is nil and it is not calibrated | the owner |
| `off=<v>` | class D only: the value at which the damper is disabled: a number, or `off=min` or `off=max` (the owner's lowest or highest legal value, read from the owner's bound and never copied into the row), `off=flat` (no curve: a list that carries no variation), or `off=-` when no value does (the damper can only be weakened). Rows that are off **only as a pair** (`off=-` on K-BEL04 and K-BEL05, K-BEL06 and K-BEL07, K-MIG03 and K-MIG04) have no value of their own that disables them: only a state of the pair does (the owner's rule says which, BEL 5 and MIG 6.5), and no calibration moves the pair there | the owner's mechanism |

VIA 11 **publishes** the list of the configuration paths the programme reads, and
`in` and `chk` follow it, so that EXP's `stale_warm_start` (EXP 14.4) and a
calibration read one list. The registry test of 11.1 (FR-CLB-009) maps each path of
its first table to a registry (scope, field) by the table-prefix rule of EXP 11.3.1
and the snake form of the field, a `.*` being every field of that scope and of the
scopes nested in it, and the README remark `(also Role, ...)` of `Upkeep` its second
scope (4.7); it fails when a data field of the list lacks its flag, or a data field
carries `in` or `chk` that the list does not give it. VIA says that the consumer
adds its own flags to the list ("none is decided here"): `out`, `tbl` and `d` are
CLB's and are not in VIA's table.

### 4.3 Classes

| Class | Name | Test | What a calibration may do |
|-------|------|------|---------------------------|
| **K** | knob | none of the other rules of the list below | search it, inside its legal range, for the reason that a fact needs it |
| **D** | damper strength | it is the parameter of a damper of the register (`Dnn` of `spec-stability.md` 5, or a limit that one counts) | search it **within the range where the damper acts**; never to its `off` value; the three checks hold at it and at both neighbours (STY 7.3.1) |
| **S** | structural choice | it defines what the setting *is*: an identity, a topology, a list, a physical or economic quantity (a duration, a yield, a quantity of upkeep, a population, a distance, a capacity), a unit (the money, the time, the distance), a policy shape (a flow of money) or a module switch | change it only in stage C1 (identity and topology) or C2 (quantities, against the solver); never as a knob; a change is a new variant of the pack (9.2) |
| **P** | presentation only | it changes no result and is not hashed, or it is a diagnostic threshold that changes no result | nothing |

The class is a statement about the *calibration*, not about importance: an S row is
as consequential as any. The test is applied **in this order**, the first rule that
holds decides, and a field has exactly one class:

1. **No effect on any result and not in the hash**, or a diagnostic threshold that
   changes no result: **P**.
2. **An identity, an ordering, a list, a unit, a module switch or a policy shape**:
   **S**, even where a damper entry names the field (`Role.Closed` of D36, a want's
   `Rank` of D31, `OutsideSide.Window` of D65: they define who, in which order, over
   which season; a calibration does not move them).
3. **A limit, a cap, a floor, a threshold, a rate or a share that a damper entry
   counts** (the parameter of its Home cell): **D**, **even where it is a quantity of
   the setting** (`OutsideSide.Capacity` of D63, `TotalUnits`, `Limit` of D41,
   `Exempt` of D43, `RateBP` of D42).
4. **A quantity of the setting that no damper counts** (a duration, a yield, a
   quantity of upkeep, a population, a distance, the capacity of a physical stock):
   **S**.
5. Otherwise **K**.

Two exceptions are stated. A **K row may carry a `D<nn>` in its Loops cell** where the
damper entry names another parameter and the knob only sits in its loop or is ignored
by it (K-SPN10 in D48). A **K or D row whose Grid is `-`** (a list or a schedule that is
not searched) MUST have an `A<nn>` link naming what is searched instead (FR-CLB-107).
**S rows at C3 and C8** are *choices* of the pack made at their stage (the scale; the
open layer's design), not knobs: a stage C3 or C8 S row is searched as a **choice
among listed variants** of the pack and each is a new variant (9.2), never a point of
a grid. The elements of a list that has a knob row (the knots of a schedule, of an
abundance curve) take **the class of that list**. **Siblings** of one mechanism are classed by
a convention of this registry, not by the owners: the member that a derived condition, an
alias or a gauge row links, and that the version 1 loops are searched through, is K (or D),
and the sibling that spells the same mechanism for another shape or fixes a physical quantity
of the pack is S: `InfoSpeed` (K, A07, I13) against a mode's `Speed` and `MoveSpeed` (S, the
physical speeds of vehicles and travellers); `Resource.RateBP` (K, I12) against `RatePerYear`
and `RegenScaleBP` (S, the rate of the other shape and a regional scale); `WalkRevertBP` (K)
against `WalkStepBP` and `TrendBP` (S, the shape of the outside's line). A reader who
disagrees moves the row: the class is a review decision (4.6). A spec that adds a field adds its
row in the same change (FR-CLB-004; N-016).

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
not forbidden to leave a default where it is. A `lin` knob at its off value
(`TargetSpread`) is searched upward from it; a `geo` knob has no level 0, so a `z` row
whose pack value is 0 has **no grid** and is not searched until the pack names a
non-zero value (4.2.1, 5.2). `ReserveTicks` at 0 (K-AGT05) is the same kind of state:
buffers are bought with no reserve held. `TargetSpread` is the first knob a
calibration is *for* (`N-015`; EXP names "`TargetSpread` experiments"): it is class
D, row K-WEL03.

### 4.4 Expected effects: signs and sources

A **move** token is `<fact>/<Measure>:<sign><src>`:

- `<fact>` is the STY fact ID without the prefix (`P02`, `S01`, `W05`) or, for a **scenario row of
  STY 5.11 that backs no fact of its own** (S23, the luxury band), that row's `S<nn>`: a
  scenario of STB is otherwise named by `S<nn>` only in a Links cell, never here, and the check
  of 11.1 admits an `S<nn>` only where STY 5.11 has a row of that ID with the band key written;
- `<Measure>` is an IND preset and field, the closed list of the registry's tokens
  (`Pinned.Share`, `Turnover.Volume`, `PriceRange`, `Offset.abs`, `Dispersion`,
  `RoleShares.Swing`, `Households`, `Unfed`, `HouseholdIncome`,
  `WelfareDistribution.Mean`, `PriceGap.GapAfterFreight`, `StockCover`, `Sectors.M`,
  `PriceIndex.Fisher`), or the **band key** of a scenario of STY that is no preset field
  (`s01.recover_ticks`, `s07.spoiled_bp`, `s14.recover_ticks`, `s17.spike_factor`,
  `s20.ticks`, `s23.band_bp`, `a1.price_band_bp`, `h02.money_bp`), or **the name of an OBS event**
  where the fact's measure is the event itself (`UpkeepShortfall` for S04: the first tick with any
  unfed household), one of the names of the table of observability-and-logging.md 4.3. The field
  names are **STY's spelling** of an IND preset's noun (IND publishes no field list: owed, 23);
  `.abs` reads the magnitude of a signed measure and is CLB's suffix;
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
the owner's bound is written `off=min` or `off=max` and not as the number, with one exception: the
numerals `0` and `1` are written as they are where the owner's lowest legal value is that numeral
(`off=0`: none, no hysteresis; `off=1`: every tick), and the check of 11.1 reads the owner's bound and
accepts both spellings.

### 4.6 How the rows are kept true

The rows are checked **in both directions** against the parameter-owner table
(4.8, 11.1): every owner row has a registry row or a group that lists it, and every
registry row names an owner row, except the fields the owner table does not hold: the
solve options of VIA 11 and the presentation-only fields. The registry is
also read **against the catalogue of `spec-config.md` 3.7**, field by field (11.1), so that a field the
README table collapses is not missed (the rows `Taste.Role` and `RegionTaste.*` that this check found
missing are in the README now). The class of a row
is not derivable by a test, and a human decides it by the order of 4.3; what the test
checks is that the decision was made.

### 4.7 The owner table's scope names

The parameter-owner table writes a **scope** per row, sometimes with a remark
(`Group; world default 1`, `Commodity (region override: ...)`, `Modules (module;
nil = off)`). The **scope token** of a row is the text up to the first `;` or
` (`, trimmed. A registry key and a group's scope use the token. The tokens in use
are listed in 6.3 with the groups that cover them, so that a new scope in the owner
table is found by the coverage check and not by a reader. A remark of the form
`(also <Scope>, ...)` names a **second scope of the same row** (`Upkeep`: `World (also
Role, replacing the world's)`), which is a pair of its own for the coverage check
(`Role.Upkeep`, S-AGT06); a remark that does not start with a capitalised scope name
(`MoneyFlow (also the ledger reason)`) names none.

### 4.8 The test's reading of the owner table

An owner-table row is `| \`Parameter\` | Scope | Owner |`. The check forms the pair
(scope token, parameter) and finds exactly one registry key `<scope token>.<parameter>`
or exactly one structure group whose scope is the token and whose fields list the
parameter. A parameter listed under two scopes (`Seasons` under Calendar and under
Yield) is two pairs, and two owner rows of one pair are one pair. The owner table
also writes a remark after the parameter (`MinAgents (K)`, `ReferenceBasket (B_r)`):
the parameter is the first backticked token, without the remark; the `(also <Scope>` remark of 4.7
adds the pair of the second scope with the same owner.

## 5. Ranges and their honesty

### 5.1 Four kinds

"Range" is two different things, and a registry that does not say which misleads. The
words of this spec are fixed:

| Kind | Name | What it is | Where it is written | May it be a search range? |
|------|------|------------|---------------------|---------------------------|
| **V** | validation bound | the **legal** range: the values `Validate` accepts. It exists to keep arithmetic from overflowing and work bounded (`spec-config.md` 4) and contains configurations that are absurd (a belief step of 25% of the centre is legal) | the owner's table, `bounds.go` | **No.** It clamps a grid and decides whether a level is legal |
| **E** | evidence range | a range or an anchor read directly in a source of `evidence.md` (status "read directly" or "read in full"; the status "read" alone is not enough), for another place and time. A snippet, a summary or an unverified row is **not** E (STY 3.3, N-018) | 6.1, with the evidence row | **Only as a sanity range**: it bounds a search from outside and calibrates nothing |
| **D** | derived condition | an inequality or a closed form that a spec derives, from other quantities of the same configuration (`EnterTicks < ExitTicks`, `ExitTicks` at most what purchases can reach, a refill rate at least 0.16 a year for a monotone approach). As strong as its derivation (H or M) | 6.4, one interaction row | **Yes**, as a constraint on the others |
| **U** | unknown | nothing supports a range. The search starts at the default and its neighbours and widens by the rule of 5.3 | the row's Search cell | By the widening rule only |

Nearly every knob is **U**. That is a result of reading the specs, not a gap to be
filled by invention: of the 165 K and D rows (108 D, 57 K) of section 6, **155 are U**, 8
have a derived condition (I07, I11, I12, I18, I27) and 2 have an evidence anchor (E5, 6.1; E1
sits on an S row, and E2 to E4 are leads, not anchors).
A **placeholder** default (N-015) is a starting point and not a range; "calibrated
once something has run" means a record (section 8) names a value and the runs it
rests on.

### 5.2 The grid and the neighbours

STY 7.3.1 asks a default to be STABLE "at the two neighbouring points of the
default's knob on the grid CLB gives it". The grid rule is therefore a **rule
constant of this spec** (FR-CLB-010, FR-CLB-011, FR-CLB-012, FR-CLB-013, FR-CLB-014, FR-CLB-108) and not a choice per run:

Let `b` be the **base value**: the value of the knob in the effective configuration
the stage starts from, which is the pack's value where the pack sets one and the
owner's default otherwise; where the owner has **no default** (`required`: `InfoSpeed`,
`TakeUnits`, `MoveSpeed`) it is the pack's value, which the owner's bound keeps at
least 1. It is recorded in the record. Let `[lo, hi]` be the validation bound of the
owner, evaluated with the other fields as configured. All division is half to even
(IND 5.2.2). A grid is built in **four steps, in this order**:

1. **The levels of the kind**, kept in `[lo, hi]`, with `b` always among them:
   - **`geo`**: `b * 2^k` for `k >= 0` and `rdiv(b, 2^-k)` for `k < 0`, at least 1 (a
     `geo` grid has no level below 1: **0 is never a geo level**, because 0 is usually an
     off value, 4.3.1), duplicates removed, ascending. Defined for `b >= 1`: the owner's
     default is at least 1, or the owner's default is `required` and its lowest legal
     value is at least 1, or the row has the flag `z` and the pack's value is non-zero
     (a `geo` row whose base is 0 has **no grid** and is not searched, below). A knob
     whose base can be 0 and has no `z` is `lin`. The factor is 2: a **placeholder**
     (P). Ratios in basis points and ticks are scale-free enough that a halving and a
     doubling are the unit step of a first look; they are not a claim that the
     response is geometric.
   - **`lin<s>`**: `b + k * s` for integer `k`. The step `s` is in the unit and is given
     in the row. `b` may be 0 (`TargetSpread`).
   - **`set:<a/b/c>`**: the listed values, in the order given (a name, a flag, a short
     enumeration).
2. **The cut.** In a D row, every level equal to the row's **off value** (`off=<v>`, or
   the owner's `lo` or `hi` for `off=min` or `off=max`), and every level beyond it on that
   side (the range where a damper acts ends at its off value, 4.3), is removed, **except
   `b`**: the base is never cut, so a default that is itself at its off value (`TargetSpread`
   0) stays a level and is moved away from, never back to (4.3.1). The cut is applied to the
   grid of step 1 and the check of 11.1 reads the owner's bound, not the row (4.5).
3. **Invalid levels.** A level is **invalid** when `Validate` rejects the configuration
   with it (a rule that involves another field: 6.4), found by `Check` (EXP 14.3)
   without a run. An invalid level is **skipped**: the neighbour on that side is the
   next valid level that is kept.
4. **The neighbours.** The neighbours of `b` are the nearest kept valid level below it
   and the nearest above. A side with none is an **edge**, recorded as `edge:bound` (the
   next level was outside `[lo, hi]`, or below the floor of a `geo` grid, which has no level under 1:
   `max(lo, 1)` is its lower bound, FR-CLB-010), `edge:off` (the next level was the off value and
   was cut) or `edge:rule` (the next level was invalid) with the rule that cut it. A
   default with an edge side is held to the **one** neighbour it has, and its stability
   record says so; that is not a pass for the missing side and not a fail for it either,
   and the default is **not called stable** on it (7.4.3).

A **`z` row whose base is 0** has no grid: the knob is at its off value (or the thing
it parameterises is absent, as a side with `Capacity` 0), a `geo` grid cannot start from
0 and there is no second level to search toward. It is **not searched**, is not a factor
of a screen, and the record says `unsearched at_off` (8.2); a pack that wants it
calibrated names a non-zero value first (a change of the pack, 9.2).

A **grid is not a range**: it says where the first look is, not how far the search may
go (5.3).

### 5.3 The widening rule

The search of a knob starts with its two neighbours (the screen, 7.2). The **explore**
step covers the interval from the second level below `b` to the second level above
(`k` from -2 to +2 of a `geo` grid: a factor of four each side; two steps of a `lin`
grid), over the **kept valid levels** of 5.2 (a cut off level or an invalid one is not a
level, so the interval is shorter, or has an edge, where one is in the way). If the
feasible points of the choice **touch an end of the interval** that is not an edge, the
interval is widened by two kept levels on that side and the choice is **made again on a
new search block** (7.1). The search never leaves `[lo, hi]` and never reaches an off
value. Both numbers (two levels, two more) are
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
(STY 3.3). **Two are anchors (E1 and E5, read directly) and three are leads (E2, E3 and
E4)**, which are not anchors in the sense of 5.1: E2's row is read from extracted text
with inferred bounds, E3's cited row holds no such number, and E4's rows have the status
"read" in `evidence.md`, from text extracted from a PDF whose tables and appendices were
not read in full. The **Status** cell says which; no row has a lead as its Search, and
the knob of a lead is `U` (an S field with a lead has Search `-`). A knob with no anchor
here is `U`.

| Anchor | Status | Knob | What the source gives | Evidence row |
|--------|--------|------|-----------------------|--------------|
| E1 | anchor | `Role.ExpectedWorkTicks` | annual income as day wage times 250 days is the usual reading (Allen; Humphries and Weisdorf), and medieval labourers worked **under** 150 days a year (Humphries and Weisdorf): two figures, an upper bound below 150 and a usual reading of 250, in **days**, which are ticks only where a tick is a day (`spec-calendar.md` 3.2; the Earth default) | `evidence.md` rows "Annual income is day wage times 250 days" and "Medieval labourers worked under 150 days" |
| E2 | lead | `World.BeliefStep` | an adjusted price changes by an amount uniform in `[0, 2%]` (Lengnick: `vartheta` 0.02, explicit in the extracted text of equation 10 and Table 1; the other bounds of the row are inferred): a **lead**, not an anchor (the row is read from extracted text, not "read directly"), an upper bound of the per-adjustment move of that model (the mean is half), not a range | `evidence.md` row "Firm price rule" (`vartheta` = 0.02 explicit; the other bounds of the row are inferred and unverified) |
| E3 | lead | `World.PriceWindow` | bazaarBot's `lookBack` default of 15 rounds and the paper's profitability window of 8 to 15 rounds (`spec-beliefs.md` 11, and `bazaarbot-reference.md`): a **lead** for the order of the window and no anchor (the row cited lists window statistics and not these numbers), not a range that transfers (the paper's rounds are not ticks) | `evidence.md` row "bazaarBot's algorithm as published" |
| E4 | lead | `Role.TrainingTicks` | apprenticeship usually lasted seven years under an indenture (Justman and van der Beek): a point, to be read in the pack's years (`TicksPerYear`) | `evidence.md` row "Apprenticeship: usually seven years" |
| E5 | anchor | `Population.BirthSchedule`, `Population.DeathSchedule` | the **shape**: births flat or rising with income, deaths falling with it, so a stable population sits where they cross, and a higher death schedule raises equilibrium living standards (Clark, chapter 2, read directly). **No rate is given**: surviving children per woman 1.93 to 2.10 is recorded and sets no default | `evidence.md` rows of Clark (2007) chapter 2 |

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
| K-WEL01 | World.WelfareWindow | D | WEL 2.3.1 | ticks | U | geo | C4 | chk off=- | D03/RoleShares.Swing:+M | L14 L19 L36 D29 | I04 A02 |
| K-WEL02 | World.WindowCycles | D | WEL 2.3.3 | count | U | lin1 | C4 | chk off=- | D03/RoleShares.Swing:+H | L19 L36 D29 | I04 A02 |
| K-WEL03 | Role.TargetSpread | D | WEL 5.1 | bp | U | lin500 | C4 | chk off=0 | P03/PriceRange:-S | L15 L34 D08 | I06 A01 |
| K-WEL04 | Role.TargetRatio | K | WEL 5 | bp | U | lin1000 | C4 | chk | M03/HouseholdIncome:0S; M05/StockCover:+S | L14 L46 | I06 |
| K-WEL05 | Role.DefaultRatio | K | WEL 2.2 | bp | U | lin1000 | C4 | - | H01/a1.price_band_bp:? | - | - |
| K-WEL06 | Role.ExpectedWorkTicks | S | WEL 4.2 | ticks | E1 | - | C2 | in | D01/Unfed:? | - | I05 |
| K-WEL07 | Role.MinWorkShare | D | WEL 5.3 | bp | U | lin1000 | C4 | off=0 | P04/Offset.abs:-M | L14 D13=HeldByFloor | I05 |
| K-WEL08 | World.WithTargetSpread | D | WEL 5.2 | bp | U | lin500 | C4 | off=0 | P03/PriceRange:-S | L15 L34 D08 | I06 A01 |

Notes. **K-SCL01**: the scale is a structural choice of resolution (STY 4.4: facts are stated at weight 1); STY-P09 is its consistency check. **K-WEL03 and K-WEL08**
are one knob in two forms (A01): the role field sets one role's spread, the option
overrides every role's and is in the hash (`spec-world.md` 4.1). Both are D with the
default **at the off value**: a calibration moves it away from 0 and never back
(4.3.1). **K-WEL04**: in a closed economy with upkeep as the only demand the
**aggregate** ratio of `HouseholdIncome` is 10 000 whatever the target says, up to the change
of stock (`spec-viability.md` 4.7, N-054; STY-M03); `WelfareDistribution.Mean` is recorded
beside it and is not bound by the identity (IND 7.7.2). `TargetRatio` is therefore **not** a
lever on `M03`; the `0S` says so. `HouseholdIncome` is not `OK` in version 1 (3.3), so
nothing measures it yet, and a measured move once it is `OK` is a finding. **K-WEL01**:
the effective window is `max(WelfareWindow, WindowCycles * longest Duration)` (WEL 2.3.3), so
at its lowest legal value the derived clause binds in every role and D29 is fully in
force: no value of `WelfareWindow` disables the damper (`off=-`, as K-WEL02). **K-WEL01, K-WEL02 and K-WEL03** carry
`chk`: VIA 11 lists `world.welfare_window`, `world.window_cycles` and `role.<id>.target_spread` as `no (a report)`
(`target_above_surplus`, `subsistence_exceeds_expected_work`, the money rule's longest job), so a change moves a report line
and not the table; **K-WEL08**, the option, is not a path and VIA lists none: it is the same quantity (A01) and
carries no flag. **K-WEL06** is read by the nominal programme (VIA 3.2, phi) and carries E1.

#### 6.2.2 Agents (AGT)

| ID | Key | Class | Owner | Unit | Search | Grid | Stage | Flags | Moves | Loops | Links |
|----|-----|-------|-------|------|--------|------|-------|-------|-------|-------|-------|
| K-AGT01 | World.CoverTicks | K | AGT 12 | ticks | U | geo | C4 | - | M05/StockCover:+H | L49 | I07 I13 |
| K-AGT02 | World.InputJobs | K | AGT 12 | count | U | lin1 | C4 | - | M05/StockCover:+H | L49 | - |
| K-AGT03 | World.OutputCoverTicks | K | AGT 12 | ticks | U | geo | C4 | - | M05/StockCover:+S | L02 L04 | I10 |
| K-AGT04 | World.FavourFloor | K | AGT 12 | bp | U | lin1000 | C6 | - | M04/HouseholdIncome:? | L49 | - |
| K-AGT05 | World.ReserveTicks | D | AGT 12 | ticks | U | lin1 | C4 | d off=0 | M06/s20.ticks:+H | L11 L45 D27 | I09 I18 |
| K-AGT06 | World.SpendMaxShare | D | AGT 12 | bp | U | lin1000 | C6 | d off=- | M04/HouseholdIncome:+H | L45 L57 D27 | I18 A04 |
| K-AGT07 | World.SpendHalfTicks | D | AGT 12 | ticks | U | geo | C6 | d off=- | M04/HouseholdIncome:+H | L45 L57 D27 | I18 A04 |
| K-AGT08 | World.RestBand | D | AGT 12 | bp | U | lin250 | C4 | off=0 | P03/PriceRange:? | L14 L15 D09 | - |
| K-AGT09 | World.ProduceHaltRatio | D | AGT 12 | bp | U | geo | C4 | off=max | M05/StockCover:+S; P02/Pinned.Share:? | L05 L06 L44 D24=ProductionHalted | I10 |
| K-AGT10 | Role.StartMoney | S | AGT 3.3 | minor | - | - | C2 | - | M06/s20.ticks:-H | L11 L45 | I09 A03 A08 |

Notes. **K-AGT09**: the halt's threshold decides which bound a pinned price reaches
(STB 10, L44) and its counter is `ProductionHalted` (OBS 4.4.6, N-067). The
sign on `StockCover` is STY 10.2's ("`ProduceHaltRatio` down lowers the unsold
stock"). **K-AGT10** is read by a cold start only: a warm-start table that names the
group overrides it (`spec-viability.md` 3.3 item 3). **K-AGT05** `ReserveTicks` is the
reserve of D27 (0 holds no reserve: its `off=0`), and `SpendMaxShare` and `SpendHalfTicks` are the cap
above it (I18, A04): the three are one budget stack, and each carries `d`: they are the budget
that decides whether an order of zero units is placed (WLD 7.3.1, "orders the agents' budgets
admit"; 4.2.1). **K-AGT07**: the cap is `SpendMaxShare * (e * H / (e + H)) / 10 000` with
`H = SpendHalfTicks * UpkeepCost` and `e` the money above the reserve (AGT 7.4.4), and
`e * H / (e + H)` rises with `H`: a larger `SpendHalfTicks` raises the cap, as a larger
`SpendMaxShare` does, so both are `+H` (A04).

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
| K-BEL13 | World.CostCeiling | D | BEL 11 | bp | U | geo | C4 | chk off=max | P04/Offset.abs:-S | L10 L52 D04=BeliefsUpdated | I02 A13 |
| K-BEL14 | World.RefBandLow | D | BEL 11 | bp | U | geo | C4 | chk off=min | P02/Pinned.Share:? | L06 L08 D04=BeliefsUpdated | I03 A13 |
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
| K-SPN01 | World.ReviewTicks | D | SPN 11 | ticks | U | geo | C4 | off=1 | D03/RoleShares.Swing:? | L36 D33 | A12 |
| K-SPN02 | World.EntryShare | D | SPN 11 | bp | U | geo | C4 | d off=max | D03/RoleShares.Swing:+S | L36 L54 L61 D14=EntryLimited | I04 A02 A12 |
| K-SPN03 | World.RoleHysteresis | D | SPN 11 | bp | U | lin500 | C4 | off=0 | D03/RoleShares.Swing:? | L36 L54 D09 | A12 |
| K-SPN04 | World.RoleCooldown | D | SPN 11 | ticks | U | geo | C4 | off=0 | D03/RoleShares.Swing:-M | L36 D34=EntryLimited | I04 A12 |
| K-SPN05 | World.EntryRatio | D | SPN 11 | bp | U | lin1000 | C4 | d off=0 | S02/Households:? | L61 D36=EntryLimited | - |
| K-SPN06 | World.ChangePrivation | D | SPN 11 | ticks | U | lin1 | C5 | off=- | S02/Unfed:+H | L35 L50 L62 D35=EntryLimited | I08 |
| K-SPN07 | World.RemovePrivation | D | SPN 11 | ticks | U | geo | C5 | off=- | S02/Unfed:+S; S02/Households:? | L35 L50 L62 D35=EntryLimited | I08 |
| K-SPN08 | World.DeathStep | D | SPN 11 | bp | U | geo | C5 | off=max | S02/Households:? | L35 D35=EntryLimited | - |
| K-SPN09 | World.ExitShare | D | SPN 11 | bp | U | geo | C5 | off=max | S02/s14.recover_ticks:? | L35 L83 D35=EntryLimited | - |
| K-SPN10 | World.Renewal | K | SPN 11 | bp | U | lin2500 | C5 | d | S02/Households:+H | L61 L79 D48 | - |
| K-SPN11 | World.EntryPoolShare | D | SPN 11 | bp | U | lin1000 | C5 | d off=max | S02/Unfed:+H | L39 L63 D37=EntryLimited | I21 |
| K-EST01 | Region.EstateMarkdown | K | EST 3 | bp | U | lin250 | C5 | - | S02/Households:? | L38 | - |
| K-EST02 | Region.PoolRelease | D | EST 2.6 | bp | U | geo | C5 | off=0 | S02/Unfed:-H | L39 L63 D11 D12 | I21 A14 |
| K-EST03 | Region.ReliefBelow | D | EST 2.6 | bp | U | lin1000 | C5 | off=0 | S02/Unfed:-H | L39 L40 D11 | A14 |
| K-SKL01 | Role.TrainingTicks | S | SKL 4.1 | ticks | - | - | C1 | - | D03/RoleShares.Swing:? | L36 L41 | I20 I23 |
| K-SKL02 | Role.MaxTrainees | D | SKL 4.4 | count | U | lin1 | C6 | off=max | D03/RoleShares.Swing:? | L36 L41 D20 | I23 |
| K-SKL03 | Role.TeachingLoad | K | SKL 4.2 | bp | U | lin500 | C6 | - | D03/RoleShares.Swing:? | L37 | I23 |
| K-SKL04 | Region.CapitalReturn | K | SKL 2.5 | bp | U | lin500 | C6 | - | D03/RoleShares.Swing:? | L36 | - |
| K-PRO01 | Recipe.TraineeYield | S | PRO 8.2 | bp | - | - | C2 | - | D03/RoleShares.Swing:? | L37 | I23 |

Notes. **K-SPN02**: `EntryShare` at 0 turns role change **off** (a structural choice);
at 10 000 it removes the cap (its off value). The condition is on the product with the
effective window (A02, `spec-stability.md` 10: "`EntryShare * T` below about 2 000");
the sign on the swing is STY 10.2's. **K-SPN03**: SPN 16.1 measures hysteresis alone at the
same amplitude as with a 10% share, so its sign is `?` until a screen reads it. **K-SPN10** is ignored while the population
module is on (`spec-population.md` 4.6; D48). **K-SKL01** is class S: the length of a
traineeship defines a barrier; E4 is a lead for it (6.1), not an anchor.

#### 6.2.5 Demand, money, population and resources (DEM, MON, POP, RES)

| ID | Key | Class | Owner | Unit | Search | Grid | Stage | Flags | Moves | Loops | Links |
|----|-----|-------|-------|------|--------|------|-------|-------|-------|-------|-------|
| K-DEM01 | Want.Elasticity | D | DEM 9 | bp | I18 | lin5000 | C6 | in off=0 | S23/s23.band_bp:-M | L14 D30=WantShortfall | I18 |
| K-DEM02 | Want.Ceiling | D | DEM 9 | bp | U | lin5000 | C6 | in off=max | S23/s23.band_bp:? | L14 D30=WantShortfall | - |
| K-DEM03 | Want.Substitution | K | DEM 9 | count | U | lin1 | C6 | in | M04/HouseholdIncome:? | L58 | - |
| K-MON01 | MoneyFlow.Limit | D | MON 4.4.2 | minor | U | geo | C8 | z off=0 | S05/Sectors.M:? | L30 L69 L72 D41=MoneyFlowEnded | - |
| K-MON02 | MoneyFlow.Exempt | D | MON 4.2 | minor | U | geo | C8 | z off=0 | S05/Sectors.M:? | L71 D43 | - |
| K-MON03 | MoneyFlow.RateBP | D | MON 4.2 | bp | U | geo | C8 | z off=0 | S05/Sectors.M:-H | L70 D42 | - |
| K-POP01 | Population.BirthSchedule | D | POP 8 | list | E5 | - | C9 | off=- | W02/Households:? | L75 D46 | I11 A15 |
| K-POP02 | Population.DeathSchedule | D | POP 8 | list | E5 | - | C9 | off=- | W02/Households:-E | L75 L77 D46 | I11 A15 |
| K-POP03 | Population.MinHouseholds | D | POP 8 | count | U | lin1 | C9 | off=0 | W06/Households:? | L82 D49=PopulationChanged | I11 |
| K-POP04 | Population.BirthStake | D | POP 8 | bp | U | lin2500 | C9 | d off=0 | W06/Households:-S | L76 L77 D51=PopulationChanged | I09 A03 |
| K-POP05 | RegionPopulation.BirthScale | K | POP 8 | bp | I11 | geo | C9 | d | W02/WelfareDistribution.Mean:-H; W02/Households:+H | L75 | I11 A15 |
| K-POP06 | RegionPopulation.DeathScale | K | POP 8 | bp | I11 | geo | C9 | - | W02/WelfareDistribution.Mean:+E; W02/Households:-E | L75 L77 | I11 A15 |
| K-POP07 | RegionPopulation.HouseholdCap | D | POP 8 | count | U | geo | C9 | d z off=0 | W06/Households:+H | L80 D50=PopulationChanged | I11 |
| K-RES01 | Resource.RateBP | K | RES 12 | bp | I12 | geo | C9 | in z | W02/Households:+H | L87 L88 | I12 |
| K-RES02 | Resource.Refuge | D | RES 12 | units | U | geo | C9 | in z off=0 | W03/Households:? | L88 D56 | I12 |
| K-RES03 | Resource.PlanShareBP | D | RES 12 | bp | I12 | lin1000 | C9 | in off=- | W02/Households:? | L87 L88 D58 | I12 |
| K-RES04 | Resource.Abundance | D | RES 4.7 | list | U | - | C9 | in off=flat | W02/WelfareDistribution.Mean:? | L87 L91 D54 | I12 A16 |
| K-RES05 | Resources.HorizonYears | D | RES 12 | years | U | geo | C2 | in off=- | - | L89 D58 | I12 |

Notes. **K-DEM01 and K-DEM02**: a luxury is stable only with `Elasticity` above
10 000 and about 10% budget headroom (`spec-demand.md` 6.3), a **derived condition**
on `Elasticity` (I18, Search of K-DEM01); the headroom is a property of the money
held above the reserve (`SpendMaxShare`, `SpendHalfTicks`, `ReserveTicks`), not of
`Ceiling`, whose Search is `U`. `Ceiling` at its highest legal value is the
non-binding end (`off=max`); D30 is switched off by `Elasticity` 0 (K-DEM01), not by `Ceiling`.
K-DEM02's sign is `?`: the only model of DEM 6.3 varies `Elasticity` and fixes a ceiling of
15 000, so a sign on `Ceiling` is a formula sentence and not a model's, and the row has no
`I18` link (I18 is the condition of K-DEM01 and of the budget knobs). **K-DEM01 to K-DEM03,
K-RES02 and K-EXT02 to K-EXT05, K-EXT08 and K-EXT09** carry `in` because VIA 11 lists their
tables with a `.*` (`modules.demand.want.<id>.*`, `modules.resources.resource.<id>.*`,
`modules.external.node.*`): every field of such a table is read by the programme when its
module is on (4.2.1). **K-POP01 and K-POP02**:
the *shape* is E5 and no rate is given; a calibration moves a schedule only through
its two region multipliers K-POP05 and K-POP06 (A15), as a composite of its knots is
not a grid. The sign on `DeathScale` is Clark's ("what raises the death rate raises
living standards", read directly), on `BirthScale` the same logic, hand-derived
(`H`). **K-POP04**: STY 10.2. **K-RES04**: a flat curve is the default and so the
damper's off value (4.3.1).

#### 6.2.6 Information, transport, merchants and the open layer (INF, TRN, MER, EXT)

| ID | Key | Class | Owner | Unit | Search | Grid | Stage | Flags | Moves | Loops | Links |
|----|-----|-------|-------|------|--------|------|-------|-------|-------|-------|-------|
| K-INF01 | Information.InfoSpeed | K | INF 8 | units | U | geo | C7 | - | P06/PriceGap.GapAfterFreight:-M | L84 L93 | I13 A07 |
| K-INF02 | Information.RumourErrorBP | D | INF 8 | bp | U | geo | C7 | off=0 | P06/PriceGap.GapAfterFreight:? | L93 L108 D60 | - |
| K-INF03 | Information.RumourMaxAge | D | INF 8 | ticks | U | geo | C7 | off=max | P06/PriceGap.GapAfterFreight:? | L23 D61=RumoursServed | I13 |
| K-MER01 | Merchants.MaxOpen | D | MER 13 | count | U | geo | C7 | off=max | P06/PriceGap.GapAfterFreight:? | L113 L114 D84=ContractRound | - |
| K-MER02 | Merchant.LotUnits | D | MER 3.2 | count | U | lin1 | C7 | off=max | P03/PriceRange:? | L116 D80=MerchantsPlanned | - |
| K-MER03 | Merchant.FeeShareSpreadBP | D | MER 3.2 | bp | U | lin500 | C7 | d off=0 | P06/PriceGap.GapAfterFreight:-S | L84 L93 D81 | I14 A10 |
| K-MER04 | Merchant.MaxSourceAge | D | MER 3.2 | ticks | U | lin1 | C7 | d off=max | P06/PriceGap.GapAfterFreight:+S | L94 D82=MerchantsPlanned | - |
| K-MER05 | Merchant.TimeCostBP | K | MER 3.2 | bp | U | lin2500 | C7 | d | P06/PriceGap.GapAfterFreight:? | - | - |
| K-MER06 | Merchant.ConsigneeCutBP | K | MER 3.2 | bp | U | lin250 | C7 | d | P06/PriceGap.GapAfterFreight:? | - | - |
| K-MER07 | Merchant.TakeMarginBP | K | MER 3.2 | bp | U | lin500 | C7 | - | P06/PriceGap.GapAfterFreight:? | L113 | - |
| K-MER08 | Role.MerchantFeeShare | D | CAR 2.2 | bp | U | lin1000 | C7 | d off=max | P06/PriceGap.GapAfterFreight:? | L16 L84 L116 D86=MerchantsPlanned | I14 A10 |
| K-MER09 | Merchant.TakeUnits | K | MER 3.2 | units | U | geo | C7 | - | P06/PriceGap.GapAfterFreight:? | L113 | - |
| K-EXT01 | OutsideGate.ScaleBP | K | EXT 3.2.4 | bp | U | lin1000 | C8 | in | T03/PriceRange:? | - | - |
| K-EXT02 | OutsideLine.FloorBP | D | EXT 6.3 | bp | U | geo | C8 | in off=min | T03/PriceRange:? | L100 D66=OutsideTraded | I17 |
| K-EXT03 | OutsideLine.CeilingBP | D | EXT 6.3 | bp | U | geo | C8 | in off=max | T03/PriceRange:? | L100 D66=OutsideTraded | I17 |
| K-EXT04 | OutsideLine.WalkRevertBP | K | EXT 6.4 | bp | U | lin500 | C8 | in | T03/PriceRange:? | L100 | - |
| K-EXT05 | OutsideLine.WalkBoundBP | D | EXT 6.4 | bp | U | geo | C8 | in z off=max | T03/PriceRange:? | L100 D66=OutsideTraded | I17 |
| K-EXT06 | OutsideSide.MarginBP | D | EXT 6.6 | bp | U | geo | C8 | in z off=- | T03/PriceRange:? | L96 L99 D62 | I17 |
| K-EXT07 | OutsideSide.Capacity | D | EXT 7.1 | units | U | geo | C8 | in z off=max | T03/PriceRange:? | L97 D63=OutsideTraded | - |
| K-EXT08 | OutsideSide.TotalUnits | D | EXT 7.3 | units | U | geo | C8 | in z off=0 | S04/UpkeepShortfall:? | L101 D63=OutsideEnded | - |
| K-EXT09 | OutsideSide.TotalValue | D | EXT 7.3 | minor | U | geo | C8 | in z off=0 | S04/UpkeepShortfall:? | L98 D63=OutsideEnded | - |

Notes. **K-INF02**: `RumourErrorBP` moves D4 and D11 only between 0 and above 0, and 0 is its off value and
never a grid level, so it is not `d` at any level of its grid. **Its sign is `?`: STY 10.2 and INF 11.2 disagree.**
STY 10.2 says that `RumourErrorBP` up "widens the converged gap of P06" and cites INF 11.2; INF 11.2 does not say so.
Its model of an invented shipper rule (placeholder numbers) gives 1.80 `f` for identical shippers and, for an
independent error of 0.05 to 0.8 `f`, 1.99, 1.94, 1.75, 1.32 and 0.60 `f`: "an independent error disperses the decisions of
shippers", and only a **shared** error widens the gap, to 2.22 and 2.43 `f` (INF-D2, "a weak damper"). The sign is STY's claim
and not INF's, the two are in conflict (X8), and a screen decides; the spread of the shippers' own thresholds
(K-MER03, A10) settles the gap where the error alone does not (INF 11.2). **K-MER03 to K-MER06 and K-MER08** carry
`d`: they are the terms of the fee-share test of MER 5.5.1 ("passes if and only if
`C * 10 000 <= phi * G`") that decides whether an offer exists (D6), and `MaxSourceAge` the bound on a rumour's
age that sets a candidate aside (`Stale`, MER 5.2). `LotUnits`, `TakeUnits` and `TakeMarginBP` set a size, a want and
a price and are not `d`, and `MaxOpen` (K-MER01) fails a formation after the draws of D6 (MER 9.5), so it is not either.
**K-INF01** is the
speed of news in the pack's distance unit (A07). **K-MER08** is `MerchantFeeShare`,
owned by CAR 2.2 and read by MER; with the default spread MER 3.2 rejects both
extremes (`ErrOutOfBounds`), so they are reachable only with a spread of 0. **K-EXT06**: the corridor widens with the margins (EXT 6.6, 19.2); a pair of
margins that sum below `MinQuoteSpreadBP` is `ErrCrossedQuotes`, so no side is moved to
0 alone. **K-EXT07**: `Capacity` 0 is an *absent* side (EXT 3.3.3), not the damper's off value: the row is `z`, a
side the pack makes present has the pack's value as its base, and an absent side has no grid. The model of EXT 19.2 shows a corridor confining a cobweb *provided the capacity covers the swing*, a statement
about the price range with the outside. **The outside knobs name T03, not P03**: P03's packs have no outside gate
(`one`, the haul overlay, the merchant lanes), and the range "with and without the outside" is STY-T03's
measure (S51, class R); T03 is a report fact, so every sign of K-EXT01 to K-EXT07 is `?` until a screen measures it.
EXT 19.2's model gives the corridor's width a `+` with the margins and a `-` with the capacity, as text and not
as a token. **K-EXT01** is the gate's scale of the quote, the same fact.

#### 6.2.7 Own work and movement (SUB, MIG)

| ID | Key | Class | Owner | Unit | Search | Grid | Stage | Flags | Moves | Loops | Links |
|----|-----|-------|-------|------|--------|------|-------|-------|-------|-------|-------|
| K-SUB01 | Subsistence.EnterTicks | D | SUB 12 | ticks | I07 | lin1 | C5 | off=- | S02/Unfed:? | L102 L103 L104 L105 D67=OwnUseChanged | I07 |
| K-SUB02 | Subsistence.ExitTicks | D | SUB 12 | ticks | I07 | lin1 | C5 | off=- | S02/Unfed:? | L102 L106 D67=OwnUseChanged | I07 |
| K-SUB03 | Subsistence.RetryJobs | D | SUB 12 | count | U | geo | C5 | off=0 | S02/Unfed:? | L102 L103 L106 D69=OwnUseChanged | I15 |
| K-MIG01 | Migration.MoveReviewTicks | D | MIG 11 | ticks | U | geo | C7 | d off=1 | G01/Households:? | L107 D72 | I16 |
| K-MIG02 | Migration.MoveBelow | K | MIG 11 | bp | U | lin1000 | C7 | d | G01/Households:? | - | - |
| K-MIG03 | Migration.MoveGain | D | MIG 11 | bp | U | geo | C7 | off=- | G01/Households:? | L107 L108 D74=MigrationReviewed | A05 I16 |
| K-MIG04 | Migration.MoveWidth | D | MIG 11 | bp | U | geo | C7 | off=- | G01/Households:? | L107 L108 D74=MigrationReviewed | A05 |
| K-MIG05 | Migration.MoveCooldown | D | MIG 11 | ticks | U | geo | C7 | d off=- | G01/Households:? | L107 L111 D73=MigrationReviewed | I04 |
| K-MIG06 | Migration.MoveHorizon | D | MIG 11 | ticks | U | geo | C7 | off=max | G01/Households:? | L107 D77=MigrationReviewed | I13 |
| K-MIG07 | Migration.MoveOutShare | D | MIG 11 | bp | U | geo | C7 | off=max | G01/Households:? | L107 L111 D75=MigrationReviewed | I16 |
| K-MIG08 | Migration.MoveInShare | D | MIG 11 | bp | U | geo | C7 | off=max | G01/Households:? | L107 L108 L109 D76=MigrationReviewed | I16 |
| K-MIG09 | RegionMove.OutShare | D | MIG 11.2 | bp | U | geo | C7 | z off=max | G01/Households:? | L107 L111 D75=MigrationReviewed | I16 |
| K-MIG10 | RegionMove.InShare | D | MIG 11.2 | bp | U | geo | C7 | z off=max | G01/Households:? | L108 L109 D76=MigrationReviewed | I16 |

Notes. The module has **no STY fact of its own**: its scenarios S61 to S66 are
exact checks and bands of `spec-stylised-facts.md` 5.11, so its knobs' `Moves` name
the umbrella fact **G01** (a healthy start stays healthy), which a migration cycle
breaks by emptying a region (L111). **K-SUB01 and K-SUB02** are a **pair** (I07): the
mode ends only when stock covers `ExitTicks` above `EnterTicks`, and `ExitTicks` is
bounded above by what purchases can reach, `CoverTicks`. **The ladder, pool, estate and
own-use rows (K-SPN06, K-SPN07, K-SPN11, K-EST02, K-EST03, K-SUB01 to K-SUB03) name
S02 and not D01**: on D01 a healthy run is fed, `Unfed` is about 0 and these rungs, the
pool and the own-use mode are dormant, so no sign can be read there; the facts that STY
sweeps them for are S02 and S06 (STY 10.1), and S02's measure holds the series, "the longest
run of ticks with any unfed (`Unfed`)" (STY 5.3).

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
| K-VIA03 | Solve option.ResidualWarnBP | P | VIA 11 | bp | - | - | C2 | chk | - | - | - |
| K-VIA04 | Solve option.MaxPivots | P | VIA 11 | count | - | - | - | - | - | - | - |
| K-VIA05 | Solve option.Lanes | S | VIA 11 | list | - | - | C1 | in | - | - | - |

Notes. `StockCoverTicks` and `MoneyBufferTicks` do not change the solved flows; they
shape **the table the solver writes** (VIA 8.3, 8.4), so a change regenerates the
table and the table is data that is hashed (`spec-viability.md` 7.4). They carry the
flag `tbl` (4.2.1): they are inputs of the solver, so not `out`, and the effect is on the table, not on a rule. **K-VIA03** changes only what
the report lists (`chk`: no table and no rule moves, only the lines of the report), and **K-VIA04** is a work cap that changes no result (class P by rule (1) of 4.3). **K-VIA05** (`Lanes`) is read by the programme (VIA 9.4) and is structure.

#### 6.2.9 Storage, employment, settlements and tenure (STO, EMP, SET, LND)

The knob rows of the fourteen modules of wave 5 (`spec-config.md` 3.1), in the order of the `Modules` row, and their structure groups in 6.3. A K or D row of these modules has the Stage `-`: no stage of 7.3 searches it (7.3.12). The class of a row is the order of 4.3, applied to the owner's own classification where that gives one; where it does not, the Notes say so.

| ID | Key | Class | Owner | Unit | Search | Grid | Stage | Flags | Moves | Loops | Links |
|----|-----|-------|-------|------|--------|------|-------|-------|-------|-------|-------|
| K-EMP01 | Employer.MaxHires | D | EMP 13 | count | U | geo | - | off=max | - | L21 L22 D105=Unfilled | - |
| K-EMP02 | Employer.WorkerShareBP | K | EMP 13 | bp | U | lin1000 | - | - | - | L148 D104 D108 | - |
| K-EMP03 | Employer.HireYieldBP | K | EMP 13 | bp | U | lin1000 | - | - | - | - | - |
| K-EMP04 | Employment.MaxOpen | D | EMP 13 | count | U | geo | - | off=max | - | L149 D110=TooMany | - |
| K-SET01 | Settlements.BandBP | D | SET 3.4.2 | bp | U | lin500 | - | off=0 | - | L151 L152 D111 | - |
| K-SET02 | Settlements.Dwell | D | SET 3.4.3 | ticks | U | geo | - | off=0 | - | L151 L152 D112=SettlementHeld | - |
| K-LND01 | Tenure.RentStepBP | D | LND 5.5 | bp | U | geo | - | z off=0 | - | L155 D116 | - |
| K-LND02 | Tenure.RentFloorBP | D | LND 5.5 | bp | U | geo | - | z off=0 | - | L155 D116 | - |
| K-LND03 | Tenure.RentCeilingBP | D | LND 5.5 | bp | U | geo | - | off=max | - | L155 D116 | - |
| K-LND04 | Tenure.MaxHolderShareBP | D | LND 6.1.2 | bp | U | geo | - | off=max | - | L158 D115 | - |
| K-LND05 | Tenure.OwnUseExempt | S | LND 5.3.4 | flag | - | - | C1 | - | - | D120 | - |
| K-LND06 | TenureTerm.Rent | S | LND 5.2 | minor | - | - | C2 | chk | - | L155 L156 | - |
| K-LND07 | TenureTerm.Closed | S | LND 4.3 | flag | - | - | C1 | chk | - | L159 | - |
| K-LND08 | Holding.ShareBP | S | LND 10 | bp | - | - | C2 | - | - | L157 L158 L160 | - |

Notes. **Storage** has no knob row: `Base`, a class's `Mass` and `Bulk`, the lists and the class's `Needs` are structural choices of the pack (STO 12.4), grouped in 6.3 (S-STO01 to S-STO07). They carry `chk` (VIA 11 lists `modules.storage.*` as `no (a check)`: the rule `room_below_warm_job`, and the room is not in the programme) and `d`: a room that cuts a bid changes the count of admitted orders, which WLD 7.4.3 says of the storage module (D5 to D8). **K-EMP01 and K-EMP04** are D by rule (3) of 4.3: `MaxHires` is the Home of D105 (EMP-D2) and `MaxOpen` of D110 (EMP-D7), as `Merchants.MaxOpen` (K-MER01) is of D84; EMP's handoff calls `MaxOpen` a structural limit and the ordered test of 4.3 decides. Neither carries `d`: the labour round takes no word of the stream and a parameter that changes the positions posted moves no later word (EMP N-124), and `MaxOpen` fails a formation after the draws of its term, as K-MER01 does (4.2.1). **K-EMP02** is a knob although D104 and D108 name `WorkerShareBP`: they are a rule constant (no bidding up) and a visible lever (the share the circuit needs, EMP 11.3.2), and neither counts it. **K-SET01 and K-SET02** are the damper strengths of D111 and D112 with the off value 0 (no band, no dwell), and the default `Dwell` equals the welfare window by design (SET 3.4.3). SET-D3, the class floor (`RegionSettlement.Floor`, S-SET04), is a bound with no D number (STB 2.3) and a structural choice. No SET field carries `d`: the module takes no draw (a gated role changes the entrants through the role's own knobs, SET 6.4). **K-LND01 to K-LND04** are the damper strengths of D116 (the rent band and the bounded step, on L155) and D115 (the share cap, on L158); `RentStepBP` 0, `RentFloorBP` 0 and `MaxHolderShareBP` 10 000 are their off values and the default state (4.3.1: a calibration moves them away from off and never back), and `RentCeilingBP` is off at its highest legal value. **K-LND05** is the switch of D120 and so S by rule (2). **K-LND06 and K-LND08** are quantities of the setting that no damper counts (a rent per unit, a holder's share) and so S by rule (4), where LND's handoff calls them knobs; the dual `Rent` of the solver's row (VIA 10.2, LND 5.6) is the guide of a pack's rent and not a search range. **K-LND06 and K-LND07** carry `chk`: VIA 11 lists `modules.tenure.term.*` as `no (a report)` (the warning `rent_off_dual`); the holdings are not read.

#### 6.2.10 Polity, credit, infrastructure, risk and fiscal (POL, CRD, CAP, RSK, FIS)

| ID | Key | Class | Owner | Unit | Search | Grid | Stage | Flags | Moves | Loops | Links |
|----|-----|-------|-------|------|--------|------|-------|-------|-------|-------|-------|
| K-CRD01 | Credit.ShieldTicks | D | CRD 8.2 | ticks | U | geo | - | z off=0 | - | L11 L45 L39 L63 D121 | - |
| K-CRD02 | Credit.WriteOffTicks | D | CRD 8.5 | ticks | U | geo | - | z off=0 | - | L166 D125 | - |
| K-CRD03 | Credit.MinPrincipal | K | CRD 14 | minor | U | geo | - | - | - | - | - |
| K-CRD04 | Credit.MaxClaims | D | CRD 6.4 | count | U | geo | - | off=max | - | D128 | - |
| K-CRD05 | Lend.RateBP | K | CRD 7.1 | bp | U | lin100 | - | - | - | L167 | - |
| K-CRD06 | Lend.MaxTermTicks | S | CRD 10.3 | ticks | - | - | C2 | - | - | - | I24 I25 |
| K-CRD07 | Lend.KeepTicks | D | CRD 10.3 | ticks | U | geo | - | z off=0 | - | L163 D124 | I24 |
| K-CRD08 | Lend.ShareBP | D | CRD 10.3 | bp | U | geo | - | off=max | - | L163 L22 L31 L45 D124 | - |
| K-CRD09 | Borrow.MaxRateBP | K | CRD 10.2 | bp | U | lin100 | - | - | - | L167 | - |
| K-CRD10 | Borrow.MaxDebtTicks | D | CRD 5.3.3 | ticks | U | geo | - | off=max | - | L11 L45 L163 D123 | I25 |
| K-CRD11 | Borrow.SlackTicks | K | CRD 10.2 | ticks | U | lin1 | - | - | - | - | I25 I26 |
| K-CAP01 | FacilityClass.Decay | D | CAP 12.2 | units | U | geo | - | in z off=0 | - | L36 L21 L170 D130 | - |
| K-CAP02 | FacilityClass.WearPerUse | D | CAP 12.2 | units | U | geo | - | in z off=0 | - | L36 L21 L170 D130 | - |
| K-CAP03 | FacilityClass.BoostBP | S | CAP 12.2 | bp | - | - | C2 | in | - | L168 | - |
| K-CAP04 | FacilityClass.Capacity | S | CAP 12.2 | units | - | - | C2 | in | - | - | - |
| K-CAP05 | FacilityClass.Durability | S | CAP 12.2 | units | - | - | C2 | in | - | L170 | - |
| K-FIS01 | Tax.Amount | K | FIS 5.1 | minor | U | geo | - | z | - | L174 L180 | - |
| K-FIS02 | Tax.RateBP | K | FIS 5.1 | bp | U | geo | - | z | - | L174 L14 | - |
| K-FIS03 | Tax.Exempt | D | FIS 5.1 | minor | U | geo | - | z off=0 | - | L174 L180 D137 | - |
| K-FIS04 | Duty.RateBP | K | FIS 6.1 | bp | U | geo | - | - | - | L175 L178 | - |
| K-FIS05 | Toll.Amount | K | FIS 7.1 | minor | U | geo | - | - | - | L179 | - |
| K-FIS06 | Payout.Amount | K | FIS 8.1 | minor | U | geo | - | z | - | L175 | - |
| K-FIS07 | Payout.Level | K | FIS 8.1 | minor | U | geo | - | z | - | L175 | - |
| K-FIS08 | Purchase.Quantity | K | FIS 9.1 | units | U | geo | - | - | - | L176 L177 | - |
| K-FIS09 | Purchase.LimitBP | K | FIS 9.1 | bp | U | geo | - | - | - | L177 | - |
| K-FIS10 | Purchase.Hold | D | FIS 9.1 | units | U | geo | - | d z off=0 | - | L176 L177 D140=TreasuryOrdered | - |
| K-FIS11 | Release.Quantity | K | FIS 9.1 | units | U | geo | - | - | - | L176 | - |
| K-FIS12 | Release.LimitBP | K | FIS 9.1 | bp | U | geo | - | - | - | L176 | - |
| K-FIS13 | Release.Reserve | D | FIS 9.1 | units | U | geo | - | d z off=0 | - | L176 D140=TreasuryOrdered | - |

Notes. **POL** has no knob row: a tree and a membership are structure, and `StartMoney` has no effect until a reader (FIS) exists, so it is S at C2 with no range (6.3). **RSK** has none: a danger is a fact of a setting, and `StrikeBP`, `SeverityBP`, `Common` and the existence of a row are structural choices with no search range (RSK 12.5.1); the two scale slots are state and not rows (3.5 item 1). `Common` and the lists of rows carry `d` (S-RSK01 to S-RSK03): a common row draws every tick by configuration (D13, D15) and a plain row only when a shipment crosses it or a store is held (D14), so the count moves with them; `StrikeBP` and `SeverityBP` do not carry it, since a row's loss probability is at least 1 and the count does not depend on its value (RSK 11.4.1). **K-CRD01 to K-CRD11**: none carries `d` (CRD 12.2, N-124: the module takes no word of the stream). `ShieldTicks` (D121) has an interior optimum (CRD 8.2.2) and its default 0 is its off value; `WriteOffTicks` 0 is "never" (D125). `MaxClaims` is D by rule (3) as the Home of D128, a state bound and not a stabiliser of a loop (CRD's handoff calls it structural); `MaxTermTicks` is S by rule (4), a duration, and pairs with `KeepTicks` (I24); `SlackTicks` is read against the recipes' `Duration` (I25) and SAV's `SafetyTicks` may not fall below it (I26). `RateBP` and `MaxRateBP` have no default (0 is written, never defaulted), so their grid is `lin` and not `geo` (5.2). **K-CAP01 and K-CAP02** are the damper strengths of D130 on L36 and L21 (decay brings the accumulation down) and the **driver**, not a damper, of L170 (the same two fields, CAP-A5; off value 0, which is also the default); they carry `in` because VIA 11 lists `modules.infrastructure.*` as moving the table (the facility row and the gains at full condition; `Decay` and the repair are not read by the programme, and the flag follows the path, not the reading). `BoostBP`, `Capacity` and `Durability` are structural with an unknown range (CAP 12.2). No CAP field carries `d` (the module takes no draw). **FIS**: `Tax.Exempt` (D137), `Purchase.Hold` and `Release.Reserve` (D140) are damper strengths whose default 0 is the off value. `Purchase.Hold` and `Release.Reserve` carry `d`: they are the caps that decide whether a treasury's standing order exists at the tick, and a treasury's order is an item of D7 (WLD 7.3.1); FIS's handoff sets no `d` and calls a comparison "not paired" (N-124), which is the same fact stated without the flag. A quantity, a limit and an amount set a size or a price and are not `d` (4.2.1). The windows, targets, payers and scopes of the six kinds are structural (6.3). `Tax.Amount`, `Tax.RateBP`, `Payout.Amount` and `Payout.Level` have a default of 0 and take `z`.

#### 6.2.11 Institutions, saving, technology, policy and insurance (ORG, SAV, TEC, PLC, INS)

| ID | Key | Class | Owner | Unit | Search | Grid | Stage | Flags | Moves | Loops | Links |
|----|-----|-------|-------|------|--------|------|-------|-------|-------|-------|-------|
| K-ORG01 | Body.Places | D | ORG 9 | count | U | geo | - | d z off=0 | - | L183 L184 D144 | - |
| K-ORG02 | Body.MaxTrainees | D | ORG 9 | count | U | geo | - | z off=max | - | L41 D20 | I23 |
| K-ORG03 | Institutions.OwnUseBound | S | ORG 5.5.2 | flag | - | - | C1 | - | - | D146 | - |
| K-ORG04 | Control.CapBP | S | ORG 6.1 | bp | - | - | C2 | chk | - | L181 | - |
| K-ORG05 | Control.FloorBP | S | ORG 6.1 | bp | - | - | C2 | chk | - | L182 | - |
| K-ORG06 | RateLimit.CapBP | S | ORG 6.1 | bp | - | - | C2 | - | - | - | - |
| K-SAV01 | Saving.SafetyTicks | S | SAV 5.6.1 | ticks | - | - | C2 | d | - | - | I26 |
| K-SAV02 | Saving.HurdleSpreadBP | D | SAV 5.5.2 | bp | U | geo | - | d off=0 | - | L36 L21 D149 | - |
| K-SAV03 | Saving.CommitBandBP | D | SAV 5.5.3 | bp | U | geo | - | d off=0 | - | L188 D148 | - |
| K-SAV04 | Saver.PrecautionShareBP | K | SAV 4.2 | bp | U | geo | - | d z | - | L45 D150 D151 | - |
| K-SAV05 | Saver.PrecautionCapTicks | D | SAV 4.2 | ticks | U | geo | - | d off=max | - | L45 D150 | - |
| K-SAV06 | Saver.HurdleBP | K | SAV 5.5.1 | bp | U | lin1000 | - | d | - | L36 L21 | - |
| K-SAV07 | Saver.PaybackTicks | K | SAV 5.3 | ticks | U | geo | - | d | - | D153 | - |
| K-SAV08 | Saver.RepairBelowBP | K | SAV 5.8 | bp | U | lin1000 | - | d | - | L189 | - |
| K-TEC01 | TechField.CapBP | D | TEC 9.2 | bp | U | geo | - | off=- | - | L191 L88 D155=ClampedBP | - |
| K-TEC02 | TechField.DoublingBP | K | TEC 9.2 | bp | U | geo | - | z | - | L191 | - |
| K-TEC03 | TechField.TrendBP | K | TEC 9.2 | bp | U | geo | - | z | - | L192 | - |
| K-TEC04 | TechField.DiscoveryBP | K | TEC 9.2 | bp | U | geo | - | z | - | L21 L22 | - |
| K-TEC05 | TechField.StepBP | K | TEC 9.2 | bp | U | geo | - | - | - | L21 L22 | - |
| K-TEC06 | TechField.CatchUpBP | K | TEC 9.2 | bp | U | geo | - | z | - | L194 | I27 |
| K-TEC07 | TechField.Reach | K | TEC 9.2 | units | I27 | geo | - | - | - | L194 | I27 |
| K-TEC08 | TechField.Base | S | TEC 9.2 | units | - | - | C2 | - | - | L191 | - |
| K-PLC01 | PolicyActor.Every | D | PLC 7.2 | ticks | U | geo | - | off=1 | - | L195 L196 D161 | - |
| K-PLC02 | Lever.MaxStep | D | PLC 6.2.2 | units | U | geo | - | z off=0 | - | L196 L198 D159=Stepped | - |
| K-PLC03 | Reaction.Deadband | D | PLC 8.3.1 | units | U | geo | - | z off=0 | - | L195 D160=InBand | - |
| K-PLC04 | Reaction.MaxAge | D | PLC 8.3.2 | ticks | U | lin1 | - | off=max | - | L196 D162=NoReading | - |
| K-PLC05 | Reaction.Span | K | PLC 8.3.1 | units | U | geo | - | - | - | L195 | A17 |
| K-PLC06 | Reaction.Move | K | PLC 8.3.1 | bp | U | lin100 | - | - | - | L195 | A17 |
| K-PLC07 | Lever.Min | S | PLC 3.4.1 | units | - | - | C2 | - | - | D158=Clamped | - |
| K-PLC08 | Lever.Max | S | PLC 3.4.1 | units | - | - | C2 | - | - | D158=Clamped | - |
| K-PLC09 | Reaction.Target | S | PLC 8.3.1 | units | - | - | C2 | - | - | L195 | - |
| K-INS01 | Insurance.ValueCapBP | D | INS 6.2 | bp | U | geo | - | off=max | - | L200 D166 | - |
| K-INS02 | Underwrite.CoverBP | D | INS 13 | bp | U | geo | - | off=max | - | L84 L85 L93 L19 D164 | - |
| K-INS03 | Underwrite.CapBP | D | INS 13 | bp | U | geo | - | off=max | - | L201 D165 | - |
| K-INS04 | Underwrite.LoadingBP | S | INS 6.4 | bp | - | - | C2 | - | - | - | - |
| K-INS05 | Underwrite.StoreRateBP | S | INS 6.4 | bp | - | - | C2 | - | - | - | - |

Notes. **K-ORG01 and K-ORG02** are D by rule (3) as the Homes of D144 and D20; ORG's handoff calls them knobs (and `MaxTrainees` "a damper strength of D20"). `Places` carries `d`: a refusal moves D6 to D8 (ORG-N5; 7.2.3), and `MaxTrainees` does not, a training place being no order. The fields of a `Right` carry `d` as well (S-ORG03): a right is a refusal by design. `OwnUseBound` is the switch of D146 (S by rule (2)). `CapBP` and `FloorBP` of a control and a rate limit are **policy values**, a pack's data that a calibration reports and never searches; a clamp changes no count (EXP 7.2.3). **K-SAV01 to K-SAV08 and every SAV group** carry `d` (SAV 9.2, N-124): a target `S*` above the reserve lowers the budgets that admit an order (D7, D8) and a planned project changes the intents considered (D5), as soon as some household of a role with a block has `S*` above its reserve. `PrecautionShareBP` is the strength of the amplifier of L45 (SAV-A1) with the off value 0 and is a knob, since an `off=` is for class D only (4.2.1); `SafetyTicks` is S with a derived condition (I26), so its Search is `-` and the condition is in Links. **TEC**: the five rates are K, and `DoublingBP`, `TrendBP`, `DiscoveryBP` and `CatchUpBP` take `z` (the owner's default 0 is "off", and a `geo` grid is built on the pack's non-zero value, 5.2); `StepBP` and `Reach` are `required` with their rate above 0 and take no `z`; `CapBP` is D with `off=-` (it is `required`, so no value disables it and it can only be weakened); `Reach` has a derived condition (I27) and is its own Search; `Base` and `Recipes` are structural. No TEC field carries `d`: the module's one word is a construction word (draw D16, WLD 7.4.2) taken only when a field has `DiscoveryBP` above 0, and 0 is never a grid level (4.2.1, K-INF02). No TEC field carries `in` or `chk`: VIA 11 lists `modules.technology.*` under "Not read" (a gain is 0 at tick 0). **PLC**: `Every`, `MaxStep`, `Deadband` and `MaxAge` are the damper strengths of D161, D159, D160 and D162; `Span` and `Move` enter a run only as the gain `Move / Span`, up to the truncation (A17); `Min` and `Max` are the mandate, a policy shape and so S by rule (2) though D158 counts them, and `Target` is the setting's goal. No PLC field carries `d`: the built-in policies take no word, and a consumer's `Policy` that draws is outside the table (PLC 7.3.1, L120). `MaxAge` is `lin1` because its owner default is `required` with a lowest legal value of 0 (5.2). **INS**: `ValueCapBP`, `CoverBP` and `CapBP` are the damper strengths of D166, D164 and D165 (INS 20.3); the posted prices and every quantity are structural. No INS field carries `d` (the module takes no word, INS 12.2).

### 6.3 Structure groups

The S and P fields of a scope that need no row of their own. A group row names the
**scope token** (4.7) and the **fields** it covers. Its class is S or P, or, for the
elements of a list that has a knob row, the class of that list (the knots of a
schedule, the knots of an abundance curve). The groups are the registry's answer to
"which fields of the owner table are not knobs": **identity and topology** (stage C1),
**quantities of the setting** (stage C2: the solver checks them) and **units**.

| ID | Scope | Fields | Class | Owner | Stage | Flags |
|----|-------|--------|-------|-------|-------|-------|
| S-CAL01 | Calendar | HoursPerDay, DaysPerWeek, Months, TicksPerDay, DaysPerTick, StartDay, Seasons, SeasonStart | S | CAL 3.1 | C1 | in |
| S-GDS01 | Commodity | ID, Classes | S | GDS 8 | C1 | - |
| S-GDS02 | Commodity | ShelfLife, Durability | S | GDS 8 | C2 | in |
| S-GDS03 | Commodity | ReferencePrice | S | GDS 8 | C2 | out chk |
| S-GDS04 | Commodity | Raw | S | GDS 8 | C1 | in |
| S-GDS05 | Commodity | Mass, Bulk | S | GDS 8 | C2 | - |
| S-CUR01 | Region | ID | S | CUR 5 | C1 | - |
| S-CUR02 | Region | ReferenceBasket | S | CUR 5 | C2 | - |
| S-AGT01 | World | Upkeep | S | AGT 5.2 | C2 | in |
| S-AGT02 | Region | ExtraUpkeep | S | AGT 5.2 | C2 | in |
| S-AGT06 | Role | Upkeep | S | AGT 5.2 12 | C2 | in |
| S-AGT03 | World | BasketTolerance | S | AGT 12 | C1 | - |
| S-AGT04 | Role | ID, Recipes | S | AGT 3.3 | C1 | in |
| S-AGT07 | Role | Trades, Strategy | S | AGT 3.3 6.6 | C1 | - |
| S-AGT05 | Role | StartStock | S | AGT 3.3 | C2 | - |
| S-SPN01 | World | Spawner | S | SPN 3.1.3 | C1 | - |
| S-SPN02 | Role | Closed | S | SPN 11 | C1 | - |
| S-CLR01 | World | Clearing | S | CLR 8 | C1 | - |
| S-WEL01 | Role | Unskilled, InKind | S | WEL 4.2 6 | C1 | - |
| S-PRO01 | Recipe | ID, Inputs, Outputs, Requires, Draws, Sited | S | PRO 3.1 6.2 | C1 | in |
| S-PRO02 | Recipe | Duration | S | PRO 5 | C2 | in |
| S-PRO03 | Recipe output | CostShare, Discard | S | PRO 3.2 7.2 | C1 | in |
| S-PRO04 | Recipe tool | WearPerUse | S | PRO 4.4 | C2 | in |
| S-PRO05 | Region and recipe | YieldProfile | S | RES 7 | C2 | in |
| S-PRO06 | Recipe | TraineeEligible | S | PRO 8 | C1 | - |
| S-RES01 | Yield | Region, Recipe, All, Seasons | S | RES 7 | C2 | in |
| S-SCL02 | Group | Households, Region, Role | S | SCL 3.1 | C2 | in d |
| S-REG01 | Config section | Edges, RegionPrices | S | REG 4.1 5.2 | C1 | - |
| S-REG02 | Edge | From, To | S | REG 4.1 | C1 | - |
| S-REG03 | Edge | Distance | S | REG 4.1 | C2 | in |
| S-REG04 | RegionPrice | Region, Commodity | S | REG 5.2 | C1 | - |
| S-REG05 | RegionPrice | Price | S | REG 5.2 | C2 | out chk |
| S-VIA01 | Config section | WarmStart | S | VIA 8.1 | C2 | out |
| S-VIA02 | WarmStart | Prices, Wages, Groups | S | VIA 8.1 | C2 | out |
| S-VIA03 | WarmStart | Pools | S | VIA 8.1 | C2 | - |
| S-DEM01 | Demand | Wants | S | DEM 9 | C1 | - |
| S-DEM02 | Want | Items, Quantity, Period, Rank | S | DEM 9 | C2 | in |
| S-DEM03 | Want item | Commodity, Weight | S | DEM 9 | C2 | in |
| S-DEM04 | Taste | Want, Scale | S | DEM 9 | C2 | in |
| S-DEM05 | Taste | Role | S | DEM 9 | C1 | - |
| S-DEM06 | RegionTaste | Region, Want, Scale | S | DEM 9 | C2 | in |
| S-MON01 | Modules.MoneySupply | Flows | S | MON 4.1 | C1 | - |
| S-MON02 | MoneyFlow | ID, Kind, Target, Amount, Level, Per, Start, Until, NotIncome | S | MON 4 7.2 | C8 | - |
| S-POP01 | Knot | Ratio, Rate | D | POP 8 | C9 | - |
| S-POP02 | RegionPopulation | Region | S | POP 8 | C1 | - |
| S-TRN01 | Transport | Modes | S | TRN 3.1 | C1 | - |
| S-TRN02 | Mode | ID | S | TRN 3.1 | C1 | - |
| S-TRN04 | Mode | Crew, Vehicle, WearPerTick, Feed | S | TRN 3.1 | C2 | in |
| S-TRN03 | Mode | Speed, UnitMass, UnitBulk, Haul, HaulSpan | S | TRN 3.1 | C2 | in |
| S-RES02 | Resources | Resources, Endowments | S | RES 3.3 | C1 | - |
| S-RES03 | Resource | ID, Shape, RatePerYear, Capacity, InitialBP, Seasons | S | RES 12 4.5 | C2 | in |
| S-RES04 | AbundanceKnot | Share, Factor | D | RES 4.7 | C9 | in |
| S-RES05 | Endowment | Region, Resource, Capacity, Initial, RegenScaleBP | S | RES 3.3.1 | C2 | in |
| S-RES06 | SeasonRegen | Season, BP | S | RES 4.5 | C2 | in |
| S-CFG01 | Modules | Information, Resources, Transport, External, Merchants, Migration, Population, Subsistence, Settlements, Storage, Employment, Tenure, Credit, Risk, Infrastructure, Insurance, Polity, Fiscal, Institutions, Policy, Saving, Technology, Estates | S | INF 8; RES 3.3; TRN 3.1; EXT 3.1; MER 13; MIG 11; POP 8; SUB 3.2; SET 8; STO 12.1; EMP 13; LND 10; CRD 14; RSK 12.1; CAP 12.1; INS 13; POL 8; FIS 14; ORG 9; SAV 10; TEC 9.1; PLC 10; EST 2.6 | C1 | - |
| S-INF01 | Role | Informed | S | INF 6.1 | C1 | d |
| S-EXT01 | External | Nodes | S | EXT 3.1 | C1 | - |
| S-EXT02 | OutsideNode | Region, Lines, Gates | S | EXT 3.1 3.3 3.2.4 | C1 | - |
| S-EXT03 | OutsideGate | Region | S | EXT 3.2.4 | C1 | - |
| S-EXT04 | OutsideLine | Commodity, Buy, Sell | S | EXT 3.3 | C1 | - |
| S-EXT05 | OutsideLine | Base, PriceBySeason, TrendBP | S | EXT 6.1 6.2 6.3 | C8 | in |
| S-EXT06 | OutsideLine | WalkStepBP | S | EXT 6.4 | C8 | in d |
| S-EXT07 | OutsideSide | Window | S | EXT 7.2 | C8 | in |
| S-EXT08 | SeasonFactor | Season, BP | S | EXT 6.2 | C8 | in |
| S-SUB01 | Subsistence | Recipes, Roles | S | SUB 3.2 | C1 | - |
| S-SUB02 | RoleOwnUse | Role, Recipes | S | SUB 3.2 | C1 | - |
| S-MIG01 | Migration | MoveSpeed, Regions | S | MIG 6.4 11.2 | C2 | - |
| S-MIG02 | RegionMove | Region | S | MIG 11.2 | C1 | - |
| S-MIG03 | Role | Fixed | S | MIG 3.1 | C1 | d |
| S-MER01 | Merchant | Modes, Consigns | S | MER 3.2 | C1 | d |
| S-MER02 | Role | Merchant | S | MER 3.2 | C1 | - |
| S-MON03 | FlowTarget | Kind, Region, Role | S | MON 4.1 | C8 | - |
| S-STO01 | Storage | Roles, Classes | S | STO 12.1 | C1 | chk d |
| S-STO02 | Storage | Base | S | STO 12.1 | C2 | chk d |
| S-STO03 | RoleRoom | Role, Stores | S | STO 12.1 | C1 | chk d |
| S-STO04 | RoleRoom | Base | S | STO 12.1 | C2 | chk d |
| S-STO05 | StoreClass | ID, Needs | S | STO 3.2 12.1 | C1 | chk d |
| S-STO06 | StoreClass | Mass, Bulk | S | STO 3.2 12.1 | C2 | chk d |
| S-STO07 | Capacity | Mass, Bulk | S | STO 12.1 | C2 | chk d |
| S-STO08 | FlowTarget | Store | S | STO 7.1.2 12.3 | C1 | - |
| S-EMP01 | Role | Hireable, Employer | S | EMP 13 | C1 | - |
| S-EMP02 | Employer | Recipes | S | EMP 13 | C1 | - |
| S-SET01 | Settlements | Classes, Regions | S | SET 8 | C1 | - |
| S-SET02 | SettlementClass | ID, Enter, Flags, Limits | S | SET 8 | C1 | - |
| S-SET03 | ClassLimit | Name, Value | S | SET 8 | C1 | - |
| S-SET04 | RegionSettlement | Region, ThresholdBP, Floor | S | SET 8 | C1 | - |
| S-SET05 | Role | Needs, Ceiling | S | SET 8 | C1 | - |
| S-LND01 | Tenure | Terms, Holdings | S | LND 10 | C1 | - |
| S-LND02 | TenureTerm | Region, Resource | S | LND 10 | C1 | chk |
| S-LND03 | Holding | Region, Resource, Holder | S | LND 10 | C1 | - |
| S-POL01 | Polity | Jurisdictions, Members | S | POL 8 | C1 | - |
| S-POL02 | Jurisdiction | ID, Parent, Seat | S | POL 8 | C1 | - |
| S-POL03 | Jurisdiction | StartMoney | S | POL 8 | C2 | - |
| S-POL04 | Member | Region, Jurisdiction | S | POL 8 | C1 | - |
| S-CRD01 | Role | Credit | S | CRD 14 | C1 | - |
| S-CRD02 | CreditTerms | Lend, Borrow | S | CRD 4.2 | C1 | - |
| S-CAP01 | Infrastructure | Classes, Initial | S | CAP 12.1 | C1 | in |
| S-CAP02 | FacilityClass | ID, Kind, Modes, Resource, Store, Needs | S | CAP 12.2 3.1 | C1 | in |
| S-CAP03 | InitialFacility | Region, Class, Toward, Role | S | CAP 3.4.2 | C1 | in |
| S-CAP04 | InitialFacility | Count | S | CAP 3.4.2 | C2 | in |
| S-CAP05 | Recipe | Needs, Builds, Repairs | S | CAP 4.1.1 | C1 | - |
| S-CAP06 | FacilityBuild | Class, Toward | S | CAP 4.1.1 | C1 | - |
| S-CAP07 | FacilityRepair | Class | S | CAP 4.1.1 | C1 | - |
| S-CAP08 | FacilityRepair | Units | S | CAP 4.1.1 | C2 | - |
| S-RSK01 | Risk | Routes, Stores | S | RSK 12.1 | C1 | d |
| S-RSK02 | RouteRisk | From, To, Mode, Common | S | RSK 3.2.1 3.2 | C1 | d |
| S-RSK03 | StoreRisk | Region, Class, Common | S | RSK 3.2.2 3.2 | C1 | d |
| S-RSK04 | RouteRisk | StrikeBP, SeverityBP | S | RSK 3.2 | C2 | - |
| S-RSK05 | StoreRisk | StrikeBP, SeverityBP | S | RSK 3.2 | C2 | - |
| S-FIS01 | Fiscal | Taxes, Duties, Tolls, Payouts, Purchases, Releases | S | FIS 14 | C1 | - |
| S-FIS02 | Tax | ID, Authority, Window, Target, Per | S | FIS 14 3.1 | C1 | - |
| S-FIS03 | Duty | ID, Authority, Window, Region, Commodity, Payer, Scope | S | FIS 14 3.1 | C1 | - |
| S-FIS04 | Toll | ID, Authority, Window, On, Mode | S | FIS 14 3.1 | C1 | - |
| S-FIS05 | Payout | ID, Authority, Window, Target, Commodity, NotIncome, Per | S | FIS 14 3.1 | C1 | - |
| S-FIS06 | Purchase | ID, Authority, Window, Commodity | S | FIS 14 3.1 | C1 | - |
| S-FIS07 | Release | ID, Authority, Window, Commodity | S | FIS 14 3.1 | C1 | - |
| S-FIS08 | Window | Start, Until, Seasons | S | FIS 14 | C2 | - |
| S-FIS09 | FlowTarget | Jurisdiction | S | FIS 14 | C1 | - |
| S-ORG01 | Institutions | Bodies, Controls, RateLimits | S | ORG 9 | C1 | - |
| S-ORG02 | Body | ID, Roles, Regions, Jurisdictions, Needs | S | ORG 9 | C1 | - |
| S-ORG03 | Right | ID, Body, Activity, Subject, Region, Jurisdiction, Who | S | ORG 9 | C1 | d |
| S-ORG04 | Control | ID, Commodity, Region, Jurisdiction | S | ORG 9 | C1 | chk |
| S-ORG05 | RateLimit | ID, Region, Jurisdiction | S | ORG 9 | C1 | - |
| S-ORG06 | Institutions | Rights | S | ORG 9 | C1 | d |
| S-SAV01 | Role | Saver | S | SAV 10 | C1 | d |
| S-SAV02 | Saver | Worth | S | SAV 10.2 | C1 | d |
| S-SAV03 | ProjectWorth | Class, PerTick, Count | S | SAV 10.2 | C2 | d |
| S-TEC01 | Technology | Fields | S | TEC 9.1 | C1 | - |
| S-TEC02 | TechField | ID, Recipes | S | TEC 3.1 | C1 | - |
| S-PLC01 | Policy | Actors | S | PLC 10 | C1 | - |
| S-PLC02 | PolicyActor | ID, Jurisdiction, Policy, ReadsRegions, Levers, Schedule, Cycle, Rules | S | PLC 10 | C1 | - |
| S-PLC03 | Lever | ID, Kind, Flow, From, To, Mode, Against, MayClose | S | PLC 3.4 3.5 | C1 | - |
| S-PLC04 | ScheduleStep | Lever, At, BP, Closed | S | PLC 8.2 | C1 | - |
| S-PLC05 | Reaction | Lever, Gauge, Commodity, Raise | S | PLC 8.3 | C1 | - |
| S-INS01 | Insurance | PayoutTicks, MinSum, MaxPolicies, MaxCoverRequests | S | INS 13 | C2 | - |
| S-INS02 | Role | Cover | S | INS 13 | C1 | - |
| S-INS03 | CoverTerms | Underwrite, Insure | S | INS 4.2 | C1 | - |
| S-INS04 | Underwrite | ShareBP | S | INS 13 | C2 | - |
| S-INS05 | Insure | MinLossBP, RuinShareBP, MaxLoadingBP, MinCoverBP, StoreShareBP, MaxStoreRateBP | S | INS 13 | C2 | - |
| S-PRS01 | Presentation | Label, UnitLabel, DistanceLabel | P | CFG 8.3 | - | - |

Notes. **S-GDS03, S-REG05, S-VIA01, S-VIA02** are `out`: the table of a
pack is **solver output** (`Pools` is not: S-VIA03, which the table may seed with a default of 0 and the solver does not write, VIA 8.3.3) and is regenerated, never written by hand (REF position 4,
FR-REF-007). **S-PRS01** is the class P group: the presentation-only fields
(`spec-config.md` 8.3), which are not rows of the owner table and are not checked
against it (the coverage check is one-directional for them). **S-SCL02** carries `d`
because the households of a group fix the number of agents and so the number of
creation draws (WLD 7.3, D1 and D10). **S-MIG01**: `MoveSpeed` is a *gauge* (A07). **Fields that a damper entry also names** and that define an identity, an order or a memory are S by rule (2) of 4.3, which is applied before rule (3): `Role.Closed` (D36), a want's `Rank` (D31) and `OutsideSide.Window` (D65) define identity, order and the memory of a line, and a calibration does not move them (C1 or C2). **S-AGT03** (`BasketTolerance`) only decides whether a configuration is accepted (`ErrBasketMismatch`) and changes no result; it is kept S at C1 with the reference basket, not P, because it is in the hash. **S-TRN04** is a quantity staged at C2 with the other quantities of the mode. **S-CUR02** (`ReferenceBasket`) is a calibration anchor and not a live value (CUR 5.4): it is not solver output (it has no `out`), the solver reads the upkeep and the reference prices (S-AGT01, S-GDS03) and not the field, and I22 checks the field against them. **S-AGT05** (`StartStock`) has no `in`: the warm start overrides a role's defaults for money and inventory (VIA 3.3 item 3). **S-AGT06** is `Role.Upkeep`, the owner row `Upkeep` with the README remark "also Role, replacing the world's" (a pair of its own, 4.7), and VIA 11 lists `role.<id>.upkeep`. **S-GDS04, S-GDS05, S-AGT07, S-PRO06 and S-VIA03** are what is left of rows split to follow the published list of VIA 11 (4.2.1): `Raw` is read and `Mass` and `Bulk` are not, `Recipes` is read and `Trades` and `Strategy` are not, `TraineeEligible` is not read, and `Pools` is seeded and not written. **S-DEM05** and **S-DEM06** are the `Tastes[]` and `RegionTastes[]` of `spec-config.md` 3.7; the README owner table has their rows since the integration of the fourteen later modules (they were owed, 23, X7), so these two groups are owner pairs like the others.

**The groups of the fourteen later modules** (S-MON03 to S-INS05) follow the order of the `Modules` row. **S-CFG01** lists the fourteen names beside the eight earlier ones and `Estates`, the 25th and last of the `Modules` row and a pointer with no field of its own (the pool policy, `spec-config.md` 6.2.4, N-220), so every `Modules` field of the owner table is a group field: 23 names in all, the 22 of the earlier rows and `Estates`; `Demand` and `MoneySupply` have no pointer row in the owner table (their tables are the scopes of S-DEM01 and S-MON01); the order of the names is the order of the hash (`spec-config.md` 3.1, 8.2). **Owed (N-220):** the owner rows that `spec-config.md` 9.2 added or split (`Scale` of `World` against `Scale` of `Group`, `Upkeep` of `Role`, the `Yields` section, and the element types `Month`, `Season`, `SeasonYield`, `Item`, `WantItem`, `Output`, `Tool`, `SeedPrice`, `SeedWage`, `WarmGroup`, `JobSpec` and `WarmPool`) have no row or group here yet, and some groups still carry the earlier scope tokens (`Recipe output`, `Recipe tool`, `Want item`, `Yield`, `Config section`); `models/clb/registry.py` reports them as the failed checks of coverage (11.1) until each is classified, which is a decision per field and not a respelling. **S-MON03** is the three old fields of `FlowTarget` (MON owns the type; STO owns `Store`, S-STO08, and FIS owns `Jurisdiction`, S-FIS09, F10); the new kinds `stores` and `treasury` are values of `Kind`, not fields. **The flags follow VIA 11 and WLD 7.3.1.** `chk`: `modules.storage.*` (S-STO01 to S-STO07), `modules.institutions.control.<id>.*` (S-ORG04 and K-ORG04, K-ORG05) and `modules.tenure.term.*` (S-LND02, K-LND06, K-LND07), each `no (a check)` or `no (a report)`; `in`: `modules.infrastructure.*` (S-CAP01 to S-CAP04 and K-CAP01 to K-CAP05). The other ten modules are under "Not read" and carry neither (the technology gain is 0 at tick 0). The `Recipe.Needs`, `Builds`, `Repairs` of S-CAP05 and the build and repair clauses (S-CAP06 to S-CAP08) carry no flag: no path of the first table of VIA 11 names a recipe's `needs`, `builds` or `repairs`, and the table says that a field no row names is not read. `d`: the room (S-STO01 to S-STO07, WLD 7.4.3), the rows and the `Common` of the risk module (S-RSK01 to S-RSK03), the fields of a right (S-ORG03, S-ORG06) and every saving field (S-SAV01 to S-SAV03, N-124); nothing else. **Stage:** C1 identity, topology and lists, C2 quantities of the setting (a room, a rent, a rate of danger, a limit, a window of ticks), as for the earlier groups (7.3.12). **S-FIS02 to S-FIS07** repeat the fields that the six kinds share (`ID`, `Authority`, `Window`) one group per kind, since a group has one scope token (4.8); the `Window` struct is S-FIS08. **S-POL03** (`StartMoney`) is S at C2 with no range until a reader exists.

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
| I19 | A change of any `in` knob, or of a solve option (`tbl`), makes the `WarmStart` table stale: the experiment regenerates it per point (`Prepare`) or says it is a perturbation | gauge | `spec-experiments.md` 3.3.3, EXP-A5 | every `in` and `tbl` row |
| I20 | A group's weight is at most `MaxWeight`; a role with `TrainingTicks` above 0 needs groups of scale 1; a group with a weight above 1 whose agents fall below `MinAgents` draws `group_below_floor` | rule, warn | `spec-scale.md` 3, 5.2; `spec-skill-barriers.md` 4.1.1 | Scale, MinAgents, Households, TrainingTicks |
| I21 | Relief and entry draw on one pool in time: entrants take at most `EntryPoolShare` of it, relief is `PoolRelease` of it, and the default keeps at least half for relief | cond | `spec-spawner.md` 7.3; `spec-estates.md` 2.6; N-082 | EntryPoolShare, PoolRelease |
| I22 | A region's `ReferenceBasket` equals the cost of the baseline `Upkeep` plus its `ExtraUpkeep` at its effective reference prices within `BasketTolerance` (`ErrBasketMismatch`) | rule | `spec-config.md` 7.4.1; `spec-currency.md` 5.3 | Upkeep, ExtraUpkeep, ReferencePrice, ReferenceBasket |
| I23 | The traineeship throttle is `trainers * MaxTrainees / TrainingTicks`; a trainer's gain per trainee-tick depends on `TraineeYield`, `TeachingLoad` and the wage; training pays only above a break-even `TraineeYield` | cond | `spec-skill-barriers.md` 4.3, 4.4, 4.9; `spec-production.md` 8.3 | TrainingTicks, MaxTrainees, TeachingLoad, TraineeYield |
| I24 | A lender with no recipe and no money flow that targets it needs `KeepTicks` at least its `MaxTermTicks`: below it the lender lends money it must eat before the money returns (`lender_keep_below_term`) | warn, cond | `spec-credit.md` 10.3, 15.2 | KeepTicks, MaxTermTicks |
| I25 | A borrower's request has the horizon `Duration` plus 1 plus `SlackTicks` (plus 1 more where the inputs cannot be bought at once, `Late`): some lender's `MaxTermTicks` must reach it (`term_exceeds_lenders`) and so must `MaxDebtTicks` of the borrower role (`cap_below_horizon`) | warn, cond | `spec-credit.md` 10.2, 15.2 | SlackTicks, Duration, MaxTermTicks, MaxDebtTicks |
| I26 | A role with a `Saver` block and a `Borrow` block has `SafetyTicks` at least its `SlackTicks` (`safety_below_slack`), so that the default borrower finds no gap for a funded project (FR-SAV-023); a role with a `Saver` block alone is not constrained | warn, cond | `spec-saving-and-investment.md` 6.1.1, 11.2; `spec-credit.md` 10.2 | SafetyTicks, SlackTicks |
| I27 | A field with `CatchUpBP` above 0 has contact only between ordinary regions at a finite distance below `Reach`, with the weight `(Reach - d) * 10 000 / Reach` and 0 at `Reach`; `Reach` is read in the pack's distance unit (A07), and the warning `diffusion_no_contact` says that no pair is within it | warn, cond | `spec-technology.md` 4.5.1, 10.2 | Reach, CatchUpBP, Distance |

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
| A17 | alias | `Reaction.Span` and `Reaction.Move` | one gain `Move / Span` per decision (the step is `trunc(e * Move / Span)`, PLC 8.3.1): the facts see the quotient and, through the truncation, the pair only at the margin | searches the gain as a composite factor and fixes `Span` by structure (the gauge's own unit); `Move` then carries the gain (the bound of the loop is on the gain, PLC-A1) |

### 6.6 Retired rows

**A11** (the composite of `Elasticity` and `Ceiling` for a luxury) is retired: its statement
was not the owner's (the headroom of DEM 6.3 is the budget, I18), and its ID is never reused,
which is the gap between A10 and A12. A row retired with its field is listed here, with the
change that removed it.

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
gate, and it recommends a **hold-out** (51 to 60), which REF 8.2 now lists as a recommendation
(a SHOULD: run only if wanted; no band and no calibration choice is set from it, and FR-REF-024
stays as written). A
calibration adds one role of its own, the **search blocks**, and uses the others for
their purposes. **The validation step of the procedure is the confirm**: a point chosen
on seeds outside every set of REF (101 and up) is run on REF's own nightly seeds 1
to 50 (tier L 1 to 10). Nothing here depends on seeds outside REF's sets for a confirm, or on a hold-out that REF does not define.


| Role | Seeds | Used for | Never used for |
|------|-------|----------|----------------|
| **search block** `b` (`b` at least 1) | `100 b + 1` to `100 b + 100`: 101 to 200, 201 to 300, and so on | the screen, the explore and the choose of **one** choice | a second choice; the confirm; a band |
| **confirm** | 1 to 50 (tier L: 1 to 10): STY's setting seeds, which are not seeds a choice was made on | the confirm of the chosen point; the stage stop; the runs STY 8.4 sets the stage's bands from | a choice |
| **gate** | 1 to 10, inside the confirm range | STY's merge-gate assertion of a set band, and the stability record of a default (STY 7.3) | an input to a choice: its verdict is a **finding** |
| **hold-out** | 51 to 60, **where it was run** (REF 8.2 lists it as a recommendation, a SHOULD; STY 8.2 recommends it and nothing here relies on it) | the out-of-sample check of a stability record (7.4.2), when it exists | a choice; a band; no band is set from it |

7.1.1. **Where a choice is made.** A choice is made on **search blocks only**, which
are disjoint from every set STY names (1 to 60). The confirm then evaluates the
chosen point on seeds it was **not chosen on** (STY 10.2: "the calibration's own separation of search and confirm; it is not the bands' hold-out"; EXP 4.6.4's example of another `Seed0` is 1 001). The confirm's per-seed values are **not read into the
next search**: a failed confirm returns to the next block **from the original base**,
and an accepted one is frozen. That is what keeps the gate, which is among the
confirm seeds, from being an input to a choice.

7.1.2. **Why a block is used once.** A point chosen as the best of `N` noisy pass
shares has its noise selected for (the winner's curse, EXP-A2). A second choice on
the same block would fit the second choice to the first's noise. A block therefore
serves **one** choice, the block is named in the record (section 8), and a failed
confirm sends the calibration to the **next** block. The arithmetic is in 18 and
19.4: of 16 points that are each really inside their band in 85% of seeds, the best
passes a 10-seed SR2 search with probability 9 999.97 bp (10 000 bp to the nearest
basis point, which is not certainty), and passes a 50-seed confirm in 22% of cases
(2 194 bp). A search that stopped at its own seeds would
call an 85% point stable.

7.1.3. **Seeds per point.** The search uses `R` replicas a point (placeholder 10: a
band is never set from fewer than 10 seeds, FR-STY-078, and a search that sees fewer
cannot tell SR2's one in ten from none); the confirm uses 50 (tier L: 10), STY's own
setting range. Both use `SameSeeds` (EXP 4.6.3) so that two points of one block are
seed-matched, except for a knob with the flag `d`, which is compared as
distributions (7.2.3).

7.1.4. **Escalation.** After **three** failed attempts at one stage (placeholder: an attempt
fails by a failed confirm or by a failure of the gate or of the hold-out, 7.4.4), the stage
is not searched again: the facts, the priors or the structure are reviewed (7.5), because
three independent blocks that all missed say that the region is empty or the target is
wrong, not that the search was unlucky.

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
and confirm: C4 is split (C4a has exactly 16 knobs) and so is C7 (C7a 12 knobs, C7b 10).

7.2.2. **Explore.** An experiment of `Spread` over the screened-in knobs as range
factors, with `N = max(16, 10 k)` points for `k` screened-in knobs (a placeholder rule), on the
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
`Pair` is refused), the alignment horizon is recorded (EXP 7.3). A binding right of the institutions module is unaligned from its first refusal (a refusal moves D6 to D8, a clamp changes no count, ORG-N5; EXP 7.2.3): the fields of a `Right` carry `d` (S-ORG03), as does `Places` (K-ORG01), so the knobs of that module are compared as distributions too.

7.2.4. **Confirm.** One experiment: the chosen configuration (the base with the
overlay of the choice) and, as a **control**, the base, each over the confirm seeds
(`Seed0` 1, `Replicas` 50, tier L: 10), the same `Ticks`, `Transient` and measures,
on the stage's packs. The verdict is the stage stop (7.4.1). The confirm's runs are
**also** the reviewed run that STY 8.4 reads the stage's band keys from, **once the choice is
frozen and in STY's order** (FR-STY-091): the confirm of a stage sets **no key by itself**, and the
keys of C2 (H01 to H04, D01) wait for STY's step 3 and those of C3 (P09) for step 5 (7.3). The
gate, which is inside the confirm range, then asserts a set band at the merge gate (STY 3.4:
a band is **set** when its key has a value and the gate asserts it, and **held** only once the
gate and the nightly have passed in later builds), and STY 8.2 says plainly that this is not
out of sample.

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
| **C2** | Viability and warm start | S rows at C2 (quantities, among them `ExpectedWorkTicks` and `StartMoney`); the three K and D rows at C2: `HorizonYears` (D) and the two solve options (K, `tbl`) | the solver report; H01 to H04; D01; G01 | a `Grid` over a pack quantity with `Prepare` = the solver; then `R` replicas | seconds of solver, then a stage run |
| **C3** | Scale | `Scale`, `MinAgents` | P09 (X parts; B part by its working prior); scenario S10 | `one` + `overlay/h200` at scales 1, 4 and 10 (`Grid`); not applicable at weight 1 | 30 runs, 0.0 core-hours (in the totals) |
| **C4a** | Price formation | the 16 BEL rows | P01 to P04, P07, P08; detectors R1, R4, R5a, R5b, R6, R7, R8 | `one`, `second`; screen, explore, choose, confirm | about 1.5 core-hours |
| **C4b** | Work and welfare | the 6 AGT and 7 WEL rows of C4 | the same, and D01, M03, M05, M06 (its X and O parts: the B part waits for a pack in which a group can start below the reserve and earn, STY tier N), H01 to H04 | `one`, `second`; M06 on `one` + `overlay/poor` and H03 on `one` + `overlay/perturbed`, each a confirm leg of the chosen point and the base; `Dist` for the `d` knob K-AGT05 | about 1.6 |
| **C4c** | Occupation | the 5 SPN rows of C4 | D03, R6 (role shares) | `one` | about 0.6 |
| **C5** | Shocks and the ladder | the 6 SPN, 3 EST and 3 SUB rows of C5 | P05, S01, S02, S06; W05 (its bound `s04.elasticity_max_bp` against its prior, and its R share) | `one`, `famine` (REF's variant already has `overlay/estates`); W05 on `one` and `one` + `overlay/nofloor`, the second a confirm leg; **branch** designs (EXP 6) | about 3.5 |
| **C6** | A larger world, barriers | the DEM, AGT and SKL rows of C6 | W01, M04, W04, D03 with the skill overlay | `one-demand` searched; `three` and `five` run once per choice on their REF invariant seeds (4 and 2) as a confirm only (the "once" of 7.3.7); `Dist` for the `d` knobs K-AGT06 and K-AGT07 | about 1.0 |
| **C7a** | Trade | the INF and MER rows (12 knobs) | P06, S03, T02 | `three` searched, confirmed on `two` and `three` with the trade overlay; `Dist` for `d` knobs | about 3.0 |
| **C7b** | Movement | the MIG rows (10 knobs), the C7a knobs fixed | G01 (umbrella), P06 held | `three` with the trade and migration overlays | about 2.7 |
| **C8** | The open layer | the EXT and MON rows | S04, S05, T01 (X), T03 (R) | `one` + `overlay/external.toml`, `overlay/faucet.toml` | about 0.8 |
| **C9** | The long facts | the POP and RES rows | W02, W03, W06, and the tier L leg of S02 (tier L) | `one-demand` + resources and population overlays; 100 years | about 24.7 |

**Departures from STY 10.3.** Bands are *set* in STY's order (FR-STY-091) whatever the
order of the search, and a stage's result is a *candidate* point whose bands are written
in STY's order when the chain is confirmed (7.2.4). The searches that run **earlier or
elsewhere** than STY's steps are these, and only these:

1. the facts H01 to H04 and D01 (STY step 3) are searched at **C2**, before P02 to P04,
   and read again at C4b;
2. the scale, P09 (STY step 5), is searched at **C3**, before C4, and its B part is read
   against its working prior until step 5 sets the band;
3. STY's step 3 is searched in **three sub-stages C4a to C4c**, each confirmed before the
   next;
4. the tier L leg of S02 (STY step 9) is searched at **C9**, on tier L, beside W02, W03
   and W06;
5. five facts are in **no step of STY 10.3**: M06 (pack `one` with `overlay/poor`; its X
   and O parts are a confirm leg at C4b, its B part is tier N until REF has a pack in which a
   group can start below the reserve and earn), P08 (pack-asserted, `second`) at C4a, S06 at
   C5, G01 at C2 and again at C7b, and W05 at C5 (the bound `s04.elasticity_max_bp` read against
   its prior on `one`, the R share also on `one` + `overlay/nofloor`, the pack of STY's scenario
   S04); H03's pack, `one` + `overlay/perturbed`, is a confirm leg at C4b, as STY's step 3
   has it. Where STY sets the bands of the five is asked of STY (23).

P05, S01 and S02 (STY step 4) are searched at C5 after step 3, as STY has them; the long
facts of step 9 at C9 are STY's own order, and neither is a departure.

The cost column is **a count times the placeholders** of `spec-scale.md` 6, STY
4.5.1, EXP 10 and CHK 10.3 (`cost.py`, 18): one core, no parallelism, a screened-in
set of a few knobs, 50 confirm seeds and a control; the overlay legs (M06 on `one` +
`overlay/poor` and H03 on `one` + `overlay/perturbed` at C4b, W05 on `one` +
`overlay/nofloor` at C5) are one more confirm each, of the chosen point and the base, at
the cost of `one`. The whole sweep of one pass is
7 292 runs and **39.5 core-hours**, C3 included (`cost.py`); the length of a run is
the scenario's own (REF 7.2 and 8.2: 1 095 ticks, three years, for a steady scenario;
1 800 ticks for S14, the shock leg; REF 8.2's 400 ticks are the golden and determinism
length, not a burn-in; C5's branch burns in 400 ticks, a placeholder of this spec), so
a band set at a confirm is asserted by the gate at the length it was set at, and
`one-demand` is 600 households of 6.5 us (REF 4.4.6, 8.2). A stability record of the 49
defaults of the claim set (7.4.2), taken on all seven packs of REF 4.4 (an upper bound:
a pack records only the knobs it sets), on the gate and, where it is run, the
hold-out of REF 8.2, is 13 860 runs and **57 core-hours** more (half that on the gate alone).
`RunUnit` is the unit a consumer fans out (EXP 5.3), so these are an afternoon on a few
cores. They are placeholders and nothing here is measured (N-015).

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
**structure**: after it, changing it changes the lineage of the pack (9.2) and every
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
and roles (`three`, `five`) and the demand and skill modules. `five` is about fourteen
times (a ratio of 13.6 ticks to one) the cost of `one` per tick (`cost.py`), so it is run **once per choice** as a
confirm, not as a search: one confirm of the chosen point and of the base, on REF's invariant seeds
(1 to 4 for `three`, 1 and 2 for `five`, STY 8.2), after the search on the small pack
(`one-demand`) has chosen.

7.3.8. **C7, Trade (C7a) and movement (C7b).** The loops with a delay: shippers, rumours,
contracts and migration (L84 to L95, L107 to L116). The packs are `two` and `three`
with `overlay/trade.toml` (and `overlay/migration.toml`). The **stability limit of a
loop with a delay is a product** (`spec-information.md` 11: the gain against the lag
plus the transit time), so the knobs are searched with the gauges of A07 fixed.
Eight knobs here have the flag `d` (K-MER03 to K-MER06, K-MER08, K-MIG01, K-MIG02 and K-MIG05), and two groups (S-INF01, S-MER01); `MaxOpen` (K-MER01) is not one, since it acts after the draws of its term (4.2.1).

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
`reentry C<n>`. A change to a knob with the flag `in` or `tbl` re-runs C2's
H facts, because the warm table is stale (I19). A change to **any** document of the pack
(an S row at C1 or C2, a unit, or a K or D value the pack itself sets) changes the lineage
(9.2) and **every stage starts again from C1** (C1 is `Check` and runs no tick).

7.3.12. **The fourteen later modules.** The K and D rows of the modules of wave 5 (6.2.9 to
6.2.11: 28 K and 32 D rows) carry the Stage `-`: no stage of this table searches them. A
stage is the one whose facts need a knob, and none of the facts of STY has a band that
these knobs move: the scenarios of the modules are harness tests or reports (STY 5.11, S91 to
S165) and STY-W07 is a report fact. A pack that wants one of them searched adds a stage after
C9 in the same change that extends the closed Stage set of 4.2 and the check of 11.1, which
no row does today. The counts of the table above (the 16 BEL rows of C4a and the others) are
unchanged, and a K or D row at Stage `-` is no factor of a screen until it has a stage. The
S rows of the modules are staged as the others: C1 for identity, topology and lists, C2 for
the quantities of the setting (a room, a rent, a rate of danger, a share, a limit, a mandate)
and the policy values of a control, which are a pack's data.


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
resource and `Rank`; the two schedule lists K-POP01 and K-POP02 (class D, Grid `-`), searched
only through their two multipliers (A15); and `MaxDispatchPerTick`, a bound, are not claims); some of these rows back facts that
the version 1 criteria of STY 8.1.1 do not list, and the claim is the sweep table's, not 8.1.1's; the rest carry no claim and are labelled placeholders). For each claimed knob and for every
shipped pack that sets it: the three checks of STY 7.1 STABLE **at the default and at
each neighbour** of its grid, on the **gate** seeds, with every key those checks read
set by review (STY 8.4), and the **stability record** of STY 7.3.3 is the output.
The gate is inside the setting range (STY 8.2), so a verdict on REF's seeds alone is
**in sample**. Where the hold-out of REF 8.2 (51 to 60) was run, the default and each neighbour
are also run on it and the three checks must hold there; where it was not, the record says
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
**finding**: the stage returns to its screen on a **new block**, and the failure
counts as **one of the three attempts of 7.1.4** (the gate is inside the confirm range, so a
failure of it is a failed confirm on a subset, and a failure on the hold-out is counted the same); no knob is moved "because seed 7 failed", and the seeds that failed are
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
experiment <step> design <oat|spread|grid|none> points <n> replicas <R>
           seed0 <s> seeding <same|per_point> ticks <T> transient <t> every <e>
           salt <s> render <16 hex>          # first 8 bytes of the SHA-256 of Render(Result), EXP 8.2.2
factor <name> key <Scope.Field> grid <geo|linN|set> base <v> lower <v or -> upper <v or ->
           edge <none|below:bound|below:rule|below:off|above:bound|above:rule|above:off>
unsearched <at_off|invisible|v1x> <knob ...>
measure <name> fact <ID> token <Measure or -> class <X|O|B|R|P> kind <fact|detector|counter>
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
          in_sample <yes|no> c_bp <STABLE|NOT_STABLE|UNDETERMINED> c_nc <v> c_ws <v> edge <none|below:bound|below:rule|below:off|above:bound|above:rule|above:off>
claim <knob> pack <name> <stable|not_stable|one_neighbour|undetermined>
unmet <fact> <measure> <values>
decision <stop|escalate|reentry C<n>|review|revert>
note <free text>
```

`design` is EXP's `OneAtATime` (`oat`), `Spread` or `Grid`, or `none` for a factor-free experiment, which is
no design (EXP 4.1.3); the confirm is a `Grid` of one factor of two levels, the choice's overlay and the
base as control. The verdict spellings `STABLE`, `NOT_STABLE` and `UNDETERMINED` are the grammar's
(STY writes NOT STABLE, FR-STY-095: one spelling in a record, never a mixed one).
A `knob` is a registry ID (`K-BEL12`), so a record names a knob by the same token as
the registry and the check of 11.2 resolves it. `document` digests and `render` are the first
8 bytes (16 hex digits) of the SHA-256 of the bytes of the document and of `Render` (EXP 8.2.2)
of the result, the width of EXP's own digests. The `token` of a `measure` line is the IND measure
of the Moves token whose sign the screen compares (a knob with two tokens for one fact, K-BEL12,
has two `measure` lines), so a `finding` maps to one token of the row; `unsearched` lists the
knobs a stage did not search and why: `at_off` (a `z` row at 0, 5.2), `invisible` (10.5) or
`v1x` (no reader in version 1). `lineage` is the first 8 bytes (16 hex digits) of the SHA-256 of the **structure
documents** of the variant (9.2): its documents in layering order, up to but not including
the first `calibrated/` overlay and without the warm-start document. It is the same in every record of one
pack until a document of the pack changes, and a change of an S row at C1 or at C2 (a quantity or a unit
included) or of a value the pack itself sets moves it; so two records of one lineage are comparable and
two of different lineages are not.

### 8.3 What the record must say

1. **The configuration**: the hash of the base of the attempt and of the chosen
   point, each the `ConfigHash` of **a world a run built** from the documents (EXP
   14.3: `Check` computes no hash and CFG exposes none; the hash is `World.ConfigHash`), and the documents
   (with their digests) the hashes came from.
2. **The experiment**: every step, as the plain-data part of an EXP `experiment.Spec`
   (design, points, replicas, `Seed0`, seeding, ticks, transient, windows, salt, the
   factors with their keys and grid, the measures and the criteria) and the digest of
   its rendered `Result`. The function values (the measures, the steps) are named by
   the labels the experiment gave them (the `Name` fields of EXP 3.1 and 3.2).
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

8. **The stability lines** (STY 7.3.3), which are the lines of the **stability record**: a file of
   this grammar that lives where STY puts it (8.4), not in a stage record. For each claimed
   default and shipped pack that sets it, one `stability` line per point (the default and
   each neighbour that exists) with the three verdicts of STY 7.2, the seed set and
   `in_sample`, and a `claim` line (`stable`, `not_stable`, `one_neighbour` or
   `undetermined`).

9. **What was not searched, and why**: an `unsearched` line for the knobs a stage did not search:
   a `z` row at 0 (`at_off`), a knob whose effect range was 0 on every measure (`invisible`,
   10.5) and a field with no reader (`v1x`).

### 8.4 Where records live

For the library's defaults, under `testdata/calibration/<pack>/<stage>-<n>.txt`
beside the golden files (`testing.md` 10.4), changed only by a reviewed change that
says what moved (`testing.md` 7.2). For a setting, wherever the consumer keeps its
pack: the grammar is the library's and the location is not. A **stability record** (STY
7.3.3) is the one other kind: a text file of the lines `calibration`, `pack`, `commit`, `config`,
`document`, `stability`, `claim` and `decision` of this grammar, **beside the verdict record**
where STY puts it (`testdata/golden/stylised/<pack>.stability.txt`, STY 8.5; the file name is
asked of STY, 23): STY owns the home and CLB the grammar, so there is one record and one
location. A stage record is **never** a golden file of the economy: it is data about a
decision, and the golden file of a pack moves only when the pack does.

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

In an **overlay document** of the pack, layered **after** the documents of the variant
(`base`, its region document and its overlays, REF 4.1) and before its warm-start document
(REF counts that one in the variant, and it is regenerated from the whole effective
configuration, so the committed one is replaced), one document for each stage that moved a knob:
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

A pack has a **lineage**: the line of its structure, the digest of the documents of its
variant that are not a calibration overlay and not the warm start (8.2). A change to an S row
(an identity, a topology, a quantity at C2, a unit), or to a K or D value that the pack's own
documents set, changes the digest and so starts a **new lineage**, with the previous one named as
`parent` in its first record; every stage is then run again from C1 (7.3.11). The lineage is
taken from the documents and not at a stage, so a quantity changed at C2 moves it as an identity
changed at C1 does, and no record of the new lineage can carry a band, a table or an overlay
of the old (FR-CLB-054). A change of the library's own defaults does not move it: it moves the
`commit` and the `config base` hash of the record. Within a lineage, a calibration overlay is versioned by its attempt number
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
| **An evidence anchor** (E1, E5; the leads E2 to E4 likewise) | as a sanity range | for another place and time (STY 3.3) |

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
- "the first consumer's data pack runs" (the wording of `vision.md` 5, item 5) is **not** a stability claim: it is
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
facts, the **elasticity** at each neighbour, `rdiv(10 000 * (Y_l * n_0 - Y_0 * n_l) * x_0,
Y_0 * n_l * (x_l - x_0))` from the exact sums of the used readings (EXP 8.5.3), which is `OK`
false for `Y_0` at or below 0, `x_0` 0 or `x_l = x_0`, and which carries the flag `Clamped` at
`+-(2^31 - 1)`; a **clamped** reading is treated as `OK` false by the alias test (10.2); and the
**effect range and share** (EXP 8.5.1). They are the **sensitivity matrix** of the stage, kept
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
128-bit integers **under a bound**: an elasticity must be at most `2^26` in absolute value for the test (EXP 8.5.3 clamps at `2^31 - 1`, a wider range) and the test takes at most `m` = 16 measures at a time (a stage with more is tested by fact group), so that `10 000 * m^2 * 2^104` stays below `2^127`. A reading above the bound, or `Clamped`, is **excluded as `OK` false** (`-` in its `screen` line) and is never wrapped: the arithmetic is checked and an overflow is an error of the function (architecture.md 3.5), which the procedure does not reach because it excluded the reading first. A zero vector is never aliased (it is **invisible**, 10.5). The
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
makes once. The alias of a later module is tested as the earlier ones are: A17 (`Span` and `Move` of PLC) says that the facts see only the gain `Move / Span` up to the truncation, so the two columns of a screen must be collinear (or one dormant), and a calibration searches the gain as a composite factor.

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
its default, and the record says `unsearched invisible` (8.2). Whether a knob's configuration change
moved the hash at all is the `no_effect` check that EXP names as a footnote and does
not specify until CFG offers a hash function (EXP 14.4 specifies `ranged_narrower_than_design`
and `duplicate_point`, and leaves `stale_warm_start` to the list of paths that VIA 11 now
publishes); it would be a stronger statement: the hash did not move.

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

**Where they live** (architecture.md 2.7, 2.9; STY 22 does the same for its checks). The
registry test, the record checks and the unit tests of the pure rules are tests **in the
package `internal/calibration`** itself: the registry test reads `../../doc/specs` by
relative path (testing.md 10.4) and needs no root type. The dry run needs `experiment`,
which nothing in the module may import except its own tests (architecture.md 2.7), so it
is a test in **`experiment`'s external test package** (`experiment/calibration_dry_test.go`,
package `experiment_test`), which imports the root, `experiment` and `internal/calibration`
and no `indicator`: its criteria are on the status of a run, not on a measure.

### 11.1 The registry test (unit, `-short`)

A test in `internal/calibration` reads this spec, the parameter-owner table of
`specs/README.md`, the catalogue of `spec-config.md` 3.7, `spec-indicators.md` 9.1,
`spec-stylised-facts.md`, `spec-stability.md` and the event table of
observability-and-logging.md 4.3 as **text**, with no dependency, as the parameter registry
test of `testing.md` 6.7 does. It builds no `World`, so it is a unit test and runs under
`-short`, though 6.7 sits in the section on integration tests. It asserts (the model of 18,
`registry.py`, implements each check of this table, reads the IND preset names from
`spec-indicators.md` 9.1 and the event names from the rule file, and does not read a field
list, because IND has none):

| Check | Fails when | Requirement |
|-------|------------|-------------|
| rows | a knob row has not 12 cells or a group row not 7 | FR-CLB-001 |
| unique IDs | an ID repeats | FR-CLB-101 |
| coverage, owner to registry | a pair (scope token, parameter) of the owner table, or of its `(also <Scope>` remark (`Role.Upkeep`), is neither a knob row's key nor a field of a group of that scope (4.7, 4.8) | FR-CLB-002 |
| coverage, registry to owner | a key or a group field names no owner row, outside the scopes `Solve option` and `Presentation` (the catalogue-only fields `Taste.Role` and `RegionTaste.*` are owner rows now) | FR-CLB-002 |
| coverage, CFG 3.7 | a field name of the catalogue of `spec-config.md` 3.7 (a field not marked (p)) is neither a registry key nor a group field, or a field listed inside a nested table of the catalogue is under no scope of that table's name | FR-CLB-002 |
| class | a class is not K, D, S or P | FR-CLB-003 |
| `off=` | a D row has no `off=`, or a non-D row has one | FR-CLB-103 |
| flags | a flag is outside the closed set of 4.2.1 | FR-CLB-104 |
| `z` and `geo` | `z` is on a row whose owner default is not 0, none or `the world's`; or a `geo` row has an owner default below 1 and no `z`; or an owner default `required` has a lower bound below 1 | FR-CLB-108 |
| `out` | an `out` flag is on a field that is not `Prices`, `Wages` or `Groups` of the WarmStart section, `Commodity.ReferencePrice` or `RegionPrice.Price` (VIA 8, REF position 4; `Pools` is seeded by the author, VIA 8.3.3) | FR-CLB-109 |
| no restated number | a column named Default or Bound exists, or a cell holds a number outside a `lin` step, an `off=` value or a `set:` level | FR-CLB-005 |
| search | a Search cell is not `U`, `E<n>`, `I<nn>` or `-`; an `E<n>` names no anchor of 6.1 whose Status is `anchor`; an `I<nn>` names no interaction; a K or D row has `-` | FR-CLB-006 |
| grid `-` | a K or D row whose Grid is `-` and whose Stage is not `-` has no `A<nn>` link | FR-CLB-107 |
| moves | a token has not the form of 4.4, names a fact (or an `S<nn>` row of STY 5.11 with its band key) that is not in STY, or a measure that is none of: a preset named in IND 9.1 with a field of the closed list of 4.4 (STY's spelling; IND publishes no field column: owed, 23), a band key written in STY, an event name of the table of observability-and-logging.md 4.3 | FR-CLB-007 |
| at most four | a row has more than four move tokens | FR-CLB-105 |
| references | a `L<nn>` or `D<nn>` is not in `spec-stability.md`, an `I<nn>` or `A<nn>` is not in 6.4 or 6.5, a `D<nn>=<Counter>` names no counter of OBS | FR-CLB-008 |
| owner | an Owner cell does not resolve to a section of the owner's spec, or is not one of the sections of the README's owner column for that pair (the README may write "section" or "sections" and a list of numbers; a README row that parses no section **fails**; a group names each of its sections) | FR-CLB-102 |
| off value, after the cut | with the grid of 5.2 built on the owner's default and cut, a level other than the base equals the off value, or `off=min`/`off=max` is reached by a kept level (read from the owner's bound, never from the row) | FR-CLB-108 |
| solver flags | a data field that a path of the first table of VIA 11 marks as moving the table lacks `in` (a `.*` naming every field of its table), a field it marks `no (a check)` or `no (a report)` lacks `chk`, or a data field carries `in` or `chk` that VIA 11 does not give it; a key and a container are not checked (4.2.1) | FR-CLB-009 |

The `in` and `chk` check is mechanical since VIA 11 publishes the list of the paths the
programme reads: the test maps each path of its first table to a (scope, field) by the
table-prefix rule of EXP 11.3.1 and the snake form, and a `.*` to every field of that scope
and of the scopes nested in it (a want's items, a resource's knots and seasons, an outside
node's gates, lines, sides and season factors). The `out` check is mechanical because the
writer of an `out` field is the solver, which VIA 8 and REF position 4 name.

### 11.2 The record check (unit)

A **parser** of the record of 8.2 is a function that reads external bytes, so it
has a **fuzz test** (`testing.md` 7.4: never panics, `Parse(Render(r))` is `r`, an
unknown keyword is an error). The checks on a record: every keyword is of the closed
set; every `knob` token resolves to a registry ID (and `unsearched v1x` names a row with
the flag `v1x`); every `fact` is a fact of STY; every `set` key names a K or D row
(FR-CLB-050's twin); `feasible n of N` has `n` at most `N`; a `chosen` index is below the
number of points; `screened_in` and `dormant` are disjoint; a `set` names no knob of
`dormant` unless the decision is `revert` (FR-CLB-062); a screen whose elasticity sign
differs from the registry's sign of the token named by its `measure` line has a
`finding` line (FR-CLB-060); the number of `screened_in` knobs of a stage is at most its
`measure` lines unless a `factor` is a composite (FR-CLB-063); and, for a stage that ends
`stop`, every `fact` line of the stage has a `result` of `pass`, `report` or `na`;
a `claim` of `stable` has three `stability` lines (default, lower, upper), every verdict
STABLE and no edge, and a `claim` of `one_neighbour` has two with one edge; a `stability`
line with `set holdout` names the seeds 51 to 60 of REF 8.2 (FR-CLB-039, FR-CLB-100). **Across
records** (8.5): a comparison of two records with different `lineage` is refused
(FR-CLB-054), and a record whose `lineage` differs from its predecessor's names that
predecessor's lineage as `parent`, and no record of the new lineage carries a `set`, a table digest or an overlay
digest of the old (FR-CLB-053); a record's `block` was not
used by an earlier record's choice, no search line names a seed of 1 to 60 and no
confirm line a search seed (FR-CLB-020, FR-CLB-021); a stage has no fourth attempt before
a `review` decision (FR-CLB-027); a `set` line moves no D knob to its `off=` value
without a `note` (FR-CLB-071). Requirements: FR-CLB-040, FR-CLB-041, FR-CLB-042,
FR-CLB-043, FR-CLB-039, FR-CLB-100, FR-CLB-053, FR-CLB-054, FR-CLB-060, FR-CLB-062,
FR-CLB-063, FR-CLB-071.

### 11.3 The pure rules (unit, `internal/calibration`)

Table-driven (`testing.md` 3.1), each with the arithmetic of its expectation
(`testing.md` 5.5):

| Test | Asserts | Requirement |
|------|---------|-------------|
| the grid | `geo`, `lin` and `set` levels for a table of (base, bound) including a base at a bound, a base of 1, a base of 0 (`lin` only; a `geo` row at 0 has no grid and is `unsearched at_off`), a half-to-even tie, and the **cut**: `RestBand` (base 500, `lin250`, off 0) loses the level 0 and has `edge:off`, `TargetSpread` (base 0, `lin500`, off 0) keeps its base and has `edge:bound`, a `geo` level that overflows is an error and not a wrapped level; the base is always a level; levels are ascending, unique and in bounds | FR-CLB-010, FR-CLB-011, FR-CLB-013, FR-CLB-014, FR-CLB-108 |
| the neighbours | the nearest kept valid level on each side; an invalid level is skipped; a side with none is an edge of the right kind (`bound`, `off`, `rule`) | FR-CLB-012 |
| the plan | the screen is the base and two neighbours of each knob with at least 10 replicas by default; the explore has `max(16, 10 k)` points on the block of the screen; an interval whose feasible points touch a non-edge end is widened by two kept levels on a new block, and never leaves `[lo, hi]` | FR-CLB-022, FR-CLB-023, FR-CLB-026 |
| the seed blocks | block `b` is `100 b + 1` to `100 b + 100`, disjoint from 1 to 60 for `b` at least 1; a block is not reused by a second choice in a record; the confirm is 1 to 50 (tier L 1 to 10) and a choice's per-seed values are not an input to the next | FR-CLB-020, FR-CLB-021 |
| the confirm | the confirm plan is the chosen configuration and the base as a control, on the confirm seeds with the measures and windows of the search | FR-CLB-025 |
| the choice | the feasible set; the base chosen when feasible; the least grid distance; the tie order; no choice when empty; the Pareto set; **no weighted sum is computed**; the seed threshold `R - floor(R / 10)` | FR-CLB-030, FR-CLB-031, FR-CLB-032, FR-CLB-033, FR-CLB-034, FR-CLB-106 |
| the alias test | collinear vectors, independent vectors, one measure, a zero vector, the 128-bit comparison at its largest inputs, a reading above `2^26` excluded | FR-CLB-061 |
| the record | render, parse, render again; the order of lines; no float and no map order; the `stability` and `claim` lines; the digest of a document and of a `Render` text; the same bytes on `amd64` and `arm64` in the nightly (testing.md 7.1) | FR-CLB-041, FR-CLB-043, FR-CLB-039, FR-CLB-100, NFR-CLB-002 |
| the process rules | a pure check over a sequence of records: a stage order with the re-entry rule, a stage of more than 16 knobs split by owner (`MaxFactors`), three failed attempts, a stage with an open X failure, a seven-condition stop | FR-CLB-027, FR-CLB-035, FR-CLB-036, FR-CLB-038 |
| the overlay | a choice renders to a document of the form of EXP 3.3.4 (a table header and a key line each, the keys grouped by table, under the header comment of 9.1), each key line byte for byte as the function of `internal/design` writes it; a key of class S or P is refused | FR-CLB-050 |
| the fuzz tests | the grid and neighbour function (any base, bound and step: levels ascending, unique, in bounds, containing the base, no panic), the choose function (any result set: the chosen point is feasible, the Pareto set is non-dominated, no panic) and the record parser | NFR-CLB-004 |

### 11.4 The dry run (integration, `-short` skip)

The procedure, run end to end on the reference chain with a **tiny** experiment,
to show it is a procedure and not a prose (`testing.md` 6.4 is its model). It is an
integration test in `experiment`'s external test package, begins with the `testing.Short()`
skip, and **asserts no economy number**. Its seeds (101 to 103 and 1 to 3), its three
replicas and its 400 ticks are outside REF 8.2, so they are **declared test constants**
and the change that adds the test lists them in its description, as FR-REF-025 asks.

- **Setup (leg A).** Pack `one`, 400 ticks, transient 100, windows of 100; a screen of
  `OneAtATime` over two knobs (`World.CostFloor`, `geo`; `World.BeliefEvidenceShare`,
  `lin1000`) at their base and neighbours, so five points; three replicas on **search
  block 1** (seeds 101 to 103); a confirm on seeds 1 to 3 (a prefix of the confirm range) with a control; criteria
  that every point meets (a run's status `ok`), and a second set that no point
  meets.
- **Asserts (leg A).** (1) the plan is the base and the two neighbours of each knob, and
  `Check` resolves them with no tick run (FR-CLB-012; the dry run sets three replicas as a declared test constant, and the default of 10 is asserted by the plan test of 11.3, FR-CLB-022); (2) the choice is
  a function of the stored result: running `choose` again on it gives the same index
  (FR-CLB-034); (3) with criteria every point meets, the **base is chosen** and the
  overlay is **empty** (FR-CLB-031); (4) with criteria none meets, the result is
  `chosen none` and a record whose `decision` is not `stop` (FR-CLB-033, FR-CLB-038);
  (5) the seeds used by each step are the ones of 7.1: the screen on block 1, the
  confirm on the confirm range, and no search seed lies in 1 to 60 (FR-CLB-020, FR-CLB-021); (6) the whole
  procedure run twice gives the same record bytes (FR-CLB-041); (7) the `render` digest of
  each step of the record is the digest of `Render` of the stored result, so a rerun
  reproduces it (FR-CLB-040); (8) the confirm's control is the base on the same seeds,
  measures and windows as the chosen point (FR-CLB-025).
- **Leg B (the rest of the procedure).** Same pack and length, driver runs with
  these stimuli: (a) a knob with the flag `d` (`World.EntryShare`, K-SPN02) is compared by
  `Dist` and `PerPointSeeds`, not paired (FR-CLB-024); (b) a pack quantity with the flag `in`
  (`Role.ExpectedWorkTicks`, K-WEL06, a `Grid` of three values as C2 uses, 7.3.3) regenerates the
  table per point with `Prepare` (the solver), the record names the digest of the table and the table is
  generated last (FR-CLB-037, FR-CLB-052, CLB-A4); (c) `World.BeliefStep` (K-BEL01) with the money unit
  of 19.1 has an invalid lower level (I01), which is **skipped and recorded, the run going on**,
  and its edge is `edge:rule` (FR-CLB-012, FR-CLB-108, FR-EXP-050); (d) a third criteria set on
  which the base fails and one neighbour passes gives that neighbour, with the overlay of
  one key that parses and validates, and the world built from it has the `ConfigHash` the
  record names as `config chosen` (FR-CLB-032, FR-CLB-050, FR-CLB-051); (e) an explore
  whose feasible points touch its end is widened by two levels on a new block
  (FR-CLB-023, FR-CLB-026); (f) three failed attempts in a row stop the stage with no
  fourth attempt before a `review` (FR-CLB-027); (g) a two-stage order in which the
  second stage moves the first one's knob needs the `reentry` (FR-CLB-035, FR-CLB-036);
  (h) a stage with a failing X fact searches nothing (FR-CLB-036); (i) the stop needs all
  seven conditions (FR-CLB-038).
- **Cost.** Leg A: 5 points of three seeds of 400 ticks and a confirm of two points of
  three: at the placeholder of 4 ms a tick about 35 seconds. Leg B: three driver runs of
  the same size, about two minutes, and the process rules (f) to (i) are pure checks on
  stored results and run no tick. A placeholder.

### 11.5 The overlay purity test (integration)

A test over every `calibrated/*.toml` of `testdata/chains/` (when any exists)
asserts that each key of each document is a K or D row of the registry (FR-CLB-050)
and that each document begins with the header comment of 9.1 (FR-CLB-051). It reads
files and builds no world, and runs in the package `internal/calibration` beside the
registry test.

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
| FR-CLB-001, 002, 003, 005, 006, 007, 008, 101, 102, 103, 104, 105, 107, 108, 109 | the registry test (11.1) |
| FR-CLB-004 | for a `Config` field the coverage check of 11.1 (FR-CLB-002) fails the change that adds it without its row; for a solve option (the scope `Solve option` is exempt from coverage) it is a review item |
| FR-CLB-009 | the registry test (11.1) |
| FR-CLB-010, 011, 013, 014 | the grid test (11.3) |
| FR-CLB-012 | the neighbours test (11.3); the dry run, leg B (c) |
| FR-CLB-020, 021 | the seed blocks test (11.3); the record check "blocks" (11.2, 14); the dry run, leg A assert 5 |
| FR-CLB-022, 023, 026 | the plan test (11.3); the dry run, leg A assert 1 and leg B (e) |
| FR-CLB-024 | the dry run, leg B (a) |
| FR-CLB-025 | the confirm test (11.3); the dry run, leg A assert 8 |
| FR-CLB-027 | the process rules test (11.3); the record check "attempts" (11.2); the dry run, leg B (f) |
| FR-CLB-035, 036, 038 | the process rules test (11.3); the dry run, leg B (g) to (i) |
| FR-CLB-030, 031, 032, 033, 034, 106 | the choice test (11.3); the dry run, leg A asserts 2 to 4 and leg B (d) |
| FR-CLB-037, 052 | the dry run, leg B (b) |
| FR-CLB-039, 100 | the record test (11.3) and the record check (11.2) |
| FR-CLB-040 | the dry run, leg A assert 7 (the digest of each step's `Render` text is reproduced) |
| FR-CLB-041, 042, 043 | the record test (11.3), the fuzz test of the parser (11.2), the dry run leg A assert 6 |
| FR-CLB-050, 051 | the overlay test (11.3), the purity test (11.5), the dry run leg B (d) |
| FR-CLB-053, 054 | the record check across records (11.2) |
| FR-CLB-060 | the record check: a `finding` line for a sign that differs (11.2) |
| FR-CLB-061 | the alias test (11.3) |
| FR-CLB-062, 063 | the record check: a `set` line of a dormant knob, a stage with fewer measures than knobs (11.2) |
| FR-CLB-070, 072, 073, 074, 080, 081 | review items (process rules of 3.5 and 7.5); the record check lists the `decision` and `unmet` lines |
| FR-CLB-071 | the record check "no off value" (11.2, 14) |
| FR-CLB-090 | the `-short` marker of the tests of 11.1 to 11.3 (review) |
| FR-CLB-091 | the dry run is skipped with `-short` (11.4) |
| NFR-CLB-001 | the registry test imports only the standard library (lint) |
| NFR-CLB-002 | the record test on two architectures in the nightly (11.3) |
| NFR-CLB-003 | `go vet` and the import check of architecture.md 2.9 (lint) |
| NFR-CLB-004 | the fuzz tests of the parser, the grid and the choose function (11.3) |
| NFR-CLB-005 | the coverage run of `testing.md` 9.1 |

## 12. What CLB hands to the first code change, and what the specs owe

### 12.1 What the first code change must create

The first change that adds the code of the experiment harness and the stylised
tests creates, in the same change:

1. **`internal/calibration/`**: the pure rules of 11.3 over primitive slices
   (standard library and `internal/design` only; it imports **none** of the root,
   `config/`, `strategy/`, `indicator/` or `experiment/`, architecture.md 2.9): the grid
   and neighbours, the seed blocks, the choose and Pareto functions, the alias test, the
   record render and parse and the overlay renderer, which calls the one function of
   `internal/design` that writes an overlay document (EXP 3.3.4) and adds the header
   comment and the grouping by table. It holds **no state, no float, no clock and no
   goroutine** (NFR-CLB-003).
2. **The registry test** of 11.1 and the record checks of 11.2, in the package
   `internal/calibration`, reading this spec, `specs/README.md` and `spec-config.md` by
   relative path (`../../doc/specs`, testing.md 10.4).
3. **`testdata/calibration/`**, with a `README.md` of the record's grammar and the
   fixture of the dry run (11.4).
4. **The dry run** (11.4), in `experiment`'s external test package, and the overlay
   purity test (11.5).
5. **The layout lines**: `internal/calibration/` in architecture.md 2.6 (which lists
   `internal/design` with its "overlay text" and `internal/verdict` already) and in
   `AGENTS.md`, and the terms of the glossary of section 23 in `doc.go`.

It creates **no public package** and **no exported identifier**. The coverage
threshold is the ordinary 90% of each package under `internal/` (`testing.md` 9.1).

### 12.2 What the other specs owe

A summary, by file; the exact change of each row is in section 23.

| To | What | Why |
|----|------|-----|
| `specs/README.md` | a sentence under "Parameter owners" that the **class, the kind of range, the expected effects and the interactions** of each row are in `spec-calibration.md` section 6, found by (scope token, parameter); the rows `Taste.Role` and `RegionTaste` (`Region`, `Want`, `Scale`) that `spec-config.md` 3.7 lists and the table lacks | the "ranges" of the roadmap row; a column would restate the owners' bounds and drift (section 4.5); the registry test reads CFG 3.7 too |
| `spec-config.md` | 9.2: the parameter registry has a second reader, the registry test of CLB 11.1; 9.3 gap list: "the solve options are rows of CLB's registry"; 6.2: the experiment path of a key is the snake-case file spelling of `<Scope>.<Field>` | FR-CFG-047; N-102 |
| `spec-viability.md` | 11: the sentence "rows in the parameter-owner table" becomes "rows of the registry of `spec-calibration.md`" (6.2.8); and two clarifications of **the list of the configuration paths the programme reads**, which VIA 11 now publishes and the registry follows (4.2.1): whether `world.belief_step` moves the table (met: 6.4 raises a seed price to the floor, and the list says `yes (a price at the floor only)`, so K-BEL01 is `in`) and whether a `.*` row names every field of its table | EXP open question 7 is answered by the list; `stale_warm_start` is EXP's to specify (EXP 14.4) |
| `spec-stylised-facts.md` | 7.3.1: an edge clause; 7.3.3: the file name of the stability record; 10.3: where M06, P08, S06, G01 and W05 are set (8.2, 10.2 and the working thresholds are met) | the edge default has one neighbour; one stability record and one home |
| `spec-stability.md` | 9 (the checklist): a step after 9.9 "**classify each new tunable and add its registry row** (`spec-calibration.md` 4.3)"; L142 and D97, whose wording ("the held-out check") the gate of STY 8.2 contradicts; the entries of 17 are merged (L139 to L142, D93, D95 to D98, S88 to S90) | FR-CLB-004 |
| `spec-indicators.md` | 9.1: a **field column**, a closed list of the result fields of each preset (without `UpkeepShortfall`, which is an OBS event) | the registry test of 11.1 |
| `spec-scale.md` | 5.2 and the owner table: a validation bound for `MinAgents` | the grid of K-SCL02 is unbounded above |
| `spec-observability.md` | 4.4.6: `ProductionHalted` is the counter of D24 (named by K-AGT09 and 15.2) | N-067 |
| `documentation_index.md`, `specs/models/README.md` | rows for the spec and `specs/models/clb/` (the roadmap already says CLB `exists`) | the index lists every file under `doc/` |
| `notes.md` | N-015's text; the new notes of 20 | |
| `doc.go`, `doc/rules/architecture.md` 2.6, `doc/rules/testing.md` 6.7, `AGENTS.md` | the glossary terms of 23; the layout line of `internal/calibration/`; the one-sentence registry-test row | the rows of 23 that FR-CLB-004 and the registry test need |

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
| explore size | `max(16, 10 k)` | 7.2.2 | a Latin hypercube of ten points a factor is the usual first design; 16 is the floor |
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
or, for the overlay, as a check of files:

| Check | Fails when | Requirement |
|-------|------------|-------------|
| registry | any failure of 11.1 | FR-CLB-001, FR-CLB-002, FR-CLB-003, FR-CLB-005, FR-CLB-006, FR-CLB-007, FR-CLB-008, FR-CLB-101, FR-CLB-102, FR-CLB-103, FR-CLB-104, FR-CLB-105, FR-CLB-107, FR-CLB-108, FR-CLB-109 |
| record | any failure of 11.2 | FR-CLB-040, FR-CLB-041, FR-CLB-042, FR-CLB-043, FR-CLB-039, FR-CLB-100, FR-CLB-053, FR-CLB-054, FR-CLB-060, FR-CLB-062, FR-CLB-063 |
| overlay | a key of a `calibrated/` document is not a K or D row, or the header comment is missing | FR-CLB-050, FR-CLB-051 |
| blocks | a record names a search block used by an earlier record's choice, or a seed of 1 to 60 in a search line, or a search seed in a confirm line | FR-CLB-020, FR-CLB-021 |
| attempts | a stage has a fourth attempt with no `review` decision before it (an attempt is a failed confirm, gate or hold-out) | FR-CLB-027 |
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
`BeliefsUpdated`, `MoneyFlowEnded`, ...). A damper whose catalogue entry
names no counter cannot be shown dormant, and the record says so (10.4). The production
halt has one: `ProductionHalted` (OBS 4.4.6, N-067), which is the `D24=ProductionHalted` of K-AGT09.

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
  is `stop.py` (an 85% point passes a 10-seed search among 16 points with probability
  9 999.97 bp and a 50-seed confirm 22% of the time).
- **CLB-A3. A damper disabled to pass a fact.** *Kind:* hazard of method. *Loop:* a
  failing fact (+) a damper moved to off (-) the failing fact, which hides the loop and
  leaves it undamped. *Owner:* CLB 4.3.1, 7.4.1. *Damper:* CLB-D3 (the off value is
  never a calibration move) and the dormant rule (a damper that never acted is not
  moved). *Undamped:* a pack that chooses the off value as structure and says so.
  *Test:* the check "no off value" of 14.
- **CLB-A4. A stale warm start across a calibration.** *Kind:* hazard (EXP-A5 for the
  procedure). *Loop:* none: a calibration moves an `in` or `tbl` knob and the table was solved
  for the old value, so the points start away from their own equilibrium. *Owner:* CLB
  4.2.1, 7.3.11, 9.1. *Damper:* the flag `in`, the re-entry rule and the warm start
  generated last. *Test:* the dry run, leg B (b): the table is regenerated per point and the
  record names its digest.
- **CLB-A5. An alias read as two knobs.** *Kind:* hazard. *Loop:* none: two knobs
  that the facts see only through a product are both "calibrated", and the values
  found are one equation in two unknowns. *Owner:* CLB 6.5, 10. *Damper:* the alias
  groups, the alias test, the composite factor. *Undamped:* an alias no spec has
  named, found only by the test. *Test:* CLB-S3.
- **CLB-A6. The gate as a training set.** *Kind:* hazard of method. *Loop:* a gate
  verdict (+) a knob moved because of it (+) the next gate verdict, which makes the
  gate (inside the setting range, so never out of sample: STY 8.2) a fitted check. *Owner:* CLB 7.1, 7.4.4. *Damper:* CLB-D4 (a gate
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

Seven scripts, standard library only, archived in `doc/specs/models/clb/` with a
`README.md`, run only in Docker (`python:3.12-slim`, `PYTHONDONTWRITEBYTECODE=1`, never
the host interpreter; the slips are in the handoff). They are models of the **rules of
this spec**, run on synthetic or invented values; nothing in them is a run of the
library, a default or a band (N-027).

| Script | Output | What it does |
|--------|--------|--------------|
| `registry.py` | `registry.out` | parses the registry of section 6 as plain text and runs the checks of 11.1 against the owner table of `specs/README.md`, the catalogue of `spec-config.md` 3.7, the facts of STY, the register of STB, the presets of IND and the event table of the rules: cells, closed sets, references, owners, `z` and `geo` against the owner's default and bound, the off number and the cut, `out`, the `in` and `chk` flags against the first table of VIA 11, and **coverage of all 549 owner pairs (the 548 rows of the table and the `(also Role` remark of `Upkeep`) both ways and of the fields of the catalogue** |
| `refs.py` | `refs.out` | the spec's own mechanical checks: requirement IDs defined and referenced, section references, files named, no shell-quoting residue, no setting words |
| `grid.py` | `grid.out` | the grid rule of 5.2 in integers: half to even, `geo`, `lin`, the cut, the skip of an invalid level, the edge record, a `z` row at 0 and the overflow error; 20 000 random (base, bound) cases check the levels are ascending, unique, in bounds and contain the base; the table of 19.1 |
| `stop.py` | `stop.out` | the exact binomial arithmetic of the stop rules: SR2 at 10 and 50 seeds, the joint pass of 9, 27, 45 and 63 conjuncts, the winner's curse of a search among 16 and 64 points, attempts |
| `cost.py` | `cost.out` | the run counts and core-hours of the stages (7.3) and of a stability record, from the placeholders of `spec-scale.md` 6, EXP 10 and CHK 10.3 |
| `choose.py` | `choose.out` | the feasible set, the minimal change, the Pareto set, the integer alias test with its exclusion of a reading above `2^26` and the Spread tie of 7.2.3 |
| `worked.py` | `worked.out` | the numbers of 19.1 to 19.4, 19.7 and 19.8 that are computed, asserted (imports the others); 19.5 is a format example of invented values and is not asserted |

What they do **not** show: that any default is stable, that any prior is right,
that a knob has the sign a row expects, that the placeholders of 13 are good ones, or
anything about the library, which has no code. They show that the rules are well
formed, that the registry is complete against the owner table and the catalogue as
they are today, and what the seed arithmetic says about the stop rules.

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

`World.RestBand`, base 500, `lin250`, off value 0, legal range 0 to 10 000: the uncut levels
are 0, 250, **500**, 750, 1 000; the cut removes 0, so the kept levels are 250, **500**, 750,
1 000, the neighbours are **250 and 750**, and below 250 there is an `edge:off` (the explore
covers 250 to 1 000, two kept levels above and one below). `Role.TargetSpread`, base 0 (its
default is its off value), `lin500`, legal range 0 to 9 999: the levels are **0**, 500, 1 000, ...;
the base is never cut, the one neighbour is **500**, and below 0 is an `edge:bound`. A
`MoneyFlow.Limit` (`geo`, `z`) with the pack's value 8 000: 2 000, 4 000, **8 000**, 16 000, 32 000;
with the pack's value 0 it has **no grid** and the record says `unsearched at_off`.

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

(Invented values; the format is 8.2. Two knobs are screened in, so the explore has `max(16, 10 * 2)` = 20 points (FR-CLB-023), and the screen shows the one factor of the example for brevity. The `claim` is `undetermined`: the third check waits for a key that is unset, STY 7.2, so no default is called stable; the line of the upper neighbour is left out for brevity, and `K-MON01` is `unsearched at_off` because the pack gives it no non-zero value.)

```
calibration 1
pack one  lineage 5a1c0e4477d3b210  stage C4a  attempt 1  block 1
commit 4f2a9c1  parent -
config base 91d3a7c2e05b4a18  chosen 0be4417f6a2d93c5
document base.toml 3b9c11aa5e07f4d2
document one.toml 77e0d2c9b3a1608e
document warm-one.toml 0be4c0319d7a52f6
experiment screen design oat points 33 replicas 10 seed0 101 seeding same ticks 1095 transient 365 every 365 salt 0 render c7f20b19a4e83d60
experiment explore design spread points 20 replicas 10 seed0 101 seeding same ticks 1095 transient 365 every 365 salt 0 render 4e9d02b6f18c7a35
experiment confirm design grid points 2 replicas 50 seed0 1 seeding same ticks 1095 transient 365 every 365 salt 0 render a07715fe62bd9c04
factor cost_floor key World.CostFloor grid geo base 7000 lower 3500 upper 14000 edge none
screen K-BEL12 measure pinned_share_bp el_lower 10000 el_upper 15000 range 1200 share_bp 7500
screened_in K-BEL07 K-BEL12
dormant K-BEL14 K-BEL15
feasible 3 of 20
chosen 1 distance 1
set World.CostFloor 3500
confirm seeds 1..50 control yes
fact STY-P02 class B key s16.pinned_share_bp target report values min 40 median 130 max 410 outside 23 result report
fact STY-P04 class B key s01.offset_bp target working values min 310 median 780 max 1210 outside - result pass
detector R5a none working yes onset -
unsearched at_off K-MON01
decision stop
note CostFloor halved; the base missed P02 on 3 of 10 search seeds
```

The **stability record** of the same default is a file of its own beside the verdict record (8.4),
with the header lines and the two kinds of line of STY 7.3.3:

```
calibration 1
pack one  lineage 5a1c0e4477d3b210  stage C4a  attempt 1  block 1
commit 4f2a9c1  parent -
config base 91d3a7c2e05b4a18  chosen same
document base.toml 3b9c11aa5e07f4d2
document one.toml 77e0d2c9b3a1608e
document warm-one.toml 0be4c0319d7a52f6
stability K-BEL12 pack one point default value 7000 set gate seeds 1..10 in_sample yes c_bp STABLE c_nc STABLE c_ws UNDETERMINED edge none
stability K-BEL12 pack one point lower value 3500 set gate seeds 1..10 in_sample yes c_bp STABLE c_nc STABLE c_ws UNDETERMINED edge none
claim K-BEL12 pack one undetermined
decision review
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
(4.5 s each at the placeholders), plus 200 confirm runs (50 seeds, the chosen point
and the base, on `one` and `second`): **about 1.5 core-hours**. The whole of the
stages in one pass is 7 292 runs, 39.5 core-hours; the stability record of 49 defaults
on the gate and the hold-out, if it is run, is 13 860 runs, 57 core-hours (half on the gate alone).

## 20. Notes settled

Read: every open note tagged CLB (N-015, N-124, N-134, N-135) and every `ALL` note (N-016,
N-017, N-027, N-103, N-126, N-128), and the notes that name what a calibration relies on
(N-018, N-028, N-061, N-084, N-085, N-086, N-092, N-102, N-113).

| Note | Disposition | Where |
|------|-------------|-------|
| N-015 placeholders to calibrate once something has run | **Left open**, updated: the registry gives every placeholder a class and says that nearly every search range is unknown (5.1); "calibrated" now means "a record names the value and the runs it rests on" (8); the new placeholders are the procedure constants of 13 | 5, 8, 13 |
| N-016 keep the parameter registry true | **Applied in part**: the registry classifies each owner row, the solve options of VIA 11 are rows of **this** registry (not of the owner table, which resolves a disagreement between VIA 11 and CFG 9.2 in CFG's favour), and a spec that adds a field adds its row (FR-CLB-004). The registry tests of `testing.md` 6.7 and CLB 11.1 are still to be written | 4, 6.2.8, 11.1 |
| N-124 a paired comparison is exact only while aligned | **Applied**: a knob with the flag `d` is compared as distributions and a paired comparison is used only for a knob that moves no count; the alignment horizon is recorded when CHK offers the read (7.2.3, FR-CLB-024). The note stays open for EXP and WLD | 4.2.1, 7.2.3 |
| N-134 a sweep that moves the steady state needs a regenerated warm start | **Applied in part**: VIA 11 now publishes the list of paths the programme reads and the flags `in` and `chk` follow it (4.2.1, FR-CLB-009); a point that changes an `in` or `tbl` knob regenerates the table or says it is a perturbation (FR-CLB-037). `stale_warm_start` is EXP's to specify (EXP 14.4): the note stays open for EXP | 4.2.1, 7.3.11 |
| N-135 no hold-out set is defined | **Read, left open**: a stability claim says `in sample` unless REF 8.2's recommended seeds 51 to 60 were run (7.4.2, FR-CLB-100); no choice and no band is made from them | 7.1, 7.4.2 |
| N-126 a new event or `Warn` adds a row | **Read, not applicable**: this spec adds no event and no `Warn` (15) | 15 |
| N-128 a hosted per-agent field is saved once | **Read, not applicable**: this spec adds no state to a checkpoint (16) | 16 |
| N-017 words and names | **Applied**: setting-agnostic names; the invented examples are flagged; the terms of section 23 go to the glossary | 19, 23 |
| N-027 no default macro constant is stable yet | **Applied**: no row holds a value; "stable" is the output of STY 7.3.1 and a record (7.4.2); the facts for which nothing is known say `?` | 4.5, 7.4 |
| N-103 running spec writers in parallel | **Applied**: only this spec, `models/clb/` and `.supervisor/clb.*` were created; handoff; models in Docker. **Slip:** the host `python3 --version` was run once, by reflex, after the instruction not to; nothing else ran on the host | 18 |
| N-084 scenarios that wait for trade, money and the coarse tick | **Read, left open**: the trade, faucet and external overlays are the packs of C7 and C8 (7.3.8, 7.3.9) and exist as REF and the owners say; the dynamic parts of X1 and X2 and the coarse-tick leg of S12 are STY's; C7a searches P06 and T02, whose dynamic parts need the trade overlay (STY 20), so until it exists they are **reported** with their observed values and are no target | 7.3.8, 7.3.9 |
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
2. *VIA, EXP*: the list of configuration paths the steady-state programme reads is
   published by VIA 11 and is the `in` and `chk` flags of the registry (met, 4.2.1);
   EXP's `stale_warm_start` is EXP's to specify (EXP open question 7, 14.4).
3. *STY, EXP, IND*: the regime detectors are `internal/verdict`; a consumer's
   calibration needs them (open question 1 below), and a promotion to a public package
   is to be decided with the first consumer's own calibration.
4. *STY, REF*: the **search blocks** (101 and up, one choice a block) are a role of seeds
   that STY 8.2 lists since the integration, and the **hold-out** 51 to 60 that STY 8.2 recommends is a
   SHOULD of REF 8.2 (no band and no choice is set from it; FR-REF-024 stays as written); CLB
   does not rely on it: a claim not checked on it says `in sample` (7.4.2). The working use of a
   prior as a threshold (3.3 item 2) is outside STY's `unknown` rule (6.5.1); STY 6.5 now says
   it is a labelled working threshold outside acceptance.
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
| FR-CLB-002 | Every (scope token, parameter) pair of the parameter-owner table (and of its `(also <Scope>` remarks) MUST be the key of exactly one knob row or a field of exactly one structure group of that scope, every field of the catalogue of `spec-config.md` 3.7 that is not presentation-only MUST be a key or a group field, and every key of the registry MUST name an owner row, except the scopes `Solve option` and `Presentation` (`RegionTaste` and `Taste.Role`, which only the catalogue listed, are owner rows of the README since the integration of the fourteen later modules). |
| FR-CLB-003 | Every row MUST have one class of K, D, S or P, decided by the order of 4.3. |
| FR-CLB-103 | A D row MUST give its `off=` value and no other row MAY. |
| FR-CLB-104 | A flag MUST be one of the closed set of 4.2.1. |
| FR-CLB-004 | A change that adds a `Config` field, or a solve option, MUST add its registry row in the same change. |
| FR-CLB-005 | A registry row MUST NOT hold a default, a validation bound or a calibrated value, and the registry MUST NOT have a column named Default or Bound. |
| FR-CLB-006 | A Search cell MUST be `U`, `E<n>`, `I<nn>` or `-`; an `E<n>` MUST name an anchor of 6.1 whose source is read directly; a K or D row MUST NOT be `-`. |
| FR-CLB-007 | A move token MUST have the form of 4.4, name a fact of STY, a measure of an IND preset named in IND 9.1 with a field of the closed list of 4.4 or a band key of STY, a sign and, for a sign other than `?`, a source. |
| FR-CLB-105 | A row MUST hold at most four move tokens. |
| FR-CLB-008 | Every `L<nn>` and `D<nn>` of a row MUST be an entry of `spec-stability.md`, and every `I<nn>` and `A<nn>` MUST be a row of 6.4 or 6.5. |
| FR-CLB-009 | The flag `in` MUST be set on every data field that a path of the first table of `spec-viability.md` 11 marks as moving the table (a `.*` naming every field of its table) and on no other data field, and the flag `chk` on every field that table marks `no (a check)` or `no (a report)` and on no other; a key and a container are not checked. |
| FR-CLB-109 | The flag `out` MUST be set only on a field the solver writes: `Prices`, `Wages` or `Groups` of the WarmStart section, `Commodity.ReferencePrice` or `RegionPrice.Price` (not `Pools`, which the table may seed and the solver does not write, VIA 8.3.3). |
| FR-CLB-010 | The levels of a `geo` grid MUST be `b * 2^k` for `k` at least 0 and `rdiv(b, 2^-k)` for `k` below 0, kept in `[max(lo, 1), hi]`, with `b` always a level, sorted and without duplicates. |
| FR-CLB-011 | The levels of a `lin<s>` grid MUST be `b + k * s` in `[lo, hi]`, and those of a `set` grid the listed values in order. |
| FR-CLB-012 | A level that `Validate` rejects MUST be skipped; the neighbours of the base MUST be the nearest kept valid level on each side; a side with none MUST be an edge, recorded as `edge:bound`, `edge:off` or `edge:rule`. |
| FR-CLB-013 | Every division of the grid, the choice and the alias test MUST be integer, rounded half to even, with no floating point (architecture.md 3.5), with two stated exceptions: the seed threshold `ceil(9 R / 10)` is exact integer arithmetic `R - floor(R / 10)` (for R of 25 it is 23, and half to even would give 22), and the mapping of a Spread value to a grid level breaks a tie to the **lower** level (a rule of its own, not a division). The checked multiply of the grid reports overflow (an error, never a wrapped level). |
| FR-CLB-014 | The base MUST be the value in the effective configuration the stage starts from (the pack's value, else the owner's default, and the pack's value where the owner has none), and MUST be recorded. |
| FR-CLB-020 | A search block `b` MUST be the seeds `100 b + 1` to `100 b + 100` for `b` at least 1, MUST be disjoint from the seeds 1 to 60, and MUST serve one choice. |
| FR-CLB-021 | A choice MUST be made on search blocks only; the confirm MUST be run on STY's setting seeds (1 to 50; tier L: 1 to 10); the per-seed values of a confirm, and the verdict of the gate, MUST NOT be read into a later choice. |
| FR-CLB-022 | The screen MUST be a one-at-a-time experiment over the base and the two neighbours of each knob, with at least 10 replicas on a search block. |
| FR-CLB-023 | The explore MUST be a spread over the screened-in knobs, with `max(16, 10 k)` points for `k` screened-in knobs on the block of the screen, an aliased knob entering as a composite. |
| FR-CLB-024 | A knob with the flag `d` MUST be compared across points as distributions, and a paired comparison MUST be used only for a knob that moves no count. |
| FR-CLB-025 | The confirm MUST run the chosen configuration and the base as a control, on the confirm seeds, with the measures and windows of the search. |
| FR-CLB-026 | If the feasible points touch an end of the explore interval that is not an edge (5.2: a bound, an off value or an invalid level), the interval MUST be widened by two levels and the choice made again on a new block; the search MUST NOT leave `[lo, hi]`. |
| FR-CLB-027 | After three failed attempts at one stage (a failed confirm, or a failure of the gate or of the hold-out) the stage MUST NOT be searched again before a review. |
| FR-CLB-030 | The feasible set MUST be the points on which every X, O and detector criterion passes on all seeds. |
| FR-CLB-106 | A feasible point MUST also pass every B criterion with a target on at least `R - floor(R / 10)` of the `R` seeds. |
| FR-CLB-107 | A K or D row whose Grid is `-` MUST carry an `A<nn>` link that names what is searched instead. |
| FR-CLB-108 | A level other than the base that equals the `off=` value of a D row MUST be cut from its grid (`edge:off`); a `geo` row MUST have an owner default of at least 1, or a `required` default whose lowest legal value is at least 1, or the flag `z` (an owner default of 0, none or `the world's`); and a `z` row whose base is 0 MUST have no grid and be recorded `unsearched at_off`. |
| FR-CLB-031 | If the base is feasible the choice MUST be the base and the overlay MUST be empty. |
| FR-CLB-032 | Otherwise the choice MUST be the feasible point with the least grid distance to the base, then the fewest knobs changed, then the lowest index. |
| FR-CLB-033 | If no point is feasible there MUST be no choice and the Pareto set of the B pass counts among the points that pass every exact criterion MUST be reported; no weighted sum of facts MUST be computed. |
| FR-CLB-034 | The choice MUST be a pure function of the stored results of the screen and the explore. |
| FR-CLB-035 | The stages MUST be run in the order C0 to C9, a stage MUST NOT move a knob of an earlier one without the re-entry rule of 7.3.11, and a stage of more than `MaxFactors` knobs MUST be split by owner. |
| FR-CLB-036 | No knob MUST be searched while an X or O fact fails on the pack. |
| FR-CLB-037 | A point that changes a knob with the flag `in` or `tbl` MUST regenerate the warm-start table from the solver, or the experiment MUST say that it is a perturbation. |
| FR-CLB-038 | A stage MUST stop only when the seven conditions of 7.4.1 hold on the confirm seeds. |
| FR-CLB-039 | A default MUST be called stable only under STY 7.3.1: STABLE at the default and at **both** neighbours of its grid on the gate seeds, with a `stability` line for each point and a `claim` line `stable` in the record; a default with an edge on one side MUST be recorded `one_neighbour` and MUST NOT be called stable. |
| FR-CLB-100 | A `stability` line MUST say `in_sample no` only for a `holdout` set of the seeds 51 to 60 of REF 8.2 that no choice has read; every other line MUST say `in_sample yes`. |
| FR-CLB-040 | A record MUST have the lines of 8.2 in order, and it MUST be reproducible: the documents, the hash, the experiment and the seeds MUST give the digest of each rendered result. |
| FR-CLB-041 | A record MUST contain no floating point, no map order and no wall-clock time, and two runs of one calibration MUST give the same bytes. |
| FR-CLB-042 | A record MUST name the configuration hashes and documents, every experiment, the results, the bands met and unmet, the detectors with `working`, the attempt and block, and the decision. |
| FR-CLB-043 | Parsing the rendering of a record MUST give the record, and an unknown keyword MUST be an error. |
| FR-CLB-050 | A calibration overlay MUST contain only keys whose registry class is K or D. |
| FR-CLB-051 | A calibration overlay MUST begin with a comment that names its record. |
| FR-CLB-052 | The warm-start document MUST be generated from the effective configuration of every other document, calibration overlays included, last. |
| FR-CLB-053 | A change to a row of class S, to a unit or to a K or D value that the pack's own documents set MUST change the `lineage` (the digest of the structure documents, 9.2), and every stage MUST be run again from C1. |
| FR-CLB-054 | A band value, a stability record, a warm-start table and a calibration overlay of one lineage MUST NOT be applied to another. |
| FR-CLB-060 | The screen's elasticities and effect ranges MUST be kept in the record, and a measured sign that differs from the registry's MUST be a finding. |
| FR-CLB-061 | Two knobs MUST be aliased when `10 000 (e_i . e_j)^2` is at least `(10 000 - alias_tol_bp) (e_i . e_i)(e_j . e_j)`, in 128-bit integers under the bound of 10.2 (an elasticity at most `2^26` in absolute value, at most 16 measures; a reading above it or `Clamped` is excluded as `OK` false), and a zero vector MUST NOT be aliased. |
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
| NFR-CLB-003 | `internal/calibration` MUST import only the standard library and `internal/design`, MUST NOT import the root package, `config/`, `strategy/`, `indicator/` or `experiment/` (architecture.md 2.9), and MUST NOT use floating point, a clock, a random source, package-level mutable state or a goroutine. |
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
| X2 | the first wording of the roadmap's CLB row ("it grows the parameter-owner table"; the row now says it adds no row) and the registry's rule that a default is defined once | a ranges column in the owner table would restate the owners' bounds and the registry's search ranges and drift | met in the roadmap; a pointer sentence under the owner table and the registry in this spec (12.2) |
| X3 | STY 8.2 (the gate 1 to 10 lies inside the setting range 1 to 50, so a set band is not asserted out of sample) against STY 7.3.1 ("on the gate seeds") | a stability verdict on the gate is `in sample`; the gate cannot be the held-out check of a claim | met in STY (7.3.1 says `in sample` unless REF has a hold-out, 8.2 has the search-block row and recommends the hold-out, 10.2 says the confirm is the calibration's own separation of search and confirm, not the bands' hold-out): search blocks of 101 and up for a choice, REF's nightly seeds 1 to 50 for the confirm, and a hold-out where it is run (7.1, 7.4.2) |
| X4 | STY 6.5.1 (an unset threshold gives `unknown`) against a search that needs a guide | a calibration cannot steer by a detector that says "I do not know" | a labelled **working** threshold (3.3), outside acceptance: met, STY 6.5.1 and its section 21 allow the priors of 6.2 as labelled working thresholds until a setting run sets them |
| X5 | EXP 13.2 `MaxFactors` 16 against stages of more than 16 knobs (C4 has 34, C7 has 22) | one experiment cannot screen C4 or C7 as written | split by owner (7.2.1); no change to EXP |
| X6 | `spec-config.md` 3.7 (`World.Scale`; `Roles[]` `Upkeep`) against the owner table (`Scale`, scope `Group; world default 1`; `Upkeep`, scope `World`) | the same field under two scopes | the registry follows the owner table (`Group.Scale`; `World.Upkeep`, and `Role.Upkeep` by the README remark "also Role", which is its own pair for the coverage check, 4.7, with the group S-AGT06) |
| X7 | the owner table (no row for the `Label`s, `Taste.Role`, `RegionTaste.*`) against `spec-config.md` 8.3 and 3.7 | fields that exist and are not owner rows | registry-only rows (6.3, S-DEM05, S-DEM06); the Taste rows are the README's (23); `World.Clearing` has its README row since the WLD integration |
| X8 | STY 10.2 ("`RumourErrorBP` up widens the converged gap of P06", citing INF 11.2) against INF 11.2 (an independent error 0.05 to 0.8 `f` gives a gap of 1.99 to 0.60 `f` against 1.80 `f` for identical shippers; only a shared error widens it) | the sign of K-INF02 is claimed by STY and shown the other way by INF's model of an invented rule | the registry says `?` until a screen (K-INF02); STY corrects its sentence or cites a model of its own (23) |

## 23. Edits owed to other files

One consolidated table: file, section, exact change. None alters a result that a test
could depend on (nothing has run); the rule files need a fresh read-receipt tag each.
The **asks** (A) are requests to the owner, with this spec's recommendation. The asks that
the integration of the working tree had already met when this revision was made (STY 8.2
search blocks, 7.3.1 `in sample`, 10.2 working thresholds; REF 4.1 `calibrated/` and records, REF
8.2 the hold-out and the search blocks; EXP 12.4 and question 7) are not repeated.

| File | Section | Change |
|------|---------|--------|
| `doc/specs/README.md` | Index and reading order | add `spec-calibration.md` (core): "The registry of tuning knobs: one row for each tunable and solve option, classified as a knob, a damper strength, a structural choice or presentation, with its owner, its kind of range (a validation bound is never a search range; nearly every search range is unknown), its grid and neighbours, its expected effects with sign and source, its loops and the pairs that must move together; the procedure that calibrates a setting against the stylised facts (search blocks, screen, explore, choose, confirm; stages C0 to C9; stop rules), the record, the packs, sensitivity and identifiability, `FR-CLB`/`NFR-CLB`"; reading order after `spec-experiments.md` |
| | Requirement IDs table | the `CLB` row: "exists; Requirements section written (FR-CLB-001 to 091 and 100 to 109, NFR-CLB-001 to 005); core; no `Config` field; all procedure constants are placeholders" |
| | Parameter owners | one sentence under the header: "The class, the kind of range, the expected effects and the interactions of each row are in `spec-calibration.md` section 6, found by (scope token, parameter)."; and the rows `Role` (scope `Taste`) and `Region`, `Want`, `Scale` (scope `RegionTaste`), owner `spec-demand.md` section 9, which `spec-config.md` 3.7 lists (met: the table has them) |
| | Decisions taken | rows: *Where is the registry of knobs?* In `spec-calibration.md` 6; the owner table is not given a column (4.1). *What is a calibration for?* Feasibility and minimal change, never an optimum (3.4). *Which seeds may a calibration choose on?* Search blocks of 100 from 101, one choice each; the confirm on STY's setting seeds 1 to 50 (REF 8.2); never the gate (7.1); a hold-out 51 to 60 is a SHOULD of REF 8.2 and no band is set from it. *What transfers between settings?* The procedure, the conditions and the regime thresholds; defaults and priors as starting points; no band, record or table (9.3) |
| | Glossary | knob, damper strength, structural choice, registry (of knobs), search range, evidence anchor, derived condition, base value, grid (of a knob), neighbour, edge (of a grid), screen, screened in, dormant, invisible, alias group, gauge, composite (knob), search block, confirm seeds, feasible region, minimal change, grid distance, Pareto set, known miss, working target, working threshold, calibration overlay, calibration record, lineage, re-entry, stage (of a calibration), attempt, off value, structure group, scope token, overlay document, claim set, sensitivity matrix, scaling check, stage stop, program stop; the terms **gate**, **hold-out**, **finding**, **stability record**, **elasticity** and **effect range** are used as STY, EXP and IND define them and are not redefined |
| `doc/documentation_index.md` | rows | `specs/spec-calibration.md` and `specs/models/clb/` |
| `doc/specs/models/README.md` | contents | a `clb/` row: `registry.py`, `refs.py`, `grid.py`, `stop.py`, `cost.py`, `choose.py`, `worked.py` with outputs |
| `doc/notes.md` | open notes, index | N-015 updated (20); the new notes of 20; the index; a fresh tag |
| `doc/rules/architecture.md` | 2.6 | add `internal/calibration/` (the grid, the choice, the alias test, the record and the overlay over primitive values; standard library and `internal/design`), created at the first code that needs it (2.1); a fresh read-receipt tag |
| `AGENTS.md` | Layout | the same line |
| `doc.go` (glossary) | | the terms above |
| `doc/specs/spec-config.md` | 9.2, 9.3 | 9.2: the registry has a second reader (CLB 11.1); the solve options are rows of CLB's registry (FR-CFG-047 stays); 9.3: a line "the solve options are CLB registry rows" |
| | 6.2 | the experiment path of a key is the snake-case file spelling of `<Scope>.<Field>` (CLB 4.2; N-102) |
| `doc/specs/spec-viability.md` (A) | 11 (the sentence before the bounds) | "They are rows in the parameter-owner table with the scope 'Solve option'" becomes "rows of the registry of `spec-calibration.md` 6.2.8"; and, in the list of the configuration paths the programme reads (met, and followed by the `in` and `chk` flags, CLB 4.2.1): (a) whether `world.belief_step` moves the table, since 6.4 raises a seed price to the resolution floor (met: the list says `yes (a price at the floor only)` and CLB follows it, K-BEL01 is `in`); (b) say that a `.*` names every field of its table (CLB reads it so) |
| `doc/specs/spec-stylised-facts.md` (A) | 7.3.1 | an **edge clause**: whether a default at the edge of a grid may be called stable on one neighbour with the beyond not examined; CLB 7.4.3 says it may not and records `one_neighbour` |
| | 7.3.3 | the file name of the stability record: `testdata/golden/stylised/<pack>.stability.txt`, in the grammar of CLB 8.2 (the `stability` and `claim` lines), one record and one home (CLB 8.4) |
| | 10.3 | where M06, P08, S06, G01 and W05 are set in the order (they are in no step); CLB 7.3 searches them at C4b (X and O parts, the B part waiting for a pack), C4a, C5, C2 and C7b, and W05 at C5 |
| `doc/specs/spec-stability.md` | 9 | a new step after 9.9 (it becomes 9.10): "classify each new tunable and add its registry row (`spec-calibration.md` 4.3)"; the CLB entries of 17 are merged (L139 to L142 with CLB-A2 in L135 and CLB-A4 in L138, D93 and D95 to D98, S88 to S90) and need nothing |
| `doc/rules/testing.md` | 6.7 | one sentence: the registry test also reads the classification of `spec-calibration.md` 6 (optional; a fresh read-receipt tag) |
| `doc/specs/spec-indicators.md` (A) | 9.1 | a **field column** (a closed list of the result fields of each preset, with the spelling STY and CLB use: `Pinned.Share`, `RoleShares.Swing`, `HouseholdIncome`), so that the registry test of CLB 11.1 can parse it instead of a list in the test; `UpkeepShortfall` is an OBS event and not a field |
| `doc/specs/spec-scale.md` (A) | 5.2 and the owner table | a validation bound for `MinAgents` (it has none in SCL or CFG 4.1, so the grid of K-SCL02 is unbounded above) |
| `doc/specs/spec-observability.md` | 4.4.6 | say that `ProductionHalted` is the counter of D24 (K-AGT09; CLB 15.2 names it) |
| `doc/specs/spec-stability.md` (A) | L142 (STB 4.12), D97 (STB 5) | "which makes the held-out check a fitted one" and "the held-out check is not trained on" become "which makes the gate a fitted check" and "the gate is not trained on": the gate is inside the setting range and is never out of sample (STY 8.2; CLB-A6) |
| `doc/specs/spec-stylised-facts.md` (A) | 10.2 (the expected-signs row) | "`RumourErrorBP` up widens the converged gap of P06 (INF 11.2)": INF 11.2 shows an independent error disperses the shippers and lowers the gap from 1.80 `f` to 0.60 `f`, and only a shared error widens it; state the sign as `?` or as INF's (CLB X8, K-INF02) |

## 24. Where this lives in the code

| Concern | Home |
|---------|------|
| the grid and neighbours, the seed blocks, the choice, the Pareto set, the alias test, the record's render and parse, the overlay renderer: pure functions over primitive values | `internal/calibration/` (`grid.go`, `choose.go`, `alias.go`, `record.go`), standard library and `internal/design` only; unit tests in the package |
| the registry test, the record checks and the overlay purity test | the package `internal/calibration` (`registry_test.go`, `record_test.go`), reading `../../doc/specs/spec-calibration.md`, `README.md` and `spec-config.md`, and `../../testdata/chains/calibrated/`, by relative path (testing.md 10.4) |
| the fuzz corpora of the parser, the grid and the choose function | `internal/calibration/testdata/fuzz/` (testing.md 7.5) |
| the dry run | `experiment`'s external test package (`experiment/calibration_dry_test.go`, package `experiment_test`), skipped with `-short` |
| the library's stage records | `testdata/calibration/<pack>/<stage>-<n>.txt` |
| the stability records | `testdata/golden/stylised/<pack>.stability.txt` (the directory is STY 7.3.3 and 8.5; the file name is this spec's and is asked of STY, 23) |
| the calibration overlays of a pack | `testdata/chains/calibrated/<stage>-<n>.toml` (REF 4.1) |
| the models | `doc/specs/models/clb/` |
