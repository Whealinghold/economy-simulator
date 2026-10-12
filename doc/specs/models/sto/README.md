# Models behind `spec-storage.md`

One small Docker Python script. It is a **model of the rules of the spec, not of the library**
(there is no library): it uses integers and placeholder parameters (note N-015), and it does not
validate any default (note N-027).

Run, never with a local interpreter. The output is captured by the host shell, so `room.out` is
owned by the user and not by root:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/sto":/w:ro -w /w python:3.12-slim python room.py > doc/specs/models/sto/room.out
```

| File | What it holds | Spec |
|------|---------------|------|
| `room.py`, `room.out` | six parts: (1) the closed-form start size of a job against the exact largest `k`, in one dimension (20 000 random cases: never above what fits, with the six cases of a gap of 3 or more listed); (2) the scaling of a target to the room, with the clamp for a negative available room; (3) a 4 000-tick random sequence of bids, settlements, job starts and dispatches that leave the upkeep room free, completions and arrivals from the pledge, the tier 1 bid and its discard, consumption and spoilage on one limited account; (4) a weight reduction: 20 000 random cases where the scaled load fits, and the case of a pledge that outlives the cut (a job's output and a cargo); (5) the rule of what goes first for a role change (5 000 random cases, and the worked case of 18.3); (6) the discard of 6.3.2 as 582 random cases and one case. The arithmetic of the worked examples 18.1 and 18.2 is at the end of the output | 5.3.2, 5.5, 6.2, 6.3, 17, 18 |

The script uses only the Python standard library and writes to standard output; `room.out` is
the captured output.
