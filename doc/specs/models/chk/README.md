# Models of `spec-checkpoint.md`

Standard-library Python, run only in Docker, never on the host
(`docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/..":/m -w /m/chk python:3.12-slim python <script>`, from `doc/specs/models/chk`; `worked.py` imports `../wld/seed.py`).
They are prototypes of the rules of the spec, not of the library (note N-027).

| Script | Output | What it does |
|--------|--------|--------------|
| `size.py` | `size.out` | the integers each section of the checkpoint writes for the first consumer's world (5 000 agents, the tradable set of 7.7), with a long-job window, with ten thousand closed accounts and for a full-detail realm of 200 000 agents, and their bytes as minimal varints and as 8-byte integers; value magnitudes are invented placeholders. Quoted in section 10.2 |
| `worked.py` | `worked.out` | the primitives of 5.2 and the spellings it rejects, the world section of the construction of `spec-world.md` 19.1 (stale under ADR 0008, see N-214: the section no longer holds a PCG state), a toy frame with its SHA-256 trailer, an account, and the reseed vectors of 6.8 (the rule of ADR 0008, computed by `../wld/seed.py`, each value asserted against a Go program's output). Quoted in section 17 |

The Go facts of section 16 (the non-minimal `binary.Uvarint`, the PCG layout, and the old reseed
vectors, which ADR 0008 replaced) were produced by `go run` of a short
standard-library program in `golang:1.25`; it is not archived (a `main` package under
`doc/` would be built by `go vet ./...`). Its outputs are quoted in the spec and
re-checked by `worked.py`.
