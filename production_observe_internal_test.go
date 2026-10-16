package economysim

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

// flushed collects the events of a flush.
func flushed(t *refusalTally, tick int64) []RecipeRefused {
	var got []RecipeRefused
	t.flush(tick, func(e RecipeRefused) { got = append(got, e) })
	return got
}

// TestRefusalTally_Rows verifies the RecipeRefused rows of a scripted tick:
// one row per (region, recipe) with a count, in ascending region then recipe,
// each count the sum of the weights refused for its reason, the five reasons
// in their columns, and a tally that is empty after the flush.
//
// Requirement: FR-PRO-009, FR-PRO-052
func TestRefusalTally_Rows(t *testing.T) {
	t.Parallel()
	tl := newRefusalTally(3, 4)
	tl.add(2, 1, ReasonNoInput, 3)
	tl.add(0, 3, ReasonNoTool, 1)
	tl.add(2, 1, ReasonNoInput, 2)
	tl.add(2, 1, ReasonZeroYield, 4)
	tl.add(0, 3, ReasonNotOnRole, 7)
	tl.add(1, 0, ReasonNoSite, 5)
	want := []RecipeRefused{
		{Tick: 9, Region: 0, Recipe: 3, NotOnRole: 7, NoTool: 1},
		{Tick: 9, Region: 1, Recipe: 0, NoSite: 5},
		{Tick: 9, Region: 2, Recipe: 1, ZeroYield: 4, NoInput: 5},
	}
	if got := flushed(tl, 9); !reflect.DeepEqual(got, want) {
		t.Errorf("rows:\n got %+v\nwant %+v", got, want)
	}
	if got := flushed(tl, 10); got != nil {
		t.Errorf("rows after a flush = %+v, want none: a tick with no refusal emits nothing", got)
	}
	tl.add(1, 1, ReasonNoInput, 1)
	if got := flushed(tl, 11); len(got) != 1 || got[0].NoInput != 1 || got[0].NoTool != 0 || got[0].Tick != 11 {
		t.Errorf("rows of the next tick = %+v, want one NoInput 1 and no carry from the flush", got)
	}
}

// TestRefusalTally_Invariants verifies that a reason the slice does not count,
// a pair outside the tables, a count below 1 and a sum past int64 are
// programmer errors that panic with the word invariant.
//
// Requirement: FR-PRO-052
func TestRefusalTally_Invariants(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		do   func(*refusalTally)
	}{
		{"later module reason", func(tl *refusalTally) { tl.add(0, 0, Reason("no_room"), 1) }},
		{"empty reason", func(tl *refusalTally) { tl.add(0, 0, "", 1) }},
		{"region past the table", func(tl *refusalTally) { tl.add(2, 0, ReasonNoInput, 1) }},
		{"negative region", func(tl *refusalTally) { tl.add(-1, 0, ReasonNoInput, 1) }},
		{"recipe past the table", func(tl *refusalTally) { tl.add(0, 2, ReasonNoInput, 1) }},
		{"negative recipe", func(tl *refusalTally) { tl.add(0, -1, ReasonNoInput, 1) }},
		{"weight 0", func(tl *refusalTally) { tl.add(0, 0, ReasonNoInput, 0) }},
		{"weight negative", func(tl *refusalTally) { tl.add(0, 0, ReasonNoInput, -1) }},
		{"sum past int64", func(tl *refusalTally) {
			tl.add(0, 0, ReasonNoInput, math.MaxInt64)
			tl.add(0, 0, ReasonNoInput, 1)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				r := recover()
				s, _ := r.(string)
				if r == nil || !strings.Contains(s, "invariant") {
					t.Errorf("recover() = %v, want a panic that says invariant", r)
				}
			}()
			tt.do(newRefusalTally(2, 2))
		})
	}
}

// TestRefusalTally_Allocs verifies that adding and flushing allocate nothing
// once the tally exists (observability-and-logging.md 8.2).
//
// Requirement: NFR-PRO-001
func TestRefusalTally_Allocs(t *testing.T) {
	tl := newRefusalTally(5, 45)
	var sink RecipeRefused
	emit := func(e RecipeRefused) { sink = e }
	if a := testing.AllocsPerRun(50, func() {
		tl.add(4, 44, ReasonNoInput, 2)
		tl.add(0, 0, ReasonNoSite, 1)
		tl.flush(3, emit)
	}); a != 0 {
		t.Errorf("AllocsPerRun(add, flush) = %v, want 0", a)
	}
	_ = sink
}
