# Model behind `spec-money-supply.md`

`flows.py` is an integer model of the money flows of `spec-money-supply.md` (the
carry of 6.1, the rate levy of 6.3, the order of grants and levies of 5.2, the
conservation of 3.1) on a few toy accounts. It is a prototype of the spec's rules,
not of the library, with Python integers (so the 128-bit intermediates are not
exercised) and placeholder numbers. `flows.out` is its output, quoted in sections
9.3, 17 and 18 of the spec.

Run it with Docker, never with a local interpreter:

```sh
docker run --rm -v "$PWD/doc/specs/models/mon":/w -w /w python:3.12-slim python flows.py
```

Standard library only. Proposed line for `doc/specs/models/README.md` (the supervisor
adds it): `| mon/ | spec-money-supply.md sections 9, 17, 18 | The integer rules of the money flows: carry, rate levy, conservation, a balanced pair (flows.py, flows.out) |`
