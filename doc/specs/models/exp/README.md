# Models of spec-experiments.md

Small Python scripts (standard library only) and their captured outputs, run in
Docker with `PYTHONDONTWRITEBYTECODE=1`. They are prototypes of the rules of
`doc/specs/spec-experiments.md`, not of the library (note N-027). See
`doc/specs/models/README.md` for how to run them:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/exp":/w -w /w python:3.12-slim python lhs.py
```

| Script | Output | What it does |
|--------|--------|--------------|
| `lhs.py` | `lhs.out` | The counter-based Latin hypercube of section 4.4: a four-round Feistel permutation with cycle walking over every `N` from 1 to 300 (and larger), exact stratification of every axis, statelessness, the midpoint value of an integer range, the empty cells of a 2-D projection and the column correlation against a stateless hash draw |
| `kronecker.py` | `kronecker.out` | The **rejected** generalised golden ratio (Kronecker) sequence in 2^64 fixed point (4.7): its multipliers computed by exact bisection (`d` = 1 gives the SplitMix64 gamma), and the one-dimensional counts per bin that show the clumping at small `N` |
| `reduce.py` | `reduce.out` | The integer reductions of section 8 checked against exact fractions: half-to-even division, the nearest integer root, the nearest-rank quantile, the sample standard deviation, the exact sign-test tail for every `n` up to 62, the effect range and share, the elasticity, and the arithmetic of the examples |
| `pairing.py` | `pairing.out` | A toy world on one stream with a state-dependent count of words: the aligned share, the first divergent sample, the paired and unpaired spread and the gain, for a calm and a busy world (section 7.4.3) |
| `cost.py` | `cost.out` | The arithmetic of section 10 from the placeholder costs of `spec-scale.md` 6: one run, a sweep, the saving of a branch and the size of a result |
| `worked.py` | `worked.out` | The points of the worked examples of section 19: a grid, one at a time, a spread of 8, the overlay documents, the seeds and a branch plan |

`worked.py` imports `lhs.py`; run it from this folder.

The constant `0x243F6A8885A308D3` in `lhs.py`, `kronecker.py` and `worked.py` is the design constant of 4.4.2 (the default salt of a design), not a reseed constant: the reseed rule of `spec-checkpoint.md` 6.8.2 has none, and no script here reseeds a world.
