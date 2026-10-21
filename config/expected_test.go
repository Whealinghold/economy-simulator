package config_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	toml "github.com/pelletier/go-toml/v2"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// expectedWarning is one warning of an expected/ file: the rule and the path
// joined with "/" (spec-reference-chain.md 8.4).
type expectedWarning struct {
	Rule string `toml:"rule"`
	Path string `toml:"path"`
}

// expectedPack is the content of testdata/chains/expected/<pack>.toml
// (spec-reference-chain.md 8.4). The top-level keys are those of the cold layer
// set, the one SolveSteadyState solves; the warm set adds its hash and warnings.
type expectedPack struct {
	ConfigHash     string            `toml:"config_hash"`
	SurplusBP      int64             `toml:"surplus_bp"`
	LabourShareBP  int64             `toml:"labour_share_bp"`
	Pivots         int64             `toml:"pivots"`
	Unpriced       []string          `toml:"unpriced"`
	UnpricedWages  []string          `toml:"unpriced_wages"`
	Warnings       []expectedWarning `toml:"warnings"`
	WarmConfigHash string            `toml:"warm_config_hash"`
	WarmWarnings   []expectedWarning `toml:"warm_warnings"`
}

// warnings converts the warnings of a configuration to the file's form.
func warningRows(c economysim.Config) []expectedWarning {
	out := []expectedWarning{}
	for _, w := range c.Warnings() {
		out = append(out, expectedWarning{Rule: w.Rule, Path: strings.Join(w.Path, "/")})
	}
	return out
}

// hashHex is the hex SHA-256 of the effective configuration.
func hashHex(c economysim.Config) string {
	h := economysim.HashConfig(c)
	return hex.EncodeToString(h[:])
}

// measureExpected computes what the load of a pack gives: the hashes and warnings
// of the cold and the warm layer set, and the solver report of the cold set.
func measureExpected(t testing.TB, cold, warm []string) expectedPack {
	t.Helper()
	fsys := os.DirFS(chains)
	c, err := config.LoadFS(fsys, cold...)
	if err != nil {
		t.Fatalf("LoadFS(%q) error = %v, want nil", cold, err)
	}
	w, err := config.LoadFS(fsys, warm...)
	if err != nil {
		t.Fatalf("LoadFS(%q) error = %v, want nil", warm, err)
	}
	for _, cfg := range []economysim.Config{c, w} {
		if err := cfg.Validate(); err != nil {
			t.Fatalf("Validate error = %v, want nil", err)
		}
	}
	st, err := economysim.SolveSteadyState(context.Background(), c)
	if err != nil {
		t.Fatalf("SolveSteadyState(%q) error = %v, want nil", cold, err)
	}
	if len(st.Programmes) != 1 {
		t.Fatalf("SolveSteadyState(%q) has %d programmes, want 1 (a single-region pack)", cold, len(st.Programmes))
	}
	p := st.Programmes[0]
	e := expectedPack{
		ConfigHash: hashHex(c), SurplusBP: p.SurplusBP, LabourShareBP: p.LabourShareBP, Pivots: p.Pivots,
		Unpriced: []string{}, UnpricedWages: append([]string{}, st.Report.UnpricedWages...),
		Warnings: warningRows(c), WarmConfigHash: hashHex(w), WarmWarnings: warningRows(w),
	}
	for _, u := range st.Report.UnpricedCommodities {
		e.Unpriced = append(e.Unpriced, u.Region+"/"+u.Commodity)
	}
	return e
}

// renderExpected writes an expectedPack in the layout of spec-reference-chain.md 8.4.
func renderExpected(e expectedPack) []byte {
	list := func(xs []string) string {
		q := make([]string, len(xs))
		for i, x := range xs {
			q[i] = fmt.Sprintf("%q", x)
		}
		return "[" + strings.Join(q, ", ") + "]"
	}
	ws := func(rows []expectedWarning) string {
		q := make([]string, len(rows))
		for i, r := range rows {
			q[i] = fmt.Sprintf("{ rule = %q, path = %q }", r.Rule, r.Path)
		}
		return "[" + strings.Join(q, ", ") + "]"
	}
	return []byte(fmt.Sprintf(`# What the load of the pack must give (spec-reference-chain.md 8.4). Recorded with
# -update by config_test TestExpected_Packs; the top-level keys are those of the
# cold layer set (the set SolveSteadyState solves), the warm_ keys those of the
# set with the warm-start document. Change it by the rule for the golden files
# (testing.md 7.2).
config_hash = %q
surplus_bp = %d
labour_share_bp = %d
pivots = %d
unpriced = %s
unpriced_wages = %s
warnings = %s
warm_config_hash = %q
warm_warnings = %s
`, e.ConfigHash, e.SurplusBP, e.LabourShareBP, e.Pivots, list(e.Unpriced), list(e.UnpricedWages), ws(e.Warnings), e.WarmConfigHash, ws(e.WarmWarnings)))
}

// TestExpected_Packs verifies, for the packs one and second, the hash of the
// effective configuration, the solver's report (surplus, labour share, pivots,
// unpriced) and the warnings against testdata/chains/expected/<pack>.toml, for
// the cold layer set and the warm one. The file is recorded with -update from the
// code under test, which proves nothing alone, so the test also asserts the
// surplus and pivots the reference chain fixes independently of the file
// (spec-reference-chain.md 5 and 7.3), and that the cold set warns
// cold_start_lockstep (FR-REF-005) on more roles than the warm set.
//
// Requirement: FR-REF-005, FR-REF-006, FR-REF-039
//
// Guard: a changed pack, default or solver shows here and not as a silent drift.
func TestExpected_Packs(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	tests := []struct {
		name      string
		cold      []string
		warm      []string
		surplusBP int64
		pivots    int64
	}{
		{"one", []string{"base.toml", "one.toml"}, []string{"base.toml", "one.toml", "warm-one.toml"}, 25_119, 10},
		{"second", []string{"second/second.toml"}, []string{"second/second.toml", "second/warm-second.toml"}, 19_663, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := measureExpected(t, tt.cold, tt.warm)
			if got.SurplusBP != tt.surplusBP || got.Pivots != tt.pivots {
				t.Errorf("%s: surplus_bp = %d and pivots = %d, want %d and %d", tt.name, got.SurplusBP, got.Pivots, tt.surplusBP, tt.pivots)
			}
			// The warm document gives a job row to the roles with a long recipe,
			// so it lifts the lockstep warning of those roles (REF 5: it stays for a
			// role the document gives no job row): the warm set warns on strictly
			// fewer roles, all of them in the cold set.
			cold, warm := rulePaths(got.Warnings, "cold_start_lockstep"), rulePaths(got.WarmWarnings, "cold_start_lockstep")
			if len(cold) == 0 || len(warm) >= len(cold) {
				t.Errorf("%s: cold_start_lockstep on %q (cold) and %q (warm), want it on every long-recipe role cold and on fewer warm", tt.name, cold, warm)
			}
			for _, p := range warm {
				if !slices.Contains(cold, p) {
					t.Errorf("%s: cold_start_lockstep on %q warm but not cold", tt.name, p)
				}
			}
			path := filepath.Join(chains, "expected", tt.name+".toml")
			if *updateFlag {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, renderExpected(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v (record it with -update after reading testing.md 7.2)", err)
			}
			var want expectedPack
			if err := toml.Unmarshal(b, &want); err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			// An empty list reads back as nil or empty; compare them as equal.
			norm := func(e expectedPack) expectedPack {
				if len(e.Unpriced) == 0 {
					e.Unpriced = nil
				}
				if len(e.UnpricedWages) == 0 {
					e.UnpricedWages = nil
				}
				if len(e.Warnings) == 0 {
					e.Warnings = nil
				}
				if len(e.WarmWarnings) == 0 {
					e.WarmWarnings = nil
				}
				return e
			}
			if g, w := norm(got), norm(want); !reflect.DeepEqual(g, w) {
				t.Errorf("%s differs from the measured load\n got  %+v\n want %+v\n(regenerate with -update only after reading testing.md 7.2)", path, g, w)
			}
		})
	}
}

// rulePaths returns the paths of the warnings of the rule.
func rulePaths(rows []expectedWarning, rule string) []string {
	var out []string
	for _, r := range rows {
		if r.Rule == rule {
			out = append(out, r.Path)
		}
	}
	return out
}
