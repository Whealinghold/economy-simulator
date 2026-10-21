package economysim_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// detTicks is the length of the determinism runs (REF 8.2).
const detTicks int64 = 400

// withoutRunBrackets drops RunStarted and RunEnded: a split Run emits one pair
// per call (FR-OBS-012), a Step loop none.
func withoutRunBrackets(res result) result {
	out := res
	out.Events = nil
	for _, e := range res.Events {
		if n := eventType(e); n != "RunStarted" && n != "RunEnded" {
			out.Events = append(out.Events, e)
		}
	}
	return out
}

// firstLineDiff names the first line at which a and b differ, or "".
func firstLineDiff(a, b []byte) string {
	if bytes.Equal(a, b) {
		return ""
	}
	al, bl := strings.Split(string(a), "\n"), strings.Split(string(b), "\n")
	for i := 0; i < max(len(al), len(bl)); i++ {
		var x, y string
		if i < len(al) {
			x = al[i]
		}
		if i < len(bl) {
			y = bl[i]
		}
		if x != y {
			return fmt.Sprintf("line %d: %q vs %q", i+1, x, y)
		}
	}
	return "length"
}

// firstTickDiff names the first tick at which the per-tick snapshots differ.
func firstTickDiff(a, b map[int64]economysim.Snapshot) string {
	for tick := int64(0); tick < int64(max(len(a), len(b))); tick++ {
		if !reflect.DeepEqual(a[tick], b[tick]) {
			return fmt.Sprintf("tick %d", tick)
		}
	}
	return ""
}

// hashingObserver hashes every event, repeating the work a varying number of
// times so the observer's speed is irregular without a sleep (testing.md 10.2).
func hashingObserver() economysim.Observer {
	var n int
	return economysim.ObserverFunc(func(e economysim.Event) {
		n++
		for i := 0; i <= n%5; i++ {
			_ = sha256.Sum256([]byte(fmt.Sprintf("%+v", e)))
		}
	})
}

// TestDeterminism_SameBytes verifies pack one, seeds 1 and 2, 400 ticks: a run
// split into any number of Run calls, with or without a logger, an observer or
// the audit, ends in the snapshot and config hash of the bare Step run, and the
// cases that can produce them match its per-tick snapshots, its events (without
// RunStarted and RunEnded) and its one.txt bytes.
//
// Requirement: FR-REF-025, FR-REF-030, FR-OBS-023, FR-WLD-063, FR-WLD-064, FR-WLD-020, FR-CFG-026, NFR-WLD-004
//
// Guard: observation, logging, the audit and the chunking of Run must not move
// the world.
func TestDeterminism_SameBytes(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOne...)
	report := solveCold(t)
	seven := []int64{1, 2, 3, 50, 100, 121, 123}
	ones := make([]int64, detTicks)
	for i := range ones {
		ones[i] = 1
	}
	type detCase struct {
		name string
		spec func() runSpec // a fresh spec per run (handlers and observers are stateful)
	}
	step := func(s runSpec) runSpec { s.Drive, s.Snapshots = driveStep, snapEvery; return s }
	run := func(observe bool, chunks ...int64) func() runSpec {
		return func() runSpec { return runSpec{Drive: driveRun(chunks...), Observe: observe} }
	}
	cases := []detCase{
		{"(a) second plain run", func() runSpec { return step(runSpec{}) }},
		{"(a) second observed run", func() runSpec { return step(runSpec{Observe: true}) }},
		{"(b) Run(400) bare", run(false, detTicks)},
		{"(b) Run(400) observed", run(true, detTicks)},
		{"(b) Run(1)+Run(399) bare", run(false, 1, detTicks-1)},
		{"(b) Run(1)+Run(399) observed", run(true, 1, detTicks-1)},
		{"(b) seven chunks bare", run(false, seven...)},
		{"(b) seven chunks observed", run(true, seven...)},
		{"(b) 400 Run(1) bare", run(false, ones...)},
		{"(b) 400 Run(1) observed", run(true, ones...)},
		{"(c) logger", func() runSpec { return step(runSpec{Logger: newCaptureHandler()}) }},
		{"(c) nil observer", func() runSpec {
			return step(runSpec{Extra: []economysim.Option{economysim.WithObserver(nil)}})
		}},
		{"(c) recording observer", func() runSpec { return step(runSpec{Observe: true}) }},
		{"(c) logger and Run(1) chunks", func() runSpec {
			s := run(true, ones...)()
			s.Logger = newCaptureHandler()
			return s
		}},
		{"(d) audit on", func() runSpec { return step(runSpec{Audit: true}) }},
		{"(d) audit on, observed", func() runSpec { return step(runSpec{Audit: true, Observe: true}) }},
		{"(e) logger, recorder, hashing observer, audit", func() runSpec {
			return step(runSpec{Logger: newCaptureHandler(), Observe: true, Audit: true,
				Extra: []economysim.Option{economysim.WithObserver(hashingObserver())}})
		}},
		{"(e) the same with Run chunks", func() runSpec {
			s := run(true, seven...)()
			s.Logger, s.Audit = newCaptureHandler(), true
			s.Extra = []economysim.Option{economysim.WithObserver(hashingObserver())}
			return s
		}},
	}
	for _, seed := range []uint64{1, 2} {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			base := runPack(t, cfg, step(runSpec{Seed: seed, Ticks: detTicks}))
			ref := runPack(t, cfg, step(runSpec{Seed: seed, Ticks: detTicks, Observe: true}))
			refEvents := renderEvents(withoutRunBrackets(ref))
			refGolden := oneTxt(cfg, seed, detTicks, ref, report)
			if d := firstTickDiff(base.Samples, ref.Samples); d != "" {
				t.Fatalf("seed %d: observed Step run differs from the bare one at %s (FR-OBS-023)", seed, d)
			}
			for _, tc := range cases {
				spec := tc.spec()
				spec.Seed, spec.Ticks = seed, detTicks
				got := runPack(t, cfg, spec)
				if !reflect.DeepEqual(got.Final, base.Final) {
					t.Errorf("seed %d, case %q: final snapshot differs from the bare run", seed, tc.name)
				}
				if economysim.HashWorld(got.World) != economysim.HashWorld(base.World) {
					t.Errorf("seed %d, case %q: state hash of the world differs from the bare run (testing.md 7.1.2)", seed, tc.name)
				}
				if got.World.ConfigHash() != base.World.ConfigHash() {
					t.Errorf("seed %d, case %q: ConfigHash differs from the bare run", seed, tc.name)
				}
				if len(spec.Drive.chunks) == 0 {
					if d := firstTickDiff(base.Samples, got.Samples); d != "" {
						t.Errorf("seed %d, case %q: snapshot differs from the bare run at %s", seed, tc.name, d)
					}
				}
				if spec.Observe {
					if d := firstLineDiff(refEvents, renderEvents(withoutRunBrackets(got))); d != "" {
						t.Errorf("seed %d, case %q: events differ from the reference at %s", seed, tc.name, d)
					}
				}
				if spec.Observe && len(spec.Drive.chunks) == 0 {
					if d := firstLineDiff(refGolden, oneTxt(cfg, seed, detTicks, got, report)); d != "" {
						t.Errorf("seed %d, case %q: one.txt differs from the reference at %s", seed, tc.name, d)
					}
				}
			}
		})
	}
}

// bodyOf returns a one.txt from its first sample on, past the header (which
// holds the hash, the seed and the solver report).
func bodyOf(b []byte) []byte {
	if i := bytes.Index(b, []byte("sample t=")); i >= 0 {
		return b[i:]
	}
	return b
}

// TestDeterminism_SeedsDiffer verifies the comparison can see a change: seeds 1
// and 2 give different one.txt bodies, and so do the warm and the cold one.
//
// Requirement: FR-REF-030, FR-REF-025
//
// Guard: if every run rendered the same bytes, the equality tests above would
// prove nothing.
func TestDeterminism_SeedsDiffer(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	warm := loadValidPack(t, packOne...)
	cold := loadValidPack(t, packOneCold...)
	report := solveCold(t)
	render := func(cfg economysim.Config, seed uint64) []byte {
		res := runPack(t, cfg, runSpec{Seed: seed, Ticks: detTicks, Drive: driveStep, Observe: true, Snapshots: snapSample})
		return bodyOf(oneTxt(cfg, seed, detTicks, res, report))
	}
	w1, w2, c1 := render(warm, 1), render(warm, 2), render(cold, 1)
	if bytes.Equal(w1, w2) {
		t.Errorf("seeds 1 and 2 give the same one.txt body, want different")
	}
	if bytes.Equal(w1, c1) {
		t.Errorf("warm and cold one (seed 1) give the same one.txt body, want different")
	}
}

// TestDeterminism_PositionLinesObservedEqualBare verifies the per-position hash
// lines of pack one, seed 1, ticks 0 to 29, are the same for a bare replay and for
// one with a recorder, a logger enabled at every level and the audit attached.
//
// Requirement: FR-WLD-064, FR-OBS-023, FR-CFG-026, testing.md 7.1.2
//
// Guard: observation must change no state at any position; a draw or a write
// that only an observed run makes shows here at its position, and a field that
// depends on observation but is not in the skip list shows here too.
func TestDeterminism_PositionLinesObservedEqualBare(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOne...)
	bare := positionLines(t, cfg, goldenSeed, posTicks)
	observed := positionLines(t, cfg, goldenSeed, posTicks,
		economysim.WithObserver(&recorder{}), economysim.WithAudit(), economysim.WithLogger(slog.New(newCaptureHandler())))
	if len(bare) != len(observed) {
		t.Fatalf("bare replay has %d lines, observed %d, want equal", len(bare), len(observed))
	}
	if tick, pos, ok := economysim.FirstDivergence(bare, observed); ok {
		t.Errorf("observed replay differs from the bare one at tick %d position %s", tick, pos)
	}
}
