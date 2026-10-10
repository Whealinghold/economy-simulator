# models/mig: prototype models behind spec-migration.md

Stylised models of the RULES of `spec-migration.md`, not of the library, and not claims that any default is stable
(notes N-015, N-027). The migration rules are the spec's integer rules; the economy is replaced by one assumed
function of the households of a region. Run each with Docker, never a local interpreter, and never leave bytecode
behind:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/mig":/w -w /w python:3.12-slim python loop.py all
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/mig":/w -w /w python:3.12-slim python worked.py
```

| Script | Section | What it holds |
|--------|---------|---------------|
| `loop.py` (`linear`, `ablation`, `caps`, `pipeline`, `herding`, `defaults`, `swing`; `all` runs them; output `loop.out`) | 16 | The delayed integrator and its limit `kappa tau = pi/2`; each damper alone against an undamped base; the outflow and pipeline caps; the invisible pipeline against the journey time; herding across two equal destinations; the defaults over seeds, elasticities and journeys; the product of outflow share and window against the margin that decides when the flow reverses. |
| `worked.py` (output `worked.out`) | 15 | The integer arithmetic of the two worked examples: the estimate, the horizon, the margin from the Weyl offset, the journey times, provisioning and the caps. |
