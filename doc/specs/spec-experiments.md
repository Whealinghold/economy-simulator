# Spec: Experiments: the sweep harness, designs and seeds, running, branching, paired runs, reductions and failure

Status: draft. Part of the economy specs: see [README.md](README.md) for the
index, the shared conventions and who owns each parameter. This is a design
reference. It does not override the rules in `doc/rules/` (architecture.md wins
on any conflict). Its requirements carry IDs of the form `FR-EXP-NNN` and
`NFR-EXP-NNN`, which tests cite (testing.md section 4). They are in section 21.

Layer: core ([ADR 0001](../adr/0001-macro-economy-as-replaceable-layers.md):
"tuning depends on it"). It is the "run again" of the tuning loop of the roadmap
("measure (IND), compare with the expected properties (STY), adjust a knob (CLB),
run again (EXP)"). It needs `spec-world.md` (WLD: one world is single-threaded and
shares nothing, the tick boundary, the stream), `spec-checkpoint.md` (CHK: branching
is restore-many), `spec-integration.md` (INT: how a shock enters) and
`spec-indicators.md` (IND: the series, the windows and the arithmetic; the harness
imports none of it, 3.4.0), and it changes no rule of theirs. Where it needs a field of theirs that does not exist, it
asks (section 22).

This spec adds **no `Config` field, no default and no row** to the
parameter-owner table: an experiment is a value a consumer builds, not part of the
configuration of a world, and the configuration hash of every point is the hash
that `spec-config.md` 8 already defines. The numbers in it are the rule constants of
section 13 (SplitMix64 and Feistel constants, the closed sets) and **bounds of the
package that are placeholders** (N-015, N-027). Nothing has run in the library and no
acceptance band exists; the scripts of section 18 are stylised Docker Python models
of the rules of sections 4, 7 and 8, not claims about the library, and the one fact
taken from `spec-scale.md` (the cost of a tick) is a placeholder of its own. The
settings of the worked examples (19) are invented **examples**: the words in them
(`cost_floor`, `belief_step`, `north`, `grow_staple`) are data, never branched on
(`AGENTS.md`, N-017).

Terms in **bold** on first use are new domain terms. They must be added to the
glossary in `doc.go` before any code uses them (architecture.md 3.2). "Consumer"
means the library consumer (a game, a server, a command-line tool), never a household
(N-001, N-017).

## 1. Purpose and scope

Tuning is a loop, and the loop needs many runs: the same chain with a parameter
moved, over several seeds, with a shock at a tick, measured the same way each time,
and the results put side by side. A consumer can write that loop with `NewWorld`,
`Run` and the indicators; this spec says what the library gives it so that every
consumer, scenario and calibration means the same thing by "a sweep", and so that a
result can be reproduced bit for bit from its description. It covers:

- **what an experiment is**: a base pack, factors, a design, replicas, a length,
  windows, a script of calls, measures and criteria (section 3);
- **designs and seeds**: the point sets the library generates, the deterministic
  generator behind them, how it differs from the world's source, and how seeds
  are derived (section 4);
- **running**: the unit of work, the sequential runner, the contract that makes a
  consumer's own parallel runner safe, cancellation (section 5);
- **branching**: one burn-in, many continuations (section 6), and **paired runs**:
  when a draw-for-draw comparison holds and how the harness sees it (section 7);
- **results and statistics**: what is recorded, the reductions, the comparison of
  two points, a sensitivity measure, and the size of a result (section 8);
- **failure** (section 9), **cost** (section 10), **how the harness is tested**
  (section 11) and **what it hands to STY and CLB** (section 12).

Out of scope, with the owner:

| Not here | Owner |
|----------|-------|
| What a measure means: the presets, the window, the price table, the arithmetic of a ratio | `spec-indicators.md` (IND) |
| Which measures a stylised fact is stated in, and the bands | `spec-stylised-facts.md` (STY) |
| The registry of knobs with ranges and expected effects, and the calibration procedure | `spec-calibration.md` (CLB) |
| The calls a shock is made of, their bounds and errors | `spec-integration.md` (INT) |
| Reading, layering and validating a configuration; its hash | `spec-config.md` (CFG) |
| The checkpoint, the restore and the reseed | `spec-checkpoint.md` (CHK) |
| Storing, shipping, plotting or encoding a result | the consumer (ADR 0002 decision 1; architecture.md 1.1) |
| Starting goroutines, a pool, a queue, a cluster | the consumer (architecture.md 12.2); section 5.3 is the contract it relies on |
| A script language, an optimiser, an adaptive design | not provided (3.7) |

## 2. Positions taken

| # | Position | Section |
|---|----------|---------|
| 1 | An **experiment** is plain data plus four function values (the calls of a script, the measures, the options of a run, an optional preparation of a configuration). It names a **base pack** (documents in layering order), **factors** of integer settings by field path, a **design**, a number of **replicas**, a length in ticks, windows, a script, measures and criteria. Everything a run needs is in it, so a run is a function of (experiment, run key) | 3 |
| 2 | A parameter change is an **overlay document** layered after the base by the configuration's own mechanism (`config.Parse`), so the loader's strictness (unknown key, not an integer), the validation, the bounds and the configuration hash apply to every point exactly as to a pack. No second override path and no reflection over `Config` exists | 3.2, 3.3 |
| 3 | The **designs** are three: a full grid, one at a time, and a Latin hypercube ("spread"). All three are pure functions of integers: no floating point, no stream, no state | 4 |
| 4 | The harness draws **no random number from any source**. Every choice that looks random (a permutation, a seed) is a counter-based function of (salt, axis, size, index), so a point is the same whatever other points exist, run, or are skipped. This differs from the world's single positional PCG on purpose | 4.5 |
| 5 | Replica seeds are consecutive integers from `Seed0`: `Seed0 + r` (the same seeds at every point, the default) or `Seed0 + p * R + r` (distinct per point). A failing run is reproduced by one `WithSeed` call | 4.6 |
| 6 | The library's runner is **sequential** and starts no goroutine. The unit of work, `RunUnit`, is a pure function that shares nothing with another unit, and `Assemble` builds the same `Result` from the records of any order, so a consumer's own goroutines (or machines) give the same bytes | 5 |
| 7 | **Branching** restores one checkpoint of a burn-in many times, each with a different **variant** of calls and, optionally, its own reseed. A branch cannot change a configuration value: the configuration hash is part of the restore, so a parameter sweep never branches | 6 |
| 8 | A **paired** comparison needs no alignment to be unbiased: seed-matched pairs are independent across seeds. What alignment (equal words drawn at every boundary) buys is exactness of the aligned prefix and a smaller variance of the difference. The harness checks alignment, when CHK offers the read (7.3.2), from a **stream fingerprint** at each sample and measures the variance reduction it got (**pairing gain**) | 7 |
| 9 | A result is plain data: no map, no float, no pointer. Reductions are integer: mean, standard deviation, nearest-rank quantiles, extremes and a pass share, with one rounding rule (half to even) taken from IND. A comparison of two points gives wins, losses and an exact sign-test tail. Sensitivity is the **range of the bin means** of a factor and its share of the total, and a one-at-a-time elasticity in basis points | 8 |
| 10 | A run that fails is **recorded, not fatal**: it keeps what was measured until the failure, a closed code, the tick and the error text. Failed runs are never dropped from a reduction: they are counted, and a pass share is over the planned replicas | 9 |
| 11 | A result is checkable: each run carries the digest of its final checkpoint, and `Render` gives a canonical text of its readings and verdicts, so "same experiment, same bytes" is a comparison of two strings | 8.2, 11 |

## 3. What an experiment is

### 3.1 The value

The package is `experiment` (a public package at `<module>/experiment`; the home
is section 23). Names are provisional until the code, as in `spec-world.md`
3.7.2; the shape and the rules are not.

```go
// Experiment describes a set of runs. The zero value is not valid.
type Experiment struct {
	Name       string   // a label for the consumer; not part of any key
	Base       [][]byte // the documents of the pack, in layering order (config 6.4)
	Factors    []Factor // none: one point, the base
	Design     Design   // Grid, OneAtATime or Spread
	DesignN    int      // Spread: the number of points
	DesignSalt uint64   // Spread: another batch of the same size
	Replicas   int      // runs per point and variant, at least 1
	Seed0      uint64   // the first replica seed; 1 in the reference chain
	Seeding    Seeding  // SameSeeds (the default) or PerPointSeeds
	Ticks      int64    // the length of a run, in ticks
	Transient  int64    // ticks before the first window opens, 0 .. Ticks-1
	Every      int64    // ticks of one sample window; 0 means one window
	Script     []Step   // calls applied between ticks in every run (3.5)
	Measures   []Measure
	Criteria   []Criterion
	Keep       Keep     // what a run record holds beyond the final readings
	Pair       bool     // record the stream fingerprint at each window, when the checkpoint offers one (7.3)
	Branch     *Branch  // nil: every run starts at tick 0 (6)
	Prepare    func(ctx context.Context, cfg economysim.Config) (economysim.Config, error) // 3.3.3
	Options    func(Key) []economysim.Option // fresh options and extensions for each world built (5.5)
}
```

3.1.1. **What makes a run.** A run is determined by the experiment and its
**key**: the point, the replica, and, with a branch, the variant and the
continuation (4.6, 6). Nothing else reaches it: no clock, no environment, no
global, no goroutine count, no result of another run (FR-EXP-001). The function
values are the consumer's code and are not part of any hash; the rest of the
experiment is plain data. The configuration hash of the world, the seed, the key and
the window settings say which run a record is about; the code behind a registered
name is in no hash (WLD 4.1; 8.1.3).

3.1.2. **Plain data and function values.** The four function values are the
reason an experiment is not itself serialisable. They are the seams that cannot be
data: a call on a world (a shock, `AddAgent`) is a method, a measure is a function
of a window of the consumer's, an extension needs a new instance per world (CHK 11 item 6), and a
consumer that regenerates the warm start of a point calls the solver
(`spec-viability.md` 7.3). Everything else (the pack, the factors, the design, the
seeds, the lengths, the criteria) is plain data and is what a consumer stores to
say what it ran.

### 3.2 Factors, levels and paths

```go
// A Factor is one axis of the design: a finite list of levels, or an integer
// range of one field path.
type Factor struct {
	Name   string  // a label; default: the first path of the first level
	Levels []Level // explicit levels, at least 2; or
	Path   string  // a field path, with Min and Max (a range factor)
	Min    int64
	Max    int64
	Steps  int     // Grid and OneAtATime with a range: the levels, at least 2
}

// A Level is one setting of a factor: integer changes, whole documents, and calls.
type Level struct {
	Label   string
	Changes []Change // a field path and an integer value
	Docs    [][]byte // documents layered after the changes (a variant, a module on)
	Steps   []Step   // calls added to the script of every run at this level
}

type Change struct {
	Path  string // "region.north.group.worker_a.households", "world.cost_floor"
	Value int64
}
```

3.2.1. **A path** is the position of a key in the file layout of
`spec-config.md` 6.2: two to eight segments of `[a-z][a-z0-9_]*`, at most 64
bytes each, separated by dots, the last being the key and the others the tables
(`world.cost_floor`, `role.worker_a.target_ratio`,
`modules.migration.move_below`). The file names are the snake case of the Go field
names by the fixed rule of `spec-config.md` 6.2, so a path is the file spelling of
a registry row. A path that is not of this syntax is `ErrBadFactor` before any run
(14).

3.2.2. **A value is an integer** in version 1: `int64`, as the loader's
integer type. A boolean, a string and a list are not swept as a value; a whole
document level (`Docs`) covers what they need (a module on or off, a role
`closed`, a strategy name): the author writes the table, and the factor chooses
between documents (open question 2).

3.2.3. **A range factor** has a `Path`, a `Min` at most a `Max`, and for a grid
`Steps` evenly spaced levels with `value(k) = Min + rdiv(k * (Max - Min), Steps - 1)`
(half to even; the end points are exact); in a spread design the range is covered
by the cell rule of 4.4. A range factor is a factor with one change per level; the
two forms exist so that a calibration over `[7 000, 10 000]` does not have to
write four thousand levels.

3.2.4. **A composite factor** is a factor whose levels carry several changes
(`households` of two groups moved together, a capacity and a rate): the
author's choice, no new mechanism. Two factors must not set one path in the
`Changes` of any of their levels (`ErrFactorOverlap`); the paths that a `Docs`
level sets are the author's and the harness does not read them, so a point whose
two factors both set a path through documents depends on the order of the
factors, which is part of the experiment (3.2.5).

3.2.5. **Level zero is the baseline** of a factor for the one-at-a-time design
(4.3), and the factor's order in the slice is the order of the points of the
grid (4.2). The order of `Factors` is therefore part of the experiment; the
order of `Changes` within a level is not (a path appears once in a level).

### 3.3 The base and its packs

3.3.1. **The base** is the documents of a pack in the order the loader layers
them (`spec-config.md` 6.4): the chain, then the overlays. The consumer supplies
the bytes (from an `fs.FS` or embedded); the library opens no file
(architecture.md 1.1).

3.3.2. **A point is resolved by the loader.** For a point the harness builds
the overlay documents (3.3.4) and calls `config.Parse` on the base followed by
the overlays in order: the documents of the levels in factor order, each level's
`Docs` first and then one document for each of its `Changes`. Everything the
configuration's own mechanism does then applies to the point (FR-EXP-002):

- the loader rejects an unknown key (`ErrUnknownKey`), a value that is not an
  integer where one is needed, a scalar over a table (`ErrMergeConflict`)
  (`spec-config.md` 6.3, 6.4): a path that is not a field is found at the first
  point, as a recorded failure, and by `Check` (14.3) before a run;
- `Validate` runs on the resolved `Config` (5.1.3 step 1) and the name checks of
  `NewWorld` run when the world is built, so a value out of its bound, a chain
  that becomes non-viable, an unknown strategy and a warning are the point's own
  validation result;
- the configuration hash that a record holds is the one of the **world** the run
  built (`World.ConfigHash`, `spec-config.md` 8.2), which includes
  `WithTargetSpread` when `Options` carries it (WLD 4.1) and so is not the hash of
  the documents alone; two points whose worlds have one hash are visible as equal
  hashes (`SameAs`, 8.2).

3.3.2a. **What a document can and cannot do** (`spec-config.md` 6.3, 6.4,
FR-CFG-021). (a) An overlay is a document of one table and one key and has no
`schema` line; the `schema` is checked on the merged tree of the base, and this
spec asks CFG to say so (22). (b) Tables merge by key, so a path that names an
entity the base lacks (`role.worker_c.target_ratio` for an absent role) creates an
entity and fails later as a missing field or a reference error (`invalid_config`),
not as an unknown key. (c) A list has no path: a path names an integer key, and
"a path the loader accepts is a field" is true of an integer key only (11.3.1).
(d) A document cannot delete a key or an entity, so a factor over documents that
switches a module or a warm start **off** is valid only when the base is the off
state and the document switches it on. (e) The call is written in Go with
`base` and the overlays as the variadic arguments of `config.Parse`; 19.4 shows
the documents, not the call.

3.3.3. **Prepare.** An optional function of the resolved `Config` returns a
`Config` or an error, applied after the overlays and before validation. Its use is
the one `spec-viability.md` 7.3 names: a parameter that moves the steady state
makes the base pack's warm start stale, and a consumer that sweeps such a parameter
regenerates `WarmStart` from the solver for each point (the table is data, hashed,
so the point's hash says which table it ran with). A sweep with no `Prepare` over
such a parameter runs every point from the base's warm start, which is a legitimate
experiment (a perturbation) and a trap (EXP-A5, 17); the harness could report `stale_warm_start` (14.4) when the base has a
`WarmStart`, a point changes a path that the solver reads and `Prepare` is nil;
the list of paths the solver reads is VIA's to publish and the warning is not
specified until it does (open question 7). `Prepare` takes the context of the run
as its first parameter (it may call the solver, architecture.md 12.3).

3.3.4. **The overlay document of a change** is one table header and one key:

```
[world]
cost_floor = 8000
```

for the path `world.cost_floor` and value 8000; for a path of `n` segments the
header is the first `n - 1` joined by dots. The loader merges tables by key and a
later scalar replaces an earlier one, so the documents of a point, applied in
order, are the point's changes. A document is generated by one function of
`internal/design` from the path and the value, with no formatting choice
(decimal, no sign for non-negative, `-` for negative); it is the same bytes on
every platform (FR-EXP-003).

3.3.5. **A Go-built configuration** has no paths, and the library does not walk a
`Config` by reflection to give it some: that would be a second override
mechanism whose naming must be kept equal to the loader's (`spec-config.md` 3.6
says file and Go agree; a second map from path to field would be a way for them
to disagree). A consumer that builds in Go calls `ExecuteUnit` (5.1) with its own
`Config` per point and uses the reductions of section 8 on the records
(open question 1).

### 3.4 Measures, samples and windows

```go
// A Measure turns one window into one integer, or says it is undefined.
type Measure struct {
	Name     string
	Observer func(Key) economysim.Observer // optional: read events (3.4.4)
	Begin    func(k Key, open economysim.Snapshot, from int64, span bool) // optional: a window opens
	Fork     func(from, to Key)            // optional: a branch copies state (6.1.1)
	Fn       func(Sample) (value int64, ok bool, err error)
}

type Sample struct {
	Config      economysim.Config
	Open, Close economysim.Snapshot
	From, To    int64 // the window is ticks [From, To)
	Span        bool  // the whole span [Transient, Ticks): the final reading
	Key         Key   // the run: a measure finds here what its Observer built
}
```

3.4.0. **The harness imports no indicator.** `spec-indicators.md` FR-IND-001 lets
no package of the module other than `indicator`, its tests, `otelobserver` and
`examples/` import `indicator`, and none but `indicator` import `internal/measure`.
`experiment/` does neither. A measure is the **consumer's** function: it imports
`indicator` itself, builds an `indicator.Tally` in its `Observer`, opens it in
`Begin`, closes it in `Fn` and computes its preset; the harness only calls the
hooks at the boundaries and stores the integers. The reductions of 8 use the
harness's own integer helpers (`internal/reduce`, 23), with the rounding rule of
IND 5.2.2.

3.4.1. **A measure is one integer.** A measure that wants a vector (a price
per commodity) is one measure per cell, named by the cell: the names table of OBS
3.2 gives the strings. This keeps a result a rectangle of integers and a reduction
a function of a column (open question 9).

3.4.2. **The windows.** The run is `Ticks` long. The first `Transient` ticks are
a burn-in that no window covers. From `Transient` the run is cut into windows of
`Every` ticks (the last one shorter if `Ticks - Transient` is not a multiple), or
into one window `[Transient, Ticks)` when `Every` is 0. At the opening of each
window the harness calls every measure's `Begin` with the opening snapshot (taken
after the steps of that boundary, 5.1.3); at its close it calls `Fn` with the two
snapshots that bound it. A window is half-open and whole ticks (IND 4.1.1); no
value exists inside a tick.

3.4.3. **Two kinds of reading.** The **series** is the reading of each measure at
each window close (kept only with `Keep.Series`). The **final** reading is one
value per measure over the **whole span** `[Transient, Ticks)`: the harness calls
`Begin` and `Fn` once more for the span (`Span` true, opened at `Transient`, closed
at `Ticks`), so that a measure of a "last two years after a transient"
(`spec-reference-chain.md` 8.2) is one call and not a reduction of windows
(FR-EXP-016). With `Every` 0 the series and the final reading coincide and are
kept once. A measure that reads events keeps a second accumulator for the span;
the events of a call made at an interior boundary (they carry the `Tick` of the
step that follows, INT 3.3.3) are in the span's accumulator and in no sample
window if the measure's accumulators follow IND 4.1.4 (no window is open while a
call is made), so the final reading then equals the sum of the windows only for a
script with no call that moves money or goods.

3.4.4. **The harness registers an observer only for a measure that has one.**
If no measure has an `Observer`, the harness registers none: a world with no
observer constructs no event (observability-and-logging.md 4.5) and the sweep
pays only for the snapshots. Each `Observer(key)` is called once for each world
built (5.5.2) and its result is registered with `WithObserver`, in measure order,
after the consumer's own. `Pair` needs no observer: the fingerprint comes from a
checkpoint (7.3). The harness's own code observes no event.

3.4.5. **Criteria.** A **criterion** is a named band on one measure, as data:

```go
type Criterion struct {
	Name    string
	Measure string // a Measure.Name
	Op      Op     // AtMost, AtLeast or Between (closed set)
	A, B    int64  // the limit, or the inclusive bounds of Between
}
```

evaluated on the **final** reading of each run: a run **passes** when the
reading is defined and inside the band, and a run that failed or whose reading is
undefined does not pass (9.3). The reduction over replicas is a **pass share**
(8.3). A band is a datum of the experiment, set by STY and not by this spec
(N-027).

### 3.5 The script

```go
// A Step is a call, or a few, applied at the boundary before a tick.
type Step struct {
	Tick  int64
	Label string
	Do    func(w *economysim.World) error
}
```

3.5.0. **A step keeps no state between runs.** `Do` returns only an error, and an
agent ID that `AddAgent` returns (INT 3.4.2: an `AgentID` has no string form) must
not be kept in a variable the closure shares with another run (5.3.3). A later step
finds the agent by a query of its own world (`Agents`, `Agent`: INT 6.2), for
example by role and region.

3.5.1. **A script is a list of (tick, call)** exactly as INT 9.1 describes: the
harness applies each call at the boundary before the tick it names, in list order
for equal ticks, with its own tick counter, and never inside a step. The calls
are INT's: `SetYieldShocks`, `SetEdgeStatuses`, `SetFlowScales`, `AddAgent`,
`Direct`, `Relocate`, `RemoveAgent` and the rest of INT 5 and 4. The sequence of
calls is an input of the run (INT 3.5.1), so the script is part of what a result
is a function of.

3.5.2. **Sources of steps.** `Experiment.Script` (every run), `Level.Steps` (the
runs of a point at that level: a shock whose size is a factor), and a
`Variant`'s steps (6). The steps of one run are merged by tick; equal ticks keep
the order script, then levels in factor order, then the variant's.

3.5.3. **A step that returns an error** fails its run with the code `script`
(9.2). Each INT call is atomic (INT 3.3.1) but a call is not a transaction across
calls (INT 3.3.4), so a `Do` that made several calls and failed on a later one
leaves the earlier ones applied; the world is healthy and the run is stopped
because its experiment did not do what it said. A `Tick` outside `[0, Ticks)` of
any step, whether of the script, of a level or of a variant, is `ErrBadSchedule`
before any run.

3.5.4. **A shock has no timer.** To end a shock at tick `t + 60` the script has a
second step that clears it (INT 3.6, 5.2.2). The harness adds none.

### 3.6 Validation of an experiment

`Validate()` checks what is a property of the value alone and runs no world and
parses no document (FR-EXP-004). It returns every error it finds, joined, each
wrapped with the path of the Go field (`Factors[2].Levels[0].Changes[1].Path`), as
`*ValidationError` values of `spec-config.md` 7.1.1 sorted by path then rule, at most
`MaxErrors` (100) and then one `ErrTooManyErrors` that says how many were left out
(`spec-config.md` 7.1.3). A record has no error channel: a failed run is a status and
the error's text (9.2).

| Rule | Error |
|------|-------|
| `Replicas` at least 1 and at most `MaxReplicas`; `Ticks` at least 1; `0 <= Transient < Ticks`; `Every >= 0` and `Ticks / Every` at most `MaxWindows`; every step tick of the script, of a level and of a variant in `[0, Ticks)`; `Pair` while the checkpoint surface offers no read of the stream state (7.3.2) | `ErrBadSchedule` |
| a factor has either at least 2 and at most `MaxLevels` levels or a path with `Min <= Max` (checked in 128 bits, 3.2.3), not both; a range factor in a grid has `Steps >= 2`; a path has the syntax of 3.2.1; at most `MaxFactors` factors | `ErrBadFactor` |
| no path is set by two factors in a `Changes` list (a `Docs` level sets paths the harness does not read: 3.2.4) | `ErrFactorOverlap` |
| `Design` is one of the three; a spread has `DesignN >= 1` | `ErrBadDesign` |
| two measures or two criteria have one name; a criterion names a measure | `ErrDuplicateID`, `ErrUnknownMeasure` |
| a branch has `1 <= At < Ticks`, at least one and at most `MaxVariants` variants, `Continuations` at most `MaxContinuations`, and no variant step below `At` (6.1) | `ErrBadBranch` |
| `Measures` at most `MaxMeasures`; the size of the experiment is within the bounds of 13.2 (points, runs, cells), with products computed by a checked multiplication | `ErrOversize` |

3.6.1. **What validation does not do.** It does not resolve a path against a
configuration (that needs the base and the loader: `Check`, 14.3), and it does not
run `Prepare`. An experiment that validates may still have a point that fails to
resolve; that is a recorded point failure and not a precondition (9).

### 3.7 What is not an experiment

- **Not a script language.** There is no loop, condition, expression or variable
  in an experiment, and no call that reads a result and changes the next run. A
  consumer that wants "bisect to the parameter at which the loop goes unstable" writes
  a loop that builds an `Experiment` per round: the loop is the consumer's, as the
  roadmap says of the calibration loop, and an adaptive design in the library
  would be an optimiser (open question 3).
- **Not file input or output.** The harness reads no file and writes none, takes
  no path, and has no cache directory. The consumer supplies bytes and stores
  `Record` and `Result` values as it stores a checkpoint (ADR 0002 decision 1).
- **Not an in-run controller.** A result never reaches a world: no slot, option
  or call of a run reads a measure (IND 3.1.1, WLD 9.4). A run that "reacts" is a
  consumer's own loop between two `RunUnit` calls and a hazard it owns (IND-A1).
- **Not a database.** The library keeps no history of experiments and no index.
  The key of a result (4.6, 8.1) is what a consumer's store indexes by.
- **Not a statistics package.** It gives the reductions of section 8 that a tuning
  loop and a stylised-fact test need, with exact integer arithmetic. A confidence
  interval of a mean, a regression or a plot are the consumer's, in floating point,
  outside the core (architecture.md 3.5).

## 4. Designs and seeds

### 4.1 Points and their order

4.1.1. A **point** is one assignment of a level (or a range value) to every
factor, and **its index** is its position in the order the design generates.
The index is stable: the same experiment gives the same points in the same order
on every platform (FR-EXP-005). The number of points is stated by the design and
checked against the bound before anything is generated (13.2).

4.1.2. A point carries, for the result: its index, the level index of each
explicit factor (or the cell, for a spread), the **value** of each range factor,
and, from the runs that built a world, the configuration hash of that world
(8.1.1, 8.2.1). Nothing else about a point is stored: the experiment is the description.

4.1.3. **A factor-free experiment** has one point, the base, whatever `Design`
says (the design of zero axes is one point). It is the shape of an experiment
that only replicates one configuration over seeds, which is most of what STY runs.

### 4.2 Grid

4.2.1. **The full grid** has `L_1 * L_2 * ... * L_F` points, `L_f` the number of
levels of factor `f` (`Steps` for a range factor). The index is the mixed-radix
number of the level indices with **the last factor fastest**:

```
index = ((l_1 * L_2 + l_2) * L_3 + l_3) ...
```

so that the points of a two-factor grid of 4 by 3 are listed as in the worked
example of 19.1. The inverse is integer division and remainder.

4.2.2. **When to use it.** A grid is for few factors with a few levels each, where
every combination is the question (S07's four `cost_floor` values, S23's
elasticity by headroom). The cost is the product, which the bound stops
(13.2); a grid of five factors of five levels is 3 125 points.

### 4.3 One at a time

4.3.1. **One at a time** (OAT) has `1 + sum over f of (L_f - 1)` points: point 0
is the **baseline**, level 0 of every factor, and then for each factor in order
each level above 0 with the other factors at level 0 (worked example 19.2).

4.3.2. **When to use it.** Screening: which of fifteen belief parameters
moves a measure at all, at a cost linear in the factors. It sees no interaction:
two factors that matter only together look inert. The result of section 8.5
for an OAT design is the elasticity at the baseline, not a main effect over the
space.

### 4.4 Spread

4.4.1. **The design.** A **spread** of `N` points over `d` factors is a **Latin
hypercube**: on every factor each of `N` equal cells is used by exactly one
point, so every one-dimensional projection is exactly stratified (a design of
`N = 10` over a range of ten values uses each value once). Which cell a point
uses on which axis is a pseudo-random permutation, generated without a stream
(4.5).

4.4.2. **The rule.** For axis `j` (the position of the factor in `Factors`,
from 0), size `N` and index `i`, the cell is

```
cell(i, j, N) = P_{k_j}(i)       with   k_j = fin(fin(S + (j + 1) * G) + N)
```

where `S = DesignSalt xor 0x243F6A8885A308D3`, `G = 0x9E3779B97F4A7C15`, `fin`
is the SplitMix64 finaliser of `spec-world.md` 7.1.2 (the same helper of
`internal/ordering`), and `P_k` is a permutation of `[0, N)` built as follows. Let
`w` be the smallest integer at least 1 with `2^(2w) >= N` (for `N >= 2`;
`N = 1` has the one cell 0). All additions and products of `S`, `G`, `k` and `t`
are modulo 2^64, by rule. A 4-round balanced
Feistel network over `2w` bits maps `x = (l, r)` (each `w` bits) through
`(l, r) -> (r, l xor (fin(k + t * G + r) mod 2^w))` for `t = 1 .. 4`;
**cycle walking** applies it again while the result is at least `N`. Because the
network is a permutation of `[0, 2^(2w))` and, for `N >= 2`, `N <= 2^(2w) < 4 N`,
the walk terminates in fewer than four applications on average and `P_k` is a permutation
of `[0, N)` (FR-EXP-007; the model of 18 checks every `N` from 1 to 300 and
larger ones).

4.4.3. **The value.** A factor with explicit levels uses level
`floor((2 * cell + 1) * L / (2 * N))`; a range factor `[Min, Max]` uses

```
value = Min + floor((2 * cell + 1) * (Max - Min + 1) / (2 * N))
```

the **midpoint** of its cell, in 128-bit arithmetic with no rounding other than
the floor (FR-EXP-008). A range narrower than `N` repeats values (several points
share a value on that axis); `Check` warns (`ranged_narrower_than_design`, 14.4)
and suggests a grid.

4.4.4. **What it is not.** The correlation between two columns of a random
Latin hypercube is as large as that of a random draw (the model of 18: the
largest of 28 column pairs at `N = 64` and 8 factors is 0.29 in the mean over
salts, against 0.27, a typical pair about 0.1), so a spread is not claimed to make the axes uncorrelated, only to
stratify each. A design that must be nearly orthogonal needs a better
construction (a maximin or an orthogonal array) that is not provided; the
batch can be repeated with another `DesignSalt` (4.4.5).

4.4.5. **Growth.** A spread of `N` is not a prefix of a spread of `N + M`: the
stratification depends on `N`. To add points to a finished spread a consumer runs a
second batch with another `DesignSalt`; the union is not stratified but is a
valid sample. A design that grows by prefix would be a low-discrepancy
sequence; the one tried (a generalised golden ratio sequence in 2^64 fixed
point) was rejected because its one-dimensional projections clump at small `N`
(4.7).

### 4.5 The generator, and how it differs from the world's source

4.5.1. **The harness has no source.** It takes no `*rand.Rand`, builds none and
reads none from a world. The brief of the specs allows "a source it is given,
seeded"; it was rejected for the reason in 4.5.3. Every value that looks random in
an experiment (a cell, a seed) is a **counter-based** function: a pure function
of its integer arguments through the finaliser, with no state between calls.

4.5.2. **How this differs from the world's source** (`spec-world.md` 7):

| | The world | The harness |
|--|-----------|-------------|
| Source | one `rand.PCG` per world, seeded by `WithSeed` | none; the finaliser of a counter |
| State | the 128-bit PCG state, saved in a checkpoint | none |
| Order | positional: the value of a draw depends on how many were taken before it (WLD-A5) | none: point `i` of axis `j` is a function of `(S, j, N, i)` alone |
| Reproduction | by the sequence of calls | by the arguments of one call |
| Reading | consumed once, in the order of the tick | any point, any time, in any order, on any machine |
| Visibility | extensions receive it (architecture.md 7.7) | no extension, strategy or world receives it |

4.5.3. **Why a stream is the wrong tool here.** A stream is positional: the
value of the `k`-th draw depends on `k` (`spec-world.md` 7.6.1). A harness that
drew its design from one stream would make point 40 depend on how many points
precede it, on how many were skipped for a failure, on which were run on which
worker and on whether the batch was resumed. A counter-based function makes each
point an independent function of its index, so a skipped or resumed or
re-ordered run gives the same point (FR-EXP-009).

4.5.4. **No leak into a world.** Nothing the harness computes is passed to a
world as randomness: a world's stream is its own (`WithSeed`, or `WithReseed`
with a salt that is a datum: 6.4). Adding a factor or a point to an experiment
therefore never moves a draw in another run.

### 4.6 Seeds and replicas

4.6.1. **A replica** is one run of one point (and variant) with one seed.
`Replicas` gives their number `R`. The seed of replica `r` of point `p` is

```
SameSeeds (default)    seed = Seed0 + r
PerPointSeeds          seed = Seed0 + p * R + r        (arithmetic modulo 2^64)
```

4.6.2. **Why consecutive integers.** `WithSeed` already turns a small integer
into an unrelated 128-bit state (`spec-world.md` 7.1.2; 7.1.4: the first outputs of neighbouring
seeds differ in about 32 bits of 64 on average, with the mix or without), so a second mixing in the harness
would add nothing, and would make a failing seed impossible to reproduce by hand.
With `Seed0` 1 the replicas are the seeds 1 to `R` of `spec-reference-chain.md`
8.2, and the message of a failing run names a seed that is a valid argument of
`WithSeed` (testing.md 6.2). A run is reproduced by `NewWorld(cfg,
WithSeed(seed))` and the point's documents (FR-EXP-010).

4.6.3. **Same or per point.** `SameSeeds` uses the same `R` seeds at every point:
the comparison of two points is then seed-matched, and a pair of runs is a pair
that the alignment check (7.3) can see. `PerPointSeeds` gives every point its
own seeds: the points are independent samples, which is what an estimate of the
spread over the whole design wants and what a paired comparison cannot be.
`SameSeeds` is the default because the cost of choosing it is nothing and the
information it keeps is the pairing.

4.6.4. **A hold-out is another `Seed0`.** A calibration that picks the best of
many points on seeds 1 to 10 has fitted those seeds as well as the knobs
(EXP-A2, 17). The check is a second experiment with the chosen point and
`Seed0` 1 001: the harness has nothing to add (FR-EXP-011 states only that the
two are independent of each other).

### 4.7 Alternatives rejected

| Alternative | Why not |
|-------------|---------|
| A seeded stream for the design (a source the harness is given) | positional, so a point depends on its predecessors (4.5.3) |
| `math/rand/v2` or `crypto/rand` | architecture.md 7.3 binds the root and `internal/`; this spec extends it to `experiment/` (owed row, 22) and `AGENTS.md` bans global and `crypto` randomness in the core |
| A Sobol sequence | extensible and stratified, but needs a table of direction numbers per dimension that this spec cannot verify; the table is a dependency of its own |
| A generalised golden ratio (Kronecker) sequence in 2^64 fixed point | extensible, one multiplier per axis computed exactly, `d = 1` gives the SplitMix64 gamma; but its one-dimensional projections are poor at small `N` for `d >= 4`: with `d = 4`, `N = 100` and ten bins one bin held 1 point and one 15, against 10 (`models/exp/kronecker.py`) |
| Halton | stratified per axis for aligned bins only; the projections of two large bases are linear stripes for `N` below their product |
| A rank-1 lattice | needs a generating vector chosen by a search per `N`; a bad one is worse than random |
| Full-factorial only | the product of levels is infeasible beyond four factors |

## 5. Running

### 5.1 The unit of work

```go
// A Key names one run.
type Key struct {
	Point, Replica int // Replica counts from 0; the seed is 4.6
	Variant, Cont  int // 0 and 0 without a branch (6)
}

// A Unit is the runs that share a burn-in: one key without a branch.
type Unit struct {
	Point, Replica int
	Keys           []Key
}

func Plan(e Experiment) ([]Unit, error)                                          // pure
func RunUnit(ctx context.Context, e Experiment, u Unit) ([]Record, error)        // one unit
func ExecuteUnit(ctx context.Context, cfg economysim.Config, e Experiment, u Unit) ([]Record, error)
func Assemble(e Experiment, recs []Record) (Result, error)                       // pure
func Run(ctx context.Context, e Experiment, opts ...RunOption) (Result, error)   // sequential
```

5.1.1. **Plan** lists the units in **plan order**: point ascending, then replica
ascending (point-major), and within a unit the keys by variant then
continuation. It resolves no document and builds no world: a plan of a million
runs is a million small structs and is checked against the bound first (13.2).

5.1.2. **RunUnit** is a pure function of the experiment and the unit
(FR-EXP-001). It resolves the point (3.3), builds the world, runs it as 5.1.3
says, and returns one record for each key of the unit. It shares nothing with any
other call (5.3). **ExecuteUnit** is the same with a `Config` the consumer built
(3.3.5): it skips the base, the factors and `Prepare`, and everything after
resolution is identical.

5.1.3. **The run of one key (no branch).**

1. *Resolve.* Parse base and overlays (3.3), apply `Prepare`, then call
   `Validate` on the result. A failure here is the point's, with the code
   `unresolved` (the loader rejected a document) or `invalid_config` (`Prepare`
   returned an error, or `Validate` returned errors); no world is built (9).
2. *Build.* The options are `Options(key)` (which carries neither `WithSeed` nor
   `WithReseed`, 5.5), then the observers of the measures
   (3.4.4), then `WithSeed(seed)` **last**, so the seed is the harness's
   whatever the options say (`spec-world.md` 4.1: the last one given wins).
   `NewWorld`: a `ValidationError` of its name checks is `invalid_config`,
   `ErrAuditFailed` is `audit` (`FailTick` 0), and any other error is `world`,
   whatever it wraps (9.1).
3. *Segments.* Let the boundaries be the union of `Transient`, every window end,
   every step tick, and `Ticks`. From tick 0 to each boundary in order the harness
   calls `w.Run(ctx, n)` for the ticks to the next boundary. At a boundary `t`
   it does, in this order: (a) if `t` ends a window, take the closing snapshot,
   call `Fn` of every measure (3.4); with `Pair` take the fingerprint (7.3); (b) apply the steps whose tick is `t`, in the order of
   3.5.2; (c) if `t` opens a window (`t` is `Transient` or a window end), take the
   opening snapshot, **after** the steps, and call every `Begin`, so the window begins from the state the
   next step will see (the totals the calls added are `Pending` in a snapshot after the
   first step and `Construction` at tick 0, `spec-world.md` 8.1.3, 5.5.2; IND
   4.1.4, 7.1.5). A segment of zero ticks (a step at tick 0 with `Transient` 0) is
   skipped: no `w.Run` is called for it.
4. *End.* At `Ticks`: the final readings (the whole-span window), the digest of
   the checkpoint (`Checkpoint`, then the last 32 bytes of its `MarshalBinary`:
   the trailer, CHK 5.4; no `io.Writer` variant exists), the final snapshot with
   `Keep.Snapshot`. A world is never checkpointed in a failed state (9.4); an
   error of `Checkpoint` on a healthy world is `checkpoint` (9.1).

5.1.4. **`Run` emits `RunStarted` and `RunEnded`** once for each segment (and
the `Info` lines of `spec-world.md` 16.3), to the consumer's observers and logger
(`spec-world.md` 3.4.1). That is the world's behaviour and a consumer's observer
that counts runs sees one per non-empty segment; the harness's own observer
ignores them.

### 5.2 The sequential runner

5.2.1. **`Run`** validates the experiment, plans, and for each unit in plan order
calls `RunUnit` on the caller's goroutine, delivers each record to the
`WithRecordFunc` callback (if any), then assembles the result. It starts no
goroutine and holds one world at a time (architecture.md 12.2).

5.2.2. **Options of a run of the sweep**:

| Option | Meaning |
|--------|---------|
| `WithRecordFunc(f func(Record))` | called synchronously, in plan order, after each record, on the caller's goroutine; it must not call the runner. It is the way to store results as they come and to report progress |
| `WithPrior(recs []Record)` | records already produced (from an earlier, cancelled or crashed sweep): a unit whose keys are all present is not run again. The records are trusted as `Assemble` trusts them (5.3.4) |

5.2.3. **The result of a sweep that stopped.** A `Run` that is cancelled
returns the `Result` assembled from the records of the **complete units**
before the cancellation, and the context's error wrapped (5.4). The keys not run
are `missing` in the result (9.2); a later `Run` with `WithPrior` of those
records completes it, and its bytes equal an uninterrupted run (FR-EXP-020).

### 5.3 The contract for a consumer's own parallel runner

The library does not parallelise (architecture.md 12.2): worlds share nothing
(`spec-world.md` 3.1.1), so a consumer that wants many cores starts its own
goroutines, or its own machines, and calls `RunUnit` on each unit of `Plan`.
This is what it may rely on, and what it must do.

5.3.1. **Shared and read-only:** the `Experiment` value, including the bytes of
`Base` and of every `Docs`, and the `Plan`. No function of the library writes
to them; the consumer must not, while a run is in flight (FR-EXP-021).

5.3.2. **Not shared, built per call:** the `Config` (parsed again for each unit),
the world, the measures' observers and their state, the record. There is no package
variable, cache, pool or `init` in the package (architecture.md 7.1, 7.2).

5.3.3. **What the consumer supplies must be safe for concurrent calls:**
`Options`, `Prepare`, every `Measure.Fn`, every `Step.Do`. Each is called with
the arguments of one run and must not write state it shares with another run
without its own synchronisation. `Options` must return **new instances** of
every extension that keeps state (CHK 11 item 6, 7.2): a shared stateful
`Strategy` corrupts both worlds, silently, and the library cannot see it
(CHK-A1). A shared `*slog.Logger` is safe if its handler is, as the standard
ones are.

5.3.4. **`Assemble` is order-insensitive.** It sorts the records by key and
checks them: a key outside the plan is `ErrUnknownRun`, a key twice with
different bytes is `ErrDuplicateRun` (the same record twice is accepted),
and a key absent is `missing`, not an error (a sweep may be partial). The
`Result` is therefore the same bytes whether the records came from `Run`, from
eight goroutines or from three machines in any order (FR-EXP-022). A record is
**trusted**: `Assemble` does not rerun it and cannot tell a forged one; the
digest in it is the consumer's means of checking a record against a rerun by one
library version (CHK 8.5: bit-identity is between two worlds of one library
version).

5.3.5. **Memory.** A unit holds one world (and one checkpoint with a branch,
about 2 MB at the first consumer's size, CHK 10.2) while it runs and nothing
after it returns. A pool of `W` workers holds `W` worlds. The records are small
(8.6).

### 5.4 Cancellation

5.4.1. **Where the context is looked at.** `Run` and `RunUnit` check `ctx` before
each unit, and inside a run `w.Run(ctx, n)` checks it before every tick
(`spec-world.md` 3.4.1, architecture.md 12.3), so a cancel takes effect within one
tick and leaves the world at a tick boundary.

5.4.2. **What a cancelled run is.** A run that was in flight when the context
ended is **discarded**: no record is produced for it, nor for the rest of its
unit, because a record cut at a tick that depends on when a signal arrived would
make the result depend on timing. What was complete is complete: the records of
earlier units are kept (5.2.3), and with the same `WithPrior` the sweep resumes.
`RunUnit` returns the records of the keys it finished (none, if it was cancelled
before the unit's end) and the context's error wrapped with `%w`.

5.4.3. **Not a failure.** A cancellation is not a world failure (`spec-world.md`
3.5.5) and does not count in `Failed` (9.3).

### 5.5 Options and extensions of a run

5.5.1. `Options(key)` returns the options of one world: the registrations of its
strategies, spawner and clearing (`strategy.Options()...` for the merchant), a
logger, an audit, its own observers, `WithTargetSpread`. It **must return neither
`WithSeed` nor `WithReseed`**: the harness owns the randomness of every world
(5.1.3 step 2, 6.3), and a restore refuses `WithSeed` with `ErrOptionContext` (CHK
6.4), so a branched unit whose `Options` carried one would record `world` for
every branch. The harness cannot strip an option it cannot see.

5.5.2. **When it is called.** `Options` is called **once for each world built**:
once for a run without a branch (the key of the run), and with a branch once for
the burn-in (the key of the unit with variant 0 and continuation 0) and once for
each restore (the key of that branch), so that every world gets its own instances
of the stateful extensions. A nil `Options` means no options: the default
strategies and a world with no logger and no observer of the consumer's.

5.5.3. **What is in a hash.** `WithTargetSpread` is a datum of the effective
configuration and is in the configuration hash; a registered name (`WithStrategy`,
`WithSpawner`, `WithClearing`) is in it and **the code behind the name is not**
(`spec-world.md` 4.1, `spec-config.md` 8.1); `WithSeed` and `WithAudit` are not.
To **sweep** an option, put it in a factor (a document or a path) rather than in
`Options`, which sees the key and not the level; to sweep the behaviour of a
strategy, register it under a distinct name per level in `Options`, keyed by a
document of the level that names it. A factor on `target_spread` is overridden by
a `WithTargetSpread` in `Options`, which is the consumer's mistake and which the
harness cannot see.

5.5.4. **A restore** (6) accepts the registered names of the world that was
checkpointed (CHK 6.2 step 4) and the same `Options` rules.

### 5.6 What observing costs a sweep

A sweep whose measures all read snapshots needs no observer, and a world with no
observer constructs no event (observability-and-logging.md 4.5): a run then costs
the ticks, one snapshot (about 1.1 ms at the first consumer's size, a
placeholder, OBS 5.5) for each window and one checkpoint at the end (about 13
ms, CHK 10.3, which is also the digest of every record). A measure with an
`Observer` brings in the consumer's observer (a tally sums a few hundred rows and
ignores the order and trade events that are about 99% of the stream, OBS 8.3), at
a placeholder of 6% of a tick. `Pair` adds one checkpoint per window and no
observer. With a
branch every restore also builds `WorldRestored` with a full `Snapshot` when any
observer was supplied (CHK 6.7.3) and logs the restore once (CHK 14.2). The sums
are in section 10.

## 6. Branching

### 6.1 The value

```go
// A Branch runs each unit's burn-in once and continues it in several ways.
type Branch struct {
	At            int64     // the tick of the checkpoint, 1 .. Ticks-1
	Variants      []Variant // at least 1
	Continuations int       // reseeded continuations of each variant; 0 or 1: none
	Salt0         uint64    // the first salt; continuation c uses Salt0 + c (mod 2^64)
}

// A Variant is a set of calls applied from the branch point.
type Variant struct {
	Label string
	Steps []Step // each with Tick >= At
}
```

6.1.1. **The shape.** For each (point, replica) the unit runs ticks `[0, At)`
once, takes a checkpoint, and then for each variant and continuation restores the
checkpoint and runs `[At, Ticks)`. A record exists for each (variant,
continuation), and carries the windows of the burn-in as the first part of its
series. The harness calls `Observer(branchKey)` for the restored world and then
`Fork(burnInKey, branchKey)` of each measure that has one, so the consumer copies
what its observer of the burn-in accumulated (a tally is plain data, IND 10.5.1);
the windows open at the branch point stay open. A branch's series is then the same
as an uninterrupted run's would be. A restore emits `WorldRestored` first (CHK
6.7.3) to the observers given to the restore; what a consumer's tally does with it
is IND's, and IND 4.2.5 (a tally fed `WorldRestored` while open starts a new
window) and IND 10.5.1, FR-IND-055 (a restore changes nothing in a straddling
window) say different things: the contradiction is routed (22), and a measure that
needs the straddling window to continue ignores `WorldRestored` in its observer.

6.1.2. **Steps.** `Experiment.Script` and the levels' steps with a tick below
`At` are applied in the burn-in; those at or above `At` and the variant's are
applied in the branch (a step at `At` right after the restore, in the order of
3.5.2). A variant step with a tick below `At` is `ErrBadBranch`.

### 6.2 What a branch can change

6.2.1. **Calls, and the reseed. Not the configuration.** `RestoreWorld` runs
`Validate` on the configuration it is given and compares its hash with the
checkpoint's (CHK 6.2 step 3); a different hash is `ErrCheckpointConfig`.
`WithTargetSpread` is a datum of the effective configuration and is in that hash; a registered name is in it and the code behind it is not (WLD 4.1).
So a branch varies the **sequence of calls** (a shock, a player, a closed road, a
stipend raised: INT 9.1) and the randomness, and nothing in the configuration. A
parameter sweep is therefore a sweep of **units**: each point has its own
burn-in, shared by that point's variants and continuations (FR-EXP-030).

6.2.2. **Why this is not a limitation worth removing.** A parameter changes what
the world is; a world restored under another configuration would be a state no
run produced. The saving a branch gives is the burn-in shared by the **shocks**
applied to one world, which is what a stylised-fact test of a shock response
needs (S01, S04, S14, S28, S61).

### 6.3 Randomness of branches

| `Continuations` | The restore | What the branches are |
|-----------------|-------------|-----------------------|
| 0 or 1 | no `WithReseed`: the stream continues from the checkpoint | the variants of a unit use the **same** words from the same position: the pairs are as aligned as the difference of their calls allows (7) |
| `C >= 2` | `WithReseed(Salt0 + c)` (mod 2^64) for `c = 0 .. C-1`, the same salt for every variant | the continuations of one variant are independent of each other; the variants of one continuation are paired |

6.3.1. **A reseed is deterministic** (CHK 6.8): a branch is reproduced from
(checkpoint, salt, calls), and the identity of a run is the key of 8.1.3. The salt is a datum of the experiment, never drawn.

6.3.2. **What continuations estimate.** Several continuations from one
burn-in are samples **conditional on that state**: their spread is the spread of
the response from that state, not of the world's unconditional distribution
(CHK-A2 is the hazard of treating them as independent replicas of a run from tick
0). For the unconditional distribution use replicas with distinct seeds, each
with its own burn-in; the two are different questions and both are offered.

### 6.4 The equality that makes branching safe

6.4.1. A branch with **no steps and no reseed** reaches the same final digest
as the uninterrupted run of the same key (FR-CHK-005, FR-CHK-050,
`spec-world.md` FR-WLD-063), so the burn-in and its checkpoint add no
information and no noise (FR-EXP-031). It is the test of the unit of 5.1 against
section 6, and a consumer who doubts a branched result reruns one branch from
tick 0 with the same calls and compares digests.

6.4.2. **Observers and extensions across the restore.** A measure's observer is
new in each branch and starts from the state `Fork` copied; `Options(key)` supplies
new instances of the stateful extensions, whose state `LoadState` takes from the
checkpoint by name (CHK 6.2 step 11). A consumer's observer sees the burn-in's
events in the burn-in world and the branch's in the branch (a restore emits
`WorldRestored` first, CHK 6.7.3).

### 6.5 Cost

Branching saves `(K - 1)` burn-ins for `K` branches and costs one checkpoint per
unit and one restore per branch (placeholders of 13 ms and 25 ms at the first
consumer's size). With a burn-in of a third of the run and ten branches the
saving is about 30% (cost.out: 10 minutes without and 7 with, for ten branches of
the five-region pack at 53 ms a tick), and with two thirds and twenty branches
about 63% (19 minutes without and 7 with).
It is worth it when `At` is a large share of `Ticks` and the variants are many.

## 7. Paired runs

### 7.1 What a pair is

A **pair** is two runs of the same seed whose difference the consumer wants to
attribute to one thing: two points of a `SameSeeds` experiment (a parameter), two
variants of a unit (a shock against none), two continuations that share a salt,
or one run observed and unobserved. The comparison is by replica index: replica
`r` of A against replica `r` of B.

### 7.2 Aligned, and what it is worth

7.2.1. **The stream is positional** (`spec-world.md` 7.6, WLD-A5): the word at
position `k` of the stream goes to whatever consumes the `k`-th word in that run.
Two runs of one seed use the same word at the same position. They **draw the
same word for the same decision** only while they have drawn the same number of
words since the start, because the count of words in a tick is a function of the
configuration and the state (`spec-world.md` 7.4): the intents considered, the
orders admitted, the agents created, the households pushed.

7.2.2. **Aligned through a boundary** means the two runs have drawn the same
number of words at every earlier boundary of the run. Equal counts are a
**necessary** condition of drawing the same word for the same decision and not a
sufficient one: WLD 7.4.1 makes the count and also "the order of every one" of the
draws functions of the configuration and the state, so a difference may add words
at one position of a tick and remove as many at a later one, and the tick-end
count is then equal while every word between goes to a different decision. The
fingerprint (7.3) can therefore **show** that two runs have diverged and cannot
**prove** that they have not; an aligned prefix as reported is an **upper bound**
of the prefix in which the two runs consume identical randomness decision by
decision (**common random numbers** in the exact sense). After the first boundary
at which the counts differ a position is used by a different decision in the two
runs, and from then on the pairing is only partial.

7.2.3. **What is aligned by construction.** The same configuration with other
observers, a logger, `WithAudit`, a split of the run into `Run` calls, a
checkpoint and restore (`spec-world.md` 7.4.1; FR-WLD-063, FR-WLD-064): the whole
run. For a **different parameter** or a **shock**, the prefix lasts until some
state that feeds a count first differs; nothing promises that is more than a few
ticks, and the model of 18 shows a parameter that is aligned for the first
sample only. That is a property of the world and not a defect of the harness.

### 7.3 How the harness checks it

7.3.1. **The fingerprint.** With `Pair`, the harness records at each window
close (and at the branch point) the **stream fingerprint**: the 16 bytes of the
state of the world's random source at that boundary. The state of a PCG is a
bijection of the number of words drawn since the seeding, over a period of 2^128,
so two worlds of one seed (and one reseed) have equal states **if and only if** they
have drawn the same number of words. The fingerprint is not a measure of the economy and is
never a measure's input.

7.3.2. **Where it comes from, and what is asked.** The public surface has no read
of the state of the source at a boundary: `CheckpointInfo` has five fields and
`Info` "reads only the frame" (CHK 3.1.2, 5.4.3, FR-CHK-006), and the state is in
section 1 of the body. This spec **does not rely** on a read that does not exist.
It asks CHK for one of two forms, and the harness uses whichever CHK adopts:
(a) `Info` decodes the first 20 bytes of section 1 and gives `Stream [16]byte`,
the 16 bytes that follow the `"pcg:"` prefix of the `MarshalBinary` of the source
(an `Info` that decodes a section needs an error result, which it does not have
today); (b) a second read of CHK's naming that returns the same 16 bytes. A copy
of the state in the header was considered and costs a format change and a
validation. Parsing format 1 in the harness was rejected: it would tie the harness
to a byte layout that CHK 9.1 allows to change before 1.0. **Without either,
`Pair` is refused at `Validate` with `ErrBadSchedule`**, the alignment is unknown,
and the pairing gain of 7.4 is the whole measure (open question 4). The digest
needs no field: it is the last 32 bytes of the `MarshalBinary` of the checkpoint
(CHK 5.4).

7.3.3. **The check.** Two runs are **aligned through sample `k`** if their
fingerprints are equal at samples `1 .. k`; the **aligned horizon** is the end
tick of the last such sample (`FirstDivergence` is the first sample at which they
differ), an **upper bound** by 7.2.2. The resolution is a sample: a run that
draws two words more in one tick and two fewer in the next has equal counts at the
sample and different words in between, and the check does not see it. A window of
one tick (`Every` 1) narrows the check to the counts of every tick and still does
not prove the decisions equal. The check costs one checkpoint (placeholder 13 ms)
per window (10).

### 7.4 What pairing is worth: the estimate and the gain

7.4.1. **The estimate does not need alignment.** Replicas are independent across
seeds, so the pairs `(A_r, B_r)` are independent across `r`, whatever the
correlation inside a pair. The mean of `d_r = B_r - A_r` is an unbiased estimate of
`E[B] - E[A]` and the sample standard deviation of `d_r` over `sqrt(n)` is a valid
standard error of it, aligned or not. WLD 7.6.1 ("a paired
comparison is valid only between runs that differ in a parameter that changes no
draw count") and WLD-A5 ("valid only for a change that moves no draw count") are
true of **exactness** and of the **variance reduction** and not of the estimate;
CHK 11 item 3 and IND 11.2 ("is valid when the difference moves no draw count")
state a sufficient condition and need no change but a word. The harness reports
alignment and the gain and does not forbid the comparison. (CHK 11 item 3's "one
parameter ... differing" cannot occur between two restores of one checkpoint: a
different configuration is `ErrCheckpointConfig`, 6.2.1.)

7.4.2. **The pairing gain.** For two groups with the sample variances `vA`, `vB` of
their used final readings, each rounded half to even as in 8.3, and `sd` the
sample standard deviation of `d_r` (8.3, `isqrt_nearest` of the rounded variance),
the **pairing gain** is

```
sd_indep = isqrt_nearest(vA + vB)
gain_bp  = rdiv(10 000 * sd_indep, sd)
```

the ratio of the spread of the difference of two **independent** runs to the
spread of the **paired** difference, in basis points (10 000: no gain; `OK` false
for `sd` 0, with `Identical` set when every `d_r` is 0). `Compare` computes it
from the readings of the records the `Result` holds, not from `Std`, and reports
`SdIndep` and `SdDiff` beside it. Variances and roots are rounded integers: when
the spread is small in the measure's own unit the gain is coarse (a `sd` of 2
against 3 moves it by half), so a measure whose spread is a few units should be
stated in a finer unit (basis points of a base, not whole units). The gain is
**measured**, not claimed.

7.4.3. **What the model shows** (`models/exp/pairing.py`, which implements 7.4.2;
a toy world of placeholder dynamics, 200 seeds): with identical settings the paired
difference is exactly 0 (`sd` 0 against 10 for two independent runs; 200 of 200
seeds aligned to the end); with a setting that moves the count of words rarely and
by one word the gain is 50 000 bp (a `sd` of 2 against 10) although only 7 of 200
seeds are aligned to the end, because a shift of one word is still mostly the same
words, and 30 000 and 20 000 bp for larger parameter steps; in a world where the
count moves by many words the gain is 12 857 bp (`sd` 7 against 9) and 11 429 for
the largest step. So the gain decays with the offset between the two streams and is
neither 10 000 nor unbounded: a consumer reads the number. Nothing here is a claim
about the library's worlds (N-027).

### 7.5 Where pairing is most useful

An **impulse response** (a shock against none, from one checkpoint, same salt): the
aligned prefix is the window over which the difference of the two series is the
effect of the shock on identical noise, and its end is a result in itself (how
many ticks the shock took to move a count). After it, compare **distributions
over replicas or continuations** (8.3, 8.4): the quantiles of each, not the
difference of two paths.

## 8. Results and statistics

### 8.1 What is recorded

8.1.1. **A record** is one run:

```go
type Reading struct {
	Value int64
	OK    bool // false: undefined (a zero base, an empty set), and Value is 0
}

type Record struct {
	Key      Key
	Seed     uint64
	Salt     uint64      // 0 with no reseed; HasSalt says which
	HasSalt  bool
	Hash     [32]byte    // World.ConfigHash of the world built (a restore: the checkpoint's); zero if none was built
	Status   Status      // 9.2
	FailTick int64       // the tick of the failure; -1 if none
	Detail   string      // the error text of a failure, or the first measure error
	Ticks    []int64     // the end tick of each window that closed
	Final    []Reading   // one per measure, over [Transient, Ticks)
	Pass     []bool      // the verdict of each criterion on this run, in criterion order
	Series   [][]Reading // [window][measure]; with Keep.Series
	Stream   [][16]byte  // [window]; with Pair
	Digest   [32]byte    // the digest of the checkpoint at the end; zero if failed
	Snapshot economysim.Snapshot // with Keep.Snapshot (HasSnapshot says which)
	HasSnapshot bool
}
```

8.1.2. **Events summarised** are the tallies' sums: whatever IND's presets read
from a window. The harness keeps no event, no order and no trade.

8.1.3. **The key of a result** is `(Hash, Seed, Variant, Salt)` together with
the experiment's script and window settings (`Ticks`, `Transient`, `Every`) and the
measures: the configuration hash does not cover the seed (WLD 4.1), nor the code
behind a registered name (WLD 4.1, CFG 8.1), nor the library and format version
(CHK 8.5), nor the length of the run. Two records with equal keys, the same
extension code and one library version are the same run (NFR-WLD-004), so a
consumer may skip a second run; the library does not (open question 5). A sweep
that varies the behaviour of a strategy registers distinct names (5.5.3).

### 8.2 The result as plain data, and its canonical rendering

8.2.1. **The result** is a tree of plain structs, in canonical order, with no
map, no pointer, no function, no float and no time (architecture.md 12.4 asks for
plain data with no pointer into live state and no function field; the rest is this
spec's choice):

```go
type Result struct {
	Format   int // 1
	Name     string
	Measures []string
	Criteria []string
	Points   []PointResult
}

type PointResult struct {
	Index    int
	Levels   []int     // per factor: the level index, or the cell of a spread
	Values   []int64   // per factor: the value of a range factor, else the level index
	Hash     [32]byte  // of the first of its runs that built a world, else zero
	Status   Status    // ok, unresolved or invalid_config (the point's resolution)
	SameAs   int       // the first earlier point with the same hash, else -1
	Warnings []Warning // the rules of 14.4 that applied to this point
	Variants []VariantResult
}

// A Warning is a rule name and a path, as in spec-config.md 7.2.
type Warning struct {
	Rule string
	Path string
}

type VariantResult struct {
	Variant    int
	Runs       []Record
	Reductions []Reduction // one per measure
	Pass       []PassShare // one per criterion
}
```

8.2.2. **`Render(r Result) string`** is the canonical text **of the readings and
verdicts** of a result: UTF-8, LF, integers in decimal, a `hash` and a `digest` as
the first 8 bytes (16 digits) in lower-case hex, a `stream` fingerprint as all its
16 bytes (32 digits), the order of the tree, `-` where a value is undefined, no
float and no time. It omits `Snapshot` and the error text of a run beyond one
`detail` line. It is **not** the event rendering of OBS 9.5.2 (one line per event);
OBS 10.3's phrase "the format of a diff between two runs of a sweep" is to read as
this format (owed row, 22). It is the form that two sweeps are diffed in and that a
golden test compares, and it is exported because a consumer needs it for the same
reason (architecture.md 2.8: an export that a consumer uses; OBS 9.5.3 left the
renderer of events to test code and said EXP may ship one). The lines, in order:

```
experiment <Name> format <Format> points <n> measures <m> criteria <c>
measure <k> <Name>
point <Index> hash <16 hex|-> status <code> same_as <n> levels <l,l,..> values <v,v,..>
warning <Rule> <Path>                            (zero or more)
variant <v>
run replica <r> cont <c> seed <s> salt <s|-> status <code> fail_tick <t> digest <16 hex|->
detail <text with line breaks replaced by a space>   (only when non-empty)
final <Measure>=<value|-> ...
verdict <Criterion>=<pass|fail> ...
window <end tick> <Measure>=<value|-> ...        (with Keep.Series)
stream <end tick> <32 hex>                       (with Pair)
reduce <Measure> planned <n> used <n> failed <n> undefined <n> missing <n> min <v|-> max <v|-> mean <v|-> std <v|-> q<bp>=<v|-> ...
pass <Criterion> passes <n> planned <n> share <bp>
```

A `-` is printed for `min`, `max`, `mean` and every quantile when `Used` is 0, for
`std` when `Used` is below 2, for `hash` and `digest` when zero because nothing was
built or the run failed, and for a reading with `OK` false. (Render is a pure
function and the golden form of an experiment, 11.)

### 8.3 Reductions over replicas

For one (point, variant) and one measure the **group** is the records of its
replicas (and continuations). The arithmetic is the harness's own (`internal/reduce`,
3.4.0, 8.3.2): 128-bit sums and the one rounding rule of IND 5.2.2, half to even;
no float (FR-EXP-040).

| Field | Definition |
|-------|------------|
| `Planned` | `max(1, Continuations)` times replicas of the experiment, for the group |
| `Used` | records with status `ok` and a defined final reading |
| `Failed` | records with a status other than `ok` or `missing` (9.3) |
| `Undefined` | records with status `ok` and an undefined reading |
| `Missing` | planned records absent from the assembly |
| `Min`, `Max` | of the used readings |
| `Mean` | `rdiv(S, Used)`, `S` the sum (128-bit) |
| `Std` | the sample standard deviation, `isqrt_nearest(rdiv(Used * SS - S^2, Used * (Used - 1)))` in 128 bits, `OK` false for `Used` below 2; two roundings (the variance, then the root), a named exception of IND 5.2.3's kind like Fisher's |
| `Q(q)` | the nearest-rank quantile (IND 5.5.2) of the used readings at `q` in basis points, for the closed list 1 000, 2 500, 5 000, 7 500, 9 000; no interpolation, so a quantile is a value some run produced |
| `Pass` | for a criterion: the runs whose reading is defined and in the band, over `Planned`: `rdiv(10 000 * passes, Planned)` |

8.3.1. **Failed runs are never dropped.** `Planned = Used + Failed + Undefined +
Missing`, always (FR-EXP-041), and a pass share is over `Planned`, so a group
in which three of ten runs failed passes at most 7 000 bp. A mean over the used
runs of a group with failures is a **survivor** mean and the result says so by
`Failed` beside it (EXP-A4, 17): the harness gives the two numbers together and
never one without the other.

8.3.2. **Bound and widths.** A reading has `|value|` at most `MaxReading` (13.2);
one beyond it is an error of the measure (`ErrRange`): the reading is undefined and
`Detail` holds the first such error. The harness's arithmetic is its own, in
`internal/reduce` (23), over `math/bits`, with one rounding rule, half to even, the
same as IND 5.2.2. The widths, each checked in the model (`reduce.py`):

| Quantity | Bound | Width |
|----------|-------|-------|
| `S`, the sum of readings | `10^6 * 2^40` | 128 |
| `SS`, the sum of squares | `10^6 * 2^80` | 128 |
| `Used * SS - S^2` | below `2^120` | 128 |
| a variance, `vA + vB` | below `2^82` | 128 (the root is a 128-bit integer root of this spec's own, not IND's 64-bit one) |
| `10^6 * 2 * tail` of the sign test | `2^83` at `n` 62 | 128 |
| `Max - Min` and `k * (Max - Min)` of 3.2.3 and 4.4.3 | `2^64` and `2^64 * 2^10` | 128, so no int64 subtraction wraps |
| the elasticity numerator | `10 000 * (y_l n_0 - y_0 n_l) * x_0` | checked 128-bit product; an overflow makes it undefined (`OK` false, `Clamped`) |

The cap of 62 on the exact sign test is a **choice**, so that `2^n` fits an
`int64`; the numerator already needs 128 bits there.

### 8.4 Comparing two points (or two variants)

```go
// A Group is the runs of one variant of one point: the unit of a reduction.
type Group struct{ Point, Variant int }

func Compare(r Result, a, b Group, measure string) (Comparison, error)
```

8.4.1. **Matching.** For `SameSeeds` (and the same continuations) the records
of the two groups are matched by replica index, and only pairs in which both are
used enter (`Pairs`). The comparison of the groups of two **different points** of a
`PerPointSeeds` experiment is **unpaired**: `Pairs` is 0, `MeanA` and `MeanB` are
the group means and the wins and the tail are not offered (`OK` false). Two
variants or continuations of **one point** share their seeds under either rule and
are matched by replica and continuation.

8.4.2. **Fields.**

| Field | Definition |
|-------|------------|
| `Pairs` | matched pairs with both readings defined |
| `MeanA`, `MeanB`, `DiffOfMeans` | the means of all used readings of each group (matched or not), and `DiffOfMeans = rdiv(S_B * n_A - S_A * n_B, n_A * n_B)` from the exact sums, one rounding: the estimate for an unmatched comparison |
| `MeanDiff` | `rdiv(sum d_r, Pairs)`, `d_r = B_r - A_r` |
| `MedianDiff` | the nearest-rank quantile of `d_r` at 5 000 |
| `Wins`, `Losses`, `Ties` | pairs with `d_r` above, below and equal to 0 |
| `SignTailPPM` | the exact two-sided tail of the sign test, ties dropped: `min(10^6, rdiv(10^6 * 2 * sum_{j >= m} C(n, j), 2^n))` with `n = Wins + Losses`, `m = max(Wins, Losses)`; `OK` false for `n` 0 or above 62 (a cap of choice, 8.3.2; use the quantiles) |
| `RelativeBP`, `Clamped` | growth, from the exact sums in one division as IND 5.2.3 asks: `rdiv(10 000 * sum d_r, sum A_r)` over the same pairs (for an unmatched comparison `rdiv(10 000 * DiffOfMeans-numerator, S_A * n_B)` likewise), clamped to `+-(2^31 - 1)` with `Clamped` set (IND 5.3.7), `OK` false for a base at or below 0 (IND 5.2.4) |
| `Dist` | for each quantile of the closed list of 8.3, the nearest-rank value in each group and their difference: the comparison of **distributions over replicas**, offered for matched and unmatched groups alike (the form for a knob that moves a draw count, 7.4.1) |
| `AlignedPairs` | with `Pair`: pairs whose fingerprints agree at every window and at the end (7.3) |
| `Horizon` | with `Pair`: the median over pairs of the aligned horizon, in ticks, an upper bound (7.3.3) |
| `PairingGain`, `SdIndep`, `SdDiff`, `Identical` | 7.4.2, in basis points, `OK` false when `SdDiff` is 0 |

8.4.3. **What it does not give.** A confidence interval of the mean difference
needs the quantile of a distribution of a rate of a square root (a float, or a
table); the harness gives `Std` and `Pairs` and the sign test, and a consumer
that wants the interval computes it outside the core (architecture.md 3.5).

### 8.5 A sensitivity measure that needs no float

8.5.1. **Main effects by bins.** (`MainEffects(r, measure, factor, bins)` and `Elasticity(r, measure, factor, level)`, names provisional.) For a factor and a measure, partition the points
by the factor's setting: by level index for a grid, by `floor(cell * B / N)`
into `B` bins for a spread (`B` at most `N`, at least 2, a datum of the call), and
for one at a time by level (the baseline and the points of this factor). The
**bin mean** is the mean of the used final readings of all runs of the points in
the bin, pooled from the exact sum and the count in one division
(`rdiv(sum, count)`), never a mean of rounded point means (IND 5.2.3). The **effect range** of the
factor is `max - min` over the non-empty bins (at least 2 needed) and its
**share** is `rdiv(10 000 * Range_f, sum over factors of Range_g)`.

8.5.2. **Reading it.** The range is in the measure's own unit and a share is
the factor's part of the total range, so a screening reads "which factor
moves the measure most, in the units of the measure" with no assumption of
linearity or of a distribution. It is not a variance decomposition: two factors
that cancel in a bin look small, and a spread's bin means are confounded by the
correlation between columns (4.4.4), by a correlation of about 0.1 for a typical pair and up to about 0.3 for the
worst of the pairs at `N = 64` and 8 factors (4.4.4). The
model of 18 checks the arithmetic on a 3 by 3 grid where a factor of range 40
has a share of 8 333 bp and one of range 8 has 1 667.

8.5.3. **Elasticity.** For a one-at-a-time design, the elasticity of a measure to
a factor at level `l` against the baseline is, from the exact sums `Y_l`, `Y_0` of
the used readings of `n_l` and `n_0` runs,

```
rdiv(10 000 * (Y_l * n_0 - Y_0 * n_l) * x_0, Y_0 * n_l * (x_l - x_0))
```

in basis points (10 000: a 1% rise of the knob moves the measure by 1%), one
rounding, `OK` false for `Y_0` at or below 0, `x_0` 0 or `x_l = x_0`, and
`Clamped` at `+-(2^31 - 1)` (8.3.2 for the width); `x` is the value of the factor's
first change. It is the finite-difference elasticity at the baseline, valid for a
small step, and says so.

### 8.6 The size of a result

8.6.1. **The formula.** A result holds `points * variants * replicas *
continuations` records. Each holds `measures` final readings and, with
`Keep.Series`, `windows * measures` more, each a `Reading` of 9 bytes packed (an
`int64` and a flag) and 16 as a Go struct, a digest and, with `Pair`, 16 bytes a
window. A result of 1 000 points, 10 replicas, 20 measures and 36 windows is 3.2 MB
of final readings as structs and about 118 MB with the series (about 67 MB packed;
cost.out). The snapshot of a record is bounded by the configuration and not by the
agents, **except** the in-flight sets (shipments, journeys, contracts, OBS 5.6.1):
about 300 rows and 60 KB at the first consumer's size (OBS 5.5.2), so
`Keep.Snapshot` over 1 000 runs is about 60 MB.

8.6.2. **The bound.** The number of cells (runs times measures times windows with
the series) is checked before anything runs (13.2); a sweep that would exceed it
is `ErrOversize`, and the way out is a coarser `Every`, fewer measures, final
readings only, or a record callback that stores each record and an assembly of
reductions the consumer writes from the stored records.

## 9. Failure

### 9.1 What can fail

A sweep runs many worlds, some with settings nobody has tried. Its rule is that
one failed run does not stop the sweep, does not vanish from it, and does not
leave a half-record that looks complete.

| What failed | Code | The world |
|-------------|------|-----------|
| a document did not parse or merge: an unknown key, a float, a scalar over a table, a path of the wrong syntax | `unresolved` | none built |
| `Prepare` returned an error, or `Validate` of the resolved configuration returned errors, or `NewWorld` returned a `ValidationError` of its name checks | `invalid_config` | none built |
| `NewWorld` returned `ErrAuditFailed` (with `WithAudit`, `spec-world.md` 3.2.2); `FailTick` 0 | `audit` | none |
| `NewWorld` or `RestoreWorld` returned any other error, **whatever it wraps**: an option, a nil extension, a refused restore, `ErrCheckpointConfig`, `ErrCheckpointCorrupt` (also when it wraps `ErrAuditFailed`, CHK 6.2 step 10) | `world` | none |
| any error out of a `Step.Do`, whatever its type (even `ErrAuditFailed` from a `w.Audit()` the step called on a healthy world) | `script` | healthy; the calls the step made before the error stay applied (3.5.3) |
| a tick (`w.Run`) returned `ErrInvalidTrade` | `invalid_trade` | failed (`spec-world.md` 3.5) |
| a tick returned `ErrAuditFailed` | `audit` | failed |
| `Checkpoint` returned an error on a healthy world (an extension's `SaveState`, FR-CHK-032; CHK names no sentinel) | `checkpoint` | healthy, dropped |
| a tick returned any other error or `ErrWorldFailed` (WLD 3.3.3 says no other condition is an error of `Step`: this row is defensive, so that a later error is classified and not lost) | `step` | failed |
| the burn-in of a unit failed: its branches inherit the burn-in's status and tick | as the burn-in | |
| the context ended | not recorded (5.4) | healthy, dropped |
| a record absent from an assembly | `missing` | |

The word "tick" in the rows above is a call of `w.Run`; "step" of a script is
`Step.Do`. The code is a function of the error and of where it came from, so two
implementations render one `Render`.

9.1.1. **A measure error is not a run failure.** A measure that returns an
error leaves that reading undefined (`Detail` holds the first, with the measure
and the window) and the run continues: the world is fine and its other readings
are valid. The group's `Undefined` counts it.

9.1.2. **A panic is not recovered.** The library does not recover a panicking
observer, strategy, spawner, clearing, measure or step (architecture.md 12.5 for
observers, `spec-world.md` 3.5.4 for all of them): it propagates to the caller of `RunUnit` and `Run`. A
consumer whose extensions might panic wraps the call to `RunUnit` in its own
`recover` and records `step`.

### 9.2 The closed set

`Status` is one of: `ok`, `unresolved`, `invalid_config`, `world`, `script`,
`invalid_trade`, `audit`, `checkpoint`, `step`, `missing` (13.4). A record with a status other
than `ok` has `FailTick` the tick of the failure (the tick of a failed step, the
tick of a refused call, 0 for a point that never built a world) and `Detail` the
error text. Its `Final` is **all undefined** (`OK` false): the whole span did not
complete. `Ticks` lists the windows that closed before the failure, and with
`Keep.Series` its `Series` holds their readings; without `Keep.Series` a failed run
keeps no reading at all (a consumer that wants the partial series of failures sets
`Keep.Series`). Its `Pass` is all false.

### 9.3 How a failed run counts

9.3.1. A failed run is in `Failed` of every reduction of its group and is **not**
in any `Used`: its partial series is kept for reading and is never reduced.
It does not pass a criterion. A point whose configuration did not resolve has
all its records with the point's status and the point's `Status`
(`PointResult.Status`); its reductions are all `Failed`.

9.3.2. **An all-failed group** has `Used` 0: every field but the counts is
undefined, and `Compare` of it returns `OK` false.

### 9.4 The failed-world rule

A tick that returned an error leaves the world failed (`spec-world.md` 3.5): the
harness reads `Err()`, takes nothing else from it (no snapshot, no checkpoint,
which would be refused), and drops it. It never retries a failed tick and never
continues the world. With `Keep.Series` the record keeps the readings of the
windows that closed before, which are valid because they were taken at healthy
boundaries.

### 9.5 What the sweep does next

The sweep goes on to the next unit. Whether many failures should stop a sweep is
a policy: the record callback sees every record in plan order, and a consumer
that wants to stop on the first failure or on a share of failures cancels the
context from it; the library has no threshold (open question 8).

## 10. Cost and budget

10.1. **What is counted.** The library measures no wall-clock time
(architecture.md 7.3), so cost is a **count** or a placeholder multiple of the
tick budget of `spec-scale.md` 6, and nothing here is measured (N-015). The
sizes: the first consumer's world, five regions, 5 000 agents, 30 commodities,
at about 53 ms a tick (`spec-scale.md` 6.3), and the `one` pack of
`spec-reference-chain.md` at about 4 ms a tick (8.2 there). The additions of a
sweep are the placeholders of OBS 5.5 (a snapshot, 1.1 ms), CHK 10.3 (a
checkpoint 13 ms, a restore 25 ms) and OBS 8.3 (an observer, 6% of a tick).

10.2. **One run, no observer, no windows** (`models/exp/cost.py`):

| Pack | Ticks | One run | Runs an hour, one core |
|------|------:|---------|-----------------------:|
| `one` | 400 | 2 s | 2 230 |
| `one` | 1 095 | 4 s | 820 |
| `five` | 400 | 21 s | 170 |
| `five` | 1 095 | 58 s | 62 |
| `five` | 3 650 | 3 min | 19 |

(At the placeholder sizes one run of the five-region pack of 1 095 ticks is about 0.017 runs a second and a run of `one` of 400 ticks about 0.6.)

A run with an observer costs 6% more, a window one snapshot, `Pair` one
checkpoint a window and the end one checkpoint. The resolution of a point is the
parse of the base and the overlays (`O(bytes)`) and a `Validate`, with the
viability programmes when they are on (NFR-CFG-001); it is paid once for each
unit and is unmeasured.

10.3. **A sweep** is `points * replicas * variants * continuations` runs.
With one core, an observer and a window of 30 ticks:

| Pack | Points | Replicas | Ticks | Runs | Total |
|------|-------:|---------:|------:|-----:|-------|
| `one` | 3 | 10 | 400 | 30 | 52 s |
| `one` | 24 | 10 | 1 095 | 240 | 19 min |
| `one` | 100 | 10 | 1 095 | 1 000 | 78 min |
| `five` | 12 | 10 | 1 095 | 120 | 2.1 h |
| `five` | 100 | 10 | 1 095 | 1 000 | 17 h |

So a scenario of STY (a grid of 3 to 24 points on `one`, ten seeds) is minutes
on one core, and a calibration of a hundred points on the five-region pack is a
day on one core and a coffee on many (`RunUnit` is the unit a consumer fans out,
5.3). The cost is linear in runs and in ticks and does not depend on the number
of households (FR-SCL-016).

10.4. **Branching** (6.5) saves the shared burn-in: 29% on `one` and 30% on
`five` for a burn-in of 365 of 1 095 ticks and ten branches, 63% for 730 ticks
and twenty.

10.5. **The harness's own cost** is a count, and none of it is quadratic in
agents or ticks: `Plan` is `O(runs)`; `Assemble` sorts the records,
`O(R log R)`; the reductions are `O(R * measures)` and a quantile a sort of
the group; a measure is a function of a snapshot (about 300 rows and 60 KB, OBS 5.5.2) and
a consumer's tally (about a thousand rows and 30 KB, IND 4.2.4). Beside a run of a few hundred ticks
it is under 1% (NFR-EXP-003).

10.6. **What is skipped under `-short`.** The tests that build a world are
integration tests and begin with the `testing.Short()` skip (testing.md 10.5;
section 11). The unit tests of the design, the seeds, the overlay, the
reductions, the comparison, the sensitivity and the rendering run in every
`go test -short`. The scenarios of STY that use the harness are STY's and skip as
`spec-reference-chain.md` 7.1.6 says.

## 11. Testing the harness

### 11.1 The pyramid

| Level | What | Examples |
|-------|------|----------|
| Unit (the large majority; `internal/design` and the reductions) | pure functions of integers; table-driven; every one fast and run under `-short` | the Feistel permutation of every `N` from 1 to 300 and the larger ones of the model, the stratification of every axis, the mixed-radix index and its inverse, the OAT order, the range values, the overlay bytes, the seeds, `rdiv` half to even, `isqrt_nearest`, the nearest-rank quantile, the sign tail against exact fractions, the effect range, the elasticity, the bounds with overflowing products, `Render` of a hand-built `Result` |
| Integration | a tiny experiment on a small pack built in Go in the test or a `testdata` fixture: two or three points, two or three seeds, tens of ticks; real worlds, the real loader. A test over several seeds names the failing seed and the point in its message (testing.md 6.2) | determinism (same bytes twice), failures recorded, cancellation and resume, parallel assembly, branching against the uninterrupted run, alignment, non-interference |
| End to end | one sweep of the reference chain `one`, 400 ticks, a few points and ten seeds; its `Render` against `testdata/golden/experiment-one.txt`, on two architectures | the golden |
| Fuzz | `FuzzOverlay` (any path and value: never panics, and a path that passes the syntax is accepted or rejected by the loader, never both), `FuzzPermutation` (any `N`, `i`, key: in range, and a bijection for small `N`), `FuzzReduce` (any readings: no panic, `Planned` identity holds) | testing.md 7.4 |

### 11.2 What is tested, and where

| Property | Test | Requirement |
|----------|------|-------------|
| a run is a pure function of the experiment and the key; `RunUnit` twice gives equal records, and `Render` equal strings | integration, two runs | FR-EXP-001, NFR-EXP-001 |
| determinism on two architectures | end to end, the golden of `Render` and the digests | NFR-EXP-001 |
| a path through the loader: an unknown key, a float and a scalar over a table are `unresolved` and recorded, a bound violation is `invalid_config`, both for one point while the others run | integration | FR-EXP-002, FR-EXP-050 |
| the overlay of a change is exactly the table header and the key (3.3.4) | unit, table of paths and values | FR-EXP-003 |
| `Validate` returns each rule's error and at most 100 | unit | FR-EXP-004 |
| the order and count of points of each design, the empty design, a factor-free experiment | unit | FR-EXP-005 |
| the spread: a permutation for every `N`, exact stratification, midpoint values, statelessness | unit (and the model's table as the oracle) | FR-EXP-007, FR-EXP-008, FR-EXP-009 |
| the seeds, `SameSeeds` and `PerPointSeeds`; **replay**: a run of a sweep is reproduced alone by `RunUnit` of its unit and by `NewWorld` with `WithSeed` and the point's documents, to the same digest | unit and integration | FR-EXP-010, FR-EXP-011 |
| the range values of a grid and the end points | unit | FR-EXP-012 |
| `Prepare` runs after the overlays and before validation | integration with a recording `Prepare` | FR-EXP-013 |
| the merge order of the steps; a step error is `script` | integration with recording steps | FR-EXP-014, FR-EXP-050 |
| the windows and the final reading; `Every` 0; a short last window | integration with a recording measure | FR-EXP-016, FR-EXP-019 |
| no observer when no measure has an `Observer`: `testing.AllocsPerRun` of a unit with only snapshot measures is below that of the same unit with one measure that has a recording observer, and both end at one digest | integration (an allocation count, observability-and-logging.md 8.2) | FR-EXP-017, FR-EXP-065 |
| a criterion is evaluated on the final reading; a failed or undefined run does not pass; `Pass` is in criterion order | unit and integration | FR-EXP-015 |
| `WithSeed` of the options is overridden by the harness's | integration | FR-EXP-018 |
| cancellation: the discarded run, the prefix, `WithPrior` completes it byte for byte | integration | FR-EXP-020, FR-EXP-052 |
| no write to the experiment; `-race` with several goroutines; `Assemble` of a shuffled set equals `Run`; an unknown key, a duplicate | integration under `-race`; unit | FR-EXP-021 to FR-EXP-023 |
| `Options` is called once for each world built (the burn-in and each restore have their own call) and stateful instances are not shared; an `Options` that carries `WithSeed` records `world` for a branched unit | integration with a counter | FR-EXP-024 |
| a unit builds its burn-in once; the branch with no steps and no reseed ends at the uninterrupted run's digest; the same salt for every variant | integration | FR-EXP-030 to FR-EXP-032 |
| a burn-in failure is inherited; a variant step below `At` is `ErrBadBranch` | integration; unit | FR-EXP-033, FR-EXP-034 |
| reductions against hand arithmetic (`reduce.out` is the oracle), `Planned` identity, the pass share over planned | unit | FR-EXP-040 to FR-EXP-043, FR-EXP-047 |
| `Compare`: fields, exact sign tail for `n` up to 62, `OK` false beyond, the gain and `Identical` | unit | FR-EXP-044, FR-EXP-045 |
| the sensitivity: the effect range, shares and the elasticity with their `OK` conditions | unit | FR-EXP-046 |
| failure: each code, the sweep goes on, the partial series is kept, no checkpoint of a failed world, a measure error leaves a reading undefined | integration with a faulty `Clearing` fake (an invalid trade) and a failing step | FR-EXP-050 to FR-EXP-053 |
| `Render`: the lines, the order, `-`, no float; a `Result` has no map, pointer or float (reflection) | unit | FR-EXP-060, FR-EXP-061 |
| the fingerprint (when CHK offers the read): equal at every window for two runs of one configuration with different observers; different at the first window after a differing count; the aligned horizon is an upper bound; `Pair` without the read is `ErrBadSchedule` | integration | FR-EXP-062 |
| `SameAs` flags equal hashes; `Check` resolves and validates every point and runs no tick; `ranged_narrower_than_design` | integration | FR-EXP-063, FR-EXP-070, FR-EXP-071 |
| the bound is checked before any run, with a checked product | unit | FR-EXP-064 |
| the paths of every shipped experiment are rows of the table (11.3.1) | unit, plain text parse | FR-EXP-072 |
| the package starts no goroutine, calls no `time.Now`, uses no `math/rand` and no float, writes no file (an import and syntax check over the package, as `go vet`-style test) | unit | NFR-EXP-002 |
| the work of the harness for a run is a count of operations that does not grow with agents or ticks beyond the shape of 10.5 (an ordinary test over counted operations, not a benchmark: testing.md 8.2); one record is within the formula of 8.6.1 | unit | NFR-EXP-003, NFR-EXP-004 |
| `internal/reduce` against `internal/measure` on the same inputs (half-to-even division, nearest-rank quantile, sums), written in the external test package of `indicator`, which may import both (IND FR-IND-001), so the two implementations of one rule cannot drift | unit | FR-EXP-040, FR-EXP-042 |
| `FuzzOverlay`, `FuzzPermutation`, `FuzzReduce` | fuzz | NFR-EXP-005 |
| coverage of `experiment/` and `internal/design` and `internal/reduce` at the threshold | the coverage gate (testing.md 9.1) | NFR-EXP-006 |

### 11.3 The registry link (testing.md 6.7)

11.3.1. **A swept field must be a registry row.** The configuration's own
guarantee is that every field of `Config` is a row of the parameter-owner table
and every row a field (`spec-config.md` 9.2, testing.md 6.7). A path the loader
accepts as an **integer key** is a field, and so a row, except a presentation-only
one (`spec-config.md` 8.3). A list has no path (3.3.2a), so "the loader accepts it"
proves a field only for an integer key. The library gives no more than the
loader's acceptance: it cannot tell a presentation-only field (a label, which
changes no hash) without a pure hash function of CFG's, which is an owed row (22).

The repository's own check, in the **tests of the shipped experiments** (those that
STY and CLB put in the repository), is the forward direction only: for each path,
(a) resolve the table prefix to a scope of the table by a small table kept in the
test (`world` to World, `role.<id>` to Role, `region.<id>` to Region,
`region.<id>.group.<id>` to Group, the module tables to their module scopes); (b)
compare the **snake form** of the row's parameter name with the last segment of the
path, by the loader's rule, which CFG is asked to state with its handling of
acronyms (`RateBP`, `ID`; 22); (c) fail for a path with no row. A bare last-segment
match is unsound (`Scale` and `Capacity` repeat across scopes), and testing.md 6.7
defines no path-to-row parse, only the table's text and the two directions between
fields and rows. It needs no dependency: the table is text (FR-EXP-072).

11.3.2. **Why not in `Validate`.** The registry is a document, and a library
function that read `specs/README.md` would read outside the package and outside
what a pure library does (architecture.md 1.1, 3.7 here). The loader is the
library's authority on what a key is. The shipped experiments live in the core
module (not under `examples/`, which are modules of their own), so their test reads
the table by a relative path inside the module.

### 11.4 What the tests do not do

They do not assert a band: no number of the economy is asserted by this spec's
tests (N-027). The scenarios that assert bands are STY's and use the harness as a
consumer. The harness's tests assert that the same experiment is the same bytes,
that a failure is recorded as itself, that a design has the property it claims and
that a reduction is the arithmetic it defines.

## 12. What EXP took from the specs before it, and what it hands to STY and CLB

### 12.1 What the written specs promised EXP, and where it is kept

| Promise | Kept in |
|---------|---------|
| Worlds share nothing, so a harness may build many and step them on separate goroutines (WLD 3.1.1, 13; CHK 11 item 1; OBS 10.3; IND 11.2) | 5.3 |
| `ConfigHash` and `Config()` key a result (WLD 13); indices are not stable across configurations (WLD 5.3.2, IND 3.3.2, OBS 3.2, 10.3) | 4.1.2, 8.1.3: a result is keyed by the world's hash with the seed and the rest of the key, and by names |
| A paired comparison needs equal draw counts (WLD 7.6.1, 13, WLD-A5; CHK 11 item 3; IND 11.2: each in its own words, 7.4.1) | 7: exactness and variance, not the estimate |
| The cost shape of a tick budgets a sweep (WLD 11, 13) | 10 |
| A failed world is detected by `Err()` and by `TickEnded.Failed` and discarded (WLD 13, 3.5; OBS 10.3) | 9.4 |
| Branching is checkpoint once, restore many, with `WithReseed(salt)` for independent continuations; a stateful extension is one instance per world (CHK 11 items 2, 4, 6) | 5.3.3, 6 |
| A scenario script is a list of (tick, call) over the public surface (INT 9.1) | 3.5 |
| A series belongs to the harness; `Stats`, `Growth` and `Change` cover what a sweep summarises (IND 10.3, 11.2) | 3.4, 8.3: the series is the record's; the reductions are over replicas |
| A tally is `O(rows)` and a sweep keeps tallies and not a point for every tick (IND 4.2.4, 11.2) | 3.4.3, 8.6: the consumer's measures keep the tallies; a record keeps readings |
| The solver is a building block of the harness and not the other way round (VIA 7.3) | 3.3.3 (`Prepare`) |
| Re-run the luxury model with DEM's schedule and the occupation model with SPN's discrete rules, and set `entry_share_window` from the run (N-086, `spec-stability.md` 15 question 6) | 12.3: the experiments are described; the run is STY's and CLB's |
| The placeholders of N-015 are calibrated once something has run | 12.4: the harness is the means; no default is set here |

### 12.2 To STY (`spec-stylised-facts.md`): the sweep forms its facts need

A stylised fact is a band on a named measure of a named window of a named pack
(IND 11.1). STY needs a form of experiment for each kind of fact; each is a value
of section 3 and nothing else is needed from this spec.

| Form | The experiment | Scenarios of the register that use it |
|------|----------------|----------------------------------------|
| **Replicate a pack** | no factors, `Replicas` 10 (nightly 50), `Ticks` and `Transient` of the scenario, measures over the final window, criteria as bands | S01 to S03, S05, S08, S09, S11, S12, S14, S16 to S22 |
| **A grid over one or two scalars** | `Grid`, `SameSeeds`, range factors with `Steps` or explicit levels | S07 (`cost_floor` at four values), S10 (scale 1, 4, 10 by an overlay), S19 (the halt ratio on and off), S23 to S25 (elasticity by headroom by perturbation), S26 (`entry_share` on and off), S33 to S35, S37, S39, S40 |
| **A shock response** | `Branch` with variants (none, the shock), `Continuations` 2 or more for a spread of response, `Pair` on when CHK offers the read (7.3.2); measures as the response window | S01, S04, S14, S28, S45, S52, S61 to S65 |
| **A shock of several sizes** | a factor whose levels carry the step (a yield shock of 5 000 and 7 000 bp, a closure at tick `t`) | S45, S52, S54 |
| **A module on and off** | a factor over two documents, `Docs` | S56, S57, S59, S24 |
| **A warm and a cold start** | a factor over documents (`WarmStart` present and absent), or `Prepare` | S17, S18, S22 |
| **An invariant over seeds** | `Options` with `WithAudit`, `Replicas` 10 to 50, criteria `status ok` (a pass share of 10 000), the digest equal across a restore | S55, S60, S66, S72, X5 |
| **A checkpoint at a tick** | `Branch` with one variant with no steps and no reseed | S49, S72, X5 |

12.2.1. **Failed runs are a result.** A scenario that expects the run to be
reported as a failure regime (F3, F5) is a criterion on a measure, and a run that
fails by `invalid_trade` or `audit` is a **fail** of the scenario with its seed
and tick in the record (testing.md 6.2).

12.2.2. **STY owns the bands and the measures**; this spec gives `Criterion`,
the pass share and the comparison as the mechanism. A band that is not set
(`spec-reference-chain.md` 7.1.5) is a criterion that is absent: the experiment
reports the reductions and asserts nothing.

12.2.3. **What `spec-stylised-facts.md` 10.1 asks, and where each is kept.**

| STY needs (10.1) | Kept in |
|------------------|---------|
| a run of a pack and overlays over a seed set, the gate 1 to 10, the setting 11 to 50, the nightly 1 to 50 (STY 8.2), shocks as INT calls at ticks, the series and the events a verdict reads as plain data | a seed set is a `Seed0` and a `Replicas`: (1, 10), (11, 40), (1, 50); shocks are steps (3.5); the series is `Keep.Series` and an event-level measure reads events through its own `Observer` and `Sample.Key` (3.4) |
| a one-at-a-time sweep of a knob at the default and its two neighbours, with the verdict of each fact and detector as the response | `OneAtATime` with three levels (level 0 the default); a fact or a detector is a `Measure` whose value is its verdict code, and a band is a `Criterion` |
| a comparison of two points as distributions over seeds, not as a pair, for a knob that moves a count | `PerPointSeeds` or `SameSeeds` with the `Dist` field of `Compare` (8.4.2), and the reductions of each group (8.3); a pair is used only for a knob that moves no count, and `AlignedPairs` and the gain say which (7) |
| the verdict record keyed by the `ConfigHash` | `Record.Hash`, `Record.Pass` and `Result`; STY's text record (STY 8.5) is a rendering of them that STY writes: the class and the band are its own, the values per seed and the seeds outside are `Final` and `Pass` |
| a failed world recorded and skipped | 9 |
| a branch from a checkpoint for the shock facts | 6 |

12.2.4. **The median.** STY 8.3.4 takes the median of an even number of seeds as
the half-to-even mean of the two middle values; `Q(5 000)` of 8.3 is the
nearest-rank value (IND 5.5.2), a value some run produced. They differ by half the
gap of the two middle readings and both are exact. STY computes its median from
the readings of the records; nothing in this spec forbids it.

### 12.3 What N-086 asks of a run

N-086 and `spec-stability.md` 15 question 6 ask that the luxury model be re-run
with DEM's schedule and the occupation model with SPN's discrete rules, and that
the threshold of `entry_share_window` be set from the run. Both are experiments
of the first two forms of 12.2 (S23 and S26 with their factors); the run is STY's
and CLB's, and this spec leaves the note open with that learned (20).

### 12.4 To CLB (`spec-calibration.md`): the interface its registry and procedure use

12.4.1. **A knob is a factor.** A row of CLB's registry (a parameter, its range,
its expected effect) is a range factor: `Path` is the file spelling of the row
(3.2.1), `Min` and `Max` its range; a composite knob is a factor whose levels
have several changes. CLB's registry may keep the paths as data.

12.4.2. **The procedure's steps and what each uses.**

| Step | Experiment | Output |
|------|------------|--------|
| Screen the knobs | `OneAtATime` over every knob at a low and a high level | the elasticity and the effect range of each (8.5): the knobs that move a measure at all |
| Explore | `Spread` of `N` points over the knobs that matter, `SameSeeds` | the main effects (8.5) and the pass share of each point against STY's criteria (8.3) |
| Choose | a consumer's loop over the result | one point, or a region |
| Confirm | `Grid` or one point with a **different `Seed0`** (4.6.4) and more replicas | the pass share of the chosen point on seeds it was not chosen on |

12.4.3. **The expected effect of a knob** is the sign of the elasticity or of the
mean difference of two points on a measure (8.4, 8.5), which is a number with a
unit, so CLB can state it and a test can check it (IND 11.3).

12.4.4. **What the library never does** is move a knob in a run, choose a point or
decide that a calibration is done (IND 11.3, 3.7): the loop is the consumer's.

## 13. Parameters

13.1. **No `Config` field, default or row.** An experiment is not configuration
(architecture.md 9.5): changing it does not change the hash of a world, and the
parameter registry test (testing.md 6.7) needs nothing from this spec.

13.2. **Bounds of the package** (rule constants of `experiment/`, placeholders,
N-015; every product is computed with a checked multiplication and an overflow is
`ErrOversize`):

| Name | Bound | Why |
|------|-------|-----|
| `MaxFactors` | 16 | a design of more factors than a consumer can read; a grid of 16 factors is two to the 16th |
| `MaxLevels` | 1 000 | per factor |
| `MaxPathSegments` | 8 | a path of 3.2.1 |
| `MaxPoints` | 100 000 | after the design's count |
| `MaxReplicas` | 10 000 | per point and variant |
| `MaxVariants`, `MaxContinuations` | 64, 1 024 | a branch |
| `MaxRuns` | 1 000 000 | points times variants times replicas times `max(1, continuations)` |
| `MaxMeasures` | 256 | |
| `MaxWindows` | 10 000 | `Ticks` over `Every`, at least |
| `MaxCells` | 10^8 | runs times measures times (windows + 1) with the series; 1.6 GB as Go structs, 900 MB packed. At the limits of the other bounds `MaxRuns` times `MaxMeasures` is 2.56 * 10^8, so `MaxCells` binds first with a series and `MaxRuns` first without |
| `MaxReading` | 2^40 | a reading's magnitude: keeps the sums below 2^126 for 10^6 used runs |
| `MaxErrors` | 100 | of `Validate`, as `spec-config.md` 7.1.3 |

13.3. **Rule constants** (beside their code, architecture.md 9.6): the salt
`0x243F6A8885A308D3` and the gamma `0x9E3779B97F4A7C15` of 4.4.2; the SplitMix64
finaliser constants (`spec-world.md` 7.1.2); the four rounds of the Feistel
network; the cell midpoint rule; the mixed-radix order; the OAT order; the seed
rules of 4.6.1; the rounding rule (half to even, IND 5.2.2); the closed list of
quantiles 1 000, 2 500, 5 000, 7 500, 9 000; the cap of 62 of the exact sign test;
the pass-share base 10 000; the lines of `Render` and the result `Format` (1).

13.4. **Closed sets**: `Design` (`Grid`, `OneAtATime`, `Spread`); `Seeding`
(`SameSeeds`, `PerPointSeeds`); `Op`
(`AtMost`, `AtLeast`, `Between`); `Status` (`ok`, `unresolved`,
`invalid_config`, `world`, `script`, `invalid_trade`, `audit`, `checkpoint`,
`step`, `missing`). Adding a member to a closed set changes a documented behaviour of an
exported identifier, so architecture.md 6.6 applies and the result `Format` is
raised.

13.5. **Not parameters:** the replica count, the ticks, the windows and the
seeds of an experiment are the experiment's data and are not defaults of the
library (N-027); the numbers in the worked examples are labelled.

## 14. Validation

14.1. **Errors this spec adds** (sentinels of `experiment/`, matched with
`errors.Is`, each with a failing-case test, testing.md 3.9): `ErrBadSchedule`,
`ErrBadFactor`, `ErrFactorOverlap`, `ErrBadDesign`, `ErrBadBranch`,
`ErrUnknownMeasure`, `ErrOversize` (an experiment over a bound),
`ErrUnknownRun` (a record whose key is outside the plan), `ErrDuplicateRun` and
`ErrRange` (a reading beyond `MaxReading`). The names differ from CFG's
`ErrUnknownKey` (a loader key) and `ErrTooLarge` (a document over
`MaxDocumentBytes`), which are different conditions. `ErrDuplicateID` and
`ErrTooManyErrors` are CFG's. None is logged by the library (architecture.md
8.3). **A record carries a `Status` and the error's text, not a matchable
sentinel**: the sentinels of the other specs (`ErrCheckpointConfig`,
`ErrInvalidTrade`, `ErrAuditFailed`) are matched by the harness to choose the
status (9.1) and are not returned through a record; `RunUnit` and `Run` return only
the errors of this section and the context's.

14.2. **`Validate()`** is 3.6: a property of the value, no parse, no world.

14.3. **`Check(ctx, e) ([]PointCheck, error)`** resolves every point (parse, the
overlays, `Prepare`, `Validate` of the resolved `Config`) and returns, per point,
its status (`ok`, `unresolved` or `invalid_config`), the error text and its
warnings; it runs no tick and builds no world. It runs **no option and no name
check**: `Options` needs a key and the three name checks of `NewWorld` need the
registrations (`spec-config.md` 7.1.4, WLD 3.2.2), so a mistake of a strategy name
is found at the first run and not by `Check`. It computes **no hash**: CFG
exposes none (the hash is `World.ConfigHash`), so `SameAs` and `duplicate_point`
are computed at `Assemble` from the hashes of built worlds (an owed row asks CFG
for a pure hash function, after which `Check` could report both). Its cost is the
parse and the validation of each point (the viability programmes included), and is
the consumer's to afford before a long sweep: a mistake of a path or a bound is
found in minutes and not at the end of a night. It is also the means of the
registry test of 11.3.

```go
type PointCheck struct {
	Point    int
	Status   Status    // ok, unresolved or invalid_config
	Detail   string    // the error text, or empty
	Warnings []Warning // 14.4
}
```

14.4. **Warnings** (plain values with a rule and a path like `spec-config.md` 7.2,
in `PointCheck.Warnings` and `PointResult.Warnings`, one spelling each; never
failures; the path is the Go path of the factor and level for a design rule and
the point index for a result rule):

| Rule | When | Where |
|------|------|-------|
| `ranged_narrower_than_design` | a spread over a range factor with fewer values than `N` (not `Validate`, which returns errors only) | `Check` |
| `duplicate_point` | two points whose worlds have one hash (`SameAs`) | `Assemble` |
| `stale_warm_start` | specified when the paths the solver reads are published, by VIA or by the `in` flag of CLB's registry rows, which `spec-calibration.md` offers as its answer to open question 7 (3.3.3); not specified here | |

A `no_effect` warning (a change that leaves the hash equal) needs a pure hash
function of the resolved configuration and is not specified until CFG offers one
(22).

14.5. **What is not validated here:** that a measure is meaningful for the pack,
that a criterion's band is reasonable (STY), that `Replicas` is enough for the
spread the experiment will see (EXP-A1, 17).

## 15. Observability

15.1. **No event, no log key, no metric and no Snapshot field.** A sweep is not a
state transition of any world: it builds worlds and reads them. The harness has no
logger and writes no line (architecture.md 11.1): a consumer observes a sweep
through the record callback (5.2.2), which is a value, and observes each world
through the options it passes (5.5), as it would any world. A failure is a
returned status and a record, never a log line (architecture.md 8.3).

15.2. **The harness observes no event itself.** The observers it registers are the
measures', the consumer's own, and obey the rules for one
(observability-and-logging.md 4.4, architecture.md 12.5): they hold no world, never
call back into one and leave every result unchanged: a run with the measures'
observers and a run with none give the same digest (FR-EXP-065,
observability-and-logging.md 1.4, 8.1 non-interference). The harness reads only
snapshots, checkpoints and the errors of `Run`.

15.3. **The Observability entry** (observability-and-logging.md 9): *None, because
...* the harness adds no failure path inside a world (every failure is a status of
a record and a returned error of the sweep), no state transition of the simulation,
no decision point of a strategy, clearing or spawner and no loop in a tick; its
loops are over runs and are not logged. It uses the `Observer` interface as a
consumer, registering the observers of the measures that have one (3.4.4); `Pair`
needs no observer and the harness constructs no event of its own.

## 16. State contributed to a checkpoint

None. An experiment is not world state and a checkpoint of a world carries
nothing of it. The progress of a sweep is the consumer's records (`WithPrior`,
5.2.2); a branch's checkpoint is internal to a unit and is dropped with it
(6.1.1). A consumer that wants to save the unit's checkpoint calls `Checkpoint`
itself; the harness does not offer it (open question 6).

## 17. Entries for the stability register

Provisional IDs (N-103); the supervisor assigns the final numbers. The checklist
of `spec-stability.md` 9 was followed. This spec adds **no feedback loop**:
the harness reads no measure and changes no parameter inside a run
(FR-STB-003), and the calibration loop is the consumer's (3.7, 12.4.4). It adds
five hazards of the practice of tuning, three dampers and four scenarios.

**EXP-A1 Too few replicas.** *Kind:* hazard. *Loop:* none: a mean and a standard
deviation of three seeds are one draw from the distribution of those statistics.
*Owner:* EXP 8.3. *Damper:* EXP-D1: the reductions report `Used` beside every
statistic and the standard deviation, so a consumer sees the sample; the bands'
seeds are `spec-reference-chain.md` 8.2 (10, nightly 50). *Undamped:* an
experiment with `Replicas` 2 reports a number. *Condition:* a band is never set
from fewer than the seeds of REF 8.2 (FR-REF-024). *Test:* EXP-S1. *Status:*
hand.

**EXP-A2 The best of many points is fitted to its seeds.** *Kind:* hazard (the
winner's curse). *Loop:* none: choosing the maximum of `N` noisy pass shares
selects the points whose noise was favourable, so the chosen point's pass share
on those seeds overstates it. *Owner:* EXP 4.6.4, 12.4.2. *Damper:* EXP-D2: the
confirm step on another `Seed0` (a second experiment, independent of the first).
*Undamped:* a calibration that stops at the choice. *Condition:* the confirm
share, not the chosen share, is what a calibration claims. *Test:* EXP-S2.
*Status:* hand.

**EXP-A3 A transient read as an equilibrium.** *Kind:* hazard. *Loop:* none: a
window opened before the run has settled measures the approach, not the
state. *Owner:* EXP 3.4.2; STY (the length of a transient). *Damper:* `Transient`
(a burn-in no window covers) and the series: a drift across successive windows is
visible (IND `Stats`, `Change`). *Undamped:* `Transient` 0. *Condition:* a band
is stated with its window (IND 11.1.1). *Test:* EXP-S3. *Status:* hand.

**EXP-A4 Survivors.** *Kind:* hazard. *Loop:* none: dropping failed runs
from a mean makes a sweep that fails often look better than one that does not.
*Owner:* EXP 8.3.1, 9.3. *Damper:* EXP-D3: `Planned = Used + Failed + Undefined +
Missing`, a pass share over `Planned`, and the counts always beside the mean.
*Undamped:* a consumer that reads `Mean` and ignores `Failed`. *Condition:* a
failure of the world is a result (9). *Test:* EXP-S4. *Status:* hand.

**EXP-A5 A stale warm start across a sweep.** *Kind:* hazard. *Loop:* none:
a parameter that moves the steady state is swept over a base whose `WarmStart`
was solved for the base's value, so every other point starts away from its own
equilibrium and the measure includes the approach (STB-L53's perturbation, the
warm start that hides an instability). *Owner:* EXP 3.3.3; VIA 8. *Damper:*
`Prepare` regenerates the table per point; `stale_warm_start` (opt-in). *Undamped:*
a sweep with no `Prepare`. *Condition:* the experiment says whether it means a
perturbation or a comparison of steady states. *Test:* S22's cold leg
(`spec-stability.md` 8). *Status:* hand.

**Dampers.** EXP-D1 *Counts beside statistics* (8.3), EXP-D2 *A confirm run on
other seeds* (12.4.2), EXP-D3 *Failures counted, never dropped* (8.3.1).

**Scenarios** (tests of the harness, written against the model of its own
rules and not against the economy): EXP-S1 *The spread of a statistic*: the
standard deviation of a mean over `R` seeds of a known toy process falls as
`1/sqrt(R)` to within the placeholders of `models/exp/pairing.py`; EXP-S2 *The
winner's curse*: of 100 points with one true pass share, the chosen one's share on
the choosing seeds exceeds its share on others; EXP-S3 *The transient*: a toy
process with a known approach, windows opened at 0 and at the settling tick
differ by the stated amount; EXP-S4 *The survivor*: a sweep whose points fail at
known rates, the `Failed` and `Planned` counts equal the rates and the mean over
used runs differs from the mean of all by the stated bias.

## 18. What was run

All in Docker (`python:3.12-slim`, `PYTHONDONTWRITEBYTECODE=1`), Python standard
library only, in `doc/specs/models/exp/`. They are models of the rules of this
spec, not of the library, and they show no default to be stable (N-027). Nothing
was run in Go.

| Script | What it does | Result quoted |
|--------|--------------|---------------|
| `lhs.py` | the counter-based Latin hypercube of 4.4: the permutation property for every `N` from 1 to 300 and for 511, 512, 513, 1 000, 4 096 and 5 000; statelessness; the cell midpoint values; the empty cells of a 2-D projection and the column correlation against a stateless hash draw, over 20 salts | permutation and stratification hold; 20.0 empty cells of 64 against 23.9 for a hash draw; the largest column correlation at `N` 64 and `d` 8 is 0.29 against 0.27, so only the stratification is claimed (4.4.4) |
| `kronecker.py` | the **rejected** generalised golden ratio sequence in 2^64 fixed point (4.7): the multipliers for `d` = 1, 2, 4, 8 computed by exact bisection (`d = 1` is the SplitMix64 gamma), the counts per bin of the one-dimensional projections, the fill of a square | at `d` 4 and `N` 100 in ten bins one bin held 1 point and another 15 against 10 |
| `reduce.py` | the integer reductions of 8.3 to 8.5 checked against exact fractions: half-to-even division (20 000 random cases), the nearest integer root (20 000), the nearest-rank quantile, the sample standard deviation, the exact sign-test tail for every `n` up to 62, the effect range and share, the elasticity; and the arithmetic of the examples of 19 | all checks pass; the sums of 10^6 readings of `2^40` stay below `2^126` |
| `pairing.py` | a toy world on one stream with a state-dependent count of words: the aligned share, the first divergent sample, the paired and unpaired spread and the gain, for a calm and a busy world | implements 7.4.2: identical settings `sd` 0 against 10; a rare one-word offset gain 50 000 bp with 7 of 200 seeds aligned at the end (30 000 and 20 000 for larger steps); a busy world 12 857 and 11 429 (7.4.3) |
| `cost.py` | the arithmetic of 10 from the placeholders of `spec-scale.md` 6, OBS 5.5, 8.3 and CHK 10.3 | the tables of 10.2 to 10.4 and 8.6 (the end checkpoint included; 16-byte readings) |
| `worked.py` | the points of the examples of 19: a grid, one at a time, a spread of 8, the overlay documents, the seeds, the plan of a branch | 19 |

What they do not show: that the Latin hypercube is better than another design for
the economy, that the pairing gain of a real world resembles the toy's, that the
costs are right (they are the placeholders of N-015), or anything about the
library, which has no code.

## 19. Worked examples

The settings are **examples**: `north` and `grow_staple` are invented;
`cost_floor` and `belief_step` are real `World` fields of `spec-beliefs.md` used as
example paths (the point (10 000, 400) of 19.1 would draw the warning
`cost_floor_at_full_cost` of `spec-config.md`); all are data, never branched on
(N-017); every number is a placeholder
(N-027). The first five are from `worked.py`.

### 19.1 A grid

`cost_floor` at 7 000, 8 000, 9 000 and 10 000, and `belief_step` at 100, 200 and
400: twelve points, the last factor fastest. Point 0 is (7 000, 100), point 1 is
(7 000, 200), point 2 is (7 000, 400), point 3 is (8 000, 100), and point 11 is
(10 000, 400). The index of (8 000, 400) is `1 * 3 + 2 = 5`.

### 19.2 One at a time

The same factors, level 0 the baseline: `1 + 3 + 2 = 6` points: point 0 is
(7 000, 100); points 1 to 3 move `cost_floor` to 8 000, 9 000, 10 000 with
`belief_step` at 100; points 4 and 5 move `belief_step` to 200 and 400 with
`cost_floor` at 7 000.

### 19.3 A spread of eight

`cost_floor` in `[6 000, 10 000]` and `belief_step` in `[100, 400]`, `N = 8`,
`DesignSalt` 0. For each axis each of the eight cells is used once. Point 0 has the
cells (4, 1), so `cost_floor = 6 000 + floor((2*4 + 1) * 4 001 / 16) = 8 250` and
`belief_step = 100 + floor((2*1 + 1) * 301 / 16) = 156`. The eight points are
(8 250, 156), (9 750, 194), (6 250, 306), (7 250, 231), (6 750, 118),
(8 750, 269), (7 750, 344) and (9 250, 382). A range of ten values with `N = 10`
uses each value once; with `N = 25` each is used two or three times
(`lhs.out`).

### 19.4 The overlay documents of a point

Point 5 of 19.1 has `cost_floor = 8 000` and `belief_step = 400`; its overlays,
after the base, are

```
[world]
cost_floor = 8000
```
```
[world]
belief_step = 400
```

and `config.Parse` of the base followed by these documents, in order, gives its
`Config`. A factor on a
role's field, `role.worker_a.target_ratio`, is `[role.worker_a]` and the key.
A path `world.cost_flor` is rejected by the loader (`ErrUnknownKey`) and the
point is `unresolved`.

### 19.5 Seeds

`Seed0` 1 and four replicas, three points: `SameSeeds` gives (1, 2, 3, 4) at every
point; `PerPointSeeds` gives (1, 2, 3, 4), (5, 6, 7, 8) and (9, 10, 11, 12). A run
of replica 2 of point 1 under `PerPointSeeds` is `WithSeed(7)` with that point's
documents.

### 19.6 A branch

A unit of seed 1 runs 365 ticks, takes a checkpoint, and has three variants: none,
`SetYieldShocks` of `north` and `grow_staple` at 5 000 bp at the branch point, and
the same with a clearing step at tick 425. With `Continuations` 1 the three share
the stream from the checkpoint and, when CHK offers the read (7.3.2), `Pair` records
the fingerprint at each window;
the aligned horizon of the second against the first is the first window at which
the shock has moved a count. With `Continuations` 5 and `Salt0` 100 each variant
has five continuations with salts 100 to 104, and continuation 2 of the three
variants is a paired set.

### 19.7 A comparison and a sensitivity

Two points and eight seed-matched replicas of a measure: A = (100, 104, 98, 110,
101, 99, 107, 103) and B = (103, 109, 97, 118, 107, 99, 113, 104). The differences
are (3, 5, -1, 8, 6, 0, 6, 1): `Pairs` 8, `MeanDiff` `rdiv(28, 8) = 4` (3.5 rounds
to the even 4), `MedianDiff` 3, `Wins` 6, `Losses` 1, `Ties` 1, `SignTailPPM`
`2 * (C(7,6) + C(7,7)) / 2^7 = 16/128` = 125 000. `RelativeBP` is
`rdiv(10 000 * 28, 822) = 341` from the exact sums of the pairs (`rdiv(10 000 * 4,
103) = 388` from the rounded means would be a second rounding and is not used,
8.4.2); `DiffOfMeans` is `rdiv(850 * 8 - 822 * 8, 64) = 4`.

Sensitivity on a 3 by 3 grid with the point means `y(i, j)` = 100, 104, 108 /
110, 114, 118 / 140, 144, 148: the bin means of the first factor are 104, 114,
144 (range 40) and of the second 117, 121, 125 (range 8); the shares are 8 333
and 1 667 bp. A knob from 7 000 to 8 000 moving a measure from 12 000 to 11 100
has, from single-run sums, the elasticity `rdiv(10 000 * (-900) * 7 000, 12 000 *
1 000) = -5 250`: a 1% rise of the knob moves the measure by -0.525%.

### 19.8 The reductions and a failure

Ten replicas of a measure (the price range of a market, in basis points of the
base): 10 100, 10 200, 10 300, 10 400, 10 500, 10 600, 10 700, 10 900, 11 800,
12 100. `Min` 10 100, `Max` 12 100, `Mean` 10 760, `Std` 674, the quantiles at
1 000, 2 500, 5 000, 7 500 and 9 000 are 10 100, 10 300, 10 500, 10 900 and
11 800. A criterion "at most 11 000" passes 8 of 10, 8 000 bp. If the run that read
10 600 had failed instead, `Used` would be 9, `Failed` 1, `Mean` 10 778 and the
same criterion would give 7 000 bp over the planned 10 (the failed run does not
pass); if the run that read 12 100 had failed, the seven passing runs and the one
reading 10 900 stay, the share is 8 000 bp and `Mean` 10 611.

### 19.9 A rendered result

Two points, two replicas, one measure `price_range_bp`, one criterion `calm`
(at most 11 000). Hashes and digests are invented.

```
experiment example format 1 points 2 measures 1 criteria 1
measure 0 price_range_bp
point 0 hash 5a1c0e44b27d9f31 status ok same_as -1 levels 0 values 7000
variant 0
run replica 0 cont 0 seed 1 salt - status ok fail_tick -1 digest 91d3a7c20f4be816
final price_range_bp=10400
verdict calm=pass
run replica 1 cont 0 seed 2 salt - status ok fail_tick -1 digest 0be4417f6c93d2a5
final price_range_bp=10900
verdict calm=pass
reduce price_range_bp planned 2 used 2 failed 0 undefined 0 missing 0 min 10400 max 10900 mean 10650 std 354 q1000=10400 q2500=10400 q5000=10400 q7500=10900 q9000=10900
pass calm passes 2 planned 2 share 10000
point 1 hash c7f20b19e84a5d06 status ok same_as -1 levels 1 values 9000
variant 0
run replica 0 cont 0 seed 1 salt - status ok fail_tick -1 digest 4e9d02b6a17c35f9
final price_range_bp=11800
verdict calm=fail
run replica 1 cont 0 seed 2 salt - status invalid_trade fail_tick 212 digest -
detail step 212: a clearing returned an invalid list of trades for a market
final price_range_bp=-
verdict calm=fail
reduce price_range_bp planned 2 used 1 failed 1 undefined 0 missing 0 min 11800 max 11800 mean 11800 std - q1000=11800 q2500=11800 q5000=11800 q7500=11800 q9000=11800
pass calm passes 0 planned 2 share 0
```

The first point: the mean is `rdiv(21 300, 2) = 10 650`, `Std` is
`isqrt_nearest(rdiv(2 * (10 400^2 + 10 900^2) - 21 300^2, 2)) =
isqrt_nearest(125 000) = 354`, the quantiles of two readings are the nearest-rank
values (rank 1 up to 5 000 bp, rank 2 above), and both readings are at most 11 000,
so the pass share is 10 000. The second point had one failed run: `used 1
failed 1`, no standard deviation, and a pass share over the planned 2, at most
5 000 and here 0 because the one used reading is above 11 000.

## 20. Notes settled

For `AGENTS.md`: each open note tagged `EXP` or `ALL`, the notes that name EXP
(N-086, N-103) and the proposed new notes of the other writers' handoffs that
address it, are applied, rejected or left open here, with the text to append in
the handoff.

| Note | Decision | Where |
|------|----------|-------|
| N-015 Placeholders to calibrate once something has run (EXP) | **Leave open**, with this added: EXP is the means, and sets no number; the placeholders it uses (the bounds of 13.2, the budget of 10) are labelled and are N-015's | 10, 13.2 |
| N-016 Keep the parameter registry true (ALL) | **Applied:** EXP adds no `Config` field and no row; a swept path is an integer key of `Config` and so a row (11.3.1); a `no_effect` warning waits for a pure hash function of CFG's (14.4) | 11.3, 13.1 |
| N-017 Words and names (ALL) | **Applied:** the terms of the list in the handoff go to the glossary; the examples are flagged | preamble, 19 |
| N-027 No default macro constant is stable yet (ALL) | **Applied:** no default is set, no number of the economy is asserted; the models show structure and the pairing gain of a toy | 13.5, 18 |
| N-086 Re-run the demand and spawner models under the register (STB, STY) | **Leave open**, with this added: the runs are experiments of two of the forms of 12.2 (a grid over the elasticity and the headroom; a spawner cap on and off), and the harness supplies the means; STY and CLB run them | 12.3 |
| N-103 Running several spec writers in parallel (ALL) | **Applied:** only this spec file, `models/exp/` and `.supervisor/exp.*` were created. **Slip:** the host `python3` was run three times, once with no argument, once with `--version` and once as `-c 1` (reflex, in shell lines around Docker runs); no script ran on the host and every model ran in Docker with no bytecode | 18 |
| New note of the WLD handoff (9) [EXP, STB] Paired runs | **Applied** with a correction: a paired comparison is **exact** only in the aligned prefix and **reduces variance** by a measured gain; the estimate of a difference of means from seed-matched pairs is unbiased whether aligned or not | 7 |
| N-CHK-7 A stateful extension is one instance per world (CHK handoff) [EXP, INT] | **Applied** | 5.3.3, 5.5 |
| N-CHK-10 Paired comparison from a checkpoint (CHK handoff) [EXP] | **Applied**, with the same correction | 6.3, 7 |
| New note of the OBS handoff (8) [EXP] Event classes at the observer | **Leave open**: a sweep with only snapshot measures registers no observer and builds no event (3.4.4); the option matters only to a sweep that needs events, where the cost is a placeholder 6% of a tick (10.1) and the measure N-015 asks for | 3.4.4, 10 |
| New note of the INT handoff (4) [STY, EXP] Not in version 1 | **Leave open**: the harness needs none of an act-now operation, a timer or a bundle; it applies a script of calls (3.5) | 3.5 |

Proposed new notes are in the handoff.

## 21. Requirements

Each is a single testable statement (testing.md 4.1). Tests cite the IDs. This spec
has no default of its own and one set of package bounds (13.2).

### Functional

| ID | Requirement |
|----|-------------|
| FR-EXP-001 | `RunUnit` MUST be a pure function of the experiment and the unit: two calls MUST give records that are equal field for field, and it MUST NOT read a clock, the environment, a global, the result of another run or the count of goroutines (3.1.1, 5.1.2). |
| FR-EXP-002 | A point MUST be resolved by `config.Parse` of the base followed by the overlay documents of its levels in factor order, so that the loader's rejections, `Validate`, the name checks and the configuration hash apply to it as to a pack; a path the loader rejects MUST be recorded as `unresolved`, and a configuration `Validate` rejects as `invalid_config`, and the other points MUST run (3.3.2, 9.1). |
| FR-EXP-003 | The overlay document of a change MUST be exactly the table header formed from all but the last segment of its path, a line break, the last segment, ` = `, the value in decimal and a line break, and MUST be the same bytes on every platform (3.3.4). |
| FR-EXP-004 | `Validate` MUST report each violation of 3.6 with its sentinel, MUST return at most 100 errors, joined and sorted, and MUST parse no document and build no world. |
| FR-EXP-005 | The points of a design MUST be generated in the order of 4.2 to 4.4 with the count of 4.2.1, 4.3.1 and the design's size, and an experiment with no factor MUST have one point, the base. |
| FR-EXP-007 | For every `N` at least 1 the cell function of a spread MUST be a permutation of `[0, N)` for every axis, built by the four-round Feistel network of 4.4.2 with cycle walking, so that every cell of every axis is used by exactly one point. |
| FR-EXP-008 | A spread MUST give an integer range factor the value `Min + floor((2 * cell + 1) * (Max - Min + 1) / (2 * N))` and an explicit-level factor the level `floor((2 * cell + 1) * L / (2 * N))`, in 128-bit arithmetic with no other rounding (4.4.3). |
| FR-EXP-009 | The index of a point and its coordinates MUST be a function of the design, the salt, the axis, `N` and the index alone: the point computed alone MUST equal the point in the full design, and no draw from any random source MUST be taken by the harness (4.5). |
| FR-EXP-010 | The seed of replica `r` of point `p` MUST be `Seed0 + r` under `SameSeeds` and `Seed0 + p * R + r` under `PerPointSeeds`, modulo 2^64, and a run MUST be reproduced by `WithSeed` of that seed with the point's documents (4.6.1). |
| FR-EXP-011 | Two experiments that differ only in `Seed0` MUST share no state, and a run's result MUST NOT depend on another experiment having run (4.6.4). |
| FR-EXP-012 | A grid over a range factor MUST give level `k` the value `Min + rdiv(k * (Max - Min), Steps - 1)`, half to even, with `Min` and `Max` exact (3.2.3). |
| FR-EXP-013 | `Prepare`, when given, MUST be applied to the resolved configuration after the overlays and before validation, and its error MUST be recorded as `invalid_config` (3.3.3). |
| FR-EXP-014 | The steps of a run MUST be merged by tick, with equal ticks in the order script, levels in factor order, variant, and applied at the boundary before the tick they name, never inside a step (3.5). |
| FR-EXP-015 | A criterion MUST be evaluated on the final reading of a run, MUST pass only when the run has status `ok`, the reading is defined and it is inside the band, and its verdict MUST be recorded in `Pass` of the record in criterion order (3.4.5). |
| FR-EXP-016 | The windows MUST be `[Transient, Transient + Every)` and so on, the last shorter if needed, or one window with `Every` 0, and the final reading of each measure MUST be over the whole span `[Transient, Ticks)` (3.4.2, 3.4.3). |
| FR-EXP-017 | When no measure has an `Observer` the harness MUST register no observer, so that no event is constructed, whether `Pair` is on or not (3.4.4). |
| FR-EXP-018 | The harness MUST pass `WithSeed` last, so that its seed is the one in effect whatever `Options` gives, and MUST NOT pass `WithSeed` to a restore (5.1.3, 5.5.1). |
| FR-EXP-019 | At a boundary the harness MUST close the window that ends there, then apply the steps of that tick, then open the next window with a snapshot taken after the steps (5.1.3). |
| FR-EXP-020 | A cancelled `Run` MUST return the result of the complete units before the cancellation and the context's error wrapped, MUST discard the run in flight, and a `Run` with `WithPrior` of those records MUST give the same `Render` as an uninterrupted `Run` (5.2.3, 5.4). |
| FR-EXP-021 | No function of the package MUST write to the experiment, its slices or the bytes of its documents (5.3.1). |
| FR-EXP-022 | `Assemble` MUST be insensitive to the order of the records, MUST return `ErrUnknownRun` for a key outside the plan and `ErrDuplicateRun` for a key twice with different contents, MUST accept an identical duplicate, and MUST report an absent key as `missing` (5.3.4). |
| FR-EXP-023 | The `Result` assembled from the records of `RunUnit` calls made on several goroutines MUST render equal to that of `Run` (5.3.4). |
| FR-EXP-024 | `Options` MUST be called once for each world built, with the key of that world (the burn-in with variant 0 and continuation 0, each restore with its own), the harness MUST NOT share an instance it returned between two worlds, and an `Options` that returns `WithSeed` or `WithReseed` MUST give `world` for a branched unit (5.5). |
| FR-EXP-030 | A unit with a branch MUST run its burn-in once and restore its checkpoint once for each variant and continuation; a configuration value MUST NOT differ between its branches (6.1, 6.2.1). |
| FR-EXP-031 | A branch with no steps and no reseed MUST end at the same digest as the uninterrupted run of the same key (6.4.1). |
| FR-EXP-032 | With `Continuations` 2 or more continuation `c` MUST be restored with `WithReseed(Salt0 + c)`, the same salt for every variant; with 0 or 1 no `WithReseed` MUST be passed (6.3). |
| FR-EXP-033 | A failure of a burn-in MUST be recorded for every key of its unit with the burn-in's status and tick, and no branch MUST be run (9.1). |
| FR-EXP-034 | A variant step with a tick below `At` MUST be `ErrBadBranch` (6.1.2). |
| FR-EXP-040 | Every reduction MUST be integer arithmetic with 128-bit sums, one rounding rule (half to even) and no floating point (8.3). |
| FR-EXP-041 | For every group and measure `Planned = Used + Failed + Undefined + Missing` MUST hold, and a failed run MUST NOT enter `Used` (8.3.1, 9.3.1). |
| FR-EXP-042 | `Mean`, `Std`, the quantiles and the extremes MUST be those defined in 8.3, with `Std` undefined for `Used` below 2 and the quantiles the nearest-rank values at the closed list. |
| FR-EXP-043 | A pass share MUST be `rdiv(10 000 * passes, Planned)` (8.3). |
| FR-EXP-044 | `Compare` MUST match records by replica and continuation, MUST give the fields of 8.4.2 with the exact sign tail for `n` up to 62 and `OK` false beyond, and MUST offer no pairs for two different points of a `PerPointSeeds` experiment (8.4.1) (8.4). |
| FR-EXP-045 | The pairing gain MUST be `rdiv(10 000 * isqrt_nearest(vA + vB), sd)` with `OK` false and `Identical` set when every difference is 0 (7.4.2). |
| FR-EXP-046 | The effect range, its share and the elasticity MUST be those of 8.5 with the stated `OK` conditions. |
| FR-EXP-047 | A reading with a magnitude above `MaxReading` MUST be undefined with an `ErrRange` detail (8.3.2). |
| FR-EXP-050 | Each failure of 9.1 MUST be recorded with its code, tick and error text and MUST NOT stop the sweep; the readings of the windows closed before a failure MUST be kept and MUST NOT enter a reduction (9.2, 9.3). |
| FR-EXP-051 | The harness MUST NOT take a snapshot or a checkpoint of a failed world and MUST NOT continue it (9.4). |
| FR-EXP-052 | A cancelled run MUST NOT be recorded and MUST NOT count in `Failed` (5.4.2, 5.4.3). |
| FR-EXP-053 | A measure that returns an error MUST leave that reading undefined, MUST keep the first error in `Detail` and MUST NOT stop the run (9.1.1). |
| FR-EXP-060 | `Render` MUST be a pure function of a `Result`, in the lines and order of 8.2.2, with no float, map order or time, and the same on every platform. |
| FR-EXP-061 | A `Result` and everything it holds MUST be plain data with no map, pointer, function or float (8.2.1). |
| FR-EXP-062 | When the checkpoint surface offers a read of the stream state (7.3.2) and `Pair` is on, the harness MUST record the stream fingerprint at each window close and at the branch point, and two runs MUST be reported aligned through a window if and only if their fingerprints are equal at it and at every earlier one (7.3). |
| FR-EXP-063 | A point whose hash equals an earlier point's MUST carry `SameAs` the index of the first (8.2.1). |
| FR-EXP-064 | The size of an experiment (points, runs, cells) MUST be checked against 13.2 with a checked multiplication before anything is run, and an overflow or an excess MUST be `ErrOversize` (3.6, 8.6.2). |
| FR-EXP-065 | A run with the measures' observers and a run with none MUST end at the same digest, and `Pair` MUST NOT change a digest (15.2). |
| FR-EXP-070 | `Check` MUST resolve and validate every point without running a tick and return its status, error text and warnings (14.3). |
| FR-EXP-071 | `Check` MUST report `ranged_narrower_than_design` for a spread over a range with fewer values than `N`, and `Assemble` MUST report `duplicate_point` with `SameAs` for points whose worlds have one hash (14.4). |
| FR-EXP-072 | The paths of every experiment shipped in the repository MUST each resolve, by the table-prefix scope and the snake form of the row's parameter against the last segment, to a row of the parameter-owner table of `specs/README.md` read as text (11.3.1). |

### Non-functional

| ID | Requirement |
|----|-------------|
| NFR-EXP-001 | For a given experiment the whole `Render` MUST be byte-identical across runs and platforms; the golden test SHOULD run on at least two architectures (architecture.md 7.4, testing.md 7.1). |
| NFR-EXP-002 | The package MUST NOT start a goroutine, call `time.Now`, use `math/rand`, keep package-level mutable state, use floating point or write to a file (architecture.md 7.1, 7.3 extended to this package by an owed row, 12.2, 3.5). |
| NFR-EXP-003 | The harness's own work for a run SHOULD be under 1% of the run at 400 ticks of the `one` pack and MUST NOT be quadratic in agents or ticks; the limit is enforced by an ordinary test of counted operations and a benchmark with `b.ReportAllocs()` reports the trend (testing.md 8.1, 8.2). |
| NFR-EXP-004 | A unit MUST hold at most one world and one checkpoint at a time and a record MUST be bounded by the formula of 8.6.1. |
| NFR-EXP-005 | The overlay generator, the permutation and the reductions MUST have fuzz tests that assert no panic and the stated ranges and identities (testing.md 7.4). |
| NFR-EXP-006 | The package, `internal/design` and `internal/reduce` MUST meet the coverage threshold of testing.md 9.1 (90%); every exported identifier MUST have a doc comment and each public entry point SHOULD have a runnable `Example` (architecture.md 6.5). |

## 22. Open questions

1. **A Go-built configuration has no path (3.3.5).** The harness takes documents and
   the loader's override; a consumer that builds a `Config` in Go calls
   `ExecuteUnit` with its own per point. A path setter over `Config` would be a
   second override mechanism. *Next step:* the first consumer that builds in Go
   and wants factors.
2. **Only integers are swept as a value (3.2.2).** A boolean, a name or a list is a
   `Docs` level. *Next step:* if document levels prove clumsy for a strategy name,
   add a string change that the loader already accepts.
3. **No adaptive design (3.7).** Finding the gain at which a loop goes unstable is a
   bisection, which is a consumer's loop of experiments. *Next step:* CLB's first
   procedure that needs it.
4. **A read of the stream state (7.3.2).** `Pair` needs one 16-byte read of the
   source's state at a boundary from CHK: `Info` decoding the first 20 bytes of
   section 1 (with an error result), or another accessor. The alternative is a
   `Draws` field on `TickEnded` (words drawn in the step): OBS refused counts that
   an observer can derive from events (4.4.3.1), and a draw count is **not**
   derivable, so the refusal does not carry over; it would give the count exactly
   at every tick. *Next step:* the supervisor's decision; without either `Pair` is
   refused at `Validate` and the gain stays.
5. **Equal `(Hash, Seed)` could be run once (8.1.3).** A spread over a narrow
   integer range repeats configurations. *Next step:* the first sweep where it
   shows (a memo keyed by hash and seed inside `Run` cannot change a result).
6. **A library-owned runner and pool (5.3).** Architecture.md 12.2 would have to
   change first: an owner with `Start` and `Close`, and every goroutine with a
   stop time. The unit of 5.1 is the seam it would use. A unit's checkpoint could
   also be handed out so that the branches of one unit run on several workers.
7. **The `stale_warm_start` warning (3.3.3, 14.4)** needs the list of paths the
   viability solver reads, which VIA does not publish. *Next step:* VIA lists them
   (the tunables of its programme: upkeep, recipes, yields), and the warning is on
   by default.
8. **A failure threshold (9.5).** Whether a sweep should stop when a share of its
   runs failed is a policy; the callback gives it. *Next step:* the first long
   calibration that wastes a night on a bad pack.
9. **Vector measures (3.4.1).** One integer a measure keeps a result a rectangle.
   A price per commodity is thirty measures. *Next step:* if the names get
   unwieldy, a measure that returns a named row, still integers.
10. **Intervals and encodings (8.4.3, 8.2).** A confidence interval needs a
    quantile table or a float; a compact binary encoding of a `Result` would be a
    second canonical form to keep equal to `Render`. *Next step:* a consumer that
    stores ten thousand runs.
11. **A variance decomposition.** The effect range is not Sobol's index. An integer
    estimator exists (a ratio of sums over bins) but needs a rounded term per bin or
    a big integer that architecture.md 3.5 allows only in `internal/steady`.
    *Next step:* the first calibration where cancelling factors matter.
12. **Prefix growth of a spread (4.4.5).** A sequence that is stratified at every
    prefix and extensible would need verified direction numbers. *Next step:* the
    first calibration that wants to add points to a finished batch.

### Conflicts found between specs

| # | Between | The conflict | Recommendation |
|---|---------|--------------|----------------|
| X1 | IND FR-IND-001 and 3.4.0 here | the import-graph test forbids any package but `indicator`, its tests, `otelobserver` and `examples/` to import `indicator`, and any but `indicator` to import `internal/measure` | **resolved on EXP's side**: `experiment/` imports neither (measures are the consumer's closures; the reductions are `internal/reduce`); no IND row |
| X2 | WLD 7.6.1 and WLD-A5 against 7.4 | "valid only between runs that differ in a parameter that changes no draw count" (7.6.1), "valid only for a change that moves no draw count" (WLD-A5); CHK 11 item 3 and IND 11.2 say "is valid when the difference moves no draw count", a sufficient condition | WLD: "**exact** draw for draw, and a variance reduction that the harness measures, only while aligned; the estimate of a difference of means from seed-matched runs is unbiased whether aligned or not"; CHK 11 item 3: "one parameter ... differing" cannot occur between two restores of one checkpoint (`ErrCheckpointConfig`) |
| X3 | CHK 3.1.2, 5.4.3, FR-CHK-006 | `Info` "reads only the frame" and has five fields; the stream state is in section 1 | the read of 7.3.2, in one of two forms, or `Pair` stays unavailable |
| X4 | architecture.md 2.6, 2.7, 7.3 and `AGENTS.md` layout | the layout is closed and has no `experiment/` or `indicator/`; 7.3 binds the root and `internal/` | add both packages and the import rule; extend 7.3 to `experiment/` |
| X5 | testing.md 9.1 | the coverage table has no row for `indicator/` or `experiment/` | add both at 90% |
| X6 | IND 4.2.5 and 10.5.1, FR-IND-055 | a tally fed `WorldRestored` while open "starts a new window"; a restore "changes nothing" in a straddling window | IND picks one; EXP does not rely on either (6.1.1) |
| X7 | CFG 6.2, 6.3, FR-CFG-021 | the snake-case rule is not stated (acronyms), the `schema` check is not said to be per merged tree, and CFG exports no pure hash function of a `Config` | state them; add the function (14.3) |

### Edits owed to other specs

One consolidated table: every file, section and change. None alters a result that
a test could depend on (nothing has run). The supervisor applies them; the handoff
is derived from this table.

| File | Section | Change |
|------|---------|--------|
| `specs/README.md` | index, reading order | add `spec-experiments.md` (core): "The sweep harness: an experiment as plain data over a base pack by field path through the loader's own layering, three designs (grid, one at a time, a counter-based Latin hypercube), consecutive-integer replica seeds, a sequential runner with a contract for a consumer's own parallel one, branching from one burn-in, aligned and paired runs with a measured gain, integer reductions, a comparison and a sensitivity, failure as a result, the cost, `FR-EXP`/`NFR-EXP`"; reading order after `spec-indicators.md` |
| `specs/README.md` | code table row `EXP` | "exists; Requirements section written (FR-EXP-001 to 072, NFR-EXP-001 to 006); core; no `Config` field; all bounds are placeholders" |
| `specs/README.md` | decisions taken | rows: *Does the harness draw randomness?* No: counter-based and stateless (4.5). *How does a point differ from the base?* By overlay documents through the loader (3.3). *Can a branch change a parameter?* No: the hash (6.2). *When is a paired comparison valid?* Exact only while aligned; the estimate is unbiased always; the gain is measured (7). *How is a failed run counted?* Never dropped (8.3.1, 9) |
| `specs/README.md` | glossary | experiment, factor, level, change, design (grid, one at a time, spread), point, replica, run key, unit (of work), plan, record, result, window (sample), transient, final reading, series, measure, criterion, pass share, reduction, branch, variant, continuation, aligned (prefix, horizon), stream fingerprint, pairing gain, effect range, elasticity, overlay |
| `roadmap.md` | tier 3 table, existing specs table, "Suggested next specs" | EXP `planned` to `exists`; a row in the existing-specs table; remove EXP from the planned list |
| `documentation_index.md` | rows | `specs/spec-experiments.md` and `specs/models/exp/` |
| `specs/models/README.md` | contents | a row for `exp/`: the counter-based Latin hypercube, the rejected Kronecker sequence, the integer reductions, the pairing toy and the cost arithmetic |
| `rules/architecture.md` | 2.6 layout, 2.7, 3.2, 7.3 | the packages `experiment/` (public) and `internal/design`, `internal/reduce`; `experiment/` may import the root and `config/`, and none of them imports it; 7.3 extended to `experiment/`; the glossary terms; a fresh read-receipt tag (X4) |
| `AGENTS.md` | layout | the same packages |
| `rules/testing.md` | 9.1 | rows for `indicator/` and `experiment/` at 90%; a fresh tag (X5) |
| `spec-indicators.md` | 4.2.5, 10.5.1, FR-IND-055; 11.2 | one rule for a tally fed `WorldRestored` (X6); the paired-comparison row reworded to "exact" (X2); `Window`'s fields defined, as EXP's measures are the consumer's and need them |
| `spec-checkpoint.md` | 3.1.2, 5.4.3, FR-CHK-006, 11 | a read of the stream state (7.3.2; `Stream [16]byte`, the 16 bytes after `"pcg:"`; `Info` then needs an error result or section 1's first bytes in the frame) (X3); a named error for `Checkpoint` of a healthy world (9.1); the EXP row (3) reworded (X2) |
| `spec-config.md` | 6.2, 6.5, 8, FR-CFG-021 | state the snake-case rule with acronyms; say the `schema` is checked on the merged tree; export a pure hash function of a `Config`; (X7) |
| `spec-viability.md` | 11 | publish the list of paths the solver reads (3.3.3, open question 7) |
| `spec-world.md` | 7.6.1, 13 (EXP row), 17 (WLD-A5 condition), 22 question 4 | the paired-comparison wording (X2); question 4 answered in part: the harness measures the pairing gain, per-purpose streams stay open |
| `spec-stability.md` | 4.8, 5, 8, 15 question 6 | merge EXP-A1 to A5 as hazards, EXP-D1 to D3, EXP-S1 to S4 (17); remove an EXP hook row if there is one; question 6 cites `spec-experiments.md` 12.3 |
| `spec-reference-chain.md` | 7.1.1, 8.2 | a scenario that varies one scalar is a `Change` of an experiment; the seeds 1 to 10 are `Seed0` 1 and `Replicas` 10 |
| `spec-observability.md` | 10.3 | a row: a sweep with no measure that has an `Observer` registers none (3.4.4); "the format of a diff between two runs of a sweep" cites `spec-experiments.md` 8.2.2, not 9.5.2 |
| `notes.md` | the open notes of 20 | update N-015, N-016, N-017, N-027, N-086, N-103; add the new notes of the handoff; the index; a fresh tag |

## 23. Where this lives in the code

| Concern | Home |
|---------|------|
| `Experiment`, `Factor`, `Level`, `Change`, `Step`, `Measure`, `Criterion`, `Branch`, `Variant`, `Validate`, the closed sets, the sentinels and the bounds | `experiment/experiment.go` |
| `Plan`, `RunUnit`, `ExecuteUnit`, `Run`, `Check`, `RunOption` | `experiment/run.go` |
| the branch procedure | `experiment/branch.go` |
| `Record`, `Result`, `Assemble`, the reductions, `Compare`, the sensitivity | `experiment/result.go`, `experiment/reduce.go` |
| `Render` | `experiment/render.go` |
| the grid, one at a time and spread index functions, the permutation, the seeds, the overlay text (primitive integers and strings) | `internal/design` |
| 128-bit sums, half-to-even division, the 128-bit integer root, the nearest-rank quantile, the exact sign tail | `internal/reduce` (stdlib and `math/bits`; IND's `internal/measure` is IND's alone, FR-IND-001) |
| the SplitMix64 finaliser | `internal/ordering` (WLD 7.1.2, N-019) |
| the models | `doc/specs/models/exp/` |

The primary type is `experiment.Experiment`, as `list.List`; no other exported name
begins with `Experiment` (architecture.md 2.5). `Step` and `Run` are this package's
names for a script call and a sweep, and are not WLD's `Step` and `Run`: they are
in another package and take other arguments (architecture.md 3.2 asks the glossary
to say so).
