package economysim

import (
	"cmp"
	"fmt"
	"slices"
)

// This file is the integration calls of spec-integration.md: what a consumer
// does to a world between two steps. It is a core concept file with no module,
// no Config field and no default (plan 2-17 D540).

// enter is checks 1 to 3 of spec-integration.md 3.2 for a call that changes the
// world, in the order guard has: a running step, a failed world, then the module
// the call needs (module names it and on says whether it is on; an empty module
// needs none). The error carries the call's name (architecture.md 8.2).
func (w *World) enter(call, module string, on bool) error {
	if err := w.guard(); err != nil {
		return fmt.Errorf("%s: %w", call, err)
	}
	if module != "" && !on {
		return fmt.Errorf("%s: module %s: %w", call, module, ErrModuleOff)
	}
	return nil
}

// argError is an error of check 4 of spec-integration.md 3.2: a *ValidationError
// whose path names the argument, one segment each (entry names the slice entry
// as "changes[2]"), with the sentinel as its rule.
func argError(rule error, value, bound string, segs ...string) error {
	return &ValidationError{Path: segs, Rule: rule, Value: value, Bound: bound}
}

// entry names the i-th entry of the slice argument name, as the segment of a
// path: "changes[2]".
func entry(name string, i int) string { return name + "[" + itoa(i) + "]" }

// keyedOnce returns the index of the first key that appears an earlier time in
// keys, or -1 if every key is distinct: the check that a key appears once in a
// call (spec-integration.md 3.3.2). It sorts a copy, so the caller's slice is
// untouched, and it uses no map (architecture.md 7.5).
func keyedOnce[K ~int | ~int32 | ~int64](keys []K) int {
	idx := make([]int, len(keys))
	for i := range idx {
		idx[i] = i
	}
	slices.SortStableFunc(idx, func(a, b int) int { return cmp.Compare(keys[a], keys[b]) })
	dup := -1
	for j := 1; j < len(idx); j++ {
		if keys[idx[j]] == keys[idx[j-1]] && (dup < 0 || idx[j] < dup) {
			dup = idx[j]
		}
	}
	return dup
}
