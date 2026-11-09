package transit

import "slices"

// Free is a sorted set of the IDs of open Transit accounts that no shipment in
// flight uses (spec-transport.md 7.1). The zero value is empty and ready.
type Free struct {
	ids []int
}

// Take removes and returns the lowest ID, or ok false when the set is empty.
func (f *Free) Take() (id int, ok bool) {
	if len(f.ids) == 0 {
		return 0, false
	}
	id = f.ids[0]
	f.ids = slices.Delete(f.ids, 0, 1) // keeps the front of the storage, so a take and put cycle does not reallocate
	return id, true
}

// Put adds id in its place. It returns false, changing nothing, when id is
// already in the set.
func (f *Free) Put(id int) bool {
	i, found := slices.BinarySearch(f.ids, id)
	if found {
		return false
	}
	f.ids = slices.Insert(f.ids, i, id)
	return true
}

// Len returns the number of IDs in the set.
func (f *Free) Len() int { return len(f.ids) }

// Rebuild replaces the set by the IDs given, in any order; a repeated ID is kept
// once. It reuses the storage of the set.
func (f *Free) Rebuild(ids []int) {
	f.ids = append(f.ids[:0], ids...)
	slices.Sort(f.ids)
	f.ids = slices.Compact(f.ids)
}
