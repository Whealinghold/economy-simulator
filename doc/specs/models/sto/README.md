# Models behind `spec-storage.md`

One small Docker Python script. It is a **model of the rules of the spec, not of the library**
(there is no library): it uses integers and placeholder parameters (note N-015), and it does not
validate any default (note N-027).

Run, never with a local interpreter:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/sto":/w -w /w python:3.12-slim python room.py
```

| File | What it holds | Spec |
|------|---------------|------|
| `room.py`, `room.out` | the closed-form start size of a job against the exact largest `k` (20 000 random cases: never above what fits), the scaling of a target to the room, a 4 000-tick random sequence of bids, settlements, job starts and completions, arrivals, consumption and spoilage on one limited account (load plus pledge plus reserved never above room; a settlement, a completion and an arrival never fail), and the arithmetic of the two worked examples | 5.3.2, 6.2, 17, 18 |

The script uses only the Python standard library and writes to standard output; `room.out` is
the captured output. Output files made by a container are owned by root; `chown` them with a
container before committing.
