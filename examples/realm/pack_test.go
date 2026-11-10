package main

import (
	"bytes"
	"flag"
	"io/fs"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// update rewrites the files of testdata/ that are recorded (testing.md 7.2).
var update = flag.Bool("update", false, "rewrite the recorded files of testdata/")

// readNames returns the rename table of pack/names.txt, old to new.
func readNames(t testing.TB) map[string]string {
	t.Helper()
	b, err := os.ReadFile("pack/names.txt")
	if err != nil {
		t.Fatalf("read pack/names.txt: %v", err)
	}
	m := make(map[string]string)
	for _, l := range strings.Split(string(b), "\n") {
		f := strings.Fields(l)
		if len(f) == 2 && !strings.HasPrefix(l, "#") {
			m[f[0]] = f[1]
		}
	}
	return m
}

// renameTokens applies the table to the whole tokens of a document, the way
// pack/rename.sh does for the layers.
func renameTokens(b []byte, m map[string]string) []byte {
	olds := make([]string, 0, len(m))
	for o := range m {
		olds = append(olds, regexp.QuoteMeta(o))
	}
	slices.SortFunc(olds, func(a, b string) int { return len(b) - len(a) })
	re := regexp.MustCompile(`\b(` + strings.Join(olds, "|") + `)\b`)
	return re.ReplaceAllFunc(b, func(tok []byte) []byte { return []byte(m[string(tok)]) })
}

// readLayers returns the bytes of the cold layers and of the module layers.
func readLayers(t testing.TB) (cold, mods [][]byte) {
	t.Helper()
	fsys, err := sub()
	if err != nil {
		t.Fatalf("sub() = %v, want nil", err)
	}
	names, err := moduleLayers(fsys)
	if err != nil {
		t.Fatalf("moduleLayers() = %v, want nil", err)
	}
	read := func(list []string) (out [][]byte) {
		for _, n := range list {
			b, err := fs.ReadFile(fsys, n)
			if err != nil {
				t.Fatalf("read %s: %v", n, err)
			}
			out = append(out, b)
		}
		return out
	}
	return read(coldLayers), read(names)
}

// TestPack_StartupSolveIsWarmFive verifies that the table Load computes at start-up
// is the table of warm-five.toml under the rename: the pack parsed with that document
// (a byte copy in testdata/, renamed through names.txt) has the same configuration
// hash as the pack Load returns.
//
// Requirement: FR-REF-007
// Guard: D684. The example ships no warm document, so a change of the layers or of
// the solver that moved the table would go unseen without this comparison.
func TestPack_StartupSolveIsWarmFive(t *testing.T) {
	t.Parallel()
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	if cfg.WarmStart == nil {
		t.Fatal("Load().WarmStart = nil, want the solved table")
	}
	doc, err := os.ReadFile("testdata/warm-five.toml")
	if err != nil {
		t.Fatalf("read testdata/warm-five.toml: %v", err)
	}
	cold, mods := readLayers(t)
	// MaxDocuments is 16 and the pack has 16 layers: the warm document rides on the calendar
	// layer, which shares no table with it.
	cold[1] = append(cold[1], renameTokens(doc, readNames(t))...)
	docs := append(cold, mods...)
	want, err := config.Parse(docs...)
	if err != nil {
		t.Fatalf("Parse(the pack with the renamed warm-five.toml) = %v, want nil", err)
	}
	if got, w := economysim.HashConfig(cfg), economysim.HashConfig(want); got != w {
		t.Errorf("HashConfig(Load()) = %x, want %x (the pack with warm-five.toml renamed)", got, w)
	}
}

// expectedRegionRe reads one region line of expected-five.toml.
var expectedRegionRe = regexp.MustCompile(`\{ region = "(\w+)", surplus_bp = (\d+), labour_share_bp = (\d+), pivots = (\d+), balanced = \[([^\]]*)\] \}`)

// TestPack_RenameIsIsomorphic verifies that the modules-off layers of the pack are
// the programme of `five` under other labels: the solver gives, for every region
// mapped through names.txt, the surplus, the labour share and the balanced census of expected-five.toml (a byte copy of the root fixture). It also
// checks that no identifier of the table remains in the pack.
//
// Requirement: FR-REF-012, FR-REF-007
// Guard: D682. A rename that merged two identifiers or missed an occurrence changes
// the programme, and the evidence the pack inherits from `five` would be void.
func TestPack_RenameIsIsomorphic(t *testing.T) {
	t.Parallel()
	names := readNames(t)
	cold, err := Cold()
	if err != nil {
		t.Fatalf("Cold() = %v, want nil", err)
	}
	st, err := economysim.SolveSteadyState(t.Context(), cold)
	if err != nil {
		t.Fatalf("SolveSteadyState(cold) = %v, want nil", err)
	}
	b, err := os.ReadFile("testdata/expected-five.toml")
	if err != nil {
		t.Fatalf("read testdata/expected-five.toml: %v", err)
	}
	rows := expectedRegionRe.FindAllStringSubmatch(string(b), -1)
	if len(rows) != 5 || len(st.Programmes) != 5 {
		t.Fatalf("regions in expected-five.toml, programmes solved = %d, %d, want 5, 5", len(rows), len(st.Programmes))
	}
	for _, r := range rows {
		region := names[r[1]]
		t.Run(region, func(t *testing.T) {
			t.Parallel()
			var p *economysim.ProgrammeResult
			for i := range st.Programmes {
				if st.Programmes[i].Regions[0] == region {
					p = &st.Programmes[i]
				}
			}
			if p == nil {
				t.Fatalf("no programme for region %s (was %s)", region, r[1])
			}
			surplus, _ := strconv.ParseInt(r[2], 10, 64)
			labour, _ := strconv.ParseInt(r[3], 10, 64)
			pivots, _ := strconv.ParseInt(r[4], 10, 64)
			if p.SurplusBP != surplus || p.LabourShareBP != labour {
				t.Errorf("%s: surplus, labour share = %d, %d, want %d, %d (as %s in five)", region, p.SurplusBP, p.LabourShareBP, surplus, labour, r[1])
			}
			// The pivots count the simplex steps, which follow the sort order of the
			// identifiers (the columns), and the rename does not preserve it: logged, not asserted.
			t.Logf("%s: %d pivots (%d as %s in five)", region, p.Pivots, pivots, r[1])
			var want, got []int64
			for _, f := range strings.Split(r[5], ",") {
				n, _ := strconv.ParseInt(strings.TrimSpace(f), 10, 64)
				want = append(want, n)
			}
			for _, c := range st.Report.Census {
				if c.Region == region && c.Role != "" {
					got = append(got, c.Balanced)
				}
			}
			slices.Sort(want)
			slices.Sort(got)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s: balanced census = %v, want %v (sorted, as %s in five)", region, got, want, r[1])
			}
		})
	}
	t.Run("no old identifier remains", func(t *testing.T) {
		t.Parallel()
		fsys, _ := sub()
		err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".toml") {
				return err
			}
			body, err := fs.ReadFile(fsys, path)
			if err != nil {
				return err
			}
			for i, line := range strings.Split(string(body), "\n") {
				line = regexp.MustCompile(`^\s*\w+\s*=`).ReplaceAllString(line, "=") // a schema key stays (rename.sh)
				for old := range names {
					if regexp.MustCompile(`\b` + regexp.QuoteMeta(old) + `\b`).MatchString(line) {
						t.Errorf("%s:%d still names %q (now %q): %s", path, i+1, old, names[old], line)
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk the pack: %v", err)
		}
	})
}

// warningLines renders the warnings of a world, one "rule path" line each.
func warningLines(ws []economysim.Warning) string {
	var sb strings.Builder
	for _, w := range ws {
		sb.WriteString(w.Rule + " " + strings.Join(w.Path, "/") + "\n")
	}
	return sb.String()
}

// TestPack_LoadsWithNoWarningsOfRuleError verifies that the pack validates and the
// world builds, and that Warnings() is the recorded list (testdata/expected-warnings.txt):
// the warnings of `five` and of the module layers, which are advice and not errors.
//
// Requirement: FR-REF-001
func TestPack_LoadsWithNoWarningsOfRuleError(t *testing.T) {
	t.Parallel()
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	w, err := economysim.NewWorld(cfg, Options(1, nil, false)...)
	if err != nil {
		t.Fatalf("NewWorld(realm) = %v, want nil", err)
	}
	got := warningLines(w.Warnings())
	t.Logf("%d warnings:\n%s", len(w.Warnings()), got)
	if *update {
		if err := os.WriteFile("testdata/expected-warnings.txt", []byte(got), 0o644); err != nil {
			t.Fatalf("write testdata/expected-warnings.txt: %v", err)
		}
	}
	want, err := os.ReadFile("testdata/expected-warnings.txt")
	if err != nil {
		t.Fatalf("read testdata/expected-warnings.txt: %v", err)
	}
	if !bytes.Equal([]byte(got), want) {
		t.Errorf("Warnings() =\n%s\nwant\n%s", got, want)
	}
}

// TestPack_SolveTimeIsReported logs the time of the start-up solve and asserts
// nothing about it (testing.md 10.2: no wall-clock dependence).
func TestPack_SolveTimeIsReported(t *testing.T) {
	t.Parallel()
	_, took, err := LoadTimed()
	if err != nil {
		t.Fatalf("LoadTimed() = %v, want nil", err)
	}
	t.Logf("start-up solve took %v", took)
}
