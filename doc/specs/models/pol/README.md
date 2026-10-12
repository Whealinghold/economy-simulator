# models/pol: prototype models behind spec-polity.md

Checks of the DEFINITIONS of `spec-polity.md` (chain, covers, common, crossing, between)
and the arithmetic of its worked examples. They are prototypes of the spec's rules,
not of the library, and not claims that any default is stable (notes N-015, N-027).
Run each with Docker, never a local interpreter, and never leave bytecode behind:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/pol":/w:ro -w /w python:3.12-slim python chain.py
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/pol":/w:ro -w /w python:3.12-slim python worked.py
```

| Script | Section | What it holds |
|--------|---------|---------------|
| `chain.py` (output `chain.out`) | 4.1, 4.3, 14 | The definitions of 4.1.1 by chains and sets, against a second formulation that climbs to the lowest common ancestor, on 2 000 random forests (up to 30 jurisdictions, 40 regions, at most 8 levels, a region with none): every ordered pair of regions compared, the reversal symmetry of a crossing, the common suffix, the covered lists by descent against by chain, and `Between` against its definition on random edges; and the depth bound (a chain of 8 entries builds; 9 levels are rejected). |
| `worked.py` (output `worked.out`) | 12 | The integer arithmetic of the worked examples: the ID tables, the chains and crossings, the `Between` lists, the agent IDs with the treasuries, and the three combining modes along a chain. |
