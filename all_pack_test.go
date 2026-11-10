package economysim_test

import (
	"encoding/hex"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/strategy"
)

// allLayers is the all-on pack of plan 2-24 D683: the warm `five` (base, lines,
// five, warm-five), its edges, and one layer for each version 1 module in the order of the
// `Modules` pointers (spec-config.md 3.1), each module after what it needs (the
// merchants need transport and information, the migration needs information). A
// layer under all/ is a retarget of a module overlay to five; the others are the
// overlays of the module plans as they are. A module added in Stage 3 appends a
// layer here and moves testdata/golden/all/ once (testing.md 7.2).
var allLayers = []string{
	"base.toml", "lines.toml", "five.toml", "warm-five.toml",
	"overlay/trade-five.toml",
	"all/demand.toml",
	"overlay/faucet.toml",
	"overlay/population.toml",
	"overlay/transport-five.toml",
	"overlay/resources-five.toml",
	"overlay/information-five.toml",
	"overlay/external.toml",
	"all/subsistence.toml",
	"overlay/migration.toml",
	"all/merchants.toml",
	"overlay/estates.toml",
}

// allColdLayers is the all-on pack without the warm start, the set the solver
// reads for the expected file.
func allColdLayers() []string {
	return slices.DeleteFunc(slices.Clone(allLayers), func(l string) bool { return l == "warm-five.toml" })
}

// allOptions are the options the all-on world is built with besides the seed: the
// registry of the strategies the pack names (the merchants), and nothing else.
func allOptions() []economysim.Option { return strategy.Options() }

// allExpectedPath is the file of what the load of the all-on pack must give (REF 8.4).
const allExpectedPath = "testdata/chains/expected/all.toml"

// expectedLine returns the value text of a top-level key of an expected/ file.
func expectedLine(t testing.TB, key string) string {
	t.Helper()
	b, err := os.ReadFile(allExpectedPath)
	if err != nil {
		t.Fatalf("read %s: %v", allExpectedPath, err)
	}
	for _, l := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(l, key+" = "); ok {
			return v
		}
	}
	t.Fatalf("%s has no key %s", allExpectedPath, key)
	return ""
}

// warningPairRe reads one { rule = "...", path = "..." } of an expected/ file.
var warningPairRe = regexp.MustCompile(`rule = "([^"]*)", path = "([^"]*)"`)

// TestAllPack_LoadsAndValidates verifies that every layer of the all-on pack loads
// with config.LoadFS, that the effective configuration validates, that its
// warnings are the list recorded in expected/all.toml (a changed pack, default or
// module shows as a diff, not as silent drift), and that its hash is the one the
// config package recorded from its own copy of the layer list, so the two lists
// cannot drift apart (the cold set too).
//
// Requirement: FR-REF-001, FR-REF-039
//
// Guard: the golden, the invariants and the realm pack read allLayers; a layer that
// stopped validating would change them without a message.
func TestAllPack_LoadsAndValidates(t *testing.T) {
	t.Parallel()
	cfg := loadValidPack(t, allLayers...) // fails on a load or validation error
	var got []string
	for _, w := range cfg.Warnings() {
		got = append(got, w.Rule+" "+strings.Join(w.Path, "/"))
	}
	var want []string
	for _, m := range warningPairRe.FindAllStringSubmatch(expectedLine(t, "warm_warnings"), -1) {
		want = append(want, m[1]+" "+m[2])
	}
	if !slices.Equal(got, want) {
		t.Errorf("warnings of the all-on pack = %q, want the %d of %s: %q", got, len(want), allExpectedPath, want)
	}
	h := economysim.HashConfig(cfg)
	if got, want := hex.EncodeToString(h[:]), strings.Trim(expectedLine(t, "warm_config_hash"), `"`); got != want {
		t.Errorf("HashConfig(all-on pack) = %s, want %s of %s", got, want, allExpectedPath)
	}
	// The cold set is the one the solver reads; the config package derives it from its own copy.
	hc := economysim.HashConfig(loadValidPack(t, allColdLayers()...))
	if got, want := hex.EncodeToString(hc[:]), strings.Trim(expectedLine(t, "config_hash"), `"`); got != want {
		t.Errorf("HashConfig(all-on pack, cold) = %s, want %s of %s", got, want, allExpectedPath)
	}
}

// TestAllPack_EveryModuleIsOn verifies that every pointer of Config.Modules is
// non-nil in the effective configuration of the all-on pack, so a layer that
// silently does nothing, or a module added without its layer, fails here.
//
// Requirement: FR-REF-002
//
// Guard: the effective configuration is read, not the text of a file, because a
// table that a later layer overrides can turn a module off again.
func TestAllPack_EveryModuleIsOn(t *testing.T) {
	t.Parallel()
	mods := reflect.ValueOf(loadValidPack(t, allLayers...).Modules)
	for i := 0; i < mods.NumField(); i++ {
		if f := mods.Type().Field(i); f.Type.Kind() == reflect.Pointer && mods.Field(i).IsNil() {
			t.Errorf("Modules.%s is off in the all-on pack, want a layer in allLayers that turns it on", f.Name)
		}
	}
}

// TestAllPack_OffGoldenUnmoved verifies that the warm five alone, every module off,
// still has the configuration hash its expected file recorded, and that the all-on
// pack differs from it. The files of the off set are pinned by TestGolden_OffUnchanged
// and TestGolden_EventsAndPositionsUnchanged (sha256 and the list of names), which own
// those hashes.
//
// Requirement: FR-REF-029
//
// Guard: architecture.md 2.10.5 and N-217. A module layer that moved the five
// alone fails here; the hash of the all-on pack moves only through the module layers.
func TestAllPack_OffGoldenUnmoved(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("testdata/chains/expected/five.toml")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^warm_config_hash = "([0-9a-f]+)"`).FindSubmatch(b)
	if m == nil {
		t.Fatal("expected/five.toml has no warm_config_hash")
	}
	five := economysim.HashConfig(loadValidPack(t, packFiveWarm...))
	all := economysim.HashConfig(loadValidPack(t, allLayers...))
	if got := hex.EncodeToString(five[:]); got != string(m[1]) {
		t.Errorf("HashConfig(warm five) = %s, want %s of expected/five.toml", got, m[1])
	}
	if five == all {
		t.Error("HashConfig(warm five) = HashConfig(all-on pack), want the module layers to move the hash")
	}
}

// TestConfigOnly_AllPackHasNoGoSpecific verifies that the all-on world is built with
// the registry of strategies and the seed alone, and that no string literal in a
// non-test Go file of the module is an identifier of the pack (a commodity, recipe,
// role, region, season, want, resource or mode): the pack is data.
//
// Requirement: FR-REF-002
//
// Guard: architecture.md 9.1 and 9.4. A branch on an identifier of the pack in
// library code would pass every test that does not look for it.
func TestConfigOnly_AllPackHasNoGoSpecific(t *testing.T) {
	t.Parallel()
	skipFiveSized(t) // builds a 5 000-household world only to read its names table
	w, err := economysim.NewWorld(loadValidPack(t, allLayers...), append(allOptions(), economysim.WithSeed(1))...)
	if err != nil {
		t.Fatalf("NewWorld(all-on pack, strategy.Options, WithSeed) error = %v, want nil", err)
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, list := range [][]string{s.Names.Regions, s.Names.Commodities, s.Names.Roles, s.Names.Recipes, s.Names.Seasons, s.Names.Wants, s.Names.Resources, s.Names.Modes} {
		for _, id := range list {
			ids[id] = true
		}
	}
	// Words that are an identifier of the pack and also a name the library owns: the
	// field of a belief (centre), the key of a transport mode (haul), the log and
	// rendering key (seed) and the seasons of the default calendar (defaults.go).
	owned := map[string]bool{"centre": true, "haul": true, "seed": true, "spring": true, "summer": true, "autumn": true, "winter": true}
	var hits []string
	for _, dir := range []string{".", "config", "strategy", "indicator", "experiment", "denomination", "internal"} {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() && path != dir && dir == "." {
				if err == nil {
					return fs.SkipDir
				}
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(f, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					if v, err := strconv.Unquote(lit.Value); err == nil && ids[v] && !owned[v] {
						hits = append(hits, path+": "+lit.Value)
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", dir, err)
		}
	}
	if len(hits) > 0 {
		t.Errorf("string literals equal to an identifier of the all-on pack in non-test Go files: %q, want none (architecture.md 9.1)", hits)
	}
}
