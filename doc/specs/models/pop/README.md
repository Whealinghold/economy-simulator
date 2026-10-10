# models/pop: prototype models behind spec-population.md

Stylised models of the RULES of `spec-population.md`, not of the library, and not claims that any default is stable
(notes N-015, N-027). Run each with Docker, never a local interpreter:

```sh
docker run --rm -v "$PWD/doc/specs/models/pop":/w -w /w python:3.12-slim python fixed_point.py
```

| Script | Section | What it holds |
|--------|---------|---------------|
| `fixed_point.py`, `sweep.py` | 5.4, 5.6, 13.1 | The Malthusian fixed point: approach from above and below, the gain-times-delay sweep, a productivity shock and a death-schedule shock. The economy is one assumed function of households. |
| `stake.py` | 4.2, 5.5, 13.2 | The stake loop: pool-funded births with deaths poorest first and spread, and the money-limited population `M / stake`. |
| `worked.py` | 12 | The integer arithmetic of the two worked examples. |
