package verdict

import (
	"fmt"
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
	parts := strings.Split(s, ";")
	if len(parts) != 3 {
		return "", fmt.Errorf("%q is not commit=<id>;config=<hash>;seeds=<a>-<b>", s)
	}
	for i, name := range []string{"commit", "config", "seeds"} {
		v, ok := strings.CutPrefix(parts[i], name+"=")
		if !ok || v == "" {
			return "", fmt.Errorf("%q: part %d is not %s=<value>", s, i+1, name)
		}
		if i < 2 && !isWord(v) {
			return "", fmt.Errorf("%s %q is not letters and digits", name, v)
		}
		if i == 2 {
			a, b, ok := strings.Cut(v, "-")
			if _, e1 := strconv.ParseUint(a, 10, 31); !ok || e1 != nil {
				return "", fmt.Errorf("seeds %q is not <a>-<b>", v)
			}
			if _, e2 := strconv.ParseUint(b, 10, 31); e2 != nil {
				return "", fmt.Errorf("seeds %q is not <a>-<b>", v)
			}
			return v, nil
		}
	}
	return "", nil
}

func isWord(s string) bool {
	for _, r := range s {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
			return false
		}
	}
	return s != ""
}

// tableName reads `[a.b]` or `[name]`: lower-case words of letters, digits and
// underscores, with at most one dot.
func tableName(line string) (string, bool) {
	if !strings.HasSuffix(line, "]") {
		return "", false
	}
	name := line[1 : len(line)-1]
	if strings.Count(name, ".") > 1 {
		return "", false
	}
	for _, part := range strings.Split(name, ".") {
		if part == "" {
			return "", false
		}
		for _, r := range part {
			if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r == '_') {
				return "", false
			}
		}
	}
	return name, true
}

func setField(k *BandKey, field, v string) error {
	switch field {
	case "unit":
		s, ok := quoted(v)
		if !ok {
			return fmt.Errorf("unit %q is not a string", v)
		}
		k.Unit = s
	case "value":
		x, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("value %q is not an integer", v)
		}
		k.Value, k.Set = x, true
	case "set_by":
		s, ok := quoted(v)
		if !ok || s == "" {
			return fmt.Errorf("set_by %q is not a non-empty string", v)
		}
		k.SetBy = s
	case "expects":
		if !strings.HasPrefix(v, "[") || !strings.HasSuffix(v, "]") {
			return fmt.Errorf("expects %q is not a list", v)
		}
		inner := strings.TrimSpace(v[1 : len(v)-1])
		for _, item := range strings.Split(inner, ",") {
			item = strings.TrimSpace(item)
			if inner == "" {
				break
			}
			s, ok := quoted(item)
			if !ok || s == "" {
				return fmt.Errorf("expects item %q is not a string", item)
			}
			k.Expects = append(k.Expects, s)
		}
	default:
		return fmt.Errorf("unknown key %q", field)
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
		if k.Set {
			t[i].Value, t[i].Set = k.Value, true
		}
	}
	return t, nil
}
