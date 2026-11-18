# colony

The second setting: the crater colony of `doc/specs/spec-reference-chain.md` 4.4.7
(10 commodities, 10 recipes, 5 roles, 1 000 households, two seasons). It is a runnable
command over an embedded pack that shares no identifier with `examples/realm`, and
it contains no Go specific to the pack: a new chain is a configuration file
(`architecture.md` 9.4).

## Build and run

```sh
go run -C examples/colony . -seed 1 -ticks 400 -every 100
```

It prints the tick, the households and the price of each commodity every `-every`
ticks. `-audit` audits the books after every step. Prices are integers in the one
denomination preset of `denomination/` (`cr`), to show that no ladder is required.

## The borrowing player

```sh
go run -C examples/colony . -borrow -ticks 60
```

`pack/credit.toml` turns the credit module on over the pack: the algae tenders may borrow and
the drillers lend (placeholder figures of the example). `borrow.go` adds one algae tender
through `AddAgent` with the directed strategy and asks for a loan through `Direct`. It prints
the player's money, the principal it owes and the tick its claim falls due, one line a tick:
the loan forms at the first round, the claim is settled when it falls due, and the player
asks again. It uses the public API only (`CreditRequest`, `Directive.Borrow`, `AgentState.Credit`).

## The pack

`pack/second.toml` and `pack/warm-second.toml` are byte copies of
`testdata/chains/second/`. The root module's `TestExamples_PackCopiesAgree` fails if
they drift, so the example cannot run a pack the reference tests no longer cover.
`pack/credit.toml` is the example's own layer and has no copy.

## What it shows

- A second, unrelated setting runs through the public API alone
  (`TestExamples_NoInternalImports`, `TestColony_SharesNoIdentifierWithTheRealm`
  in the root module).
- A commodity, a recipe and a role added by one more layer need no Go code
  (`TestColony_ExtraLayerNeedsOnlyConfiguration`).

## Test

```sh
go test -C examples/colony -race ./...
```
