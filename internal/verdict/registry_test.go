package verdict_test

import (
	"errors"
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/verdict"
)

// The registry tests read the spec and the band files by relative path
// (testing.md 6.7, 10.4), as the parameter registry test does.
const (
	specPath  = "../../doc/specs/spec-stylised-facts.md"
	bandsDir  = "../../testdata/chains/bands"
	repoRoot  = "../.."
	regimesFn = "regimes.toml"
)

// release is the flag of STY 8.1.1: with it the keys of the version 1
// criteria must have values; without it the test logs the unset ones.
var release = flag.Bool("stylised.release", false, "require a value for every key of STY 8.1.1")

var ws = regexp.MustCompile(`\s+`)

// spec returns the spec text with its line wrapping collapsed to single spaces.
func spec(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read %s: %v", specPath, err)
	}
	return ws.ReplaceAllString(string(b), " ")
}

// section returns the text from the heading that starts with head to the next
// heading of the same or a higher level.
func section(t *testing.T, text, head string) string {
	t.Helper()
	i := strings.Index(text, head)
	if i < 0 {
		t.Fatalf("spec has no heading %q", head)
	}
	rest := text[i+len(head):]
	level := len(head) - len(strings.TrimLeft(head, "#"))
	end := regexp.MustCompile(` #{1,` + strconv.Itoa(level) + `} [0-9]`).FindStringIndex(rest)
	if end == nil {
		return rest
	}
	return rest[:end[0]]
}

// bandFile is one parsed file of bands/.
type bandFile struct {
	Name string
	Keys []verdict.BandKey
}

func loadBandFiles(t *testing.T) []bandFile {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(bandsDir, "*.toml"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("glob %s: %v, %d files, want files", bandsDir, err, len(paths))
	}
	var out []bandFile
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		keys, err := verdict.LoadBands(string(b))
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		out = append(out, bandFile{strings.TrimSuffix(filepath.Base(p), ".toml"), keys})
	}
	return out
}

var (
	factRe = regexp.MustCompile(`\*\*(STY-[A-Z][0-9]{2}) `)
	keyRe  = regexp.MustCompile("`([a-z][a-z0-9]*\\.[a-z0-9_]+)`")
	// factFld are the fields of 3.1 that every entry carries. "Pass and fail"
	// and "Loops" are left out of the exact facts' entries (S04, M01, M02 and
	// others); factOpt are read when present.
	factFld = []string{"Property", "Measure", "Pack", "Seeds", "Band", "Source", "Confidence", "Class"}
	factOpt = []string{"Pass and fail", "Loops"}
)

type fact struct {
	ID     string
	Fields map[string]string
}

// facts splits section 5.1 to 5.8 into entries and their fields (3.1).
func facts(t *testing.T, text string) []fact {
	t.Helper()
	i, j := strings.Index(text, " 5.1 Price level and dynamics "), strings.Index(text, " 5.9 Facts that cannot both hold")
	if i < 0 || j < i {
		t.Fatalf("spec has no section 5.1 to 5.9")
	}
	body := text[i:j]
	idx := factRe.FindAllStringSubmatchIndex(body, -1)
	var out []fact
	for n, m := range idx {
		end := len(body)
		if n+1 < len(idx) {
			end = idx[n+1][0]
		}
		entry := body[m[0]:end]
		f := fact{ID: body[m[2]:m[3]], Fields: map[string]string{}}
		for _, name := range append(slices.Clone(factFld), factOpt...) {
			if k := strings.Index(entry, "*"+name+":*"); k >= 0 {
				f.Fields[name] = entry[k+len(name)+3:]
			}
		}
		out = append(out, f)
	}
	return out
}

// knownKeys is every key the spec names in backticks, plus the thresholds.
func knownKeys(text string) map[string]bool {
	known := map[string]bool{}
	for _, m := range keyRe.FindAllStringSubmatch(text, -1) {
		known[m[1]] = true
	}
	for _, n := range verdict.ThresholdNames() {
		known[n] = true
	}
	return known
}

// TestRegistry_Spec parses the spec and the band files and runs the checks of
// STY 12: fields, keys, shape, seeds, coverage, closed sets; and the pairs of
// 5.9.
//
// Requirement: FR-STY-001, FR-STY-002, FR-STY-003, FR-STY-004, FR-STY-010, FR-STY-098
//
// Guard: a fact without its fields, a band key that belongs to no fact or a
// scenario with no home is found by this test and by nothing else.
func TestRegistry_Spec(t *testing.T) {
	t.Parallel()
	text := spec(t)
	fs := facts(t, text)
	if len(fs) != 39 {
		t.Fatalf("parsed %d facts, want 39 (P 9, W 7, S 6, D 3, M 6, T 3, H 4, G 1)", len(fs))
	}
	seen := map[string]bool{}
	classRe, confRe := regexp.MustCompile(`^\W*[XOBRP]\b`), regexp.MustCompile(`\b[IHMEP]\b`)
	tierRe := regexp.MustCompile(`tier [GNLS]`)
	table36 := section(t, text, "### 3.6 The band keys of REF, mapped")
	for _, f := range fs {
		if seen[f.ID] {
			t.Errorf("fact %s appears twice", f.ID)
		}
		seen[f.ID] = true
		for _, name := range factFld {
			if f.Fields[name] == "" {
				t.Errorf("fact %s lacks the field %q of 3.1", f.ID, name)
			}
		}
		cls, conf := firstSentence(f.Fields["Class"]), firstSentence(f.Fields["Confidence"])
		if !classRe.MatchString(cls) || !tierRe.MatchString(cls) {
			t.Errorf("fact %s: Class %q is not X, O, B, R or P with a tier", f.ID, cls)
		}
		if !confRe.MatchString(conf) && !(strings.HasPrefix(strings.TrimSpace(conf), "none") && regexp.MustCompile(`\bR\b`).MatchString(cls)) {
			t.Errorf("fact %s: Confidence %q is not one of I, H, M, E, P", f.ID, conf)
		}
		if regexp.MustCompile(`\bB\b`).MatchString(cls) && len(keyRe.FindAllString(firstField(f.Fields["Band"]), -1)) == 0 && !strings.Contains(table36, f.ID) {
			t.Errorf("fact %s is class B and has no band key in its entry or in 3.6", f.ID)
		}
	}
	for _, id := range []string{"STY-P01", "STY-S06", "STY-D03", "STY-M06", "STY-H04", "STY-G01"} {
		if !seen[id] {
			t.Errorf("fact %s not found, want it among the %d parsed", id, len(fs))
		}
	}
	t.Run("pairs of 5.9", func(t *testing.T) { exclusivePairs(t, text, loadBandFiles(t)) })
	t.Run("keys of 3.6", func(t *testing.T) { refKeys(t, table36, seen) })
	t.Run("band files", func(t *testing.T) { bandFileKeys(t, text, loadBandFiles(t)) })
	t.Run("coverage of 5.11", func(t *testing.T) { scenarioCoverage(t, text) })
	t.Run("closed sets", func(t *testing.T) { closedSets(t, text) })
}

// firstSentence returns the text up to the first period followed by a space or
// the end: the class or confidence letters come first.
func firstSentence(s string) string {
	if i := strings.Index(s, ". "); i >= 0 {
		return s[:i]
	}
	return s
}

// firstField cuts an entry field at the next field marker.
func firstField(s string) string {
	if i := strings.Index(s, ":* "); i >= 0 {
		return s[:strings.LastIndex(s[:i], "*")]
	}
	return s
}

// exclusivePairs checks 5.9: the eight rows are present and no band file holds
// the keys of both facts of a pair.
func exclusivePairs(t *testing.T, text string, files []bandFile) {
	t.Helper()
	sec := section(t, text, "### 5.9 Facts that cannot both hold in one pack")
	for _, c := range []string{"C1", "C2", "C3", "C4", "C5", "C6", "C7", "C8"} {
		if !strings.Contains(sec, "| "+c+" |") {
			t.Errorf("5.9 has no row %s", c)
		}
	}
	// The facts of a pair that carry keys: M03 and W02/G01 (C1), H02 and W02
	// (C5), P04 and the faucet facts (C7, no key), M03 and M04 (C8, no key on M04).
	pairs := [][2][]string{
		{{"s19.ratio_band_bp"}, {"w02.n_band_bp", "w02.ratio_band_bp"}},
		{{"h02.offset_bp", "h02.money_bp"}, {"w02.n_band_bp", "w02.ratio_band_bp"}},
		{{"s01.offset_bp"}, {"t03.quote_band_bp", "t03.pull_bp"}},
	}
	for _, f := range files {
		have := map[string]bool{}
		for _, k := range f.Keys {
			have[k.Name] = true
		}
		for _, p := range pairs {
			if anyOf(have, p[0]) && anyOf(have, p[1]) {
				t.Errorf("band file %s declares the keys of both facts of the pair %v and %v (FR-STY-010)", f.Name, p[0], p[1])
			}
		}
	}
}

func anyOf(have map[string]bool, keys []string) bool {
	return slices.ContainsFunc(keys, func(k string) bool { return have[k] })
}

// refRowRe reads a row of the table of 3.6: the keys, the fact.
var refRowRe = regexp.MustCompile("\\| ((?:`[a-z0-9]+\\.[a-z0-9_]+`(?:, | and )?)+) \\| ([^|]+?) \\|")

// refKeys checks that the 37 keys of REF map: 35 to a fact of the spec and the
// two cost-budget keys to none.
func refKeys(t *testing.T, table string, facts map[string]bool) {
	t.Helper()
	mapped, unmapped := 0, 0
	for _, m := range refRowRe.FindAllStringSubmatch(table, -1) {
		n := len(keyRe.FindAllString(m[1], -1))
		if strings.HasPrefix(m[2], "STY-") {
			if !facts[m[2]] {
				t.Errorf("3.6 maps %s to %s, which is not a fact of 5", m[1], m[2])
			}
			mapped += n
		} else {
			unmapped += n
		}
	}
	if mapped != 35 || unmapped != 2 {
		t.Errorf("3.6 maps %d keys to a fact and %d to none, want 35 and 2 (37 keys of REF)", mapped, unmapped)
	}
}

// bandFileKeys checks keys, shape and ordering of the band files.
func bandFileKeys(t *testing.T, text string, files []bandFile) {
	t.Helper()
	known := knownKeys(text)
	names := verdict.ThresholdNames()
	regimes := map[string]int64{}
	for _, f := range files {
		if f.Name != strings.TrimSuffix(regimesFn, ".toml") {
			continue
		}
		var got []string
		for _, k := range f.Keys {
			got = append(got, k.Name)
			if k.Value != 0 && k.Set {
				regimes[k.Name] = k.Value
			}
		}
		if !slices.Equal(got, names) {
			t.Errorf("%s tables = %v, want one table per key of STY 6.2 in its order %v", regimesFn, got, names)
		}
	}
	for _, f := range files {
		for _, k := range f.Keys {
			if !known[k.Name] {
				t.Errorf("%s: key %s belongs to no fact or detector", f.Name, k.Name)
			}
			if k.Unit == "" {
				t.Errorf("%s: key %s has no unit", f.Name, k.Name)
			}
			if f.Name != "regimes" && slices.Contains(names, k.Name) {
				t.Errorf("%s: key %s is a regime threshold; a pack may not set its own (6.5.1)", f.Name, k.Name)
			}
		}
		values := maps(regimes)
		for _, k := range f.Keys {
			if k.Set {
				values[k.Name] = k.Value
			}
		}
		if err := verdict.CheckOrdering(values, 0); err != nil {
			t.Errorf("%s: %v", f.Name, err)
		}
	}
}

func maps(m map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// harness lists the scenarios of the class "harness test" that 5.11 leaves out
// (S77, S78, S84 to S90 and the list under the table) and S15, unused.
var harness = []int{15, 77, 78, 84, 85, 86, 87, 88, 89, 90, 101, 105, 106, 112, 113, 119, 125, 132, 141, 142, 147, 148, 153, 155, 158, 159}

var scenRe = regexp.MustCompile(`S([0-9]+)(?: to S([0-9]+))?`)

// scenarioCoverage checks that every scenario S01 to S165 has a row in 5.11,
// except S15 and the harness tests (FR-STB-005, FR-STY-098).
func scenarioCoverage(t *testing.T, text string) {
	t.Helper()
	raw, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	i, j := strings.Index(string(raw), "### 5.11 "), strings.Index(string(raw), "## 6. The failure regimes")
	covered := map[int]bool{}
	for _, row := range regexp.MustCompile(`(?m)^\| (S[0-9][^|]*)\|`).FindAllStringSubmatch(string(raw)[i:j], -1) {
		for _, m := range scenRe.FindAllStringSubmatch(row[1], -1) {
			a, _ := strconv.Atoi(m[1])
			b := a
			if m[2] != "" {
				b, _ = strconv.Atoi(m[2])
			}
			for n := a; n <= b; n++ {
				covered[n] = true
			}
		}
	}
	for n := 1; n <= 165; n++ {
		if !covered[n] && !slices.Contains(harness, n) {
			t.Errorf("scenario S%02d has no row in 5.11 and is not a harness test", n)
		}
	}
	_ = text
}

// closedSets checks the verdict classes and the detectors against 6.1.2 and 6.2.
func closedSets(t *testing.T, text string) {
	t.Helper()
	i := strings.Index(text, "6.1.2. **A verdict**")
	if i < 0 {
		t.Fatalf("spec has no 6.1.2")
	}
	classes := text[i : i+strings.Index(text[i:], "6.1.3.")]
	for _, c := range []verdict.Class{verdict.None, verdict.Detected, verdict.Unknown} {
		if !strings.Contains(classes, "`"+c.String()+"`") {
			t.Errorf("6.1.2 does not name the class %q", c)
		}
	}
	rules := section(t, text, "### 6.2 The detectors")
	for _, d := range verdict.Detectors() {
		if !strings.Contains(rules, "| **"+string(d)+"** |") {
			t.Errorf("6.2 has no row for detector %s", d)
		}
	}
}

// priorRes finds the prior of a key: the number that follows the key, or the
// number after the word "prior" in the clause that follows it.
func priorOf(text, key string) (int64, bool) {
	q := regexp.QuoteMeta("`" + key + "`")
	num := `([0-9]+(?: [0-9]{3})*)`
	for _, re := range []string{q + ` ` + num + `\b`, q + ".{0,200}?prior " + num + `\b`} {
		if m := regexp.MustCompile(re).FindStringSubmatch(text); m != nil {
			v, err := strconv.ParseInt(strings.ReplaceAll(m[1], " ", ""), 10, 64)
			return v, err == nil
		}
	}
	return 0, false
}

// priors of the keys of the pairs of STY 6.5.2, as the spec states them.
var specPriors = map[string]int64{
	"s07.freeze_ticks": 30, "r5.freeze_ticks": 60, "s03.pinned_ticks": 30, "s04.pinned_ticks": 30, "r5.pinned_ticks": 60,
	"s05.range_factor": 20000, "s12.range_factor": 20000, "x4.range_factor": 20000, "r6.factor": 30000,
	"s05.range_growth_bp": 1000, "r6.growth_bp": 2500, "s06.swing_bp": 300, "r6.swing_bp": 1000,
	"s01.offset_bp": 1500, "h02.offset_bp": 2000, "r8.offset_bp": 3000, "s01.unmet_bp": 1500, "r8.unmet_bp": 3000,
	"p07.drift_bp": 500, "r1.rise_bp": 3000, "r9.fall_bp": 3000, "s10.dispersion_min_bp": 100, "r7.disp_bp": 50,
	"m05.pool_share_bp": 3000, "r2.share_bp": 5000, "x4.households_bp": 9000, "r3.fall_bp": 2000,
	"s16.pinned_share_bp": 1000, "x4.pinned_share_bp": 1000, "r3.horizon_factor": 50000,
}

// TestRegistry_SharedKeys verifies that the priors the test holds are the
// priors of the spec, and that the two priors of the recovery band are equal.
//
// Requirement: FR-STY-072
func TestRegistry_SharedKeys(t *testing.T) {
	t.Parallel()
	text := spec(t)
	for _, k := range sortedKeys(specPriors) {
		if got, ok := priorOf(text, k); !ok || got != specPriors[k] {
			t.Errorf("prior of %s in the spec = %d (found %v), want %d", k, got, ok, specPriors[k])
		}
	}
	a, okA := priorOf(text, "s14.recover_band_bp")
	b, okB := priorOf(text, "r3.recover_bp")
	if !okA || !okB || a != b {
		t.Errorf("prior of s14.recover_band_bp = %d, of r3.recover_bp = %d (found %v, %v), want equal", a, b, okA, okB)
	}
}

func sortedKeys(m map[string]int64) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// TestOrdering_Pairs verifies that every pair of STY 6.5.2 holds for the
// specPriors, that the implication pair does (60 ticks are 1 643 bp of a year),
// that the recovery horizon pair holds for the same t_min, and that a
// violating value is rejected with the pair named.
//
// Requirement: FR-STY-072
func TestOrdering_Pairs(t *testing.T) {
	t.Parallel()
	if err := verdict.CheckOrdering(maps(specPriors), 0); err != nil {
		t.Errorf("CheckOrdering(specPriors) error = %v, want nil", err)
	}
	withRecovery := maps(specPriors)
	withRecovery["s14.recover_ticks"] = 400
	if err := verdict.CheckOrdering(withRecovery, 100); err != nil {
		t.Errorf("CheckOrdering(4 t_min against 5 t_min) error = %v, want nil", err)
	}
	withRecovery["s14.recover_ticks"] = 501
	if err := verdict.CheckOrdering(withRecovery, 100); err == nil {
		t.Errorf("CheckOrdering(s14.recover_ticks 501, t_min 100) error = nil, want the horizon pair rejected")
	}
	for _, tc := range []struct {
		name string
		key  string
		v    int64
	}{
		{"upper bound above its threshold", "s03.pinned_ticks", 61},
		{"range factor above r6.factor", "x4.range_factor", 30001},
		{"floor below its threshold", "s10.dispersion_min_bp", 49},
		{"households below the complement", "x4.households_bp", 7999},
		{"pinned share above a pinned run of a year", "s16.pinned_share_bp", 1644},
		{"drift above the fall", "p07.drift_bp", 3001},
	} {
		v := maps(specPriors)
		v[tc.key] = tc.v
		err := verdict.CheckOrdering(v, 0)
		var oe *verdict.OrderingError
		if !errors.Is(err, verdict.ErrOrdering) || !errors.As(err, &oe) || oe.Pair.Band != tc.key {
			t.Errorf("CheckOrdering(%s: %s = %d) error = %v, want a violation naming %s", tc.name, tc.key, tc.v, err, tc.key)
		}
	}
	// An error reads as the pair and the values; a huge pinned run saturates and holds.
	v := maps(specPriors)
	v["s03.pinned_ticks"] = 61
	if err := verdict.CheckOrdering(v, 0); err == nil || !strings.Contains(err.Error(), "s03.pinned_ticks 61 against r5.pinned_ticks 60") {
		t.Errorf("CheckOrdering(61 against 60) error = %v, want the pair and the values in the message", err)
	}
	v = maps(specPriors)
	v["r5.pinned_ticks"] = 1 << 62
	if err := verdict.CheckOrdering(v, 0); err != nil {
		t.Errorf("CheckOrdering(r5.pinned_ticks 2^62) error = %v, want nil (the implication saturates)", err)
	}
	for _, th := range []int64{-5, -1 << 63} { // a forged threshold: no pass by overflow
		for _, p := range verdict.OrderingPairs() {
			if p.Rel != verdict.PinnedRunShare {
				continue
			}
			if err := verdict.CheckOrdering(map[string]int64{p.Band: 1, p.Threshold: th}, 0); err == nil {
				t.Errorf("CheckOrdering(%s = 1, %s = %d) error = nil, want a violation", p.Band, p.Threshold, th)
			}
		}
	}
	if err := verdict.CheckOrdering(map[string]int64{"s03.pinned_ticks": 99}, 0); err != nil {
		t.Errorf("CheckOrdering(a band with no threshold) error = %v, want nil (an unset key asserts nothing)", err)
	}
	if got := 60 * 10_000 / 365; got != 1643 {
		t.Errorf("60 ticks of 365 = %d bp, want 1643", got)
	}
}

// TestOrdering_EveryPair verifies each pair of OrderingPairs on its own: the
// band at the threshold holds, one past it is rejected naming the pair, and a
// pair with a side missing is skipped.
//
// Requirement: FR-STY-072
func TestOrdering_EveryPair(t *testing.T) {
	t.Parallel()
	pairs := verdict.OrderingPairs()
	if len(pairs) != 18 {
		t.Fatalf("OrderingPairs() has %d pairs, want the 18 of STY 6.5.2", len(pairs))
	}
	wantPairs := []verdict.Pair{
		{Band: "s07.freeze_ticks", Threshold: "r5.freeze_ticks", Rel: verdict.AtMost},
		{Band: "s03.pinned_ticks", Threshold: "r5.pinned_ticks", Rel: verdict.AtMost},
		{Band: "s04.pinned_ticks", Threshold: "r5.pinned_ticks", Rel: verdict.AtMost},
		{Band: "s05.range_factor", Threshold: "r6.factor", Rel: verdict.AtMost},
		{Band: "s12.range_factor", Threshold: "r6.factor", Rel: verdict.AtMost},
		{Band: "x4.range_factor", Threshold: "r6.factor", Rel: verdict.AtMost},
		{Band: "s05.range_growth_bp", Threshold: "r6.growth_bp", Rel: verdict.AtMost},
		{Band: "s06.swing_bp", Threshold: "r6.swing_bp", Rel: verdict.AtMost},
		{Band: "s01.offset_bp", Threshold: "r8.offset_bp", Rel: verdict.AtMost},
		{Band: "h02.offset_bp", Threshold: "r8.offset_bp", Rel: verdict.AtMost},
		{Band: "s01.unmet_bp", Threshold: "r8.unmet_bp", Rel: verdict.AtMost},
		{Band: "p07.drift_bp", Threshold: "r1.rise_bp", Rel: verdict.AtMost},
		{Band: "p07.drift_bp", Threshold: "r9.fall_bp", Rel: verdict.AtMost},
		{Band: "s10.dispersion_min_bp", Threshold: "r7.disp_bp", Rel: verdict.AtLeast},
		{Band: "m05.pool_share_bp", Threshold: "r2.share_bp", Rel: verdict.AtMost},
		{Band: "x4.households_bp", Threshold: "r3.fall_bp", Rel: verdict.AtLeastComplement},
		{Band: "s16.pinned_share_bp", Threshold: "r5.pinned_ticks", Rel: verdict.PinnedRunShare},
		{Band: "x4.pinned_share_bp", Threshold: "r5.pinned_ticks", Rel: verdict.PinnedRunShare},
	}
	if !slices.Equal(pairs, wantPairs) {
		t.Errorf("OrderingPairs() = %v, want the pairs of STY 6.5.2 in the order of the spec %v", pairs, wantPairs)
	}
	// A pinned run of a whole year is the whole year (10 000 bp), not one less.
	if err := verdict.CheckOrdering(map[string]int64{"r5.pinned_ticks": 365, "s16.pinned_share_bp": 10_000}, 0); err != nil {
		t.Errorf("CheckOrdering(a pinned run of 365 ticks against 10000 bp) error = %v, want nil", err)
	}
	for _, p := range pairs {
		th := specPriors[p.Threshold]
		var at, past int64
		switch p.Rel {
		case verdict.AtMost:
			at, past = th, th+1
		case verdict.AtLeast:
			at, past = th, th-1
		case verdict.AtLeastComplement:
			at, past = 10_000-th, 10_000-th-1
		case verdict.PinnedRunShare:
			at = th * 10_000 / 365
			past = at + 1
		}
		if err := verdict.CheckOrdering(map[string]int64{p.Band: at, p.Threshold: th}, 0); err != nil {
			t.Errorf("CheckOrdering(%s = %d at %s = %d) error = %v, want nil", p.Band, at, p.Threshold, th, err)
		}
		err := verdict.CheckOrdering(map[string]int64{p.Band: past, p.Threshold: th}, 0)
		var oe *verdict.OrderingError
		if !errors.As(err, &oe) || oe.Pair != p || oe.Band != past || oe.Threshold != th {
			t.Errorf("CheckOrdering(%s = %d past %s = %d) error = %v, want a violation of that pair with its values", p.Band, past, p.Threshold, th, err)
		}
		if err := verdict.CheckOrdering(map[string]int64{p.Threshold: th}, 0); err != nil {
			t.Errorf("CheckOrdering(%s unset) error = %v, want nil", p.Band, err)
		}
	}
	// The horizon pair: only with t_min, and only with both keys.
	for _, tc := range []struct {
		name string
		in   map[string]int64
		tMin int64
		bad  bool
	}{
		{"4 against 5", map[string]int64{"s14.recover_ticks": 500, "r3.horizon_factor": 50000}, 100, false},
		{"just over", map[string]int64{"s14.recover_ticks": 501, "r3.horizon_factor": 50000}, 100, true},
		{"no t_min", map[string]int64{"s14.recover_ticks": 501, "r3.horizon_factor": 50000}, 0, false},
		{"no factor", map[string]int64{"s14.recover_ticks": 501}, 100, false},
		{"no recover ticks", map[string]int64{"r3.horizon_factor": 50000}, 100, false},
		{"one part in ten thousand over", map[string]int64{"s14.recover_ticks": 100000, "r3.horizon_factor": 999_995_000}, 1, true},
		{"at the limit", map[string]int64{"s14.recover_ticks": 100000, "r3.horizon_factor": 1_000_000_000}, 1, false},
	} {
		if err := verdict.CheckOrdering(tc.in, tc.tMin); (err != nil) != tc.bad {
			t.Errorf("CheckOrdering(horizon, %s) error = %v, want an error: %v", tc.name, err, tc.bad)
		}
	}
}

// TestSteps_EveryStep verifies the order of STY 10.3 step by step: a key of
// each step set while a key of the step before is unset is rejected, and the
// same with a key of an earlier step set is not.
//
// Requirement: FR-STY-091
func TestSteps_EveryStep(t *testing.T) {
	t.Parallel()
	first := map[int]string{}
	for _, s := range verdict.StepOrder() {
		if _, ok := first[s.Step]; !ok {
			first[s.Step] = s.Key
		}
	}
	for step := 3; step <= 9; step++ {
		prev, cur := first[step-1], first[step]
		if prev == "" || cur == "" {
			t.Fatalf("steps %d or %d have no key, want keys at every step from 2 to 9", step-1, step)
		}
		if err := verdict.CheckSteps(map[string]bool{prev: false, cur: true}); !errors.Is(err, verdict.ErrStepOrder) {
			t.Errorf("CheckSteps(%s unset, %s set) error = %v, want ErrStepOrder", prev, cur, err)
		}
		if err := verdict.CheckSteps(map[string]bool{prev: true, cur: false}); err != nil {
			t.Errorf("CheckSteps(%s set, %s unset) error = %v, want nil (the later step may wait)", prev, cur, err)
		}
	}
	if err := verdict.CheckSteps(nil); err != nil {
		t.Errorf("CheckSteps(nil) error = %v, want nil", err)
	}
}

// TestBandFiles_Order verifies the unit order of STY 10.3 over the files and
// on built cases.
//
// Requirement: FR-STY-091
func TestBandFiles_Order(t *testing.T) {
	t.Parallel()
	declared := map[string]bool{}
	for _, f := range loadBandFiles(t) {
		for _, k := range f.Keys {
			declared[k.Name] = declared[k.Name] || k.Set
		}
	}
	if err := verdict.CheckSteps(declared); err != nil {
		t.Errorf("CheckSteps(band files) error = %v, want nil", err)
	}
	text := spec(t)
	known := knownKeys(text)
	for _, s := range verdict.StepOrder() {
		if !known[s.Key] {
			t.Errorf("step %d lists %s, which the spec does not name", s.Step, s.Key)
		}
	}
	for _, tc := range []struct {
		name string
		in   map[string]bool
		bad  bool
	}{
		{"later set, earlier unset", map[string]bool{"s03.pinned_ticks": false, "w02.n_band_bp": true}, true},
		{"threshold unset, band set", map[string]bool{"r1.rise_bp": false, "s03.pinned_ticks": true}, true},
		{"both set", map[string]bool{"s03.pinned_ticks": true, "w02.n_band_bp": true}, false},
		{"later set, earlier not declared", map[string]bool{"w02.n_band_bp": true}, false},
		{"same step", map[string]bool{"s03.pinned_ticks": false, "s04.pinned_ticks": true}, false},
		{"all unset", map[string]bool{"s03.pinned_ticks": false, "w02.n_band_bp": false}, false},
	} {
		err := verdict.CheckSteps(tc.in)
		if (err != nil) != tc.bad || (tc.bad && !errors.Is(err, verdict.ErrStepOrder)) {
			t.Errorf("CheckSteps(%s) error = %v, want an error: %v", tc.name, err, tc.bad)
		}
	}
}

// versionOneKeys are the keys of STY 8.1.1 that must have values before version
// 1 is called done.
var versionOneKeys = []string{
	"s03.pinned_ticks", "s04.pinned_ticks", "s07.freeze_ticks", "s07.spoiled_bp", "s16.pinned_share_bp", "x4.pinned_share_bp",
	"s05.range_factor", "s05.range_growth_bp", "s12.range_factor", "x4.range_factor",
	"x1.margin_bp", "x1.worst_bp", "x1.warmup",
	"s01.band_bp", "s01.recover_ticks", "s01.unmet_bp",
	"s14.recover_ticks", "s14.recover_band_bp", "s14.fall_bp",
	"a1.price_band_bp", "h02.offset_bp", "h02.money_bp", "s17.spike_factor", "s17.control_gap_bp",
	"s01.offset_bp", "x4.households_bp",
}

// TestRelease_VersionOneKeys verifies that the keys of STY 8.1.1 exist in the
// band files and, with -stylised.release, that each has a value in every file
// that declares it; without the flag it logs the unset ones. The keys of
// regimes.toml whose detector separates are not listed: which separate is
// known only after the setting run.
//
// Requirement: FR-STY-012
func TestRelease_VersionOneKeys(t *testing.T) {
	t.Parallel()
	set, declared := map[string]bool{}, map[string]bool{}
	for _, f := range loadBandFiles(t) {
		for _, k := range f.Keys {
			v, seen := set[k.Name]
			set[k.Name], declared[k.Name] = (!seen || v) && k.Set, true
		}
	}
	var unset []string
	for _, k := range versionOneKeys {
		switch {
		case !declared[k]:
			t.Errorf("version 1 key %s is declared in no band file", k)
		case !set[k]:
			unset = append(unset, k)
		}
	}
	if len(unset) == 0 {
		return
	}
	if *release {
		t.Errorf("-stylised.release: %d keys of STY 8.1.1 have no value: %v", len(unset), unset)
		return
	}
	t.Logf("%d keys of STY 8.1.1 have no value yet (report; -stylised.release makes it an error): %v", len(unset), unset)
}

// claimRe finds "stable" used as a claim about a default, a pack, a
// configuration or a run (STY 7.3.2). The sort is stable and "not stable across
// configurations" (an ID) are not claims.
var claimRe = regexp.MustCompile(`\bstable (default|pack|configuration|config|run|economy|world|state)\b|\b(default|pack|configuration|config|run|economy|world)s? (is|are|remains?|stays?) stable\b`)

// TestWord_Stable verifies by a source text check that no exported doc
// comment, error string or log message of the module uses "stable" as a claim
// about a default, a pack or a run; the three words of a check's verdict are
// upper case and are not matched.
//
// Requirement: FR-STY-096
func TestWord_Stable(t *testing.T) {
	t.Parallel()
	fset := token.NewFileSet()
	files := 0
	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "testdata" || d.Name() == ".git" || d.Name() == "examples") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if perr != nil {
			return perr
		}
		files++
		for _, msg := range claimTexts(f) {
			if claimRe.MatchString(msg.text) {
				t.Errorf("%s: %s claims stability: %q (STY 7.3.2)", fset.Position(msg.pos), msg.kind, msg.text)
			}
		}
		return nil
	})
	if err != nil || files < 100 {
		t.Fatalf("walk %s: %d files, error %v, want at least 100 files", repoRoot, files, err)
	}
	for _, s := range []string{"The default is stable.", "the pack is stable", "a stable configuration", "stable state"} {
		if !claimRe.MatchString(s) {
			t.Errorf("claimRe does not match %q, want it to", s)
		}
	}
	for _, s := range []string{"The sort is stable.", "It is not stable across configurations.", "NOT STABLE", "so IDs stay stable"} {
		if claimRe.MatchString(s) {
			t.Errorf("claimRe matches %q, want it not to", s)
		}
	}
}

type claimText struct {
	pos  token.Pos
	kind string
	text string
}

// claimTexts returns the package comment, the doc comments of exported
// declarations and the first string literal of every call of errors.New,
// fmt.Errorf or a log method.
func claimTexts(f *ast.File) []claimText {
	var out []claimText
	if f.Doc != nil {
		out = append(out, claimText{f.Doc.Pos(), "package comment", ws.ReplaceAllString(f.Doc.Text(), " ")})
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if x.Name.IsExported() && x.Doc != nil {
				out = append(out, claimText{x.Doc.Pos(), "doc comment of " + x.Name.Name, ws.ReplaceAllString(x.Doc.Text(), " ")})
			}
		case *ast.GenDecl:
			if x.Doc != nil && len(x.Specs) > 0 && exportedSpec(x.Specs[0]) {
				out = append(out, claimText{x.Doc.Pos(), "doc comment", ws.ReplaceAllString(x.Doc.Text(), " ")})
			}
		case *ast.CallExpr:
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok && logMethods[sel.Sel.Name] && len(x.Args) > 0 {
				if lit, ok := x.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					out = append(out, claimText{lit.Pos(), sel.Sel.Name + " message", lit.Value})
				}
			}
		}
		return true
	})
	return out
}

var logMethods = map[string]bool{"New": true, "Errorf": true, "Debug": true, "Info": true, "Warn": true, "Error": true}

func exportedSpec(s ast.Spec) bool {
	switch x := s.(type) {
	case *ast.TypeSpec:
		return x.Name.IsExported()
	case *ast.ValueSpec:
		return len(x.Names) > 0 && x.Names[0].IsExported()
	}
	return false
}
