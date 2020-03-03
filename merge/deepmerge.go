package merge

import (
	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/api"
)

// Deep will merge the values 'a' and 'b' if both values are hashes or both values are
// arrays. When this is not the case, no merge takes place and the 'a' argument is returned.
// The second bool return value true if a merge took place and false when the first argument
// is returned.
//
// When both values are hashes, Deep is called recursively entries with identical keys.
// When both values are arrays, the merge creates a union of the unique elements from the two arrays.
// No recursive merge takes place for the array elements.
func Deep(a, b dgo.Value, opi interface{}) (dgo.Value, bool) {
	var options dgo.Map
	if opi != nil {
		options = api.ToMap(`deep merge options`, options)
	}
	return deep(a, b, options)
}

func deep(a, b dgo.Value, options dgo.Map) (dgo.Value, bool) {
	switch a := a.(type) {
	case dgo.Map:
		if hb, ok := b.(dgo.Map); ok {
			es := vf.MapWithCapacity(a.Len() + hb.Len())
			a.EachEntry(func(e dgo.MapEntry) {
				if bv := hb.Get(e.Key()); bv != nil {
					if m, mh := deep(e.Value(), bv, options); mh {
						es.Put(e.Key(), m)
						return
					}
				}
				es.Put(e.Key(), e.Value())
			})
			hb.EachEntry(func(e dgo.MapEntry) {
				if !a.ContainsKey(e.Key()) {
					es.Put(e.Key(), e.Value())
				}
			})
			if !a.Equals(es) {
				return es, true
			}
		}

	case dgo.Array:
		if ab, ok := b.(dgo.Array); ok && ab.Len() > 0 {
			if a.Len() == 0 {
				return ab, true
			}
			an := a.WithAll(ab).Unique()
			if !an.Equals(a) {
				return an, true
			}
		}
	}
	return a, false
}
