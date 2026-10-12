# models/lnd: prototype models behind spec-land.md

Stylised models of the RULES of `spec-land.md`, not of the library, and not claims that any default is
stable (notes N-015, N-027). The rules are integer; the economy around them is replaced by an assumed
demand. Run each with Docker, never a local interpreter, and never leave bytecode behind:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/lnd":/w:ro -w /w python:3.12-slim python tiers.py all
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/lnd":/w:ro -w /w python:3.12-slim python rent.py
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/lnd":/w:ro -w /w python:3.12-slim python erosion.py
```

| Script | Section | What it holds |
|--------|---------|---------------|
| `tiers.py` (`rule`, `worked`, `worked_b`, `all`; output `tiers.out`) | 4, 5.4, 15 | The entitlement by largest remainder, the own, commons and let tiers, the let demand clipped by cash, the starts cut by the money left, the lessor supply on the units taken and the attribution to tenants, with the conservation and bound checks over 20 000 random scenarios (run with and without cash), the arithmetic of worked example A and of example B before and after the release |
| `rent.py` (output `rent.out`) | 5.5, 16 | The posted rent that moves by a bounded step with the units unserved, against tenants that respond with a delay: the limit cycle against the step and the delay, a band that binds below the market-clearing rent, and the half-span of the cycle against `3 * step * (delay + 1) / 4` |
| `erosion.py` (output `erosion.out`) | 6.4, 16 | The share of titles that has passed to the pool when each removal hands its titles to the estate and every birth is landless, against the closed form |
