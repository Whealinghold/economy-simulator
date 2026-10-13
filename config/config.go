// Package config loads a configuration from TOML documents into the plain
// structs of the root package (spec-config.md 6, ADR 0009).
//
// It is the only package of the module that imports a third-party parser. It
// never opens a path: the caller supplies bytes (Parse) or an fs.FS (LoadFS).
// The documents of a pack are merged in the order given (6.4), the merged tree
// is checked and mapped by hand to economysim.Config, each entity starting from
// its Default constructor (5.4), and every rejection is a root sentinel wrapped
// in a *economysim.ValidationError whose Path is the key path in the file
// (6.3). The loader does not check bounds or cross-references: that is
// Config.Validate (section 7), the same code whether the Config came from a file
// or from Go.
//
// Every byte is untrusted: the limits of 6.3.1 are checked before the work they
// bound, and the text of an error never carries more than a cut, quoted path
// segment of the input (6.3.2).
package config

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strconv"
	"unicode/utf8"

	toml "github.com/pelletier/go-toml/v2"

	economysim "github.com/Whealinghold/economy-sim"
)

// schemaVersion is the one version of the file format the loader reads.
const schemaVersion = 1

// maxReported is the most errors the loader returns before ErrTooManyErrors
// (6.3.3, the same cap as Validate).
const maxReported = 100

// Parse merges the documents in order, later over earlier, checks and maps the
// merged tree and returns the Config, or an error. It does not validate beyond
// spec-config.md 6.3.
//
// A limit (6.3.1) or a syntax error is returned alone. The rejections of the
// merge and the mapping are collected over the whole tree, sorted by path and
// then rule, at most 100, joined with errors.Join and followed by one
// ErrTooManyErrors that says how many were left out. Each is a
// *economysim.ValidationError whose Rule is the sentinel, so errors.Is and
// errors.As work on the result.
//
// On any error the Config is the zero value. The configuration it returns is not canonical (spec-config.md 3.4), but every
// entity list is in the ID order of its table, and the lists a file writes as
// arrays are in the order written. An empty list is nil.
func Parse(docs ...[]byte) (economysim.Config, error) {
	var zero economysim.Config
	if len(docs) > economysim.MaxDocuments {
		return zero, limitError("MaxDocuments", economysim.MaxDocuments, len(docs))
	}
	total := 0
	for i, d := range docs {
		if len(d) > economysim.MaxDocumentBytes {
			return zero, fmt.Errorf("document %d: %w", i+1, limitError("MaxDocumentBytes", economysim.MaxDocumentBytes, len(d)))
		}
		total += len(d)
		if total > economysim.MaxMergedBytes {
			return zero, limitError("MaxMergedBytes", economysim.MaxMergedBytes, total)
		}
	}
	for i, d := range docs {
		if err := guard(d); err != nil {
			return zero, fmt.Errorf("document %d: %w", i+1, err)
		}
	}
	merged := map[string]any{}
	var l loader
	for i, d := range docs {
		var tree map[string]any
		if err := toml.Unmarshal(d, &tree); err != nil {
			return zero, fmt.Errorf("document %d: %w", i+1, syntaxError(err))
		}
		l.merge(merged, tree, nil)
	}
	if v, ok := merged["schema"].(int64); !ok || v != schemaVersion {
		return zero, &economysim.ValidationError{Path: []string{"schema"}, Rule: economysim.ErrUnsupportedSchema}
	}
	cfg := l.config(merged)
	if err := l.err(); err != nil {
		return zero, err
	}
	return cfg, nil
}

// LoadFS reads the named files from fsys, in order, and calls Parse. It never
// opens a path itself. The count of names is checked before any file is opened,
// and each file is read through a reader bounded by MaxDocumentBytes and by
// what is left of MaxMergedBytes, whatever fs.FileInfo says of its size. A name
// that cannot be read is the fs error wrapped with %w and the name written as a
// path segment.
func LoadFS(fsys fs.FS, names ...string) (economysim.Config, error) {
	var zero economysim.Config
	if len(names) > economysim.MaxDocuments {
		return zero, limitError("MaxDocuments", economysim.MaxDocuments, len(names))
	}
	docs := make([][]byte, 0, len(names))
	budget := economysim.MaxMergedBytes
	for _, name := range names {
		f, err := fsys.Open(name)
		if err != nil {
			// A *fs.PathError repeats the name unquoted and uncut, so its
			// cause is wrapped instead; errors.Is still matches it.
			var pe *fs.PathError
			if errors.As(err, &pe) {
				err = pe.Err
			}
			return zero, fmt.Errorf("open %s: %w", segment(name), err)
		}
		limit := min(economysim.MaxDocumentBytes, budget)
		b, err := io.ReadAll(io.LimitReader(f, int64(limit)+1))
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return zero, fmt.Errorf("read %s: %w", segment(name), err)
		}
		if len(b) > economysim.MaxDocumentBytes {
			return zero, fmt.Errorf("%s: %w", segment(name), limitError("MaxDocumentBytes", economysim.MaxDocumentBytes, len(b)))
		}
		if len(b) > budget {
			return zero, fmt.Errorf("%s: %w", segment(name), limitError("MaxMergedBytes", economysim.MaxMergedBytes, economysim.MaxMergedBytes-budget+len(b)))
		}
		budget -= len(b)
		docs = append(docs, b)
	}
	return Parse(docs...)
}

// limitError is ErrTooLarge naming the constant, its value and the measured
// value (spec-config.md 6.3.1).
func limitError(name string, limit, got int) error {
	return fmt.Errorf("%w: %s is %d, got %d", economysim.ErrTooLarge, name, limit, got)
}

// syntaxError is ErrSyntax with the line and column of the library's error and
// none of its text, which can hold a whole key (spec-config.md 6.3.2).
func syntaxError(err error) error {
	var de *toml.DecodeError
	if errors.As(err, &de) {
		row, col := de.Position()
		return fmt.Errorf("%w at line %d, column %d", economysim.ErrSyntax, row, col)
	}
	// Not reached by any input so far: the library reports every syntax error as
	// a *toml.DecodeError, and FuzzLoad has found none that is not. It is kept so
	// that a future library error that is not one is still ErrSyntax, with no
	// text of the input.
	return economysim.ErrSyntax
}

// segment writes s cut at the last rune boundary at or before
// economysim.MaxIDLength bytes and quoted, followed by three dots outside the
// quotes when it was cut (spec-config.md 6.3.2).
func segment(s string) string {
	cut := len(s) > economysim.MaxIDLength
	if cut {
		n := economysim.MaxIDLength
		for n > 0 && !utf8.RuneStart(s[n]) {
			n--
		}
		s = s[:n]
	}
	q := strconv.Quote(s)
	if cut {
		q += "..."
	}
	return q
}

// loader collects the rejections of the merge and the mapping.
type loader struct {
	errs []*economysim.ValidationError
}

// fail records one rejection at the key path p.
func (l *loader) fail(rule error, p []string) {
	l.errs = append(l.errs, &economysim.ValidationError{Path: slices.Clone(p), Rule: rule})
}

// err returns nil, or the rejections sorted by path then rule, cut at
// maxReported and followed by one ErrTooManyErrors (spec-config.md 6.3.3).
func (l *loader) err() error {
	if len(l.errs) == 0 {
		return nil
	}
	slices.SortStableFunc(l.errs, func(a, b *economysim.ValidationError) int {
		if c := slices.Compare(a.Path, b.Path); c != 0 {
			return c
		}
		return cmp.Compare(a.Rule.Error(), b.Rule.Error())
	})
	shown := l.errs
	if len(shown) > maxReported {
		shown = shown[:maxReported]
	}
	out := make([]error, 0, len(shown)+1)
	for _, e := range shown {
		out = append(out, e)
	}
	if left := len(l.errs) - len(shown); left > 0 {
		out = append(out, fmt.Errorf("%w: %d more not shown", economysim.ErrTooManyErrors, left))
	}
	return errors.Join(out...)
}

// merge folds src into dst (spec-config.md 6.4): tables merge recursively by
// key, a scalar or a list replaces the earlier value, and a key that is a table
// in one and not a table in the other is ErrMergeConflict (the earlier value
// stays). The recursion is as deep as the tables of one document, at most
// MaxNestingDepth, which guard has checked.
func (l *loader) merge(dst, src map[string]any, path []string) {
	for k, sv := range src {
		dv, ok := dst[k]
		if !ok {
			dst[k] = sv
			continue
		}
		dm, dIsTable := dv.(map[string]any)
		sm, sIsTable := sv.(map[string]any)
		switch {
		case dIsTable && sIsTable:
			l.merge(dm, sm, append(path[:len(path):len(path)], k))
		case dIsTable != sIsTable:
			l.fail(economysim.ErrMergeConflict, append(path[:len(path):len(path)], k))
		default:
			dst[k] = sv
		}
	}
}
