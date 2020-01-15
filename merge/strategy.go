package merge

import (
	"fmt"
	"reflect"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/api"
)

type (
	deepMerge struct{ opts dgo.Map }

	hashMerge struct{}

	firstFound struct{}

	unique struct{}
)

// GetStrategy returns the merge.MergeStrategy that corresponds to the given name. The
// options argument is only applicable to deep merge
func GetStrategy(n string, opts dgo.Map) api.MergeStrategy {
	switch n {
	case `first`:
		return &firstFound{}
	case `unique`:
		return &unique{}
	case `hash`:
		return &hashMerge{}
	case `deep`:
		if opts == nil {
			opts = vf.Map()
		}
		return &deepMerge{opts}
	default:
		panic(fmt.Errorf(`unknown merge strategy '%s'`, n))
	}
}

type merger interface {
	api.MergeStrategy

	merge(a, b dgo.Value) dgo.Value

	mergeSingle(v reflect.Value, vf func(l interface{}) dgo.Value) dgo.Value

	convertValue(v dgo.Value) dgo.Value
}

func doLookup(s merger, vs interface{}, ic api.Invocation, vf func(l interface{}) dgo.Value) dgo.Value {
	vsr := reflect.ValueOf(vs)
	if vsr.Kind() != reflect.Slice {
		return nil
	}
	top := vsr.Len()
	switch top {
	case 0:
		return nil
	case 1:
		return s.mergeSingle(vsr.Index(0), vf)
	default:
		return ic.WithMerge(s, func() dgo.Value {
			var memo dgo.Value
			for idx := 0; idx < top; idx++ {
				v := variantLookup(vsr.Index(idx), vf)
				if v != nil {
					if memo == nil {
						memo = s.convertValue(v)
					} else {
						memo = s.merge(memo, v)
					}
				}
			}
			if memo != nil {
				ic.ReportMergeResult(memo)
			}
			return memo
		})
	}
}

func variantLookup(v reflect.Value, vf func(l interface{}) dgo.Value) dgo.Value {
	if v.CanInterface() {
		return vf(v.Interface())
	}
	return nil
}

func (d *firstFound) Name() string {
	return `first`
}

func (d *firstFound) Label() string {
	return `first found strategy`
}

func (d *firstFound) MergeLookup(vs interface{}, ic api.Invocation, f func(location interface{}) dgo.Value) dgo.Value {
	vsr := reflect.ValueOf(vs)
	if vsr.Kind() != reflect.Slice {
		return nil
	}
	top := vsr.Len()
	switch top {
	case 0:
		return nil
	case 1:
		return variantLookup(vsr.Index(0), f)
	default:
		var v dgo.Value
		for idx := 0; idx < top; idx++ {
			v = variantLookup(vsr.Index(idx), f)
			if v != nil {
				break
			}
		}
		if v != nil {
			ic.ReportMergeResult(v)
		}
		return v
	}
}

func (d *firstFound) Options() dgo.Map {
	return vf.Map()
}

func (d *firstFound) mergeSingle(v reflect.Value, vf func(l interface{}) dgo.Value) dgo.Value {
	return variantLookup(v, vf)
}

func (d *firstFound) convertValue(v dgo.Value) dgo.Value {
	return v
}

func (d *firstFound) merge(a, b dgo.Value) dgo.Value {
	return a
}

func (d *unique) Name() string {
	return `unique`
}

func (d *unique) Label() string {
	return `unique merge strategy`
}

func (d *unique) MergeLookup(vs interface{}, ic api.Invocation, f func(location interface{}) dgo.Value) dgo.Value {
	return doLookup(d, vs, ic, f)
}

func (d *unique) Options() dgo.Map {
	return vf.Map()
}

func (d *unique) mergeSingle(rv reflect.Value, vf func(l interface{}) dgo.Value) dgo.Value {
	v := variantLookup(rv, vf)
	if av, ok := v.(dgo.Array); ok {
		return av.Flatten().Unique()
	}
	return v
}

func (d *unique) convertValue(v dgo.Value) dgo.Value {
	if av, ok := v.(dgo.Array); ok {
		return av.Flatten()
	}
	return vf.Values(v)
}

func (d *unique) merge(a, b dgo.Value) dgo.Value {
	return d.convertValue(a).(dgo.Array).WithAll(d.convertValue(b).(dgo.Array)).Unique()
}

func (d *deepMerge) Name() string {
	return `deep`
}

func (d *deepMerge) Label() string {
	return `deep merge strategy`
}

func (d *deepMerge) MergeLookup(vs interface{}, ic api.Invocation, f func(location interface{}) dgo.Value) dgo.Value {
	return doLookup(d, vs, ic, f)
}

func (d *deepMerge) Options() dgo.Map {
	return d.opts
}

func (d *deepMerge) mergeSingle(v reflect.Value, vf func(l interface{}) dgo.Value) dgo.Value {
	return variantLookup(v, vf)
}

func (d *deepMerge) convertValue(v dgo.Value) dgo.Value {
	return v
}

func (d *deepMerge) merge(a, b dgo.Value) dgo.Value {
	v, _ := Deep(a, b, d.opts)
	return v
}

func (d *hashMerge) Name() string {
	return `hash`
}

func (d *hashMerge) Label() string {
	return `hash merge strategy`
}

func (d *hashMerge) MergeLookup(vs interface{}, ic api.Invocation, f func(location interface{}) dgo.Value) dgo.Value {
	return doLookup(d, vs, ic, f)
}

func (d *hashMerge) Options() dgo.Map {
	return vf.Map()
}

func (d *hashMerge) mergeSingle(v reflect.Value, vf func(l interface{}) dgo.Value) dgo.Value {
	return variantLookup(v, vf)
}

func (d *hashMerge) convertValue(v dgo.Value) dgo.Value {
	return v
}

func (d *hashMerge) merge(a, b dgo.Value) dgo.Value {
	if ah, ok := a.(dgo.Map); ok {
		var bh dgo.Map
		if bh, ok = b.(dgo.Map); ok {
			return bh.Merge(ah)
		}
	}
	return a
}
