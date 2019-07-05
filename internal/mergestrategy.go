package internal

import (
	"reflect"

	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/pcore/types"

	"github.com/lyraproj/issue/issue"

	"github.com/lyraproj/pcore/px"
)

func init() {
	hieraapi.GetMergeStrategy = getMergeStrategy
}

func getMergeStrategy(n hieraapi.MergeStrategyName, opts map[string]px.Value) hieraapi.MergeStrategy {
	switch n {
	case `first`:
		return &firstFound{}
	case `unique`:
		return &unique{}
	case `hash`:
		return &hashMerge{}
	case `deep`:
		return &deepMerge{opts}
	default:
		panic(px.Error(hieraapi.UnknownMergeStrategy, issue.H{`name`: n}))
	}
}

type merger interface {
	hieraapi.MergeStrategy

	merge(a, b px.Value) px.Value

	mergeSingle(v reflect.Value, vf func(l interface{}) px.Value) px.Value

	convertValue(v px.Value) px.Value
}

type deepMerge struct{ opts map[string]px.Value }

type hashMerge struct{}

type firstFound struct{}

type unique struct{}

func doLookup(s merger, vs interface{}, ic hieraapi.Invocation, vf func(l interface{}) px.Value) px.Value {
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
		return ic.WithMerge(s, func() px.Value {
			var memo px.Value
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

func variantLookup(v reflect.Value, vf func(l interface{}) px.Value) px.Value {
	if v.CanInterface() {
		return vf(v.Interface())
	}
	return nil
}

func (d *firstFound) Label() string {
	return `first found strategy`
}

func (d *firstFound) Lookup(vs interface{}, ic hieraapi.Invocation, f func(location interface{}) px.Value) px.Value {
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
		var v px.Value
		return ic.WithMerge(d, func() px.Value {
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
		})
	}
}

func (d *firstFound) Options() px.OrderedMap {
	return px.EmptyMap
}

func (d *firstFound) mergeSingle(v reflect.Value, vf func(l interface{}) px.Value) px.Value {
	return variantLookup(v, vf)
}

func (d *firstFound) convertValue(v px.Value) px.Value {
	return v
}

func (d *firstFound) merge(a, b px.Value) px.Value {
	return a
}

func (d *unique) Label() string {
	return `unique merge strategy`
}

func (d *unique) Lookup(vs interface{}, ic hieraapi.Invocation, f func(location interface{}) px.Value) px.Value {
	return doLookup(d, vs, ic, f)
}

func (d *unique) Options() px.OrderedMap {
	return px.EmptyMap
}

func (d *unique) mergeSingle(rv reflect.Value, vf func(l interface{}) px.Value) px.Value {
	v := variantLookup(rv, vf)
	if av, ok := v.(*types.Array); ok {
		return av.Flatten().Unique()
	}
	return v
}

func (d *unique) convertValue(v px.Value) px.Value {
	if av, ok := v.(*types.Array); ok {
		return av.Flatten()
	}
	return types.WrapValues([]px.Value{v})
}

func (d *unique) merge(a, b px.Value) px.Value {
	return d.convertValue(a).(px.List).AddAll(d.convertValue(b).(px.List)).Unique()
}

func (d *deepMerge) Label() string {
	return `deep merge strategy`
}

func (d *deepMerge) Lookup(vs interface{}, ic hieraapi.Invocation, f func(location interface{}) px.Value) px.Value {
	return doLookup(d, vs, ic, f)
}

func (d *deepMerge) Options() px.OrderedMap {
	if len(d.opts) > 0 {
		return types.WrapStringToValueMap(d.opts)
	}
	return px.EmptyMap
}

func (d *deepMerge) mergeSingle(v reflect.Value, vf func(l interface{}) px.Value) px.Value {
	return variantLookup(v, vf)
}

func (d *deepMerge) convertValue(v px.Value) px.Value {
	return v
}

func (d *deepMerge) merge(a, b px.Value) px.Value {
	v, _ := DeepMerge(a, b, d.opts)
	return v
}

func (d *hashMerge) Label() string {
	return `hash merge strategy`
}

func (d *hashMerge) Lookup(vs interface{}, ic hieraapi.Invocation, f func(location interface{}) px.Value) px.Value {
	return doLookup(d, vs, ic, f)
}

func (d *hashMerge) Options() px.OrderedMap {
	return px.EmptyMap
}

func (d *hashMerge) mergeSingle(v reflect.Value, vf func(l interface{}) px.Value) px.Value {
	return variantLookup(v, vf)
}

func (d *hashMerge) convertValue(v px.Value) px.Value {
	return v
}

func (d *hashMerge) merge(a, b px.Value) px.Value {
	if ah, ok := a.(*types.Hash); ok {
		var bh *types.Hash
		if bh, ok = b.(*types.Hash); ok {
			return bh.Merge(ah)
		}
	}
	return a
}
