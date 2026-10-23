# inspector

A command-line tool that runs the economy simulator and shows what happened:
how many households each role has, who is going hungry, where the money is,
what the markets are doing, and which events (deaths, role changes, shortfalls)
drove it. It has two modes:

- **report mode** (the default) prints text or JSON lines, so you can read it,
  `grep` it, diff two runs, or feed it to a script;
- **TUI mode** (`-tui`) is an interactive terminal UI over the same engine.

It is an example of the public API: its own Go module (`go.mod`), importing the
library as any consumer would. The core library stays standard-library only.

## Contents

1. [Build and run](#build-and-run)
2. [Reading the output](#reading-the-output)
3. [Choosing what to run](#choosing-what-to-run)
4. [Recipes](#recipes)
5. [Flag reference](#flag-reference)
6. [Event counters](#event-counters)
7. [Terminal UI](#terminal-ui)
8. [Troubleshooting](#troubleshooting)
9. [Tests](#tests)

## Build and run

From the repository root, build once and run the binary from the root (so the
default `-chains testdata/chains` finds the packs):

```sh
go build -C examples/inspector -o /tmp/inspector .
/tmp/inspector -pack one -seed 1 -ticks 300 -every 100
```

Or run it in place from the example's directory, pointing at the packs:

```sh
cd examples/inspector
go run . -chains ../../testdata/chains -pack one -ticks 300 -every 100
```

A run is deterministic: the same pack, seed and flags give the same bytes.
A 600-tick run of the `one` pack takes about a second.

## Reading the output

Each snapshot starts with a header, then a role table and a market table.

```
== tick 100  households 557  unfed 60  money 24638321 (estates and tracking error 506993)
  role       region     hh  unfed  ratioBP expectBP viable     wage        money   money/hh
  cook       north     150      1    14094    12904   true     1641     16107712     107384
  ...
  commodity  region   price  volume     stock   estate  unmetBid  unmetAsk bestBid bestAsk  anchor
  staple     north       34    1090     20850       25         2     17525      16      24      35
```

**Header.** The tick is the boundary the snapshot is taken at (tick 100 means 100
steps have run). `households` is the sum of agent weights. `unfed` is the
households that lacked at least one upkeep unit in the last step. `money` is the
total of the households and the estates. `estates and tracking error` is that
total less what the live agents hold, as the tool derives it from events: it is
the money in the estates (what dead households left behind), so it is never
negative, and it is exactly 0 when nobody has died (for example with `-freeze`).
A negative value would mean the tool booked a trade to the wrong agent.

**Role table** (one row per region and role).

| Column | Meaning |
|--------|---------|
| `hh`, `unfed` | households in the role, and how many of them were unfed |
| `ratioBP` | mean welfare ratio in basis points: income against upkeep cost over the role's window (10 000 = it covers its upkeep). 0 when none exists yet |
| `expectBP` | the spawner's expected ratio for the role: what it uses to rank roles for entry and role change |
| `viable` | the spawner's gate: whether an entrant may enter the role |
| `wage` | observed wage of the role, money per working household-tick |
| `money`, `money/hh` | money held by the role's live agents, and per household. **Derived from events**; the snapshot itself has only the total |

**Market table** (one row per region and commodity).

| Column | Meaning |
|--------|---------|
| `price` | the published price: the volume-weighted mean of the tick's trades. It can sit far from the real margin (see `bestBid`) |
| `volume` | units traded in the last clearing |
| `stock`, `estate` | units held by agents and by the estates |
| `unmetBid`, `unmetAsk` | units still unfilled after the last clearing |
| `bestBid`, `bestAsk` | the best *unmatched* bid and ask. Close to each other means the market sits at a thin edge; 0 means that side is empty |
| `anchor` | the cost anchor: the price floor logic's unit cost for the commodity; **0 means unknown** |

**Event line** (printed every `-win` ticks): `events before tick N: key=value ...`
are the counters of the window that just ended. See [Event counters](#event-counters).

**Trace line** (with `-trace staple`): the last clearing of that commodity at each
snapshot, with the best bid and ask and the anchor.

## Choosing what to run

The tool builds the configuration by layering TOML files, later layers winning,
exactly as the library's `config.Parse` does.

| You give | Layers loaded, in order |
|----------|-------------------------|
| `-pack one` (default) | `base.toml`, `one.toml`, `warm-one.toml` |
| `-pack five` | `base.toml`, `lines.toml`, `five.toml`, `warm-five.toml` |
| `-pack one -cold` | `base.toml`, `one.toml` (no warm start) |
| `-layers a.toml,b.toml` | exactly those files, ignoring `-pack` |

Then, on top of whichever you chose: every `-layer FILE`, every `-toml 'DOC'`,
and finally `-freeze`. A file name with no `/` is looked up in `-chains`; a path
is used as given.

The packs are in `testdata/chains/` (`one`, `two`, `three`, `five`, `famine`,
`one-demand`). `-seed` picks the random seed (default 1).

## Recipes

These are the questions the tool was built to answer.

**Does the economy survive?** Households over time, unfed alongside:

```sh
inspector -pack one -ticks 600 -every 100 -show roles | grep '== tick'
```

**Is it the markets or the spawner?** Freeze the households (no role change, no
deaths) and see whether they are fed:

```sh
inspector -pack one -freeze -ticks 300 -every 100 -show roles | grep '== tick'
```

If `unfed` stays high with the population frozen, the markets and production fail
on their own; the spawner is not the cause.

**Where is the money?** Read `money` and `money/hh` per role:

```sh
inspector -pack one -freeze -ticks 300 -every 100 -show roles
```

**What is killing each role?** Show only the deaths, moves and shortfalls:

```sh
inspector -pack one -ticks 300 -win 50 -every 1000 -events dead,move,short
```

**Why is a good not trading?** Look at the best bid and ask and the anchor:

```sh
inspector -pack one -ticks 200 -every 50 -show markets -trace staple
```

**Try a parameter change without editing a file.** `-toml` adds a layer (use `\n`
for a newline):

```sh
inspector -pack one -ticks 300 -every 100 -toml '[world]\nentry_share = 0'
```

Or keep the variant in a file and add it with `-layer variant.toml`, or copy the
pack directory and point `-chains` at the copy.

**Compare seeds.** Run several and compare the last line:

```sh
for s in 1 2 3; do
  echo "seed $s: $(inspector -seed $s -ticks 600 -every 600 -show roles | grep '== tick 600')"
done
```

**Machine-readable output.** `-json` prints one JSON object per line, of kind
`snapshot` or `events`. With `jq`:

```sh
inspector -pack one -ticks 600 -every 50 -json \
  | jq -c 'select(.Kind=="snapshot") | [.Tick, .Households, .Unfed]'
```

The `snapshot` object has `Tick`, `Households`, `Unfed`, `Money`, `Pool` (the
estates' share), `Roles` and `Markets`. The `events` object has `Until` and a
`Counts` map.

## Flag reference

| Flag | Default | Meaning |
|------|---------|---------|
| `-chains DIR` | `testdata/chains` | directory of the pack's layers |
| `-pack NAME` | `one` | pack name (see [Choosing what to run](#choosing-what-to-run)) |
| `-cold` | off | leave out the warm-start layer |
| `-layers a,b,c` | | exact layer files, instead of the pack's |
| `-layer FILE` | | extra TOML file on top; repeatable |
| `-toml 'DOC'` | | extra inline TOML on top, `\n` is a newline; repeatable |
| `-freeze` | off | no role change and no deaths (`entry_share = 0`, `exit_share = 0`) |
| `-seed N` | `1` | random seed |
| `-ticks N` | `600` | ticks to run |
| `-every N` | `100` | print a snapshot every N ticks (and at the last tick) |
| `-win N` | `100` | length of the event window in ticks |
| `-show LIST` | `roles,markets,money,events` | any of `roles`, `markets`, `money`, `events`. `roles` and `money` both print the role table (which includes the money columns) |
| `-events PREFIXES` | all | only counters whose key starts with one of these, e.g. `dead,move,short` |
| `-trace LIST` | | commodities whose last clearing is printed at each snapshot |
| `-json` | off | JSON lines instead of text |
| `-tui` | off | the interactive UI instead of a report |

Exit codes: `0` success, `1` the run or the configuration failed (a missing
file, an invalid pack, a step error), `2` a bad flag value.

## Event counters

Counters are summed over the window and printed sorted. Filter them with
`-events`; each key is a prefix, then dot-separated parts.

| Key | Meaning |
|-----|---------|
| `dead.ROLE.CAUSE` | households removed; cause is `insolvent`, `starved`, `consumer` or `population` |
| `cut.ROLE.CAUSE` | households removed from an agent without removing it (a reduction) |
| `move.FROM>TO` | households that changed role |
| `exec.RECIPE` | executions of a recipe that completed |
| `aband.RECIPE.REASON` | executions abandoned (`agent_removed`, `role_changed`, ...) |
| `refuse.RECIPE.KIND` | households refused a recipe: `no_tool`, `no_input`, `no_site`, `zero_yield` |
| `short.ROLE.COMMODITY` | units of upkeep the role's households were short |
| `entry.*` | the spawner: `entered`, `removed`, `unfunded`, `no_viable_role`, `stranded` |
| `vol.COMMODITY`, `val.COMMODITY` | units traded and money value traded |
| `rev.ROLE`, `spend.ROLE` | money the role's agents received from sales and paid for purchases |

## Terminal UI

`-tui` opens three views of the same session, with the same flags choosing what
to load:

```sh
inspector -pack one -tui
```

- **Overview:** roles with households, unfed, money and a bar for each role's
  share of all money; markets with a price-history sparkline.
- **Events:** the counters of the last window, with filter presets.
- **Market:** the selected market in detail: best bid and ask, unmet bid and ask,
  stock, anchor and the range of its price history.

| Key | Action |
|-----|--------|
| space | run or pause; `+` / `-` change the ticks stepped per frame (1, 2, 5, 10, 25) |
| `s` / `S` | step 1 or 10 ticks |
| `b` / `B` | go back 10 or 100 ticks. The world is built again from the seed and stepped forward, so the state is exact |
| `[` / `]` | previous or next seed, returning to the same tick |
| `f` | freeze or unfreeze role change and deaths, returning to the same tick |
| `r` | back to tick 0 |
| tab, `1` `2` `3` | change view; `e` cycles the event filter; up and down pick the market |
| `?` or `h` | the help screen: every key and what each view shows (any key closes it) |
| `q` | quit |

The top of the screen shows the three views with their keys (`1 overview  2 events
3 market  ? help`) and highlights the open one.

Stepping happens on the UI's own goroutine because `World` is not safe for
concurrent use. Going back takes as long as running to that tick again, a second
or two at tick 500 and beyond.

## Troubleshooting

- **`open testdata/chains/base.toml: no such file or directory`**: the default
  `-chains` is relative to where you run it. Run from the repository root, or
  pass `-chains ../../testdata/chains` from the example's directory.
- **`unknown key` or a validation error naming a path**: a `-toml` or `-layer`
  document has a key the configuration does not know; the message names the
  path and the rule.
- **Money numbers look wrong in a multi-region pack**: the money by role sums
  across regions; the role table has one row per region and role, so a role's
  money repeats on each of its rows.
- **`estates and tracking error` is large**: it is mostly estate money from
  dead households, not an error. A *negative* value would be a tracking bug.
- **No `anchor` (0)**: the cost anchor is unknown for that commodity right now
  (see the cost-anchor rules in `doc/specs/spec-production.md` 9.3).
- **The TUI shows nothing or garbled output**: it needs a real terminal (it uses
  the alternate screen). For scripts and logs use report mode.

## Tests

```sh
cd examples/inspector
go test -short ./...    # the quick checks
go test -race ./...     # integration tests too, about 45 s with -race
```

The tests read their own copy of the `one` pack in `testdata/chains/` (a test may
not read outside its module), so they do not follow later changes to the
repository's packs.
