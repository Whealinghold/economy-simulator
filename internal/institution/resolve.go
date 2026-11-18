package institution

// Scope is the key of a row: a jurisdiction (Jurisdiction true) or a single place
// (Jurisdiction false), by index.
type Scope struct {
	Jurisdiction bool
	Key          int32
}

// Row is one row of a table keyed by (subject, scope key). The table holds at most
// one row for each key (the validator rejects a second, spec-institutions.md 10.1),
// so the first row found for a key is the row. Suspended is a row that a consumer
// has set aside between ticks.
type Row struct {
	Subject   int32
	Scope     Scope
	Suspended bool
}

// Resolve returns the index in rows of the row in force for a subject at a place
// with the given chain of jurisdictions, innermost first (spec-institutions.md
// 5.4 and 6.1.3; the innermost mode of spec-polity.md 6.1.2): the row keyed by the
// place itself, else the row of the first jurisdiction of the chain that has one,
// else none. A row that is suspended leaves its key open: the result is none and no
// outer row is consulted (FR-ORG-015). ok is false for none.
func Resolve(rows []Row, subject, at int32, chain []int32) (ref int, ok bool) {
	if i := find(rows, subject, Scope{Key: at}); i >= 0 {
		return i, !rows[i].Suspended
	}
	for _, j := range chain {
		if i := find(rows, subject, Scope{Jurisdiction: true, Key: j}); i >= 0 {
			return i, !rows[i].Suspended
		}
	}
	return 0, false
}

// find returns the index of the first row with the subject and the scope, or -1.
func find(rows []Row, subject int32, sc Scope) int {
	for i := range rows {
		if rows[i].Subject == subject && rows[i].Scope == sc {
			return i
		}
	}
	return -1
}

// Overlap returns the first pair of lists that share an item and a place, in the
// order of the lowest first list, then the lowest second, then the lowest shared
// item, then the lowest shared place (FR-ORG-004, spec-institutions.md 3.3.2).
// items[i] and places[i] are the sorted, ascending items and places of list i; the
// result is the two list indexes with a < b and the item and the place they share.
// ok is false when no two lists overlap.
func Overlap(items, places [][]int32) (a, b int, item, at int32, ok bool) {
	for a = range items {
		for b = a + 1; b < len(items); b++ {
			if item, ok = firstShared(items[a], items[b]); !ok {
				continue
			}
			if at, ok = firstShared(places[a], places[b]); ok {
				return a, b, item, at, true
			}
		}
	}
	return 0, 0, 0, 0, false
}

// firstShared returns the lowest value that two ascending slices both hold.
func firstShared(x, y []int32) (int32, bool) {
	i, j := 0, 0
	for i < len(x) && j < len(y) {
		switch {
		case x[i] < y[j]:
			i++
		case x[i] > y[j]:
			j++
		default:
			return x[i], true
		}
	}
	return 0, false
}
