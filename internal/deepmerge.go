package internal

import (
	"github.com/lyraproj/pcore/types"

	"github.com/lyraproj/pcore/px"
)

// DeepMerge will merge the values 'a' and 'b' if both values are hashes or both values are
// arrays. When this is not the case, no merge takes place and the 'a' argument is returned.
// The second bool return value true if a merge took place and false when the first argument
// is returned.
//
// When both values are hashes, DeepMerge is called recursively entries with identical keys.
// When both values are arrays, the merge creates a union of the unique elements from the two arrays.
// No recursive merge takes place for the array elements.
func DeepMerge(a, b px.Value, options map[string]px.Value) (px.Value, bool) {
	switch a := a.(type) {
	case *types.Hash:
		if hb, ok := b.(*types.Hash); ok {
			es := make([]*types.HashEntry, 0, a.Len()+hb.Len())
			mergeHappened := false
			a.Each(func(ev px.Value) {
				e := ev.(*types.HashEntry)
				if bv, ok := hb.Get(e.Key()); ok {
					if m, mh := DeepMerge(e.Value(), bv, options); mh {
						es = append(es, types.WrapHashEntry(e.Key(), m))
						mergeHappened = true
						return
					}
				}
				es = append(es, e)
			})
			hb.Each(func(ev px.Value) {
				e := ev.(*types.HashEntry)
				if !a.IncludesKey(e.Key()) {
					mergeHappened = true
					es = append(es, e)
				}
			})
			if mergeHappened {
				return types.WrapHash(es), true
			}
		}

	case *types.Array:
		if ab, ok := b.(*types.Array); ok && ab.Len() > 0 {
			if a.Len() == 0 {
				return ab, true
			}
			es := a.AppendTo(make([]px.Value, 0, a.Len()+ab.Len()))
			mergeHappened := false
			ab.Each(func(e px.Value) {
				if !a.Any(func(v px.Value) bool { return v.Equals(e, nil) }) {
					es = append(es, e)
					mergeHappened = true
				}
			})
			if mergeHappened {
				return types.WrapValues(es), true
			}
		}
	}
	return a, false
}
