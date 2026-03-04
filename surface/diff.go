package surface

import "sort"

// DiffResult contains the differences between two surface snapshots.
type DiffResult struct {
	Added   []Entry // Entries in new but not in old
	Removed []Entry // Entries in old but not in new (breaking changes)
}

// HasBreakingChanges returns true if any entries were removed.
func (d DiffResult) HasBreakingChanges() bool {
	return len(d.Removed) > 0
}

// Diff compares two snapshots and returns additions and removals.
func Diff(old, new []Entry) DiffResult {
	oldSet := make(map[string]Entry, len(old))
	for _, e := range old {
		oldSet[e.String()] = e
	}

	newSet := make(map[string]Entry, len(new))
	for _, e := range new {
		newSet[e.String()] = e
	}

	var result DiffResult

	// Find removals (in old but not in new)
	for key, e := range oldSet {
		if _, ok := newSet[key]; !ok {
			result.Removed = append(result.Removed, e)
		}
	}

	// Find additions (in new but not in old)
	for key, e := range newSet {
		if _, ok := oldSet[key]; !ok {
			result.Added = append(result.Added, e)
		}
	}

	sort.Slice(result.Added, func(i, j int) bool { return result.Added[i].String() < result.Added[j].String() })
	sort.Slice(result.Removed, func(i, j int) bool { return result.Removed[i].String() < result.Removed[j].String() })

	return result
}
