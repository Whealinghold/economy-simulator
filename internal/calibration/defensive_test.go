package calibration

import (
	"errors"
	"math"
	"testing"
)

// TestDefensive_ForgedInput verifies that a hand-built value no builder would
// make is answered with an empty result or an error, never a panic or a wrap.
func TestDefensive_ForgedInput(t *testing.T) {
	forged := Grid{Levels: []int64{1, 2}, Base: 5}
	if got := ScreenLevels(forged); got != nil {
		t.Errorf("ScreenLevels(forged) = %v, want none", got)
	}
	if got := Explore(Grid{Levels: []int64{1}, Base: math.MaxInt}); got != (Span{Lo: 0, Hi: -1}) {
		t.Errorf("Explore(forged) = %v, want the empty span", got)
	}
	h := Header{Stage: "C1", Attempt: 1, Lineage: "5a1c0e4477d3b210", Record: "r"}
	if _, err := Overlay(h, nil, nil); !errors.Is(err, ErrOverlay) {
		t.Errorf("Overlay with a nil class = %v, want ErrOverlay", err)
	}
	rs := []Record{
		{Pack: "p", Lineage: "a", Stage: "C1", Attempt: math.MaxInt, Decision: "revert"},
		{Pack: "p", Lineage: "a", Stage: "C1", Attempt: math.MinInt, Decision: "revert"},
	}
	var found bool
	for _, p := range CheckSequence(rs, Known{}, 1) {
		found = found || (p.Record == 1 && p.Rule == RuleAttemptNumber)
	}
	if !found {
		t.Error("an attempt after MaxInt was accepted: the successor wrapped")
	}
	if _, err := Parse(make([]byte, maxRecordBytes+1)); !errors.Is(err, ErrRecord) {
		t.Errorf("Parse(oversized) = %v, want ErrRecord", err)
	}
}
