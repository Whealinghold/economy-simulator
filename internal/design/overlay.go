package design

import (
	"errors"
	"strconv"
	"strings"
)

// The path syntax of spec-experiments.md 3.2.1, rule constants.
const (
	minSegments = 2
	maxSegments = 8
	maxSegBytes = 64
)

// ErrPath is returned by Overlay for a path that ValidPath refuses.
var ErrPath = errors.New("not a valid configuration path")

// ValidPath reports whether path has the syntax of spec-experiments.md 3.2.1: two
// to eight dot-separated segments, each of [a-z][a-z0-9_]* and at most 64 bytes.
func ValidPath(path string) bool {
	segs := strings.Split(path, ".")
	if len(segs) < minSegments || len(segs) > maxSegments {
		return false
	}
	for _, s := range segs {
		if !validSegment(s) {
			return false
		}
	}
	return true
}

func validSegment(s string) bool {
	if s == "" || len(s) > maxSegBytes || !isLower(s[0]) {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isNameByte(s[i]) {
			return false
		}
	}
	return true
}

func isLower(c byte) bool { return c >= 'a' && c <= 'z' }

func isNameByte(c byte) bool { return isLower(c) || (c >= '0' && c <= '9') || c == '_' }

// Overlay returns the document of one change: the table header of all but the
// last segment of path, and the key set to value in decimal, no sign for a
// non-negative value and "-" for a negative one, with LF line ends
// (spec-experiments.md 3.3.4):
//
//	[world]
//	cost_floor = 8000
//
// The bytes are the same on every platform (FR-EXP-003). It returns ErrPath for
// a path that is not valid.
func Overlay(path string, value int64) ([]byte, error) {
	if !ValidPath(path) {
		return nil, ErrPath
	}
	i := strings.LastIndexByte(path, '.')
	var b strings.Builder
	b.WriteString("[" + path[:i] + "]\n" + path[i+1:] + " = ")
	b.WriteString(strconv.FormatInt(value, 10))
	b.WriteByte('\n')
	return []byte(b.String()), nil
}
