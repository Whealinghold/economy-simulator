# realm

The first consumer: a fantasy realm served over HTTP. It carries its own pack (the
rebalanced `five` chain with every identifier renamed to the realm's words), adds a
player to the world, takes the player's orders, lets you shock a harvest or close a
road, and saves and resumes the whole run. It is its own Go module and uses the
public API alone (`TestExamples_NoInternalImports` in the root module checks it), so
it is the proof that a server can be built without touching `internal/`
(`architecture.md` 1.3).

The setting vocabulary (grain, ore, copper pieces) is allowed here because this is an
example; the core library stays generic.

## Build and run

```sh
go run -C examples/realm . -seed 1 -state /tmp/realm.ckpt
```

The start loads the pack, solves the warm start once (about 0.3 s) and builds the
world (about 3 s). Then, in another terminal:

```sh
curl localhost:8080/v1/world
curl -X POST 'localhost:8080/v1/step?n=10'
```

Flags: `-addr` (default `127.0.0.1:8080`), `-seed`, `-pace` (a duration: step the
world once every `pace`; 0, the default, steps it only on `POST /v1/step`), `-state`
(a checkpoint file: restored at start if it exists, written by
`POST /v1/checkpoint?save=true`). The path is a flag of the process, never a field of
a request. Interrupt the process to stop it.

The same calls from Go, which is `ExampleGame`:

```go
g, _ := NewGame(1)
g.AddPlayer(PlayerSpec{Region: "crownlands", Role: "traveller", Money: 12_345})
g.Advance(10)
p, _ := g.Player() // the tick is 10; the balance is in minor units
```

## Routes

Every response is JSON and carries `tick`, the tick of the data in it. Prices and
amounts are `{"value": <minor units>, "text": "<ladder>"}`, the text in the `RPG`
preset of `denomination/` (1 460 is `1 sp 4 cp`, 12 345 is `1 gp 2 sp 3 cp`).

| Route | Does |
| --- | --- |
| `GET /v1/world` | config hash, warnings, regions with households and agents, the harvest factors and roads in force |
| `GET /v1/market?region=&commodity=` | price, age, volume, unmatched units, best bid and ask |
| `GET /v1/region?region=` | households, money, roles and markets of one region |
| `POST /v1/player` | add the player: `{"region", "role", "money", "goods": {name: quantity}, "mortal"}` |
| `GET /v1/player` | the player's money, holdings, hunger |
| `POST /v1/player/directive` | the orders of the next step: `{"job", "intents": [{"commodity", "side": "bid" or "ask", "quantity", "limit"}], "discards", "hold", "sticky"}` |
| `GET /v1/events?since=` | the player's own orders, rejections and trades from tick `since`, kept in a ring of 1 024 |
| `POST /v1/shock/harvest` | `{"shocks": [{"region", "recipe", "bp"}]}`: 0 is a failed harvest, 10 000 clears the slot |
| `POST /v1/shock/road` | `{"roads": [{"from", "to", "closed", "factor_bp"}]}`; an omitted factor is 10 000 |
| `POST /v1/step?n=` | run `n` ticks (default 1, at most 1 000) |
| `POST /v1/checkpoint[?save=true]` | the encoded checkpoint as the response; with `save=true` also written to `-state` |
| `POST /v1/restore` | replace the world by the checkpoint in the body |

A directive acts in the next step only, one tick of latency (`spec-integration.md`
3.6), unless it is `sticky`. The player's fallback strategy trades by itself when the
directive does not say otherwise; `hold` names the commodities it must leave alone.

## Bad input

The server takes untrusted input and answers it with a 4xx and the name of a
sentinel, never a 5xx:

```json
{"tick": 10, "error": "ErrBadInput", "message": "n \"1001\", want 0 to 1000: bad input"}
```

An unknown region, commodity, recipe or role is 404 `ErrUnknownName`; a negative
amount, a count past the cap, a malformed body or a second JSON value is 400
`ErrBadInput`; a number past a library bound is 400 `ErrOutOfBounds`; a body past
64 KiB (32 MiB for a restore) is 413 `ErrBodyTooLarge`; a restore of bytes that are no
checkpoint is 400 `ErrCheckpointCorrupt` and leaves the world as it was. A refusal
the table in `server.go` does not list is 422 `ErrRefused`. A path or method no
route has is 404 `ErrUnknownName`. `FuzzServerRequests` throws arbitrary methods,
paths, queries and bodies at the router.

The world is single-threaded (`architecture.md` 12.1). The server holds a read-write
lock: reads share it, a step, a restore, a shock and an added player take it alone,
so a `POST /v1/step?n=1000` (about a minute) holds every other request until it is
done. There is no write timeout for that reason. Steps do not queue: a step asked for
while another runs is 429 `ErrBusy`. A POST whose `Origin` header names another host
(a web page in a browser) is 403 `ErrForbidden`. SIGINT and SIGTERM stop a running
step at a tick boundary and drain the requests before the process exits.

## The pack

`pack/cold/` is the modules-off chain and `pack/modules/` the twelve module layers,
both made from `testdata/chains/` by `pack/rename.sh` and the table in
`pack/names.txt`. `Load` solves the cold layers once for the warm start and merges
the module layers over them. `ExampleGame` and the server tests use that world: 5 000
households in five regions and an overseas node, every module on.

## What it shows

- A player in the market through the public API alone: `AddAgent`, `Direct`,
  `Agent`, `Market`, `Region`, an `Observer` that keeps the player's events.
- Shocks and save and resume: `SetYieldShocks`, `SetEdgeStatuses`, `Checkpoint`,
  `RestoreWorld` with the observer registered again (extensions are not saved,
  `spec-integration.md` 8). `TestServer_CheckpointMidRunResumesBitIdentical` shows
  that a run resumed from a checkpoint over HTTP is byte-identical to the run that
  was not interrupted.
- Prices shown in a ladder with `denomination/`.

## Test

```sh
go test -C examples/realm -race ./...
```

Coverage of the example, without `main.go` (flags and wiring only): **91.1%** (86.6% with `main.go`) of
statements (`go test -coverprofile`, then `go tool cover -func` over the profile
without `main.go`). The 90% of `testing.md` 9.1 binds the library; the example's own
floor is 80%. `testdata/transcript.golden` is the recorded session of
`TestServer_Transcript`; it is rewritten only with `-update`, after the cause of a
difference is understood.
