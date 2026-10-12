# models/plc: prototype models behind spec-policy.md

Checks of the RULES of `spec-policy.md` (an integrating rule on a plant with a delay,
the integer rule with its dampers, and the arithmetic of its worked examples). They are
prototypes of the spec's rules, not of the library, and not claims that any default is
stable (notes N-015, N-027). Run each with Docker, never a local interpreter, and never
leave bytecode behind:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/plc":/w:ro -w /w python:3.12-slim python delay.py
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/plc":/w:ro -w /w python:3.12-slim python cycle.py
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/plc":/w:ro -w /w python:3.12-slim python worked.py
```

| Script | Section | What it holds |
|--------|---------|---------------|
| `delay.py` (output `delay.out`) | 8.3.5, 15 (PLC-A1), 16 | The integral rule `u[k+1] = u[k] - K (y[k] - r)` on a pure delay of `d` decisions: the largest stable `K` by simulation and bisection against the closed form `2 sin(pi / (2 (2d + 1)))` for `d` 0 to 8; and the largest gain per decision and per tick for a plant lag of 30 and 60 ticks at several cadences. Floating point, because it models the rule and is not the library. |
| `cycle.py` (output `cycle.out`) | 8.3, 15 (PLC-A1, A2, A4), 16 | The integer rule (truncation toward zero, basis points) on a linear plant with a delay of 0 or 2 decisions after a persistent shock: with a gain below and above the bound, with and without the step limit, the deadband and the lever's bounds, with and without noise; and a stuck reading (a stale price) with and without the age guard. |
| `worked.py` (output `worked.out`) | 14 | The arithmetic of the worked examples: the buckets and due ticks of three actors, a schedule's applications, five decisions of a rule, two weighted gauges with their rounding, the expansion of an `Against` lever in the kingdom of `spec-polity.md` 12.1 and the checks of a returned list. |
