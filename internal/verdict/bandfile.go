package verdict

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// BandKey is one table of a band file (testdata/chains/bands/*.toml). A key
// with Set false has no value line: it is unset, and a value of 0 is a value
// (STY 6.5.1, FR-STY-070).
type BandKey struct {
	Name  string
	Unit  string
	Value int64
	Set   bool
	// SetBy is the run that set the value: commit, configuration hash and seeds.
	SetBy string
	// Expects names the regimes a scenario expects (STY 6.4), lower case.
	Expects []string
	// Line is the line of the table header, for a message.
	Line int
}

// ParseError is a malformed line of a band file.
type ParseError struct {
	Line int
	Msg  string
}

// Error names the line.
func (e *ParseError) Error() string {
	return "verdict: band file line " + strconv.Itoa(e.Line) + ": " + e.Msg
}

// The setting seed sets of STY 12 "seeds": 1 to 50, and 1 to 10 for a key of
// tier L.
const (
	settingSeedsG = "1-50"
	settingSeedsL = "1-10"
)

// tierLong reports whether the key is read by a fact of tier L (STY 4.5): the
// population facts W02 and W06 (D589).
func tierLong(name string) bool {
	return strings.HasPrefix(name, "w02.") || strings.HasPrefix(name, "w06.")
}

// LoadBands parses the plain grammar of bands/ (STY 6.5.1, REF 7.1.5): tables
// `[a.b]` (or `[name]` for a shared key), the lines `unit`, `value`, `set_by`
// and `expects`, and `#` comment lines. It returns the keys in file order. A
// `value` needs a `set_by` of the form
// `commit=<id>;config=<hash>;seeds=<a>-<b>` whose seeds are exactly 1-50 (1-10
// for a tier L key); a malformed line is an error with its number.
func LoadBands(src string) ([]BandKey, error) {
	p := bandParser{seen: map[string]bool{}}
	for i, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		var err error
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
		case strings.HasPrefix(line, "["):
			err = p.header(i+1, line)
		default:
			err = p.field(i+1, line)
		}
		if err != nil {
			return nil, err
		}
	}
	if err := p.finish(); err != nil {
		return nil, err
	}
	return p.keys, nil
}

// bandParser is the state of LoadBands: the keys so far, the tables seen and
// the fields of the open table.
type bandParser struct {
	keys   []BandKey
	seen   map[string]bool
	fields map[string]bool
}

// finish checks the shape of the open table, if any.
func (p *bandParser) finish() error {
	if len(p.keys) == 0 {
		return nil
	}
	return checkKey(&p.keys[len(p.keys)-1])
}

func (p *bandParser) header(n int, line string) error {
	if err := p.finish(); err != nil {
		return err
	}
	name, ok := tableName(line)
	if !ok {
		return &ParseError{n, fmt.Sprintf("%q is not a table header", line)}
	}
	if p.seen[name] {
		return &ParseError{n, "table [" + name + "] twice"}
	}
	p.seen[name] = true
	p.keys = append(p.keys, BandKey{Name: name, Line: n})
	p.fields = map[string]bool{}
	return nil
}

func (p *bandParser) field(n int, line string) error {
	if len(p.keys) == 0 {
		return &ParseError{n, "a line before the first table"}
	}
	k, v, ok := strings.Cut(line, "=")
	k, v = strings.TrimSpace(k), strings.TrimSpace(v)
	if !ok {
		return &ParseError{n, fmt.Sprintf("%q is not key = value", line)}
	}
	if p.fields[k] {
		return &ParseError{n, "key " + k + " twice"}
	}
	p.fields[k] = true
	if err := setField(&p.keys[len(p.keys)-1], k, v); err != nil {
		return &ParseError{n, err.Error()}
	}
	return nil
}

// checkKey enforces the shape of a finished table: a value has a set_by of the
// right shape, and a set_by needs a value.
func checkKey(k *BandKey) error {
	switch {
	case k.Set && k.SetBy == "":
		return &ParseError{k.Line, "[" + k.Name + "] has a value and no set_by"}
	case !k.Set && k.SetBy != "":
		return &ParseError{k.Line, "[" + k.Name + "] has a set_by and no value"}
	case !k.Set:
		return nil
	}
	seeds, err := setByShape(k.SetBy)
	if err != nil {
		return &ParseError{k.Line, "[" + k.Name + "] set_by: " + err.Error()}
	}
	want := settingSeedsG
	if tierLong(k.Name) {
		want = settingSeedsL
	}
	if seeds != want {
		return &ParseError{k.Line, "[" + k.Name + "] set_by seeds " + seeds + ", want " + want}
	}
	return nil
}

// setByShape checks `commit=<id>;config=<hash>;seeds=<a>-<b>` and returns the
// seed set. An id and a hash are non-empty words of letters and digits.
func setByShape(s string) (string, error) {
	m := setByRe.FindStringSubmatch(s)
	if m == nil {
		return "", fmt.Errorf("%q is not commit=<id>;config=<hash>;seeds=<a>-<b>", s)
	}
	return m[3], nil
}

var (
	setByRe = regexp.MustCompile(`^commit=([A-Za-z0-9]+);config=([A-Za-z0-9]+);seeds=([0-9]{1,9}-[0-9]{1,9})$`)
	// tableRe is a table name: lower-case words of letters, digits and
	// underscores with at most one dot.
	tableRe = regexp.MustCompile(`^\[([a-z0-9_]+(?:\.[a-z0-9_]+)?)\]$`)
)

func tableName(line string) (string, bool) {
	m := tableRe.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	return m[1], true
}

func setField(k *BandKey, field, v string) error {
	switch field {
	case "unit":
		return setString(&k.Unit, field, v, false)
	case "set_by":
		return setString(&k.SetBy, field, v, true)
	case "value":
		x, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("value %q is not an integer", v)
		}
		k.Value, k.Set = x, true
		return nil
	case "expects":
		return setExpects(k, v)
	}
	return fmt.Errorf("unknown key %q", field)
}

// setString reads a quoted string into dst; nonEmpty refuses "".
func setString(dst *string, field, v string, nonEmpty bool) error {
	s, ok := quoted(v)
	if !ok || (nonEmpty && s == "") {
		return fmt.Errorf("%s %q is not a string", field, v)
	}
	*dst = s
	return nil
}

// setExpects reads `["r3", "r4"]`.
func setExpects(k *BandKey, v string) error {
	if !strings.HasPrefix(v, "[") || !strings.HasSuffix(v, "]") {
		return fmt.Errorf("expects %q is not a list", v)
	}
	inner := strings.TrimSpace(v[1 : len(v)-1])
	if inner == "" {
		return nil
	}
	for _, item := range strings.Split(inner, ",") {
		s, ok := quoted(strings.TrimSpace(item))
		if !ok || s == "" {
			return fmt.Errorf("expects item %q is not a string", strings.TrimSpace(item))
		}
		k.Expects = append(k.Expects, s)
	}
	return nil
}

// quoted reads a double-quoted string with no escape.
func quoted(v string) (string, bool) {
	if len(v) < 2 || v[0] != '"' || v[len(v)-1] != '"' || strings.ContainsAny(v[1:len(v)-1], "\"\\") {
		return "", false
	}
	return v[1 : len(v)-1], true
}

// ThresholdsOf fills a Thresholds from the keys of regimes.toml. A table that
// is not a key of STY 6.2 is an error: a pack may not invent a threshold
// (6.5.1).
func ThresholdsOf(keys []BandKey) (Thresholds, error) {
	t := NewThresholds()
	for _, k := range keys {
		i := -1
		for j := range t {
			if t[j].Name == k.Name {
				i = j
			}
		}
		if i < 0 {
			return nil, fmt.Errorf("verdict: [%s] is not a regime threshold", k.Name)
		}
		if k.Set && k.Name == kR3Factor && k.Value < 0 {
			return nil, fmt.Errorf("verdict: [%s] value %d is negative: the horizon could never be reached", k.Name, k.Value)
		}
		if k.Set {
			t[i].Value, t[i].Set = k.Value, true
		}
	}
	return t, nil
}
