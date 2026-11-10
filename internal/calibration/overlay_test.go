package calibration

import (
	"errors"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/design"
)

func classOf(key string) byte {
	switch {
	case strings.HasPrefix(key, "K."):
		return 'K'
	case strings.HasPrefix(key, "D."):
		return 'D'
	case strings.HasPrefix(key, "S."):
		return 'S'
	case strings.HasPrefix(key, "P."):
		return 'P'
	}
	return 0
}

func ovHeader() Header {
	return Header{Stage: "C4a", Attempt: 1, Lineage: "5a1c0e4477d3b210", Record: "testdata/calibration/one/c4a-1.txt"}
}

// TestOverlay_KeysAndHeader verifies the overlay of 19.6: the header comment naming
// the stage, the attempt, the first 8 digits of the lineage and the record, then
// the table and the key.
//
// Requirement: FR-CLB-050, FR-CLB-051
func TestOverlay_KeysAndHeader(t *testing.T) {
	t.Parallel()
	got, err := Overlay(ovHeader(), []Change{{"K.cost_floor", "world.cost_floor", 3500}}, classOf)
	want := "# calibration C4a attempt 1 of lineage 5a1c0e44: see testdata/calibration/one/c4a-1.txt\n[world]\ncost_floor = 3500\n"
	if err != nil || string(got) != want {
		t.Errorf("Overlay(19.6) = %q, %v, want %q", got, err, want)
	}
	empty, err := Overlay(ovHeader(), nil, classOf)
	if err != nil || string(empty) != "# calibration C4a attempt 1 of lineage 5a1c0e44: see testdata/calibration/one/c4a-1.txt\n" {
		t.Errorf("Overlay(no change) = %q, %v, want the comment alone", empty, err)
	}
}

// TestOverlay_RefusesSAndP verifies that a key of class S or P, or of no class, is
// refused, and that K and D are accepted.
//
// Requirement: FR-CLB-050
func TestOverlay_RefusesSAndP(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"S.units", "P.label", "X.unknown"} {
		_, err := Overlay(ovHeader(), []Change{{key, "world.x", 1}}, classOf)
		if !errors.Is(err, ErrOverlay) {
			t.Errorf("Overlay(%s) error = %v, want ErrOverlay", key, err)
		}
	}
	for _, key := range []string{"K.a", "D.b"} {
		if _, err := Overlay(ovHeader(), []Change{{key, "world.x", 1}}, classOf); err != nil {
			t.Errorf("Overlay(%s) error = %v, want nil", key, err)
		}
	}
	// One refused key refuses the whole overlay, whatever its position.
	if out, err := Overlay(ovHeader(), []Change{{"K.a", "world.a", 1}, {"S.b", "world.b", 2}}, classOf); err == nil || out != nil {
		t.Errorf("Overlay(K then S) = %q, %v, want nil and an error", out, err)
	}
}

// TestOverlay_ByteForByte verifies that each key line is the second line of the
// document internal/design writes for the same path and value, signs and zero
// included.
//
// Requirement: FR-CLB-050
func TestOverlay_ByteForByte(t *testing.T) {
	t.Parallel()
	for _, c := range []Change{
		{"K.a", "world.cost_floor", 8000}, {"K.a", "world.belief_step", 0}, {"K.a", "modules.migration.move_below", -3},
		{"K.a", "role.worker_a.target_ratio", 12500}, {"K.a", "world.x", -9223372036854775808},
	} {
		doc, err := design.Overlay(c.Path, c.Value)
		if err != nil {
			t.Fatal(err)
		}
		_, wantKey, _ := strings.Cut(strings.TrimSuffix(string(doc), "\n"), "\n")
		got, err := Overlay(ovHeader(), []Change{c}, classOf)
		if err != nil || !strings.HasSuffix(string(got), "\n"+wantKey+"\n") {
			t.Errorf("Overlay(%+v) = %q, %v, want it to end with the key line %q", c, got, err, wantKey)
		}
	}
}

// TestOverlay_GroupsByTable verifies that keys are grouped under one header per
// table, tables and keys are in byte order whatever the order given, and a blank
// line separates the tables.
//
// Requirement: FR-CLB-050
func TestOverlay_GroupsByTable(t *testing.T) {
	t.Parallel()
	in := []Change{
		{"K.b", "world.b_key", 2}, {"K.r", "role.baker.ratio", 5}, {"K.a", "world.a_key", 1}, {"K.m", "modules.migration.gain", 7},
	}
	want := "# calibration C4a attempt 1 of lineage 5a1c0e44: see testdata/calibration/one/c4a-1.txt\n" +
		"[modules.migration]\ngain = 7\n\n[role.baker]\nratio = 5\n\n[world]\na_key = 1\nb_key = 2\n"
	got, err := Overlay(ovHeader(), in, classOf)
	if err != nil || string(got) != want {
		t.Errorf("Overlay(four keys) = %q, %v, want %q", got, err, want)
	}
	rev := []Change{in[3], in[2], in[1], in[0]}
	if again, _ := Overlay(ovHeader(), rev, classOf); string(again) != want {
		t.Errorf("Overlay(reversed) = %q, want the same bytes", again)
	}
}

// TestOverlay_Errors verifies the refusals that are not about a class: a path the
// design refuses, the same path twice, and a header that names no stage, no
// attempt, no lineage or no record.
//
// Requirement: FR-CLB-051
func TestOverlay_Errors(t *testing.T) {
	t.Parallel()
	ok := []Change{{"K.a", "world.a", 1}}
	for name, c := range map[string][]Change{
		"a path that is not valid": {{"K.a", "World.A", 1}},
		"a path of one segment":    {{"K.a", "world", 1}},
		"the same path twice":      {{"K.a", "world.a", 1}, {"K.b", "world.a", 2}},
	} {
		if _, err := Overlay(ovHeader(), c, classOf); !errors.Is(err, ErrOverlay) {
			t.Errorf("Overlay(%s) error = %v, want ErrOverlay", name, err)
		}
	}
	for name, mutate := range map[string]func(*Header){
		"no stage":       func(h *Header) { h.Stage = "C99" },
		"attempt 0":      func(h *Header) { h.Attempt = 0 },
		"short lineage":  func(h *Header) { h.Lineage = "5a1c0e44" },
		"no record":      func(h *Header) { h.Record = "" },
		"record newline": func(h *Header) { h.Record = "a\nb" },
	} {
		h := ovHeader()
		mutate(&h)
		if _, err := Overlay(h, ok, classOf); !errors.Is(err, ErrOverlay) {
			t.Errorf("Overlay(header with %s) error = %v, want ErrOverlay", name, err)
		}
	}
}
