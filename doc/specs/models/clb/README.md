# Models behind `spec-calibration.md`

Seven standard-library Python scripts, models of the **rules of the spec** on synthetic
or invented values. They are not runs of the library and show no default to be stable
(N-027). Run only in Docker, never with the host interpreter, and with
`PYTHONDONTWRITEBYTECODE=1` (read-only mount, so no `__pycache__` is left):

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/clb":/w:ro \
  -w /w python:3.12-slim python grid.py
# registry.py and refs.py read the spec files two levels up and the rule files:
# mount `doc`.
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc":/d:ro \
  -w /d/specs/models/clb python:3.12-slim python registry.py
```

| Script | Output | What it does | Spec |
|--------|--------|--------------|------|
| `registry.py` | `registry.out` | parses the registry of section 6 as plain text and runs the checks of 11.1: cells, closed sets, references to STY facts and STB loops, the IND presets and the OBS events read from their files, the anchors with Status `anchor`, owner sections (the README's "section" and "sections", failing a row that parses none), `z` and `geo` against the owner's default and bound, the off number against the owner's bound and the cut, `out` (the WarmStart `Prices`, `Wages` and `Groups`, `ReferencePrice`, `RegionPrice.Price`), the `in` and `chk` flags against the first table of VIA 11 (a key and a container are not checked), coverage of the owner-table pairs (and the `(also Role` remark of `Upkeep`) both ways and of the catalogue of `spec-config.md` 3.7 | 4, 6, 11.1 |
| `refs.py` | `refs.out` | mechanical checks of the spec itself: requirement IDs defined and referenced, section references, files named, no shell-quoting residue, no setting words | 21 |
| `grid.py` | `grid.out` | the grid rule of 5.2 in integers: `geo` and `lin` levels, the cut of an off value and of the levels beyond it, the invalid-level skip, the edge record (`bound`, `off`, `rule`), a `z` row at 0 (no grid), the overflow error of the checked multiply, 20 000 random cases and the table of 19.1; `set:` and the widening of 5.3 are not modelled | 5.2, 19.1 |
| `stop.py` | `stop.out` | exact binomial arithmetic of SR2 at 10 and 50 seeds, joint pass of up to 63 conjuncts, the winner's curse, attempts | 7.1.2, 7.4, 19.4 |
| `cost.py` | `cost.out` | run counts and core-hours of the stages from the placeholders of `spec-scale.md` 6, EXP 10 and CHK 10.3 | 7.3, 19.8 |
| `choose.py` | `choose.out` | the feasible set, the minimal change, the Pareto set (non-dominated B-count vectors), the integer alias test with its bounds, the exclusion of a reading above `2^26` or clamped, the Spread tie to the lower level, and half to even; it does not model the `d` comparison by distributions | 7.2.3, 7.5, 10.2, 19.3, 19.7 |
| `worked.py` | `worked.out` | the numbers of the worked examples 19.1 to 19.8 that are computed (grids and the cut, the screen, the choice table, the seed arithmetic, the overlay, the alias test, the cost count and the stability-record cost), asserted; 19.5 (a record with invented values) is a format example and is not asserted | 19 |
