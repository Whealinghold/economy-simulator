package indicator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/indicator"
)

// goldenHeader says what the file is and how it is recorded.
const goldenHeader = `# Presets 1 to 21 of spec-indicators.md 9.1 for the second half of a run of pack one,
# warm (seed 1, 400 ticks, two windows of 200; the first is the base of the price
# index). Integers only; a field that is 0, false or empty is not written. It
# records no world state, so testdata/golden/off/ does not move (plan 2-18 D577).
# Inequality is not here: it takes the household rows of AgentState (plan 2-17).
`

// TestGolden_IndicatorPresetsOnOne verifies the rendering of presets 1 to 21 for a
// window of pack one, warm, seed 1, against testdata/golden/indicator/one.txt. A
// change of a preset or of the pack data moves the file; regenerate it with
// -update only after reading testing.md 7.2 and understanding the diff.
//
// Requirement: spec-indicators.md 23
func TestGolden_IndicatorPresetsOnOne(t *testing.T) {
	if testing.Short() {
		t.Skip("run-level test, skipped with -short")
	}
	cfg := loadPack(t, runPacks[0].files...)
	w, rec := newRun(t, cfg, 1, false)
	wins := stepWindows(t, w, rec, 200, 2, nil)
	p0, err := indicator.PricesOfWindow(wins[0].w)
	if err != nil {
		t.Fatalf("PricesOfWindow(base) error = %v, want nil", err)
	}
	base, err := indicator.BaseOf(wins[0].w, p0)
	if err != nil {
		t.Fatalf("BaseOf() error = %v, want nil", err)
	}
	got := append([]byte(goldenHeader), render(t, presetsOf(t, w.Config(), w.Graph(), base, wins[1]))...)
	path := filepath.Join("..", "testdata", "golden", "indicator", "one.txt")
	if *updateFlag {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v, want nil", err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v, want nil", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v; record it with -update after reading testing.md 7.2", path, err)
	}
	if string(want) != string(got) {
		wl, gl := strings.Split(string(want), "\n"), strings.Split(string(got), "\n")
		for i := 0; i < max(len(wl), len(gl)); i++ {
			var a, b string
			if i < len(wl) {
				a = wl[i]
			}
			if i < len(gl) {
				b = gl[i]
			}
			if a != b {
				t.Fatalf("golden %s differs at line %d:\n want %q\n  got %q\n(%d lines wanted, %d got; regenerate with -update only after reading testing.md 7.2)", path, i+1, a, b, len(wl), len(gl))
			}
		}
	}
}
