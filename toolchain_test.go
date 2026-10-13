package economysim_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestToolchain_PinnedGoVersion verifies that go.mod declares exactly "go
// 1.25" and has no toolchain line.
//
// Guard: the vectors of the world's random streams (internal/ordering, WLD)
// were computed on Go 1.25, and a change of the toolchain that moves them is a
// behaviour change. Raising the go directive is a breaking change that
// architecture.md 5.4 requires the changelog to call out, and a toolchain line
// could lower or move the toolchain without the directive changing. The pinned
// vectors in internal/ordering are the check that the generator did not move;
// this test is the check that nobody moved the toolchain by accident.
func TestToolchain_PinnedGoVersion(t *testing.T) {
	t.Parallel()
	mod := readFile(t, "go.mod")
	goRe := regexp.MustCompile(`(?m)^go\s+(\S+)\s*$`)
	m := goRe.FindAllStringSubmatch(mod, -1)
	if len(m) != 1 || m[0][1] != "1.25" {
		t.Errorf("go.mod go directive = %v, want exactly one, \"go 1.25\" (architecture.md 5.4)", m)
	}
	if strings.Contains(mod, "\ntoolchain ") || strings.HasPrefix(mod, "toolchain ") {
		t.Errorf("go.mod has a toolchain line: it can move the toolchain the vectors were computed on (architecture.md 5.4)")
	}
}
