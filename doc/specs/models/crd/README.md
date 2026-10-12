# models/crd: prototype models behind spec-credit.md

Stylised models of the RULES of `spec-credit.md`, not of the library, and not claims that any default is
stable (notes N-015, N-027). The rules are integer; the economy around them is replaced by an assumed
income. Run each with Docker, never a local interpreter, and never leave bytecode behind:

```sh
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/crd":/w:ro -w /w python:3.12-slim python alloc.py
docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs/models/crd":/w:ro -w /w python:3.12-slim python loans.py
```

| Script | Section | What it holds |
|--------|---------|---------------|
| `alloc.py` (output `alloc.out`) | 3.6, 7, 8.3, 9.1, 20 | The interest rule with its half-to-even rounding, the pro rata allocation with the largest remainder and the stateless key, interest first inside a claim, the waterfall of a removal, the figures of the worked examples of 20.1, 20.2 and 20.3 (the request, the interest, the pot table, the shares); 20 000 random allocations (sum equals the payment, no share above what is owed, no claim moves with the order of the others) and a mini ledger of 6 000 drawn operations (4 128 applied) (`Lend`, `Settle`, `WriteOff`, `Reassign`) with the total of money and the pair identity of 3.6 asserted after every one |
| `loans.py` (output `loans.out`) | 10.2, 10.3, 8.2, 18 | Sixteen borrowers with a 60-tick job and a lump at its end, four lenders, a market account that pays every income so money is conserved: no credit; credit with the horizon of 10.2.1.1 and without its structural tick (a tick short); one bad cycle with the shield at 0, 1 and 3 ticks and a sweep of the shield at `SlackTicks` 0 and 5; a persistent bad income with and without a write-off; start money below the inputs with `Tier2` as what tier 2 can afford and in full, at two values of `MinPrincipal`. Cases H0 and H1 apply the library's delay of the inputs (bought at P3.2, the loan spendable a tick later) with and without the `Late` tick of 10.2.1.2. It shows the cash gap closed by the default request, the default cascade to the lenders, that a write-off lifts the bar and costs the lenders, and that the best shield depends on the horizon |
