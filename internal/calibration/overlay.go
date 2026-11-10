package calibration

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/Whealinghold/economy-sim/internal/design"
)

// ErrOverlay is returned for an overlay that cannot be written: a key whose class
// is not K or D (FR-CLB-050), a path internal/design refuses, the same path twice,
// or a header that does not name a stage, an attempt and a lineage.
const ErrOverlay = calError("calibration: invalid overlay")

// lineageShort is the digits of the lineage the header comment names (19.6).
const lineageShort = 8

// Header names the record an overlay came from, for the comment on its first line
// (spec-calibration.md 9.1, FR-CLB-051).
type Header struct {
	Stage   string // C1 to C9 with its sub-stage
	Attempt int
	Lineage string // 16 hex digits; the comment names the first 8
	Record  string // the path of the record, as the comment prints it
}

// Change is one key an overlay writes: the registry key, whose class decides
// whether it may be written, the configuration path of EXP 3.2.1 (the caller
// resolves the instance of a table keyed by an ID, and answers for the pair: the
// path is the key's, as Known.Paths says) and the value.
type Change struct {
	Key   string
	Path  string
	Value int64
}

// Overlay renders the calibration overlay of a choice: the header comment
// `# calibration <stage> attempt <n> of lineage <8 hex>: see <record path>`, then
// one table header and the key lines of that table (a blank line between tables),
// tables and keys in byte order, each key line byte for byte as internal/design writes it (EXP 3.3.4,
// FR-CLB-050, FR-CLB-051). class returns the registry class of a key ('K', 'D',
// 'S', 'P', or 0 for none); a key of any class but K and D is refused. No change
// gives the comment alone: the empty overlay of a base that was feasible.
func Overlay(h Header, changes []Change, class func(key string) byte) ([]byte, error) {
	if err := checkHeader(h); err != nil {
		return nil, err
	}
	tables := map[string][]string{}
	seen := map[string]bool{}
	for _, ch := range changes {
		if c := class(ch.Key); c != 'K' && c != 'D' {
			if c == 0 {
				return nil, fmt.Errorf("%w: %s is not a registry key", ErrOverlay, ch.Key)
			}
			return nil, fmt.Errorf("%w: %s is class %q, not K or D", ErrOverlay, ch.Key, string(rune(c)))
		}
		if seen[ch.Path] {
			return nil, fmt.Errorf("%w: %s is written twice", ErrOverlay, ch.Path)
		}
		seen[ch.Path] = true
		doc, err := design.Overlay(ch.Path, ch.Value)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrOverlay, ch.Path, err)
		}
		table, keyLine, _ := strings.Cut(strings.TrimSuffix(string(doc), "\n"), "\n")
		tables[table] = append(tables[table], keyLine)
	}
	var out bytes.Buffer
	fmt.Fprintf(&out, "# calibration %s attempt %d of lineage %s: see %s\n", h.Stage, h.Attempt, h.Lineage[:lineageShort], h.Record)
	names := make([]string, 0, len(tables))
	for t := range tables {
		names = append(names, t)
	}
	slices.Sort(names)
	for i, t := range names {
		slices.Sort(tables[t])
		if i > 0 {
			out.WriteByte('\n') // a blank line between tables
		}
		out.WriteString(t + "\n" + strings.Join(tables[t], "\n") + "\n")
	}
	return out.Bytes(), nil
}

// checkHeader refuses a header that names no stage, no attempt, no lineage or a
// record path that is empty or has a line break.
func checkHeader(h Header) error {
	var w writer
	w.stage(h.Stage)
	w.hex("lineage", h.Lineage)
	if h.Attempt < 1 || h.Record == "" || strings.ContainsAny(h.Record, "\r\n") {
		w.fail("attempt %d or record path %q", h.Attempt, h.Record)
	}
	if w.err != nil {
		return fmt.Errorf("%w: %w", ErrOverlay, w.err)
	}
	return nil
}
