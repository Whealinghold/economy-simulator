package calibration

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "records", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// specRecords returns the two fenced blocks of 19.5 of the spec, the stage record
// and the stability record.
func specRecords(t *testing.T) (stage, stability []byte) {
	t.Helper()
	spec, err := os.ReadFile("../../doc/specs/spec-calibration.md")
	if err != nil {
		t.Fatal(err)
	}
	sec := between(string(spec), "### 19.5 A record", "### 19.6 The overlay")
	parts := strings.Split(sec, "```")
	if len(parts) < 5 {
		t.Fatalf("19.5 has %d fenced parts, want at least 5", len(parts))
	}
	return []byte(strings.TrimPrefix(parts[1], "\n")), []byte(strings.TrimPrefix(parts[3], "\n"))
}

// TestRecord_RenderParseRender verifies that a record parses and renders back to
// its own bytes, and that rendering the parsed value and parsing it again gives
// the same value: the minimal record of every keyword and the two records of 19.5.
//
// Requirement: FR-CLB-041, FR-CLB-043
func TestRecord_RenderParseRender(t *testing.T) {
	t.Parallel()
	stage, stab := specRecords(t)
	min := readFixture(t, "valid/minimal.txt")
	none := strings.Replace(string(min), "chosen 1 distance 1", "chosen none distance 0", 1)
	tests := map[string][]byte{"minimal": min, "19.5 stage record": stage, "19.5 stability record": stab, "chosen none": []byte(none)}
	for name, in := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r, err := Parse(in)
			if err != nil {
				t.Fatalf("Parse(%s) error = %v", name, err)
			}
			out, err := Render(r)
			if err != nil || string(out) != string(in) {
				t.Fatalf("Render(Parse(%s)) = %q, %v, want the input", name, out, err)
			}
			again, err := Parse(out)
			if err != nil || !reflect.DeepEqual(again, r) {
				t.Errorf("Parse(Render(r)) = %+v, %v, want %+v", again, err, r)
			}
		})
	}
}

// TestRecord_Fixture19_5 verifies that the worked record of 19.5 parses and
// renders unchanged and satisfies the record checks of 11.2 (E6), against a
// registry that holds the rows it names.
//
// Requirement: FR-CLB-039, FR-CLB-041, FR-CLB-100
func TestRecord_Fixture19_5(t *testing.T) {
	t.Parallel()
	stage, stab := specRecords(t)
	for name, in := range map[string][]byte{"stage": stage, "stability": stab} {
		r, err := Parse(in)
		if err != nil {
			t.Fatalf("Parse(19.5 %s) error = %v", name, err)
		}
		if ps := CheckRecord(r, specKnown()); len(ps) != 0 {
			t.Errorf("CheckRecord(19.5 %s) = %+v, want none", name, ps)
		}
	}
}

// TestRecord_UnknownKeyword verifies that a keyword outside the closed set, in
// the middle of a record or alone, is ErrRecord and names its line.
//
// Requirement: FR-CLB-043
func TestRecord_UnknownKeyword(t *testing.T) {
	t.Parallel()
	for _, in := range [][]byte{readFixture(t, "invalid/unknown-keyword.txt"), []byte("flavour\n"), []byte("Pack one\n"), []byte("\n")} {
		_, err := Parse(in)
		if !errors.Is(err, ErrRecord) {
			t.Errorf("Parse(%q) error = %v, want ErrRecord", in, err)
		}
	}
	_, err := Parse(readFixture(t, "invalid/unknown-keyword.txt"))
	if err == nil || !strings.Contains(err.Error(), "line 3") || !strings.Contains(err.Error(), `"flavour"`) {
		t.Errorf("Parse(unknown keyword) error = %v, want it to name line 3 and the keyword", err)
	}
}

// TestRecord_ParseErrors verifies that Parse refuses everything that is not the
// canonical form: files of the invalid directory, and the minimal record with one
// line changed in each way the grammar forbids.
//
// Requirement: FR-CLB-041, FR-CLB-043
func TestRecord_ParseErrors(t *testing.T) {
	t.Parallel()
	min := string(readFixture(t, "valid/minimal.txt"))
	swap := func(old, new string) string {
		if !strings.Contains(min, old) {
			t.Fatalf("minimal record has no %q", old)
		}
		return strings.Replace(min, old, new, 1)
	}
	tests := map[string]string{
		"float in a field":           string(readFixture(t, "invalid/float-ticks.txt")),
		"keywords out of order":      string(readFixture(t, "invalid/out-of-order.txt")),
		"no decision":                string(readFixture(t, "invalid/no-decision.txt")),
		"no final line feed":         strings.TrimSuffix(min, "\n"),
		"empty":                      "",
		"crlf":                       strings.ReplaceAll(min, "\n", "\r\n"),
		"plus sign":                  swap("points 5 ", "points +5 "),
		"leading zero":               swap("attempt 1 ", "attempt 01 "),
		"one space in a pack line":   swap("pack one  lineage", "pack one lineage"),
		"three spaces":               swap("commit 4f2a9c1  parent", "commit 4f2a9c1   parent"),
		"extra field":                swap("document base.toml 3b9c11aa5e07f4d2", "document base.toml 3b9c11aa5e07f4d2 x"),
		"missing field":              swap("document base.toml 3b9c11aa5e07f4d2", "document base.toml"),
		"wrong keyword in a line":    swap("seeding same ticks", "seeding same tick"),
		"design outside the set":     swap("design oat", "design bogus"),
		"edge outside the set":       swap("edge below:rule", "edge below:other"),
		"uppercase hex":              swap("lineage 5a1c0e4477d3b210", "lineage 5A1C0E4477D3B210"),
		"short hex":                  swap("lineage 5a1c0e4477d3b210", "lineage 5a1c0e44"),
		"stage outside the set":      swap("stage C4a", "stage C10"),
		"class outside the set":      swap("class B kind fact", "class b kind fact"),
		"sign outside the set":       swap("expected +", "expected x"),
		"claim outside the set":      swap("pack one undetermined", "pack one maybe"),
		"result outside the set":     swap("result report", "result skipped"),
		"decision outside the set":   swap("decision stop", "decision halt"),
		"reentry of no stage":        swap("decision stop", "decision reentry C10"),
		"yes or no":                  swap("control yes", "control maybe"),
		"seeds range":                swap("seeds 1..50", "seeds 1-50"),
		"seed list":                  swap("outside 5,9", "outside 5;9"),
		"chosen index":               swap("chosen 1 distance", "chosen one distance"),
		"empty knob list":            swap("unsearched at_off K-MON01", "unsearched at_off"),
		"empty pareto":               swap("pareto 0 1", "pareto"),
		"repeated pack line":         swap("commit 4f2a9c1", "pack one  lineage 5a1c0e4477d3b210  stage C4a  attempt 1  block 1\ncommit 4f2a9c1"),
		"negative count":             swap("replicas 50", "replicas -50"),
		"wrong format version":       swap("calibration 1", "calibration 2"),
		"note with a carriage":       swap("note CostFloor", "note \rCostFloor"),
		"finding without a slash":    swap("P02/Pinned.Share", "P02"),
		"unmet with no values":       swap("unmet P04 offset_bp min 310 median 780", "unmet P04 offset_bp"),
		"grid outside the set":       swap("grid geo", "grid log"),
		"seed0 negative":             swap("seed0 101 seeding same ticks 1095 transient 365 every 365 salt 0 render c7f20b19a4e83d60", "seed0 -1 seeding same ticks 1095 transient 365 every 365 salt 0 render c7f20b19a4e83d60"),
		"int too long":               swap("points 5 ", "points 99999999999999999999 "),
		"class of two characters":    swap("class B kind fact", "class BB kind fact"),
		"seeds not numbers":          swap("seeds 1..50", "seeds a..b"),
		"seeds with a third part":    swap("seeds 1..50", "seeds 1..2..3"),
		"outside list":               swap("outside 5,9", "outside 5,x"),
		"pareto index":               swap("pareto 0 1", "pareto 0 x"),
		"no fields at all":           swap("unsearched at_off K-MON01", "unsearched"),
		"absent value with a number": swap("lower 3500 upper 14000", "lower x upper 14000"),
	}
	for name, in := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := Parse([]byte(in)); !errors.Is(err, ErrRecord) {
				t.Errorf("Parse(%s) error = %v, want ErrRecord", name, err)
			}
		})
	}
}

// TestRecord_NoFloatNoMapOrder verifies that a rendered record holds no float and
// no number with an exponent or a sign it does not need, and that rendering it
// again gives the same bytes: no field depends on a map order.
//
// Requirement: FR-CLB-041
func TestRecord_NoFloatNoMapOrder(t *testing.T) {
	t.Parallel()
	r, err := Parse(readFixture(t, "valid/minimal.txt"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := Render(r)
	if err != nil {
		t.Fatal(err)
	}
	for range 50 {
		if again, _ := Render(r); string(again) != string(first) {
			t.Fatalf("Render twice gave different bytes:\n%s\n%s", first, again)
		}
	}
	float := regexp.MustCompile(`[0-9]+\.[0-9]+`) // a seed range 1..50 and a name like s01.offset_bp are not floats
	for _, line := range strings.Split(string(first), "\n") {
		if !strings.HasPrefix(line, "note ") && float.MatchString(line) {
			t.Errorf("line %q holds a float", line)
		}
	}
}

// TestRecord_Digest verifies the digest against the SHA-256 test vectors of FIPS
// 180-2: the first 8 bytes, as 16 lower-case hex digits.
//
// Requirement: FR-CLB-040
func TestRecord_Digest(t *testing.T) {
	t.Parallel()
	tests := []struct{ in, want string }{
		{"", "e3b0c44298fc1c14"},    // SHA-256 of the empty string starts e3b0c44298fc1c14
		{"abc", "ba7816bf8f01cfea"}, // FIPS 180-2 vector: ba7816bf 8f01cfea 414140de ...
	}
	for _, tt := range tests {
		if got := Digest([]byte(tt.in)); got != tt.want {
			t.Errorf("Digest(%q) = %s, want %s", tt.in, got, tt.want)
		}
	}
}

// TestRecord_RenderIsStable verifies that a fixed record renders to a fixed
// literal: the bytes do not change with the platform or the Go version.
//
// Requirement: NFR-CLB-002
func TestRecord_RenderIsStable(t *testing.T) {
	t.Parallel()
	r := Record{
		Pack: "one", Lineage: "5a1c0e4477d3b210", Stage: "C1", Attempt: 1, Block: 0, Commit: "4f2a9c1",
		Base: "91d3a7c2e05b4a18", Chosen: "same",
		Documents:  []Document{{"base.toml", "3b9c11aa5e07f4d2"}},
		Stability:  []Stability{{Knob: "K-BEL12", Pack: "one", Point: "default", Value: 7000, Set: "gate", First: 1, Last: 10, InSample: true, CBP: "STABLE", CNC: "STABLE", CWS: "UNDETERMINED", Edge: "none"}},
		Claims:     []Claim{{"K-BEL12", "one", "undetermined"}},
		Decision:   "review",
		Notes:      []string{"a note"},
		ScreenedIn: nil,
	}
	want := "calibration 1\n" +
		"pack one  lineage 5a1c0e4477d3b210  stage C1  attempt 1  block 0\n" +
		"commit 4f2a9c1  parent -\n" +
		"config base 91d3a7c2e05b4a18  chosen same\n" +
		"document base.toml 3b9c11aa5e07f4d2\n" +
		"stability K-BEL12 pack one point default value 7000 set gate seeds 1..10 in_sample yes c_bp STABLE c_nc STABLE c_ws UNDETERMINED edge none\n" +
		"claim K-BEL12 pack one undetermined\n" +
		"decision review\n" +
		"note a note\n"
	if got, err := Render(r); err != nil || string(got) != want {
		t.Errorf("Render(fixed record) = %q, %v, want %q", got, err, want)
	}
}

// TestRecord_RenderErrors verifies that Render refuses a record the grammar cannot
// write, so a record that renders always parses back to itself.
//
// Requirement: FR-CLB-041, FR-CLB-043
func TestRecord_RenderErrors(t *testing.T) {
	t.Parallel()
	base := func() Record {
		r, err := Parse(readFixture(t, "valid/minimal.txt"))
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	tests := map[string]func(*Record){
		"empty pack":                func(r *Record) { r.Pack = "" },
		"space in a token":          func(r *Record) { r.Pack = "o ne" },
		"non-ASCII token":           func(r *Record) { r.Pack = "café" },
		"control in a token":        func(r *Record) { r.Commit = "a\tb" },
		"attempt 0":                 func(r *Record) { r.Attempt = 0 },
		"negative block":            func(r *Record) { r.Block = -1 },
		"bad lineage":               func(r *Record) { r.Lineage = "xyz" },
		"bad parent":                func(r *Record) { r.Parent = "xyz" },
		"bad chosen hash":           func(r *Record) { r.Chosen = "samey" },
		"bad stage":                 func(r *Record) { r.Stage = "" },
		"bad document digest":       func(r *Record) { r.Documents[0].Digest = "1234" },
		"unsearched without knobs":  func(r *Record) { r.Unsearched[0].Knobs = nil },
		"absent opt with a value":   func(r *Record) { r.Factors[1].Lower = Opt{Value: 3} },
		"feasible values, no line":  func(r *Record) { r.Feasible = Feasible{N: 1} },
		"negative feasible":         func(r *Record) { r.Feasible.N = -1 },
		"chosen values, no line":    func(r *Record) { r.Pick = Pick{Index: 2} },
		"chosen none with an index": func(r *Record) { r.Pick = Pick{Has: true, None: true, Index: 2} },
		"negative pareto":           func(r *Record) { r.Pareto = []int{-1} },
		"confirm values, no line":   func(r *Record) { r.Confirm = ConfirmLine{First: 1} },
		"note with a line feed":     func(r *Record) { r.Notes = []string{"a\nb"} },
		"unmet values with spaces":  func(r *Record) { r.Unmet[0].Values = "a  b" },
		"empty unmet values":        func(r *Record) { r.Unmet[0].Values = "" },
		"no decision":               func(r *Record) { r.Decision = "" },
		"bad grid step":             func(r *Record) { r.Factors[0].Grid = "lin" },
		"grid step with a letter":   func(r *Record) { r.Factors[0].Grid = "lin5x" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r := base()
			mutate(&r)
			if out, err := Render(r); !errors.Is(err, ErrRecord) || out != nil {
				t.Errorf("Render(%s) = %q, %v, want nil and ErrRecord", name, out, err)
			}
		})
	}
	// An empty-valued optional is fine: no screened_in, no pareto, no claim.
	r := base()
	r.ScreenedIn, r.Pareto, r.Claims = nil, nil, nil
	if _, err := Render(r); err != nil {
		t.Errorf("Render(no screened_in, pareto, claim) error = %v, want nil", err)
	}
}

// FuzzParse verifies that Parse never panics, that every input it accepts renders
// back to the same bytes, and that every input it refuses is ErrRecord.
//
// #lizard forgives: one fuzz body, one property per block.
func FuzzParse(f *testing.F) {
	for _, name := range []string{"valid/minimal.txt", "invalid/unknown-keyword.txt", "invalid/float-ticks.txt", "invalid/out-of-order.txt", "invalid/no-decision.txt"} {
		if b, err := os.ReadFile(filepath.Join("testdata", "records", name)); err == nil {
			f.Add(b)
		}
	}
	f.Add([]byte("calibration 1\npack one  lineage 5a1c0e4477d3b210  stage C1  attempt 1  block 0\ncommit x  parent -\nconfig base 91d3a7c2e05b4a18  chosen same\ndecision stop\n"))
	f.Add([]byte("note \n"))
	f.Add([]byte(""))
	f.Fuzz(func(t *testing.T, b []byte) {
		r, err := Parse(b)
		if err != nil {
			if !errors.Is(err, ErrRecord) {
				t.Fatalf("Parse(%q) error = %v, want ErrRecord", b, err)
			}
			return
		}
		out, err := Render(r)
		if err != nil || string(out) != string(b) {
			t.Fatalf("Render(Parse(%q)) = %q, %v, want the input", b, out, err)
		}
	})
}
