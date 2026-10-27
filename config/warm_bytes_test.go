package config_test

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// updateFlag rewrites the files this package records (the warm documents and
// expected/): with it, the tests write what they compare (testing.md 7.2: read
// it first, and explain the diff in the PR). It is defined once for the
// config_test binary; a package that does not define it fails under
// `go test -update ./...`, so run it per package.
var updateFlag = flag.Bool("update", false, "rewrite the recorded files (testing.md 7.2: read it first)")

// warmRegion renders the warm-start table of one region in the house format of
// the warm-*.toml documents: sorted keys (the table is in canonical order),
// inline arrays for stock and jobs, and a blank line after the region.
func warmRegion(b *strings.Builder, ws economysim.WarmStart, region string, wage economysim.Money) {
	fmt.Fprintf(b, "[warm_start.region.%s]\nwage = %d\n", region, wage)
	for _, p := range ws.Pools {
		if p.Region == region {
			fmt.Fprintf(b, "pool = %d\n", p.Money)
		}
	}
	fmt.Fprintf(b, "[warm_start.region.%s.price]\n", region)
	for _, p := range ws.Prices {
		if p.Region == region {
			fmt.Fprintf(b, "%s = %d\n", p.Commodity, p.Price)
		}
	}
	for _, g := range ws.Groups {
		if g.Region != region {
			continue
		}
		fmt.Fprintf(b, "[warm_start.region.%s.group.%s]\nmoney = %d\n", region, g.Role, g.Money)
		if len(g.Stock) > 0 {
			items := make([]string, len(g.Stock))
			for i, it := range g.Stock {
				items[i] = fmt.Sprintf("{ commodity = %q, quantity = %d }", it.Commodity, it.Quantity)
			}
			fmt.Fprintf(b, "stock = [ %s ]\n", strings.Join(items, ", "))
		}
		if len(g.Jobs) > 0 {
			jobs := make([]string, len(g.Jobs))
			for i, j := range g.Jobs {
				jobs[i] = fmt.Sprintf("{ recipe = %q, in_progress = %d }", j.Recipe, j.InProgressBP)
			}
			fmt.Fprintf(b, "jobs = [ %s ]\n", strings.Join(jobs, ", "))
		}
	}
	b.WriteString("\n")
}

// renderWarm is the test-only canonical TOML writer of a warm-start table
// (FR-REF-007, plan 1-10 D133): the bytes of the document a table is committed
// as. The regions are those of the wage rows, in the table's order.
func renderWarm(ws economysim.WarmStart) []byte {
	var b strings.Builder
	for _, w := range ws.Wages {
		warmRegion(&b, ws, w.Region, w.Wage)
	}
	return []byte(b.String())
}

// TestRenderWarm_Format verifies the writer on a hand-built table: the layout of
// a region, a pool line, a group with neither stock nor jobs (no key written),
// and a second region separated by a blank line.
//
// Requirement: FR-REF-007
func TestRenderWarm_Format(t *testing.T) {
	t.Parallel()
	ws := economysim.WarmStart{
		Prices: []economysim.SeedPrice{{Region: "a", Commodity: "x", Price: 5}, {Region: "b", Commodity: "y", Price: 7}},
		Wages:  []economysim.SeedWage{{Region: "a", Wage: 10}, {Region: "b", Wage: 20}},
		Pools:  []economysim.WarmPool{{Region: "b", Money: 99}},
		Groups: []economysim.WarmGroup{
			{Region: "a", Role: "r", Money: 3, Stock: []economysim.Item{{Commodity: "x", Quantity: 2}, {Commodity: "y", Quantity: 1}},
				Jobs: []economysim.JobSpec{{Recipe: "j", InProgressBP: 40}}},
			{Region: "b", Role: "s", Money: 4},
		},
	}
	want := `[warm_start.region.a]
wage = 10
[warm_start.region.a.price]
x = 5
[warm_start.region.a.group.r]
money = 3
stock = [ { commodity = "x", quantity = 2 }, { commodity = "y", quantity = 1 } ]
jobs = [ { recipe = "j", in_progress = 40 } ]

[warm_start.region.b]
wage = 20
pool = 99
[warm_start.region.b.price]
y = 7
[warm_start.region.b.group.s]
money = 4

`
	if got := string(renderWarm(ws)); got != want {
		t.Errorf("renderWarm =\n%s\nwant\n%s", got, want)
	}
	// The writer's output loads back to the table.
	back, err := config.Parse(append([]byte("schema = 1\n"), renderWarm(ws)...))
	if err != nil {
		t.Fatalf("Parse(renderWarm) error = %v, want nil", err)
	}
	if back.WarmStart == nil || len(back.WarmStart.Groups) != 2 || back.WarmStart.Pools[0].Money != 99 {
		t.Errorf("Parse(renderWarm).WarmStart = %+v, want the two groups and the pool", back.WarmStart)
	}
}

// warmPacks are the packs whose committed warm document is the solver's table
// for the layered configuration without it. one-demand is solved with the wants as fixed demand.
var warmPacks = []struct {
	name string
	cold []string
	warm string // path of the document relative to chains
}{
	{"one", []string{"base.toml", "one.toml"}, "warm-one.toml"},
	{"two", []string{"base.toml", "two.toml"}, "warm-two.toml"},
	{"three", []string{"base.toml", "three.toml"}, "warm-three.toml"},
	{"five", []string{"base.toml", "lines.toml", "five.toml"}, "warm-five.toml"},
	{"second", []string{"second/second.toml"}, "second/warm-second.toml"},
	{"one-demand", []string{"base.toml", "demand.toml", "one-demand.toml"}, "warm-one-demand.toml"},
	{"famine", []string{"base.toml", "famine.toml", "overlay/estates.toml"}, "warm-famine.toml"},
}

// TestWarmDocuments_Bytes verifies that each committed warm-start document is,
// byte for byte, the canonical rendering of the table SolveSteadyState produces
// for the pack's layers without it (FR-REF-007): the document is the solver's
// output and never hand-written. A difference is a changed pack, default or
// solver; regenerate with -update only after reading testing.md 7.2, and say why
// in the PR. The table is that of the host architecture (the two-architecture run
// is a hand run).
//
// Requirement: FR-REF-007
//
// Guard: a solver or default that moved the seed would leave the committed
// document stale, and the world would start from a state the solver no longer
// calls steady.
func TestWarmDocuments_Bytes(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	for _, tt := range warmPacks {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := config.LoadFS(os.DirFS(chains), tt.cold...)
			if err != nil {
				t.Fatalf("LoadFS(%q) error = %v, want nil", tt.cold, err)
			}
			st, err := economysim.SolveSteadyState(context.Background(), cfg)
			if err != nil {
				t.Fatalf("SolveSteadyState(%s) error = %v, want nil", tt.name, err)
			}
			got := renderWarm(st.Table)
			path := filepath.Join(chains, tt.warm)
			if *updateFlag {
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("%s differs from the rendering of SolveSteadyState(%s)\n%s\n(regenerate with -update only after reading testing.md 7.2)",
					tt.warm, tt.name, firstDifference(string(want), string(got)))
			}
		})
	}
}

// firstDifference names the first line at which want and got differ.
func firstDifference(want, got string) string {
	wl, gl := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := range max(len(wl), len(gl)) {
		var w, g string
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if w != g {
			return fmt.Sprintf("line %d:\n  document: %q\n  rendered: %q", i+1, w, g)
		}
	}
	return "no line differs"
}
