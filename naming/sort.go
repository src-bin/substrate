package naming

import "sort"

// Index returns the index where x can be found in canonical, which is _not_
// presumed to be sorted. This function is intended to be used to implement
// sort by environment or quality using the canonical order found in the
// substrate.environments and substrate.qualities files. See accounts.Sort
// for a more complex example.
func Index(canonical []string, x string) int {
	for i, s := range canonical {
		if s == x {
			return i
		}
	}
	return -1
}

// IndexedSort sorts a slice of strings in place according to the order
// prescribed in canonical (via Index).
func IndexedSort(slice, canonical []string) {
	sort.Slice(slice, func(i, j int) bool {
		return Index(canonical, slice[i]) < Index(canonical, slice[j])
	})
}
