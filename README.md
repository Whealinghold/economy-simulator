# economy-sim

A Go library that simulates a market economy at the level of a realm: several
connected regions, each with its own market, where prices respond to events,
regions specialise, and shortages and famines matter. Supply chains are data:
commodities, recipes and roles come from configuration, so the same engine runs
any setting by changing files, not code.

It is inspired by bazaarBot (agents with price beliefs trading in a double
auction, supply chains emerging from recipes) and is for games and simulations,
not for forecasting a real economy.

- **A library, not an application.** No server, transport or database. Servers
  and tools are separate modules that import it.
- **Deterministic.** The same seed, configuration and calls give the same bytes.
- **Fixed-point.** Money, prices and quantities are integers; there is no
  floating point in anything that affects results.
- **Standard library only** in the core. A TOML parser is allowed in `config/`
  alone.

## Status

The first stage (the "slice": one region, the reference supply chain, clearing,
price beliefs, the default strategy and spawner, the stability checks) is
built. **The reference packs do not yet reach a stable economy:** they decay
toward collapse under the engine's own rules (see
[doc/notes.md](doc/notes.md) N-312 and the stability scenarios in
`testdata/chains/expected/one.scenarios.txt`). Finding and fixing why is the
current work. The delivery plan is [doc/roadmap-v1.md](doc/roadmap-v1.md).

## Use it as a library

The module path is `github.com/Whealinghold/economy-sim` and the package name
is `economysim`, so import it with an explicit name. Go 1.25 or later.

```go
import (
	"context"
	"fmt"
	"log"
	"os"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

func main() {
	// Configuration layers merge in order: later files win.
	cfg, err := config.LoadFS(os.DirFS("testdata/chains"), "base.toml", "one.toml", "warm-one.toml")
	if err != nil {
		log.Fatal(err)
	}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
	if err != nil {
		log.Fatal(err)
	}
	if _, err := w.Run(context.Background(), 300); err != nil {
		log.Fatal(err)
	}
	snap, err := w.Snapshot()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("tick", snap.Tick, "households", snap.Households)
}
```

Observability is optional and a no-op by default: pass `WithLogger(*slog.Logger)`
and `WithObserver(Observer)` to see what the world does. A `World` is not safe for
concurrent use.

## Run the inspector, including the terminal UI

`examples/inspector` is a command-line tool that runs the simulator and shows the
households and money of each role, the markets, and the events behind them. It
has a text and JSON report mode and an interactive terminal UI (TUI). It is its
own Go module.

Build it once from the repository root:

```sh
go build -C examples/inspector -o /tmp/inspector .
```

**Open the terminal UI** (run it in a real terminal, about 100 columns by 40
rows or larger):

```sh
/tmp/inspector -pack one -tui
```

Without building a binary, from the example's directory:

```sh
cd examples/inspector
go run . -chains ../../testdata/chains -pack one -tui
```

Keys in the UI: **space** run or pause (`+` and `-` change the speed), **s** and
**S** step 1 or 10 ticks, **b** and **B** go back 10 or 100 ticks, **tab** or
**1 2 3** change view (overview, events, one market), **f** freeze role change
and deaths, **[** and **]** change the seed, **?** the help screen, **q** quit.
The views and their keys are also shown across the top of the screen.

**Print a report instead** (readable by a person, a script or an agent):

```sh
/tmp/inspector -pack one -seed 1 -ticks 600 -every 100
/tmp/inspector -pack one -freeze -ticks 300 -show roles      # no role change or deaths: markets alone
/tmp/inspector -pack one -ticks 300 -json | jq -c 'select(.Kind=="snapshot")'
```

The packs are in `testdata/chains/`: `one`, `two`, `three`, `five`, `famine` and
`one-demand`. The inspector's flags, the meaning of every column and counter, and
recipes for the common questions are in
[examples/inspector/README.md](examples/inspector/README.md).

## Develop

```sh
gofmt -l .                     # must print nothing
go vet ./...
staticcheck ./...
go test -short -race ./...     # unit tests only, fails fast
go test -race -timeout 60m ./...   # gate R: reduced seeds under -race
go test -covermode=atomic -coverpkg=./... -coverprofile=coverage.out -timeout 120m ./...   # gate F
go mod tidy                    # must produce no diff
```

Read [AGENTS.md](AGENTS.md) before changing anything: it lists the hard
constraints (pure library, standard library only in the core, supply chains as
data, setting-agnostic names, determinism, fixed-point money) and the rules
files every contributor reads first:

- [doc/rules/architecture.md](doc/rules/architecture.md)
- [doc/rules/observability-and-logging.md](doc/rules/observability-and-logging.md)
- [doc/rules/testing.md](doc/rules/testing.md)

Mutation testing (`gomu`) runs only inside a CPU- and memory-capped Docker
container, never directly on the host (testing.md 9.4).

## Layout

```
economy-sim/
├── *.go                 the root package economysim: World, Config, Snapshot, events
├── config/              TOML files -> Config (the only package with a third-party import)
├── internal/            the rules over primitive structs (ledger, clearing, beliefs, ...)
├── testdata/chains/     the reference supply chains (the packs) and their expected results
├── testdata/golden/     golden files
├── examples/inspector/  the CLI and terminal UI (its own module)
└── doc/                 overview, vision, specs, ADRs, rules, roadmap, notes
```

The full target layout is in `AGENTS.md` and `doc/rules/architecture.md` 2.6.

## Documentation

Start with [doc/overview.md](doc/overview.md) (the approach in one page), then
[doc/vision.md](doc/vision.md) (the goals and version 1 scope). Every file under
`doc/` is summarised in [doc/documentation_index.md](doc/documentation_index.md).
Design decisions are in [doc/adr/](doc/adr/README.md) and the specs, one area per
file, in [doc/specs/](doc/specs/README.md).
