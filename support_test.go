package economysim_test

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// This file is the support layer of the pack-run tests of stage 1-12
// (doc/plans/plan-phase-1-12.md, phase 2): the pack loader, the pack runner,
// the plain-text reader of the bands and scripts, and the golden comparison. It
// is test code only. The recorder and the captureHandler are those of
// observe_helpers_test.go.

// updateFlag is the one -update flag of the package: with it, checkGolden
// writes the file it compares (testing.md 7.2). A _test.go file is outside the
// determinism guard (determinism_guard_test.go, guardedFiles).
var updateFlag = flag.Bool("update", false, "rewrite the golden files (testing.md 7.2: read it first)")

// The layers of the packs.
var (
	packOne     = []string{"base.toml", "one.toml", "warm-one.toml"}
	packOneCold = []string{"base.toml", "one.toml"}
	packSecond  = []string{"second/second.toml", "second/warm-second.toml"}
)

const chainsDir = "testdata/chains"

// loadValidPack loads and validates the layers under testdata/chains.
func loadValidPack(t testing.TB, layers ...string) economysim.Config {
	t.Helper()
	cfg, err := config.LoadFS(os.DirFS(chainsDir), layers...)
	if err != nil {
		t.Fatalf("LoadFS(%q) error = %v, want nil", layers, err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate(%q) error = %v, want nil", layers, err)
	}
	return cfg
}

// loadPackBytes loads the layers, layers the in-memory document extra last and
// validates the result. LoadFS takes one fs.FS, so a MapFS cannot be layered
// over os.DirFS; the files are read here and parsed together with extra.
func loadPackBytes(t testing.TB, extra []byte, layers ...string) economysim.Config {
	t.Helper()
	docs := make([][]byte, 0, len(layers)+1)
	for _, l := range layers {
		b, err := os.ReadFile(filepath.Join(chainsDir, l))
		if err != nil {
			t.Fatalf("read layer %s: %v", l, err)
		}
		docs = append(docs, b)
	}
	docs = append(docs, extra)
	cfg, err := config.Parse(docs...)
	if err != nil {
		t.Fatalf("Parse(%q + extra) error = %v, want nil", layers, err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate(%q + extra) error = %v, want nil", layers, err)
	}
	return cfg
}

// driveMode says how runPack advances the world.
type driveMode struct{ chunks []int64 }

// driveStep is a Step loop: no RunStarted and no RunEnded.
var driveStep = driveMode{}

// driveRun is one Run call per chunk; the chunks add up to Ticks.
func driveRun(chunks ...int64) driveMode { return driveMode{chunks: chunks} }

// snapMode says which snapshots runPack takes between steps.
type snapMode int

const (
	// snapNone takes the final snapshot only.
	snapNone snapMode = iota
	// snapSample takes the snapshots at the ticks of REF 8.5.
	snapSample
	// snapEvery takes a snapshot after every step.
	snapEvery
)

// sampleTicks are the sample ticks of REF 8.5, before T-1.
var sampleTicks = []int64{0, 1, 2, 5, 10, 20, 50, 100, 200, 300}

// runSpec describes one run of a pack.
type runSpec struct {
	Seed      uint64
	Ticks     int64
	Drive     driveMode
	Observe   bool         // attach the recorder
	Audit     bool         // WithAudit
	Logger    slog.Handler // nil: no logger
	Snapshots snapMode     // needs driveStep unless snapNone
	Extra     []economysim.Option
}

// result is what runPack kept. Samples is keyed by the tick t of the step the
// snapshot follows (so Samples[t].Tick = t+1).
type result struct {
	World   *economysim.World
	Events  []economysim.Event
	Samples map[int64]economysim.Snapshot
	Final   economysim.Snapshot
	Ticks   int64
}

// isSample reports whether the snapshot after step t is asked for.
func (s runSpec) isSample(t int64) bool {
	switch s.Snapshots {
	case snapEvery:
		return true
	case snapSample:
		if t == s.Ticks-1 {
			return true
		}
		for _, x := range sampleTicks {
			if x == t {
				return true
			}
		}
	}
	return false
}

// runPack builds a world over cfg and runs it as spec says. A bare run
// (Observe, Audit and Logger all off) is the plain baseline.
func runPack(t testing.TB, cfg economysim.Config, spec runSpec) result {
	t.Helper()
	opts := []economysim.Option{economysim.WithSeed(spec.Seed)}
	rec := &recorder{}
	if spec.Observe {
		opts = append(opts, economysim.WithObserver(rec))
	}
	if spec.Audit {
		opts = append(opts, economysim.WithAudit())
	}
	if spec.Logger != nil {
		opts = append(opts, economysim.WithLogger(slog.New(spec.Logger)))
	}
	opts = append(opts, spec.Extra...)
	w, err := economysim.NewWorld(cfg, opts...)
	if err != nil {
		t.Fatalf("NewWorld(seed %d) error = %v, want nil", spec.Seed, err)
	}
	res := result{World: w, Samples: map[int64]economysim.Snapshot{}, Ticks: spec.Ticks}
	if len(spec.Drive.chunks) == 0 {
		for i := int64(0); i < spec.Ticks; i++ {
			if err := w.Step(); err != nil {
				t.Fatalf("Step at tick %d (seed %d) error = %v, want nil", i, spec.Seed, err)
			}
			if spec.isSample(i) {
				s, err := w.Snapshot()
				if err != nil {
					t.Fatalf("Snapshot after tick %d error = %v, want nil", i, err)
				}
				res.Samples[i] = s
			}
		}
	} else {
		if spec.Snapshots != snapNone {
			t.Fatalf("runPack: snapshots between steps need driveStep")
		}
		var sum int64
		for _, c := range spec.Drive.chunks {
			sum += c
			if n, err := w.Run(context.Background(), c); err != nil || n != c {
				t.Fatalf("Run(%d) = %d, %v, want %d, nil", c, n, err, c)
			}
		}
		if sum != spec.Ticks {
			t.Fatalf("runPack: chunks add up to %d, want Ticks %d", sum, spec.Ticks)
		}
	}
	res.Final, err = w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot error = %v, want nil", err)
	}
	res.Events = rec.events
	return res
}

// tomlTable is one table of a plain-text document, in file order.
type tomlTable struct {
	name  string
	array bool
	kv    map[string]string
}

var (
	tomlTableRe = regexp.MustCompile(`^\[(\[)?([A-Za-z0-9_.-]+)(\])?\]$`)
	tomlKeyRe   = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

// plainTOML reads the grammar of bands/ and scripts/ only: [table], [[array]]
// and key = value with a number, a string or a list of them; no third-party
// code (A4). A line outside the grammar is a t.Fatal.
func plainTOML(t testing.TB, path string) []tomlTable {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	tables, err := parsePlainTOML(string(b))
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return tables
}

// parsePlainTOML is plainTOML on a string. Keys before the first header belong
// to a table named "".
func parsePlainTOML(src string) ([]tomlTable, error) {
	tables := []tomlTable{{kv: map[string]string{}}}
	for i, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
		case strings.HasPrefix(line, "["):
			m := tomlTableRe.FindStringSubmatch(line)
			if m == nil || (m[1] == "[") != (m[3] == "]") {
				return nil, fmt.Errorf("line %d: %q is not a table header", i+1, line)
			}
			tables = append(tables, tomlTable{name: m[2], array: m[1] == "[", kv: map[string]string{}})
		default:
			k, v, ok := strings.Cut(line, "=")
			k, v = strings.TrimSpace(k), strings.TrimSpace(v)
			if !ok || !tomlKeyRe.MatchString(k) {
				return nil, fmt.Errorf("line %d: %q is not key = value", i+1, line)
			}
			if err := checkPlainValue(v); err != nil {
				return nil, fmt.Errorf("line %d: %w", i+1, err)
			}
			cur := &tables[len(tables)-1]
			if _, dup := cur.kv[k]; dup {
				return nil, fmt.Errorf("line %d: key %q twice in [%s]", i+1, k, cur.name)
			}
			cur.kv[k] = v
		}
	}
	if len(tables[0].kv) == 0 {
		tables = tables[1:]
	}
	return tables, nil
}

// checkPlainValue accepts a number, a double-quoted string without an escape, or
// a [list] of those with no nesting.
func checkPlainValue(v string) error {
	if strings.HasPrefix(v, "[") {
		if !strings.HasSuffix(v, "]") {
			return fmt.Errorf("list %q is not closed", v)
		}
		inner := strings.TrimSpace(v[1 : len(v)-1])
		if inner == "" {
			return nil
		}
		for _, item := range splitPlainList(inner) {
			item = strings.TrimSpace(item)
			if strings.HasPrefix(item, "[") {
				return fmt.Errorf("nested list in %q", v)
			}
			if err := checkPlainValue(item); err != nil {
				return err
			}
		}
		return nil
	}
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' && !strings.ContainsAny(v[1:len(v)-1], "\"\\") {
		return nil
	}
	if _, err := strconv.ParseFloat(v, 64); err == nil && !strings.ContainsAny(v, "xXpP_nNiI") {
		return nil
	}
	return fmt.Errorf("value %q is not a number, a string or a list", v)
}

// splitPlainList splits on the commas outside a string.
func splitPlainList(s string) []string {
	var out []string
	in, start := false, 0
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '"':
			in = !in
		case s[i] == ',' && !in:
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// goldenPath is the path of a file of the off set (testing.md 7.1.1).
func goldenPath(name string) string { return filepath.Join("testdata", "golden", "off", name) }

// checkGolden compares got with the golden file name, or writes it with -update.
func checkGolden(t testing.TB, name string, got []byte) {
	t.Helper()
	if err := compareGolden(goldenPath(name), got, *updateFlag); err != nil {
		t.Error(err)
	}
}

// compareGolden is checkGolden on a path and a flag, so it can be tested alone.
func compareGolden(path string, got []byte, update bool) error {
	if update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, got, 0o644)
	}
	want, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("golden %s does not exist; record it with -update after reading testing.md 7.2", path)
	}
	if err != nil {
		return err
	}
	if bytes.Equal(want, got) {
		return nil
	}
	wl, gl := strings.Split(string(want), "\n"), strings.Split(string(got), "\n")
	for i := 0; i < max(len(wl), len(gl)); i++ {
		var w, g string
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if w != g || i >= len(wl) || i >= len(gl) {
			return fmt.Errorf("golden %s differs at line %d:\n want %q\n  got %q\n(%d lines wanted, %d got; regenerate with -update only after reading testing.md 7.2)",
				path, i+1, w, g, len(wl), len(gl))
		}
	}
	return nil
}

// TestPlainTOML_Grammar verifies the plain-text reader on the grammar of bands/
// and scripts/ and on lines outside it.
//
// Requirement: FR-REF-022
func TestPlainTOML_Grammar(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		src     string
		want    []string // "name:array:k=v,..." per table
		wantErr bool
	}{
		{"empty", "", nil, false},
		{"table and keys", "# c\n[a.b]\nx = 1\ny = \"s\"\nz = [1, \"a,b\", 3]\n", []string{"a.b:false:x=1,y=\"s\",z=[1, \"a,b\", 3]"}, false},
		{"array repeats", "[[shock]]\nt = 5\n[[shock]]\nt = 9\n", []string{"shock:true:t=5", "shock:true:t=9"}, false},
		{"top-level keys", "k = -2.5\n", []string{":false:k=-2.5"}, false},
		{"empty list", "k = []\n", []string{":false:k=[]"}, false},
		{"inline table", "k = {a = 1}\n", nil, true},
		{"bare word", "k = yes\n", nil, true},
		{"mismatched header", "[[a]\n", nil, true},
		{"no equals", "just text\n", nil, true},
		{"duplicate key", "[a]\nk = 1\nk = 2\n", nil, true},
		{"nested list", "k = [[1]]\n", nil, true},
		{"unclosed list", "k = [1\n", nil, true},
		{"escape in string", "k = \"a\\n\"\n", nil, true},
		{"hex number", "k = 0x10\n", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parsePlainTOML(tt.src)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parsePlainTOML error = %v, want error = %t", err, tt.wantErr)
			}
			var lines []string
			for _, tb := range got {
				var kv []string
				for _, k := range sortedKeys(tb.kv) {
					kv = append(kv, k+"="+tb.kv[k])
				}
				lines = append(lines, fmt.Sprintf("%s:%t:%s", tb.name, tb.array, strings.Join(kv, ",")))
			}
			if strings.Join(lines, "|") != strings.Join(tt.want, "|") {
				t.Errorf("tables = %q, want %q", lines, tt.want)
			}
		})
	}
}

// sortedKeys returns the keys of m in order.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

// TestPlainTOML_File verifies the file wrapper reads a file in file order.
//
// Requirement: FR-REF-022
func TestPlainTOML_File(t *testing.T) {
	t.Parallel()
	p := filepath.Join(t.TempDir(), "s.toml")
	if err := os.WriteFile(p, []byte("[[shock]]\nt = 1\n[[shock]]\nt = 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := plainTOML(t, p)
	if len(got) != 2 || got[0].kv["t"] != "1" || got[1].kv["t"] != "2" || !got[0].array {
		t.Errorf("plainTOML = %+v, want two [[shock]] entries t=1, t=2 in order", got)
	}
}

// FuzzPlainTOML verifies the plain-text reader never panics and that a document
// it accepts has exactly the tables it declares, in order.
//
// Requirement: FR-REF-022
func FuzzPlainTOML(f *testing.F) {
	for _, p := range []string{"testdata/chains/one.toml", "testdata/chains/base.toml"} {
		if b, err := os.ReadFile(p); err == nil {
			f.Add(string(b))
		}
	}
	f.Add("[[shock]]\nt = 1\n[[shock]]\nt = 2\n")
	f.Add("[a.b]\nk = [1, \"x\"]\n")
	f.Add("[[a]\n")
	f.Fuzz(func(t *testing.T, src string) {
		tables, err := parsePlainTOML(src)
		if err != nil {
			return
		}
		var declared []string
		for _, line := range strings.Split(src, "\n") {
			if line = strings.TrimSpace(line); strings.HasPrefix(line, "[") {
				declared = append(declared, strings.Trim(line, "[]"))
			}
		}
		var got []string
		for _, tb := range tables {
			if tb.name != "" {
				got = append(got, tb.name)
			}
		}
		if strings.Join(got, "|") != strings.Join(declared, "|") {
			t.Errorf("tables = %q, declared %q", got, declared)
		}
	})
}

// TestCheckGolden_Compare verifies the golden comparison: equal, different with
// the line number and both lines, absent, and -update writes.
//
// Requirement: FR-REF-029
func TestCheckGolden_Compare(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "sub", "g.txt")
	if err := compareGolden(p, []byte("a\nb\n"), false); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("absent file error = %v, want does-not-exist", err)
	}
	if err := compareGolden(p, []byte("a\nb\n"), true); err != nil {
		t.Fatalf("update error = %v, want nil", err)
	}
	if b, _ := os.ReadFile(p); string(b) != "a\nb\n" {
		t.Errorf("file after update = %q, want a\\nb\\n", b)
	}
	if err := compareGolden(p, []byte("a\nb\n"), false); err != nil {
		t.Errorf("equal error = %v, want nil", err)
	}
	err := compareGolden(p, []byte("a\nX\n"), false)
	if err == nil || !strings.Contains(err.Error(), "line 2") || !strings.Contains(err.Error(), `want "b"`) ||
		!strings.Contains(err.Error(), `got "X"`) || !strings.Contains(err.Error(), "testing.md 7.2") {
		t.Errorf("different error = %v, want line 2, both lines and the testing.md 7.2 hint", err)
	}
	if err := compareGolden(p, []byte("a\nb\nc\n"), false); err == nil || !strings.Contains(err.Error(), "line 3") {
		t.Errorf("longer error = %v, want a difference at line 3", err)
	}
	if got := goldenPath("x.txt"); got != filepath.Join("testdata", "golden", "off", "x.txt") {
		t.Errorf("goldenPath = %q, want testdata/golden/off/x.txt", got)
	}
}

// TestLoadPack_Packs verifies loadPack and loadPackBytes load and validate the
// packs, and that an extra in-memory layer is applied last.
//
// Requirement: FR-REF-001
func TestLoadPack_Packs(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		layers []string
	}{{"one", packOne}, {"one cold", packOneCold}, {"second", packSecond}} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, tt.layers...)
			if len(cfg.Regions) == 0 {
				t.Errorf("loadValidPack(%q) has no region", tt.layers)
			}
		})
	}
	t.Run("extra layer last", func(t *testing.T) {
		t.Parallel()
		extra, err := os.ReadFile(filepath.Join(chainsDir, "overlay/nofloor.toml"))
		if err != nil {
			t.Fatal(err)
		}
		got := loadPackBytes(t, extra, packOneCold...)
		want := loadValidPack(t, append(append([]string{}, packOneCold...), "overlay/nofloor.toml")...)
		if economysim.HashConfig(got) != economysim.HashConfig(want) {
			t.Errorf("loadPackBytes hash differs from LoadFS with the same layers")
		}
		if economysim.HashConfig(got) == economysim.HashConfig(loadValidPack(t, packOneCold...)) {
			t.Errorf("the extra layer changed nothing")
		}
	})
}

// TestRunPack_Run verifies the runner on pack one for three ticks: events are
// kept only with Observe, samples are taken at the asked ticks, and a Run split
// reaches the same final tick.
//
// Requirement: FR-REF-029
func TestRunPack_Run(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("runs a world, skipped with -short")
	}
	cfg := loadValidPack(t, packOne...)
	bare := runPack(t, cfg, runSpec{Seed: 1, Ticks: 3, Drive: driveStep})
	if len(bare.Events) != 0 || len(bare.Samples) != 0 || bare.Final.Tick != 3 {
		t.Errorf("bare run: %d events, %d samples, final tick %d, want 0, 0, 3", len(bare.Events), len(bare.Samples), bare.Final.Tick)
	}
	obs := runPack(t, cfg, runSpec{Seed: 1, Ticks: 3, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapSample})
	if len(obs.Events) == 0 || len(obs.Samples) != 3 || obs.Samples[2].Tick != 3 {
		t.Errorf("observed run: %d events, %d samples, want events and samples at ticks 0, 1, 2", len(obs.Events), len(obs.Samples))
	}
	run := runPack(t, cfg, runSpec{Seed: 1, Ticks: 3, Drive: driveRun(1, 2), Observe: true})
	if run.Final.Tick != 3 {
		t.Errorf("split run final tick = %d, want 3", run.Final.Tick)
	}
}
