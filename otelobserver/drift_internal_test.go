package otelobserver

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// TestCatalogue_CopyMatchesExpectedTable verifies that the expected table the
// catalogue test compares with is the table of the copy of the spec in testdata:
// the same rows in the same order, with the same kind, unit and attribute keys.
// The root test TestCatalogueOtelCopy_MatchesSpec ties the copy to the spec.
//
// Requirement: FR-OBS-050
// Guard: the module cannot read doc/, so without this and the root test the
// expected table could drift from spec-observability.md 6.2 unseen.
func TestCatalogue_CopyMatchesExpectedTable(t *testing.T) {
	text, err := os.ReadFile("testdata/catalogue_copy.md")
	if err != nil {
		t.Fatalf("read copy: %v", err)
	}
	rows := copyRows(t, text)
	if len(rows) != len(want) {
		t.Fatalf("copy has %d rows, expected table has %d", len(rows), len(want))
	}
	for i, r := range rows {
		w := want[i]
		if r.name != w.name || r.kind != w.kind || r.unit != w.unit || !slices.Equal(r.attrs, w.attrs) {
			t.Errorf("row %d of the copy = %+v, want %+v", i, r, w)
		}
	}
}

// rowNames is the instruments a name cell names: one name, or a first name and
// the last segments of the others (`economysim.a.x`, `.y`, `.z`).
func rowNames(cell string) []string {
	var out []string
	for _, m := range regexp.MustCompile("`([^`]+)`").FindAllStringSubmatch(cell, -1) {
		n := m[1]
		if strings.HasPrefix(n, ".") && len(out) > 0 {
			n = out[0][:strings.LastIndex(out[0], ".")] + n
		}
		out = append(out, n)
	}
	return out
}

// copyRows parses the rows of the copy of the spec tables.
func copyRows(t *testing.T, text []byte) []wantRow {
	t.Helper()
	attrRe := regexp.MustCompile("`(economysim\\.[a-z_.]+)`")
	var rows []wantRow
	for _, l := range strings.Split(string(text), "\n") {
		if !strings.HasPrefix(l, "| `economysim.") {
			continue
		}
		c := strings.Split(strings.Trim(strings.TrimSpace(l), "|"), "|")
		attrs := []string{}
		for _, m := range attrRe.FindAllStringSubmatch(c[3], -1) {
			attrs = append(attrs, m[1])
		}
		for _, n := range rowNames(c[0]) {
			rows = append(rows, wantRow{n, strings.TrimSpace(c[1]), strings.Trim(strings.TrimSpace(c[2]), "`"), attrs})
		}
	}
	return rows
}
