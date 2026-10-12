# Models behind `spec-employment.md`

Prototypes of the spec's rules, not of the library (see `../README.md`). Standard
library only. Run each in Docker, never with a local interpreter, and without
writing bytecode:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/emp":/w -w /w python:3.12-slim python round.py
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/emp":/w -w /w python:3.12-slim python circuit.py
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/emp":/w -w /w python:3.12-slim python labour.py
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/emp":/w -w /w python:3.12-slim python worked.py
```

| Script | Output | What it shows |
|--------|--------|---------------|
| `round.py` | `round.out` | Sections 5 and 6 and EMP-A4: the labour round (admission, keys, first-fit matching, the wage `R + phi * (Bid - R)` half to even, formation with its refusals, the escrow), the release of a tick's wage, the completion with the refund, and the three interruptions, over 500 random scenarios, with the check after every step that the money of all accounts is constant, no balance is negative, the escrow holds `wage * k * ticks left` and the wage lies between the reservation and the bid. Integer arithmetic; ties are broken by the stateless key `K(t, id)` of `spec-world.md` 10.1.1 (no draw); `random.Random` only builds the scenarios |
| `circuit.py` | `circuit.out` | Section 11 and EMP-A3: the money circuit of a world with employers who spend only an upkeep and the cap of AGT 7.4.4, with the production halt. The mean number of hired households settles at `h* = min(places, E * u_e * (1 + SpendMaxShare * SpendHalfTicks / 10 000) / ((1 - phi) * a))` (exact for `SpendMaxShare` 0, an upper bound otherwise: the asymptote of the cap is reached only at a large excess). Prices are fixed to isolate the quantity adjustment |
| `labour.py` | `labour.out` | Section 17, EMP-A1 and EMP-A2: the price channel of hired labour with an inelastic staple, a midpoint wage and a margin-seeking employer, over ten seeds: a glut sends the price to the employers' zero-margin price `c / a` (and the wage with it), a shortage to the ceiling, and the hired households per tick of a glut run between none and all the places (the output prints the minimum and maximum of the last 100 ticks). A caricature of the price rule of `spec-beliefs.md`, with a seeded gate |
| `worked.py` | `worked.out` | Section 19: the integer arithmetic of the two worked examples (the bid, the wage, the positions, the escrow, the pay schedule, a fall of weight) and the closed form of section 11.2. Every number is a labelled placeholder |
