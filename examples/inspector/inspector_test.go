package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// runArgs runs the inspector on the fixture pack with extra arguments and
// returns its exit code and streams.
func runArgs(t *testing.T, extra ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	args := append([]string{"-chains", "testdata/chains", "-pack", "one"}, extra...)
	code = run(args, &out, &errb)
	return code, out.String(), errb.String()
}

// TestRun_TextReportHasEverySnapshot verifies that a run prints a snapshot at
// each multiple of -every and at the last tick, with the role and market
// tables.
//
// Guard: the report is what a reader (or a script) gets from the program; a
// missing final snapshot would hide the state the run ended in.
func TestRun_TextReportHasEverySnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	code, out, errs := runArgs(t, "-seed", "1", "-ticks", "50", "-every", "20", "-win", "20")
	if code != 0 {
		t.Fatalf("run(...) = %d, want 0, stderr %q", code, errs)
	}
	for _, want := range []string{"== tick 0 ", "== tick 20 ", "== tick 40 ", "== tick 50 ", "cook", "staple", "events before tick 20"} {
		if !strings.Contains(out, want) {
			t.Errorf("run(...) output lacks %q", want)
		}
	}
}

// TestRun_SameSeedSameOutput verifies that two runs with one seed print the
// same bytes.
//
// Guard: the simulator is deterministic (architecture.md 7.4); a report that
// iterated a map in random order would differ between runs and make every
// comparison of two runs useless.
func TestRun_SameSeedSameOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	_, a, _ := runArgs(t, "-seed", "3", "-ticks", "60", "-every", "30")
	_, b, _ := runArgs(t, "-seed", "3", "-ticks", "60", "-every", "30")
	if a != b {
		t.Errorf("two runs with seed 3 differ:\n%s\n---\n%s", a, b)
	}
	_, c, _ := runArgs(t, "-seed", "4", "-ticks", "60", "-every", "30")
	if a == c {
		t.Errorf("seeds 3 and 4 gave the same report; the seed has no effect")
	}
}

// TestRun_JSONLinesAreValidAndMoneyIsTracked verifies that -json prints one
// valid JSON object per line, and that the money tracked by role never
// exceeds the snapshot's total: what is left over is the estates' share, so
// it cannot be negative.
//
// Guard: the money by role is derived from events, not read from the
// snapshot. A negative remainder would mean a trade or removal was booked to
// the wrong agent.
func TestRun_JSONLinesAreValidAndMoneyIsTracked(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	code, out, errs := runArgs(t, "-json", "-ticks", "120", "-every", "40", "-win", "40")
	if code != 0 {
		t.Fatalf("run(-json) = %d, stderr %q", code, errs)
	}
	snaps := 0
	for i, line := range strings.Split(strings.TrimSpace(out), "\n") {
		var v struct {
			Kind        string
			Tick        int64
			Money, Pool int64
			Roles       []struct{ Money int64 }
		}
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			t.Fatalf("line %d is not JSON: %v: %q", i, err, line)
		}
		if v.Kind != "snapshot" {
			continue
		}
		snaps++
		var sum int64
		for _, r := range v.Roles {
			sum += r.Money
		}
		if v.Pool < 0 || sum+v.Pool != v.Money {
			t.Errorf("tick %d: role money %d + pool %d != total %d (pool must be >= 0)", v.Tick, sum, v.Pool, v.Money)
		}
	}
	if snaps != 4 { // ticks 0, 40, 80, 120
		t.Errorf("got %d snapshots, want 4", snaps)
	}
}

// TestRun_FreezeKeepsTheHouseholds verifies that -freeze stops role change
// and deaths, so the household count stays at the pack's 600 while the
// unfrozen run loses households.
//
// Guard: the frozen run is the way to tell a market failure from a spawner
// effect; if freeze silently did nothing, that conclusion would be wrong.
func TestRun_FreezeKeepsTheHouseholds(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	_, frozen, _ := runArgs(t, "-freeze", "-ticks", "100", "-every", "100", "-show", "roles")
	if !strings.Contains(frozen, "== tick 100  households 600 ") {
		t.Errorf("frozen run at tick 100 lacks 600 households:\n%s", frozen)
	}
	_, free, _ := runArgs(t, "-ticks", "100", "-every", "100", "-show", "roles")
	if strings.Contains(free, "== tick 100  households 600 ") {
		t.Errorf("unfrozen run still has 600 households at tick 100; the pack no longer loses any, so the freeze test proves nothing")
	}
}

// TestRun_BadInputIsAnErrorNotAPanic verifies the exit codes: 2 for a bad
// flag value, 1 for a pack that cannot be loaded.
//
// Guard: a script reads the exit code to tell a bad call from a failed run.
func TestRun_BadInputIsAnErrorNotAPanic(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want int
	}{
		{"zero every", []string{"-every", "0"}, 2},
		{"unknown flag", []string{"-nope"}, 2},
		{"missing pack", []string{"-pack", "absent"}, 1},
		{"bad inline toml", []string{"-toml", "[world"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if code, _, errs := runArgs(t, tt.args...); code != tt.want {
				t.Errorf("run(%v) = %d, want %d, stderr %q", tt.args, code, tt.want, errs)
			}
		})
	}
}
