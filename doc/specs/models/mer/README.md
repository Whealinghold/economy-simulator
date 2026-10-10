# Models behind `spec-merchants.md`

Prototypes of the spec's rules, not of the library (see `../README.md`). Standard
library only. Run each in Docker, never with a local interpreter, and without
writing bytecode:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/mer":/w -w /w python:3.12-slim python lane.py
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/mer":/w -w /w python:3.12-slim python contract.py
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/mer":/w -w /w python:3.12-slim python worked.py
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/mer":/w -w /w python:3.12-slim python backhaul.py
```

| Script | Output | What it shows |
|--------|--------|---------------|
| `lane.py` | `lane.out` | Sections 7.2, 7.3 and register entry MER-A1: merchants that ship a lot when the rumoured gap exceeds their bar, with identical bars, the lane lock, a one-sided spread of the bars (even, and by the Weyl value of the agent ID), an independent error, a share of the effect at dispatch, longer delays, a larger lot, a lot cap, the number of merchants, eight seeds, the smallest spread that settles the gap against the slope condition, and how even the Weyl spread of IDs is. A linearised gap, floating point inside the model only, seeded `random.Random` (the library draws from the world's source). Extends `../inf/shippers.py` |
| `contract.py` | `contract.out` | Section 9 and MER-A5: the matching of indivisible offers against divisible takes on a lane, the formation, and the settlement in four cases, with the check after each step that the money of all accounts is constant and the escrow account is empty. Integer arithmetic. Ties are broken by agent ID here and by a drawn key in the spec (9.4.2); no number quoted depends on it |
| `worked.py` | `worked.out` | Section 19: the integer arithmetic of the two worked examples (travel times, lags, the lot per carrying unit, the cautious revenue, the cost of carriage, the fee share of four agents, the haul ceiling, the break-even and the offer limit, the contract price, the cost comparison of a mode, the Weyl values). Every number is a labelled placeholder |
| `backhaul.py` | `backhaul.out` | Section 10: the unit cost of a haul unit in a round-trip and a one-way reading of the carrier's recipe and the smallest gap a merchant ships on. Integer arithmetic. Re-reads the example of `../trn/worked.py` |
