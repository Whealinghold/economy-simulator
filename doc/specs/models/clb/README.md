# Models behind `spec-calibration.md`

Seven standard-library Python scripts, models of the **rules of the spec** on synthetic
or invented values. They are not runs of the library and show no default to be stable
(N-027). Run only in Docker, never with the host interpreter:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/clb":/w \
  -w /w python:3.12-slim python grid.py
# registry.py and refs.py read the spec files two levels up (mount `doc/specs`):
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs":/s:ro \
  -w /s/models/clb python:3.12-slim python registry.py
```

| Script | Output | What it does | Spec |
|--------|--------|--------------|------|
| `registry.py` | `registry.out` | parses the registry of section 6 as plain text and runs the checks of 11.1: cells, closed sets, references to STY facts and STB loops, owner sections, coverage of the 255 pairs of the parameter-owner table both ways | 4, 6, 11.1 |
| `refs.py` | `refs.out` | mechanical checks of the spec itself: requirement IDs defined and referenced, section references, files named, no shell-quoting residue, no setting words | 21 |
| `grid.py` | `grid.out` | the grid and neighbour rule in integers (`geo`, `lin`; `set:` and the widening of 5.3 are not modelled), with `bound`, `rule` and `off` edges, 20 000 random cases and the table of 19.1 | 5.2, 19.1 |
| `stop.py` | `stop.out` | exact binomial arithmetic of SR2 at 10 and 50 seeds, joint pass of up to 63 conjuncts, the winner's curse, attempts | 7.1.1, 7.4, 19.4 |
| `cost.py` | `cost.out` | run counts and core-hours of the stages from the placeholders of `spec-scale.md` 6, EXP 10 and CHK 10.3 | 7.3, 19.8 |
| `choose.py` | `choose.out` | the feasible set, the minimal change, the Pareto set, the integer alias test with its bounds and overflow error, and half to even; it does not model the `d` comparison by distributions or the equal-vector Pareto rule | 7.2.3, 7.5, 10.2, 19.3, 19.7 |
| `worked.py` | `worked.out` | the numbers of the worked examples 19.1 to 19.8 that are computed (grids, the screen, the choice table, the seed arithmetic, the overlay, the alias test, the cost count and the stability-record cost), asserted; 19.5 (a record with invented values) is a format example and is not asserted | 19 |
