# Models behind `spec-subsistence.md`

Two small Docker Python scripts. They are **models of the rules of the spec, not of the
library** (there is no library): they use integers, placeholder parameters (note N-015) and
they do not validate any default (note N-027).

Run, never with a local interpreter:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/sub":/w -w /w python:3.12-slim python staple.py
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/sub":/w -w /w python:3.12-slim python worked.py
```

| File | What it holds | Spec |
|------|---------------|------|
| `staple.py`, `staple.out` | the pinned staple market of `spec-stability.md` 6.1 with 1 370 consumer households added, with and without the own-use mode: the floor off, `f` = 2, 3 and 4 units a tick, the retry on and off, the work floor on and off, an outage of the market, the refused exit by purchase and an exit cover above what purchases build | 11.2, 17.1 |
| `worked.py`, `worked.out` | the integer rules of 4.1, 5.2, 5.3 and 5.5 applied to the two worked examples: the hand-solved surplus factors, `Cov`, the nine-tick trace of one household with its carried remainder, the ramp of a cohort of 40 and the unfed ticks before a first output | 16, 17.2 |

The scripts use only the Python standard library. Each writes to standard output; the `.out`
files are the captured output. A script changes together with the spec section that quotes it.
Output files made by a container are owned by root; `chown` them with a container before
committing (`docker run --rm -v "$PWD/doc/specs/models/sub":/w alpine chown -R "$(id -u):$(id -g)" /w`).
