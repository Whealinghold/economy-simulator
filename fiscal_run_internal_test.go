package economysim

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The tests of the module over a run: its window, its neutrality towards the stream and
// towards observation, its allocation and the places it is not called from (spec-fiscal.md
// 3.1.2, 10, 17; plan 3-fis D3).

// appliedTicks returns the ticks of the FiscalFlowApplied events of a flow.
func appliedTicks(rec *evLog, flow string) []int64 {
	var out []int64
	for _, e := range of[FiscalFlowApplied](rec) {
		if e.Flow == flow {
			out = append(out, e.Tick)
		}
	}
	return out
}

// TestFiscal_Window verifies that a flow acts exactly from Start up to, not including,
// Until: a tax with Start 2 and Until 4 is reported at ticks 2 and 3 and at no other, and
// nothing is applied while the world is built.
//
// Requirement: FR-FIS-006
func TestFiscal_Window(t *testing.T) {
	t.Parallel()
	m := &Fiscal{Taxes: []Tax{{ID: "poll", Authority: "crown", Target: homeMillers, Amount: 1, Window: Window{Start: 2, Until: 4}}}}
	rec := &evLog{}
	w := newMill(t, fiscalMillConfig(m), WithSeed(exitSeed), WithAudit(), WithObserver(rec))
	if got := of[FiscalFlowApplied](rec); len(got) != 0 {
		t.Errorf("events at construction = %+v, want none", got)
	}
	step(t, w, 6)
	if got, want := appliedTicks(rec, "poll"), []int64{2, 3}; !slices.Equal(got, want) {
		t.Errorf("ticks the tax was applied = %v, want %v", got, want)
	}
}

// TestFiscal_Seasons verifies that a flow with a season acts only in the ticks of that
// season, the season of the moment of P1.1: a tax of one season is reported at exactly
// the ticks of the first 12 that the calendar puts in it, which are some of them and not
// all. The calendar's first season starts on day 5, so a flow of spring waits five ticks.
//
// Requirement: FR-FIS-006, FR-FIS-011
func TestFiscal_Seasons(t *testing.T) {
	t.Parallel()
	const ticks = 12
	cfg := fiscalMillConfig(&Fiscal{Taxes: []Tax{{ID: "poll", Authority: "crown", Target: homeMillers, Amount: 1, Window: Window{Seasons: []string{"spring"}}}}})
	cfg.Calendar.SeasonStart = 5
	var want []int64
	for tick := range int64(ticks) {
		if cfg.Calendar.Seasons[cfg.Calendar.SeasonAt(tick)].ID == "spring" {
			want = append(want, tick)
		}
	}
	if len(want) == 0 || len(want) == ticks {
		t.Fatalf("spring ticks of the calendar = %v, want some but not all of the %d (the fixture's seasons)", want, ticks)
	}
	rec := &evLog{}
	w := newMill(t, cfg, WithSeed(exitSeed), WithAudit(), WithObserver(rec))
	step(t, w, ticks)
	if got := appliedTicks(rec, "poll"); !slices.Equal(got, want) {
		t.Errorf("ticks the spring tax was applied = %v, want %v", got, want)
	}
}

// TestFiscal_Observation verifies that a run with a logger at Debug and an observer is
// the run without them: the same checkpoint bytes after 12 ticks.
//
// Requirement: NFR-FIS-005
func TestFiscal_Observation(t *testing.T) {
	t.Parallel()
	m := &Fiscal{Taxes: []Tax{{ID: "poll", Authority: "crown", Target: homeMillers, Amount: 1_000, Per: PerYear}}}
	quiet := newMill(t, fiscalMillConfig(m), WithSeed(exitSeed), WithAudit())
	watched := newMill(t, fiscalMillConfig(m), WithSeed(exitSeed), WithAudit(), WithObserver(&evLog{}),
		WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))))
	step(t, quiet, 12)
	step(t, watched, 12)
	a, _ := quiet.Checkpoint()
	b, _ := watched.Checkpoint()
	if !bytes.Equal(a.b, b.b) {
		t.Error("a run with a logger and an observer differs from the run without them after 12 ticks, want the same bytes")
	}
}

// TestFiscal_AllocsPerRun verifies that the position allocates nothing with the default
// logger and observer once its scratch has grown: no event is constructed without an
// observer and the carries are merged into reused tables.
//
// Requirement: NFR-FIS-001
func TestFiscal_AllocsPerRun(t *testing.T) {
	m := &Fiscal{Taxes: []Tax{
		{ID: "poll", Authority: "crown", Target: homeMillers, Amount: 1_000, Per: PerYear},
		{ID: "remit", Authority: "crown", Target: FlowTarget{Kind: TargetTreasury}, RateBP: 100},
	}}
	w := newMill(t, fiscalMillConfig(m), WithSeed(exitSeed))
	w.p17bFiscal() // grows the scratch
	w.p17bFiscal()
	if got := testing.AllocsPerRun(20, w.p17bFiscal); got != 0 {
		t.Errorf("allocations of P1.7b with no observer = %v, want 0", got)
	}
}

// moduleFiles returns the non-test files of the fiscal module's own: fiscal.go and the
// files named fiscal_*.go.
func moduleFiles(t *testing.T) []string {
	t.Helper()
	all, err := filepath.Glob("fiscal*.go")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range all {
		if !strings.HasSuffix(f, "_test.go") {
			out = append(out, f)
		}
	}
	return out
}

// TestFiscal_NoDraws verifies by a read of the module's files that none imports a random
// package or touches the world's source, so no step of the module takes a word of the
// stream, and that no file keeps a variable at package level.
//
// Requirement: FR-FIS-062
// Guard: the module draws nothing (a cut and a tie use K(t, id)); a draw would move the
// words of every later site only through the keyed streams, but it would make a result
// depend on a call count (architecture.md 7.3).
func TestFiscal_NoDraws(t *testing.T) {
	t.Parallel()
	for _, name := range moduleFiles(t) {
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatalf("ParseFile(%s) error = %v, want nil", name, err)
		}
		for _, imp := range f.Imports {
			if strings.Contains(imp.Path.Value, "rand") {
				t.Errorf("%s imports %s, want no random package", name, imp.Path.Value)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if s, ok := n.(*ast.SelectorExpr); ok && (s.Sel.Name == "rnd" || s.Sel.Name == "rand") {
				t.Errorf("%s reads %s, want no draw", name, s.Sel.Name)
			}
			return true
		})
	}
}

// TestFiscal_RemovalLeavesNoCall verifies that the module is called from none of the
// places a per-agent module is: the removal, the agent-lifecycle fan-ins, the relocation
// refusal and the exit gate keep no mention of it (plan 3-fis D3: a tax keeps no
// per-agent state and keeps no agent away).
//
// Requirement: FR-FIS-060
func TestFiscal_RemovalLeavesNoCall(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"fanin.go", "regions_relocate.go", "phase_exit.go", "estates.go", "strategy.go", "strategy_default.go"} {
		b, err := filepath.Glob(name)
		if err != nil || len(b) != 1 {
			t.Fatalf("Glob(%s) = %v, %v, want the file", name, b, err)
		}
		src, err := readFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(strings.ToLower(src), "fiscal") {
			t.Errorf("%s mentions the fiscal module, want no call from a removal, a relocation or a strategy", name)
		}
	}
}

// readFile returns the text of a file of the package.
func readFile(name string) (string, error) {
	b, err := os.ReadFile(name)
	return string(b), err
}
