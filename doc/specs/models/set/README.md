# models/set: prototype models behind spec-settlements.md

Stylised models of the RULES of `spec-settlements.md`, not of the library, and not claims that any default is
stable (notes N-015, N-027). The review is the spec's integer rule; the economy is replaced by an assumed function
of the household count. Run each with Docker, never a local interpreter, and never leave bytecode behind:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/set":/w:ro -w /w python:3.12-slim python band.py all
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/set":/w:ro -w /w python:3.12-slim python worked.py
```

| Script | Section | What it holds |
|--------|---------|---------------|
| `band.py` (`rule`, `noise`, `loop`; `all` runs them; output `band.out`) | 4, 13, 14 | The review rule in integers and its invariant over every household count; class changes under a mean-reverting census noise for four bands and two dwells; the staircase of an assumed growth by class, and an absorbing class with and without a class floor. |
| `worked.py` (output `worked.out`) | 12 | The integer arithmetic of the worked examples: effective thresholds with a region scale, the floor, the trace of one region through a change and a held change, the one-class setting. |
