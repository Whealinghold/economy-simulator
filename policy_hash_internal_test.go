package economysim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

// TestPolicyHashEncodingPinned verifies the bytes of the policy module under the ids
// of spec-config.md 8.2.2: the module is id 22 of Modules, written even when empty
// (`0016 0000`), between Institutions 21 and Technology 24 whatever the order of
// building; a PolicyConfig has Actors 1; an actor has ID 1 to Rules 9 in declaration
// order, with Label having no id; a lever has ID 1 to MayClose 11; a step has Lever 1 to
// Closed 4 and a reaction Lever 1 to Raise 9; a field at its off value writes nothing.
//
// Requirement: FR-PLC-003
//
// Guard: the ids are part of the hash of every configuration with the module on; an id
// moved, reused or written for a field at its off value would change the hash of a
// stored configuration without a test noticing.
func TestPolicyHashEncodingPinned(t *testing.T) {
	t.Parallel()
	id := func(n uint16) []byte { return binary.BigEndian.AppendUint16(nil, n) }
	u64 := func(n uint64) []byte { return binary.BigEndian.AppendUint64(nil, n) }
	str := func(s string) []byte { return append(u64(uint64(len(s))), s...) }
	end := []byte{0, 0}

	e := &encoder{}
	pairsModules(e, Modules{Institutions: &Institutions{}, Policy: &PolicyConfig{}, Technology: &Technology{}})
	pair := append(id(22), 0, 0)
	at := bytes.Index(e.buf, pair)
	if at < 0 || bytes.Index(e.buf, append(id(21), 0, 0)) > at || at > bytes.Index(e.buf, append(id(24), 0, 0)) {
		t.Errorf("pairsModules(Institutions, Policy, Technology) = % x, want the empty pair % x after id 21 and before id 24", e.buf, pair)
	}

	e = &encoder{}
	pairsPolicy(e, PolicyConfig{Actors: []PolicyActor{{
		ID: "a", Jurisdiction: "j", Policy: "rule", Every: 3, ReadsRegions: true, Cycle: 9, Label: "ignored",
		Levers:   []Lever{{ID: "l", Kind: LeverEdge, Flow: "f", From: "r", To: "s", Mode: "m", Against: "k", Min: 1, Max: 2, MaxStep: 3, MayClose: true}},
		Schedule: []ScheduleStep{{Lever: "l", At: 4, BP: 5, Closed: true}},
		Rules:    []Reaction{{Lever: "l", Gauge: GaugePrice, Commodity: "c", MaxAge: 6, Target: 7, Deadband: 8, Span: 9, Move: 10, Raise: true}},
	}}})
	lever := bytes.Join([][]byte{
		id(1), str("l"), id(2), str("edge"), id(3), str("f"), id(4), str("r"), id(5), str("s"), id(6), str("m"), id(7), str("k"),
		id(8), u64(1), id(9), u64(2), id(10), u64(3), id(11), {1}, end,
	}, nil)
	step := bytes.Join([][]byte{id(1), str("l"), id(2), u64(4), id(3), u64(5), id(4), {1}, end}, nil)
	rule := bytes.Join([][]byte{
		id(1), str("l"), id(2), str("price"), id(3), str("c"), id(4), u64(6), id(5), u64(7), id(6), u64(8),
		id(7), u64(9), id(8), u64(10), id(9), {1}, end,
	}, nil)
	want := bytes.Join([][]byte{
		id(1), u64(1),
		id(1), str("a"), id(2), str("j"), id(3), str("rule"), id(4), u64(3), id(5), {1},
		id(6), u64(1), lever,
		id(7), u64(1), step,
		id(8), u64(9),
		id(9), u64(1), rule,
		end,
	}, nil)
	if !bytes.Equal(e.buf, want) {
		t.Errorf("pairsPolicy = % x\nwant        % x", e.buf, want)
	}

	// A field at its off value writes nothing: an empty actor is its two-byte terminator.
	e = &encoder{}
	pairsPolicy(e, PolicyConfig{Actors: []PolicyActor{{}}})
	if want := bytes.Join([][]byte{id(1), u64(1), end}, nil); !bytes.Equal(e.buf, want) {
		t.Errorf("pairsPolicy(empty actor) = % x, want % x", e.buf, want)
	}
}

// TestPolicyCanonical_Sorted verifies the canonical order of spec-policy.md 3.1.2 and
// FR-PLC-003: the actors by ID, the levers of an actor by ID, the steps by (Lever, At)
// and the reactions by Lever, and that two spellings of one module hash alike.
//
// Requirement: FR-PLC-003
func TestPolicyCanonical_Sorted(t *testing.T) {
	t.Parallel()
	m := &PolicyConfig{Actors: []PolicyActor{
		{ID: "b", Levers: []Lever{{ID: "y"}, {ID: "x"}}, Schedule: []ScheduleStep{{Lever: "y", At: 5}, {Lever: "x", At: 9}, {Lever: "y", At: 1}}, Rules: []Reaction{{Lever: "y"}, {Lever: "x"}}},
		{ID: "a"},
	}}
	got := m.canonical()
	if got.Actors[0].ID != "a" || got.Actors[1].ID != "b" {
		t.Errorf("canonical actors = %q, %q, want a, b", got.Actors[0].ID, got.Actors[1].ID)
	}
	b := got.Actors[1]
	if b.Levers[0].ID != "x" || b.Rules[0].Lever != "x" {
		t.Errorf("canonical levers and rules of b = %v, %v, want x first", b.Levers, b.Rules)
	}
	wantSteps := []ScheduleStep{{Lever: "x", At: 9}, {Lever: "y", At: 1}, {Lever: "y", At: 5}}
	if !reflect.DeepEqual(b.Schedule, wantSteps) {
		t.Errorf("canonical schedule = %v, want %v", b.Schedule, wantSteps)
	}
	if m.Actors[0].ID != "b" || m.Actors[0].Levers[0].ID != "y" {
		t.Error("canonical changed the module it was called on")
	}
	var nilM *PolicyConfig
	if nilM.canonical() != nil {
		t.Error("canonical of a nil module is not nil")
	}
	h1, h2 := &encoder{}, &encoder{}
	pairsPolicy(h1, *m.canonical())
	pairsPolicy(h2, *(&PolicyConfig{Actors: []PolicyActor{m.Actors[1], m.Actors[0]}}).canonical())
	if !bytes.Equal(h1.buf, h2.buf) {
		t.Error("two spellings of one module encode differently")
	}
}
