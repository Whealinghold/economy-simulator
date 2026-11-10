package economysim_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestPacks_BandFilesLoad verifies that overlay/sink.toml layers over pack one,
// loads, validates, puts the money supply module on with one levy and no grant,
// and hashes; and that every file under bands/ parses with plainTOML, so the
// tables the stylised facts add (plan 2-19 D597) keep the grammar the scenario
// tests read.
//
// Requirement: FR-REF-022, architecture 2.10.5
//
// Guard: R9's provoked run and the band tables of the root tests read these
// files; a table outside the plain grammar fails the scenario tests with no
// message about the cause.
func TestPacks_BandFilesLoad(t *testing.T) {
	t.Parallel()
	cfg := loadValidPack(t, append(slices.Clone(packOne), "overlay/sink.toml")...)
	m := cfg.Modules.MoneySupply
	if m == nil || len(m.Flows) != 1 || m.Flows[0].Kind != "levy" {
		t.Fatalf("pack one + sink: money supply %+v, want one levy flow", m)
	}
	if economysim.HashConfig(cfg) == economysim.HashConfig(loadValidPack(t, packOne...)) {
		t.Errorf("HashConfig(one + sink) = HashConfig(one), want the overlay to change the hash")
	}
	files, err := filepath.Glob(filepath.Join(chainsDir, "bands", "*.toml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("glob bands: %v, %d files, want files", err, len(files))
	}
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			t.Fatal(err)
		}
		if tb := plainTOML(t, f); len(tb) == 0 {
			t.Errorf("%s: no table, want at least one", f)
		}
	}
}
