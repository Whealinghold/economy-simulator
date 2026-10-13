# Models of spec-world.md

Small Python scripts (standard library only) and their captured outputs, run in
Docker with `PYTHONDONTWRITEBYTECODE=1`. They are prototypes of the rules of
`doc/specs/spec-world.md`, not of the library (note N-027). See
`doc/specs/models/README.md` for how to run them:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/wld":/w -w /w python:3.12-slim python seed.py
```

`seed.py` is the one source of the vectors of the keyed streams (ADR 0008, WLD 7):
the finaliser, the key of a stream, the site constants, `K(t, id)`, the reseed rule and a
PCG-DXSM. `worked.py` and `stream.py` import it, and so does `../chk/worked.py`
(CHK 6.8, 17.5), so no rule is written twice.

| Script | Output | What it does |
|--------|--------|--------------|
| `seed.py` | `seed.out` | The finaliser, the key of a stream and the site constants (7.1.2 to 7.1.4), the vectors of 7.1.5 (the four bare seeds, nine keys, `K(t, id)`, the reseed of CHK 6.8), asserted against the values Go printed; the bits that differ between the first outputs of neighbouring seeds, ticks and IDs |
| `rotation.py` | `rotation.out` | A cap that binds on tied candidates, under the lower ID, the keyed tie `K(t, id)` and a rotation (10.1.1, WLD-S1) |
| `cadence.py` | `cadence.out` | The reviews per tick of a rotation cadence by raw ID, `Weyl32 mod P`, the Weyl32 bucket (correlated with MIG's margin), the hash bucket (not adopted) and the rule bucket of 9.1.1 (a second multiplier, `0x6A09E667`), over strides 1 to 31 and periods 10 and 30; the correlation of each with the migration margin; the share of coinciding 30 and 10 tick reviews; a group smaller than its period; the drift against a year (9, WLD-S2) |
| `stream.py` | `stream.out` | The share of later draws that differ after one extra draw, with one positional stream and with a stream per tick (both rejected) and with a stream per (site, tick, agent) (adopted, 7.6, WLD-A5, WLD-S4) |
| `worked.py` | `worked.out` | The arithmetic of the worked examples of section 19: the rule bucket of ids 10 to 15, the creation draws of example A and the streams of one tick of example B |

`seed.out` and `worked.out` carry the vector lines that a short Go program (standard
library `math/rand/v2` only, `PCG.Seed` for the reseeded streams) printed too; the two
outputs were equal, on Go 1.25.14 and Go 1.27.2 (PCG is a specified generator, so
the words do not depend on the toolchain). The program is not archived (a `main`
package under `doc/` would be built by `go vet ./...`); it is a restatement of the
rules of 7.1 in 60 lines, and the first code carries the vectors as its tests. One trap
it found: in Go, `2 * 0x9E3779B97F4A7C15` is a constant expression that overflows
`uint64`, so the second increment is `s + gamma + gamma` or a variable.
