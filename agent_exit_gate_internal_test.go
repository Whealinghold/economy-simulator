package economysim

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// rowFiles lists the non-test Go files of this row: agent.go, the strategy
// files and the pure rules of internal/household.
func rowFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	for _, pat := range []string{"agent.go", "strategy*.go", "internal/household/*.go"} {
		m, err := filepath.Glob(pat)
		if err != nil {
			t.Fatalf("glob %s: %v", pat, err)
		}
		for _, f := range m {
			if !strings.HasSuffix(f, "_test.go") {
				files = append(files, f)
			}
		}
	}
	if len(files) < 8 {
		t.Fatalf("found %d files of the row, want at least 8 (agent.go, 6 strategy files, household)", len(files))
	}
	return files
}

var chainIDRe = regexp.MustCompile(`(?m)^\[(?:commodity|recipe|role)\.([a-z0-9_]+)[\].]`)

// TestRowFiles_NoBranchOnAnIdentifier verifies by parsing agent.go, the
// strategy files and internal/household that none has a string literal equal
// to a commodity, recipe or role identifier of testdata/chains/.
//
// Requirement: FR-AGT-065
//
// Guard: supply chains are data. A literal such as "staple" in the agent or the
// default strategy would be a branch on one setting's vocabulary that no other
// pack could follow, and no Stage 0 guard covers these files.
func TestRowFiles_NoBranchOnAnIdentifier(t *testing.T) {
	t.Parallel()
	ids := map[string]bool{}
	err := filepath.WalkDir("testdata/chains", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".toml") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for _, m := range chainIDRe.FindAllStringSubmatch(string(b), -1) {
			ids[m[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk testdata/chains: %v", err)
	}
	if len(ids) < 20 || !ids["staple"] {
		t.Fatalf("read %d identifiers (staple present: %v), want a real vocabulary", len(ids), ids["staple"])
	}
	for _, name := range rowFiles(t) {
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if s, err := strconv.Unquote(lit.Value); err == nil && ids[s] {
					t.Errorf("%s has the string literal %q, the identifier of a chain, want no branch on an identifier", name, s)
				}
			}
			return true
		})
	}
}

// TestRowFiles_NoLoggerNoObserver verifies that agent.go, the strategy files
// and internal/household import no logger, call no Observer and do not read an
// Enabled flag: the strategy is pure and the world logs for it, and the view
// does not depend on observation.
//
// Requirement: FR-AGT-062, FR-AGT-063
//
// Guard: a log line or an observer read inside the per-agent step would be both
// a per-agent line outside the world's Debug guard and a way for a run with a
// logger to differ from one without.
func TestRowFiles_NoLoggerNoObserver(t *testing.T) {
	t.Parallel()
	for _, name := range rowFiles(t) {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		src := string(b)
		for _, banned := range []string{`"log/slog"`, "slog.", "Enabled(", "Observe(", ".observer", "os.Stdout", "os.Stderr", "fmt.Print"} {
			if strings.Contains(src, banned) {
				t.Errorf("%s contains %q, want no logger, observer or output", name, banned)
			}
		}
	}
}

// TestTrade_CostAndDrawsDoNotDependOnTheWeight verifies that, for one state at
// weights 1, 7 and MaxWeight, the fill and Trade are not weight-dependent in
// work: both allocate nothing with a warmed scratch, and Trade takes the same
// number of words (one per intent considered, a function of the view's gaps and
// surpluses, not of the weight).
//
// Requirement: NFR-AGT-002
//
// Guard: the benchmark reports the trend but never fails; this test fails if
// the cost grew with the weight (an allocation or a draw per unit of weight).
func TestTrade_CostAndDrawsDoNotDependOnTheWeight(t *testing.T) {
	// Not parallel: testing.AllocsPerRun panics in a parallel test.
	var wantWords uint64
	for k, w := range []int64{1, 7, MaxWeight} {
		h := NewViewHarness(t, viewConfig())
		i := h.Spawn(t, "baker", "south", w)
		h.Give(t, i, "flour", 20)
		h.Fund(t, i, 4000)
		v, err := h.Fill(i)
		if err != nil {
			t.Fatalf("Fill at weight %d: %v", w, err)
		}
		if got := testing.AllocsPerRun(20, func() {
			if _, err := h.Fill(i); err != nil {
				t.Fatalf("Fill at weight %d: %v", w, err)
			}
		}); got != 0 {
			t.Errorf("Fill at weight %d allocates %v times, want 0", w, got)
		}
		plan := h.Plan()
		rng := rand.New(rand.NewPCG(1, 2))
		if got := testing.AllocsPerRun(20, func() { DefaultStrategy{}.Trade(v, rng, plan) }); got != 0 {
			t.Errorf("Trade at weight %d allocates %v times, want 0", w, got)
		}
		cnt := &countingSource{src: rand.NewPCG(1, 2)}
		DefaultStrategy{}.Trade(v, rand.New(cnt), plan)
		if k == 0 {
			wantWords = cnt.n
			continue
		}
		if cnt.n != wantWords {
			t.Errorf("Trade at weight %d took %d words, want %d as at weight 1", w, cnt.n, wantWords)
		}
	}
}

type countingSource struct {
	src *rand.PCG
	n   uint64
}

func (c *countingSource) Uint64() uint64 { c.n++; return c.src.Uint64() }
