package verdict_test

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/verdict"
)

const okSetBy = `"commit=abc123;config=9f8e;seeds=1-50"`

// TestBandFile_Grammar verifies the plain grammar of bands/: tables, unit,
// value, set_by and expects, comments, a value of 0 as a value, a key with no
// value as unset, a value without set_by rejected, a set_by whose seeds are not
// exactly 1-50 (1-10 for a tier L key) rejected, and a malformed line reported
// with its number.
//
// Requirement: FR-STY-004, FR-STY-070, FR-STY-005 (STY 12 "shape" and "seeds"), FR-REF-022
//
// Guard: a gate that reads a half-set or wrongly-seeded band asserts a bound
// nobody reviewed.
func TestBandFile_Grammar(t *testing.T) {
	t.Parallel()
	ok := `# comment
[s01.band_bp]
unit = "bp"
value = 0
set_by = ` + okSetBy + `

[s01.recover_ticks]
unit = "ticks"
[s14.fall_bp]
unit = "bp"
expects = ["r3", "r4"]
[min_windows]
unit = "windows"
expects = []
[w02.n_band_bp]
unit = "bp"
value = -7
set_by = "commit=a;config=b;seeds=1-10"
`
	keys, err := verdict.LoadBands(ok)
	if err != nil {
		t.Fatalf("LoadBands(ok) error = %v, want nil", err)
	}
	want := []verdict.BandKey{
		{Name: "s01.band_bp", Unit: "bp", Value: 0, Set: true, SetBy: "commit=abc123;config=9f8e;seeds=1-50", Line: 2},
		{Name: "s01.recover_ticks", Unit: "ticks", Line: 7},
		{Name: "s14.fall_bp", Unit: "bp", Expects: []string{"r3", "r4"}, Line: 9},
		{Name: "min_windows", Unit: "windows", Line: 12},
		{Name: "w02.n_band_bp", Unit: "bp", Value: -7, Set: true, SetBy: "commit=a;config=b;seeds=1-10", Line: 15},
	}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("LoadBands(ok) = %+v, want %+v", keys, want)
	}
	bad := []struct{ name, src, msg string }{
		{"value without set_by", "[a.b]\nvalue = 1\n", "no set_by"},
		{"set_by without value", "[a.b]\nset_by = " + okSetBy + "\n", "no value"},
		{"seeds 5-20", "[a.b]\nvalue = 1\nset_by = \"commit=a;config=b;seeds=5-20\"\n", "want 1-50"},
		{"seeds 1-30", "[a.b]\nvalue = 1\nset_by = \"commit=a;config=b;seeds=1-30\"\n", "want 1-50"},
		{"seeds 1-10 on a tier G key", "[a.b]\nvalue = 1\nset_by = \"commit=a;config=b;seeds=1-10\"\n", "want 1-50"},
		{"seeds 1-50 on a tier L key", "[w06.money_cap_bp]\nvalue = 1\nset_by = " + okSetBy + "\n", "want 1-10"},
		{"set_by with no hash", "[a.b]\nvalue = 1\nset_by = \"commit=a;seeds=1-50\"\n", "not commit="},
		{"set_by with empty commit", "[a.b]\nvalue = 1\nset_by = \"commit=;config=b;seeds=1-50\"\n", "not commit="},
		{"set_by seeds not a range", "[a.b]\nvalue = 1\nset_by = \"commit=a;config=b;seeds=x\"\n", "seeds"},
		{"line before table", "unit = \"bp\"\n", "line 1"},
		{"not key = value", "[a.b]\nunit\n", "line 2"},
		{"unknown field", "[a.b]\nunits = \"bp\"\n", "unknown key"},
		{"value not an integer", "[a.b]\nvalue = 1.5\n", "not an integer"},
		{"unit not a string", "[a.b]\nunit = bp\n", "not a string"},
		{"expects not a list", "[a.b]\nexpects = \"r3\"\n", "not a list"},
		{"expects item not a string", "[a.b]\nexpects = [r3]\n", "not a string"},
		{"value without set_by before the next table", "[a.b]\nvalue = 1\n[c.d]\n", "no set_by"},
		{"unit one quote", "[a.b]\nunit = \"\n", "not a string"},
		{"unit quote only at the end", "[a.b]\nunit = bp\"\n", "not a string"},
		{"unit quote only at the start", "[a.b]\nunit = \"bp\n", "not a string"},
		{"unit with a quote inside", "[a.b]\nunit = \"a\"b\"\n", "not a string"},
		{"unit with a backslash", "[a.b]\nunit = \"a\\\\b\"\n", "not a string"},
		{"set_by empty", "[a.b]\nvalue = 1\nset_by = \"\"\n", "not a string"},
		{"expects without a close", "[a.b]\nexpects = [\"r3\"\n", "not a list"},
		{"expects without an open", "[a.b]\nexpects = \"r3\"]\n", "not a list"},
		{"expects empty item", "[a.b]\nexpects = [\"r3\", \"\"]\n", "not a string"},
		{"table twice", "[a.b]\n[a.b]\n", "twice"},
		{"field twice", "[a.b]\nunit = \"bp\"\nunit = \"bp\"\n", "twice"},
		{"two dots", "[a.b.c]\n", "not a table header"},
		{"upper case", "[A.b]\n", "not a table header"},
		{"empty part", "[a.]\n", "not a table header"},
		{"unclosed", "[a.b\n", "not a table header"},
	}
	for _, tc := range bad {
		_, err := verdict.LoadBands(tc.src)
		var pe *verdict.ParseError
		if !errors.As(err, &pe) || !strings.Contains(err.Error(), tc.msg) {
			t.Errorf("LoadBands(%s) error = %v, want a ParseError containing %q", tc.name, err, tc.msg)
		}
	}
}

// TestBandFile_Values verifies that a value is read in base 10 over the whole
// int64 range and that a header after a half-set table names that table.
//
// Requirement: FR-STY-070
func TestBandFile_Values(t *testing.T) {
	t.Parallel()
	for _, v := range []int64{0, 7, 8, 9, 10, 98765, -98765, 1<<63 - 1, -1 << 63} {
		src := "[a.b]\nvalue = " + strconv.FormatInt(v, 10) + "\nset_by = " + okSetBy + "\n"
		keys, err := verdict.LoadBands(src)
		if err != nil || len(keys) != 1 || keys[0].Value != v || !keys[0].Set {
			t.Errorf("LoadBands(value = %d) = %+v, %v, want the value read", v, keys, err)
		}
	}
	// A header after a table without its set_by names that table's line.
	_, err := verdict.LoadBands("[a.b]\nvalue = 1\n[c.d]\nunit = \"bp\"\n")
	var pe *verdict.ParseError
	if !errors.As(err, &pe) || pe.Line != 1 {
		t.Errorf("LoadBands(two tables, first without set_by) error = %v, want a ParseError at line 1", err)
	}
	// The error reads line and message.
	if got := (&verdict.ParseError{Line: 3, Msg: "bad"}).Error(); got != "verdict: band file line 3: bad" {
		t.Errorf("ParseError.Error() = %q, want the line and the message", got)
	}
}

// TestBandFile_ThresholdsOf verifies that the keys of regimes.toml fill the
// thresholds, that a value of 0 is a value, and that a table that is not a
// threshold is refused.
//
// Requirement: FR-STY-070, FR-STY-060
func TestBandFile_ThresholdsOf(t *testing.T) {
	t.Parallel()
	keys, err := verdict.LoadBands("[r1.rise_bp]\nvalue = 0\nset_by = " + okSetBy + "\n[r1.windows]\n")
	if err != nil {
		t.Fatal(err)
	}
	th, err := verdict.ThresholdsOf(keys)
	if err != nil {
		t.Fatalf("ThresholdsOf error = %v, want nil", err)
	}
	if v, set := th.Get("r1.rise_bp"); v != 0 || !set {
		t.Errorf("Get(r1.rise_bp) = %d, %v, want 0, true", v, set)
	}
	if _, set := th.Get("r1.windows"); set {
		t.Errorf("Get(r1.windows) set, want unset")
	}
	if _, err := verdict.ThresholdsOf([]verdict.BandKey{{Name: "s01.band_bp"}}); err == nil {
		t.Errorf("ThresholdsOf(s01.band_bp) error = nil, want a refusal")
	}
	if _, err := verdict.ThresholdsOf([]verdict.BandKey{{Name: "r3.horizon_factor", Value: -1, Set: true}}); err == nil {
		t.Errorf("ThresholdsOf(r3.horizon_factor -1) error = nil, want a refusal: it could never fire")
	}
}

// FuzzBandFile verifies that no input makes the parser panic and that what it
// accepts has a value only with a set_by.
//
// Requirement: NFR-STY-003
func FuzzBandFile(f *testing.F) {
	f.Add("[a.b]\nunit = \"bp\"\n")
	f.Add("[a.b]\nvalue = 1\nset_by = " + okSetBy + "\n")
	f.Add("[x]\nexpects = [\"r3\", \"r4\"]\n")
	f.Fuzz(func(t *testing.T, src string) {
		keys, err := verdict.LoadBands(src)
		if err != nil {
			return
		}
		for _, k := range keys {
			if k.Set != (k.SetBy != "") {
				t.Fatalf("LoadBands(%q): %+v has a value and set_by out of step", src, k)
			}
		}
	})
}

func TestBandFile_ThresholdsOf_Rejects(t *testing.T) {
	for name, keys := range map[string][]verdict.BandKey{
		"duplicate": {{Name: "r1.rise_bp", Set: true, Value: 1}, {Name: "r1.rise_bp"}},
		"negative":  {{Name: "r1.windows", Set: true, Value: -1}},
	} {
		if _, err := verdict.ThresholdsOf(keys); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
