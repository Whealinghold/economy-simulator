# Calibration records

This directory holds the records of the library's own calibration
(`doc/specs/spec-calibration.md` 8). It is empty of records today: nothing has
been calibrated yet, and a record is written only by a reviewed run (N-027).

## A record is data about a decision, not a golden file

A stage record says what a calibration of a pack at one stage tried, on which
seeds, what it chose and why it stopped. It is **never a golden file of the
economy** (spec-calibration.md 8.4): nothing compares a run to it, and no test
regenerates it with `-update`. It changes only by a reviewed change that says
what moved (`doc/rules/testing.md` 7.2), and the golden files of a pack move
only when the pack does. No test asserts a number of the economy from a record
(N-027).

## Layout

```text
testdata/calibration/
  README.md             this file
  dryrun.shape.txt      the shape of the dry run's record (Phase 7 of plan 2-21)
  <pack>/<stage>-<n>.txt  one record per (pack, stage, attempt): c1-1.txt, c4a-2.txt
testdata/golden/stylised/<pack>.stability.txt   a stability record (STY 8.5)
```

`<stage>` is the stage in lower case (`c0` to `c9`, with `c4a` to `c4c` and
`c7a`, `c7b`) and `<n>` the attempt, so the files of a pack sort in the order
of the stages. `dryrun.shape.txt` sits beside the pack directories, not in one,
so the glob of the library check never reads it: it holds masks that no
`Parse` accepts.

`internal/calibration` checks every file here (`TestLibraryRecords_Checked`):
each record is parsed, checked one by one (`CheckRecord`) and, per pack in file
order, as a sequence (`CheckSequence`: the stage order, the three attempts, the
search blocks and seeds, the lineage rule). A calibration overlay of a pack
lives in `testdata/chains/calibrated/<stage>-<n>.toml` and is checked by
`TestOverlays_Purity`: only K and D keys, and a header comment that names its
record.

## The grammar by example

A record is plain text: one `<keyword> <fields>` per line, integers in decimal,
digests as 16 lower-case hex digits, `-` for none, LF line ends, and the lines
in a fixed order. The groups of the `pack`, `commit` and `config` lines are
separated by two spaces and every other field by one. `note` is the only line
with free text. There is no float, no time and no map order, so two runs of the
same experiment give the same bytes.

```text
calibration 1
pack one  lineage 5a1c0e4477d3b210  stage C4a  attempt 1  block 1
commit 4f2a9c1  parent -
config base 91d3a7c2e05b4a18  chosen 0be4417f6a2d93c5
document base.toml 3b9c11aa5e07f4d2
experiment screen design oat points 5 replicas 10 seed0 101 seeding same ticks 1095 transient 365 every 365 salt 0 render c7f20b19a4e83d60
factor cost_floor key World.CostFloor grid geo base 7000 lower 3500 upper 14000 edge none
set World.CostFloor 3500
confirm seeds 1..50 control yes
detector R5a none working yes onset -
decision stop
note CostFloor halved
```

The full grammar, with every keyword, is spec-calibration.md 8.2 and the worked
record is 19.5. `Render` writes exactly these bytes and `Parse` accepts exactly
what `Render` writes, so a record that parses renders back to itself.

## Seeds

A search runs on its own block `b`, the seeds `100 b + 1` to `100 b + 100`, and
a block serves one choice. A confirm runs on the seeds 1 to 50 (tier L: 1 to
10). No search seed is in 1 to 60, and no confirm seed is a search seed
(spec-calibration.md 7.1).
