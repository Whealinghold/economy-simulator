# models/org: prototype models behind spec-institutions.md

Checks of the RULES of `spec-institutions.md` (the legal price as a clamp of one side's
limit, and the membership round) and the arithmetic of its worked examples. They are
prototypes of the spec's rules, not of the library, and not claims that any default is
stable (notes N-015, N-027). Run each with Docker, never a local interpreter, and never
leave bytecode behind:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/org":/w:ro -w /w python:3.12-slim python clamp.py
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/org":/w:ro -w /w python:3.12-slim python admit.py
```

| Script | Section | What it holds |
|--------|---------|---------------|
| `clamp.py` (output `clamp.out`) | 6, 13.2, 15 | The default clearing (sort, cross, midpoint half to even) over 20 000 random books with a cap, a floor or both: every trade price is inside the legal range when only the bids above a cap and the asks below a floor are clamped; clamping trades at least the units that rejecting the same orders does and exactly the units an independent count of the k-th clamped bid against the k-th ask gives; every trade price is between its two (clamped) limits, so the world's check of `spec-clearing.md` 8.2 holds. P5: the range in minor units is bounded by `MaxPrice` and a limit times a quantity fits `int64`. The arithmetic of example B (a cap at 12 000 and at 9 000 basis points of a reference price). |
| `admit.py` (output `admit.out`) | 4, 13.1, 15 | The membership round of 4.2 and 4.3 over 3 000 random populations of a body with three regions in its scope, with removals, role changes and moves in between: members of a region never exceed the places from a start with no member, every member is eligible and in the region it was admitted in, the result does not depend on the order of the list, a second round changes nothing, the tie key is the SplitMix64 finaliser (no draw), and a held completion (removals, arrivals that bypass the headroom, queue order, entrants after completions) never takes the households above the places. The arithmetic of example A (a craft body of 12 places and 14 households). |
