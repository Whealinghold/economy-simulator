package economysim_test

import (
	"encoding/hex"
	"fmt"
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
// five, warm-five), its edges, and one layer for each version 1 module, each module after
// what it needs (the merchants need transport and information, the migration needs
// information). The order is not the field order of `Modules` (estates and external come
// later here than there). A
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
		t.Errorf("HashConfig(all-on pack) = %s, want %s of %s (allLayers here and allWarm in config/expected_test.go must list the same layers)", got, want, allExpectedPath)
	}
	// The cold set is the one the solver reads; the config package derives it from its own copy.
	hc := economysim.HashConfig(loadValidPack(t, allColdLayers()...))
	if got, want := hex.EncodeToString(hc[:]), strings.Trim(expectedLine(t, "config_hash"), `"`); got != want {
		t.Errorf("HashConfig(all-on pack, cold) = %s, want %s of %s (allLayers here and allCold in config/expected_test.go must list the same layers)", got, want, allExpectedPath)
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
		f := mods.Type().Field(i)
		if f.Type.Kind() != reflect.Pointer {
			continue
		}
		off := mods.Field(i).IsNil()
		switch listed := allPackNotYetOn[f.Name]; {
		case off && !listed:
			t.Errorf("Modules.%s is off in the all-on pack, want a layer in allLayers that turns it on", f.Name)
		case !off && listed:
			t.Errorf("Modules.%s is on in the all-on pack and listed in allPackNotYetOn: delete the entry", f.Name)
		}
	}
}

// allPackNotYetOn lists the built modules whose layer for the all-on pack is not
// written yet, in the staged form of architecture.md 9.7: the change that writes
// the layer deletes the entry, and the test fails on a stale one. The employment
// module's layer comes with its overlay and its module-alone golden (plan 2-28
// Phase 3), which move golden/all with their explanation (testing.md 7.1.1). The
// polity module's layer joins the all-on set once, jointly, at the close of Stage 3
// (the owner decision of 2026-11-13: no Stage 3 module moves golden/all on its own).
var allPackNotYetOn = map[string]bool{"Employment": true, "Polity": true}

// TestAllPack_OffGoldenUnmoved verifies that the warm five alone, every module off,
// still has the configuration hash its expected file recorded, and that the all-on
// pack differs from it. The files of pack one in the off set are pinned by
// TestGolden_OffUnchanged and TestGolden_EventsAndPositionsUnchanged (sha256, and the list
// of names); the three files of the warm five are checked against themselves by
// TestGolden_FiveOff and its siblings, so they move only with -update and the explanation
// of testing.md 7.2.
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

// collectIDs adds to ids every string held by a field named ID or Name anywhere in v: the
// identifiers of the commodities, recipes, roles, regions, seasons, wants, resources and
// modes of a configuration, without building a world to read its names table.
func collectIDs(v reflect.Value, ids map[string]bool) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			collectIDs(v.Elem(), ids)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			collectIDs(v.Index(i), ids)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if f.Type.Kind() == reflect.String && (f.Name == "ID" || f.Name == "Name") && f.IsExported() {
				ids[v.Field(i).String()] = true
				continue
			}
			if f.IsExported() {
				collectIDs(v.Field(i), ids)
			}
		}
	}
}

// TestConfigOnly_AllPackHasNoGoSpecific verifies that no string literal in a non-test Go
// file of the module is an identifier of the all-on pack (a commodity, recipe, role,
// region, season, want, resource or mode): the pack is data. It reads the identifiers
// from the configuration, so it is cheap and runs in every gate.
//
// Requirement: FR-REF-002
//
// Guard: architecture.md 9.1 and 9.4. A branch on an identifier of the pack in
// library code would pass every test that does not look for it.
func TestConfigOnly_AllPackHasNoGoSpecific(t *testing.T) {
	t.Parallel()
	cfg := loadValidPack(t, allLayers...)
	ids := map[string]bool{}
	collectIDs(reflect.ValueOf(cfg), ids)
	// Words that are an identifier of the pack and also a name the library owns: the
	// field of a belief (centre), the key of a transport mode (haul), the log and
	// rendering key (seed) and the seasons and months of the default calendar (defaults.go).
	owned := map[string]bool{"centre": true, "haul": true, "seed": true, "spring": true, "summer": true, "autumn": true, "winter": true}
	for m := 1; m <= 12; m++ {
		owned[fmt.Sprintf("m%02d", m)] = true // the month names of the default calendar
	}
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
