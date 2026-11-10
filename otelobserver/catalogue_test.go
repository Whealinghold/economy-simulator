package otelobserver

import (
	"slices"
	"strings"
	"testing"
)

var kindName = map[kind]string{kCounter: "Counter", kUpDown: "UpDownCounter", kGauge: "Gauge", kHistogram: "Histogram"}

// built is the catalogue by name.
func built() map[string]spec {
	m := map[string]spec{}
	for _, s := range catalogue {
		m[s.name] = s
	}
	return m
}

// TestCatalogue_NamesUnitsKindsAttributes verifies that every row of the
// expected table of spec-observability.md 6.2 that is not on notBuilt is in the
// catalogue with the same kind, unit and attribute keys, in the same order, and
// that no row is built twice. Without it a dashboard would read a unit or an
// attribute the spec does not promise.
//
// Requirement: FR-OBS-050
func TestCatalogue_NamesUnitsKindsAttributes(t *testing.T) {
	got := built()
	if len(got) != len(catalogue) {
		t.Fatalf("catalogue has %d rows and %d names, want equal", len(catalogue), len(got))
	}
	for _, w := range want {
		if notBuilt[w.name] {
			continue
		}
		s, ok := got[w.name]
		if !ok {
			t.Errorf("catalogue lacks %s, want it (or an entry on notBuilt)", w.name)
			continue
		}
		if kindName[s.kind] != w.kind || s.unit != w.unit || !slices.Equal(s.attrs, w.attrs) {
			t.Errorf("%s = %s %q %v, want %s %q %v", w.name, kindName[s.kind], s.unit, s.attrs, w.kind, w.unit, w.attrs)
		}
	}
}

// TestCatalogue_NoTotalNoUnitInName verifies that every name is in the
// economysim namespace, none ends in _total, none ends in a unit, every attribute
// key is in the namespace and not in the reserved otel one, and every row has a
// unit and a one-line description.
//
// Requirement: FR-OBS-050
func TestCatalogue_NoTotalNoUnitInName(t *testing.T) {
	units := []string{".seconds", ".bytes", ".total", "_total", "_seconds", "_bp", "_minor_unit"}
	for _, s := range catalogue {
		if !strings.HasPrefix(s.name, "economysim.") {
			t.Errorf("name %q is outside the economysim namespace", s.name)
		}
		for _, u := range units {
			if strings.HasSuffix(s.name, u) {
				t.Errorf("name %q ends in %q, want no unit and no _total (rules 5.1.3)", s.name, u)
			}
		}
		if s.unit == "" || s.desc == "" || strings.Contains(s.desc, "\n") {
			t.Errorf("%s has unit %q and description %q, want a unit and a one-line description (rules 5.1.5)", s.name, s.unit, s.desc)
		}
		for _, a := range s.attrs {
			if !strings.HasPrefix(a, "economysim.") {
				t.Errorf("%s has attribute %q, want the economysim namespace", s.name, a)
			}
		}
	}
}

// TestCatalogue_NotBuiltIsExact verifies the staged form of the catalogue test
// (architecture.md 9.7): it fails on a notBuilt entry the catalogue has and on
// an expected row that is neither built nor listed. It prints the count.
//
// Guard: a stale entry would hide a missing instrument, and a typo in the list
// would hide a gap.
func TestCatalogue_NotBuiltIsExact(t *testing.T) {
	got := built()
	expected := map[string]bool{}
	for _, w := range want {
		expected[w.name] = true
		if _, ok := got[w.name]; !ok && !notBuilt[w.name] {
			t.Errorf("%s is neither built nor on notBuilt", w.name)
		}
	}
	for n := range notBuilt {
		if !expected[n] {
			t.Errorf("notBuilt names %s, which is not a row of the expected table", n)
		}
		if _, ok := got[n]; ok {
			t.Errorf("notBuilt names %s, which the catalogue builds: delete the entry", n)
		}
	}
	for n := range got {
		if !expected[n] {
			t.Errorf("catalogue builds %s, which is not a row of the expected table", n)
		}
	}
	t.Logf("expected %d rows, built %d, not built %d", len(want), len(got), len(notBuilt))
}
