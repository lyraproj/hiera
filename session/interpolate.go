package session

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/api"
)

var iplPattern = regexp.MustCompile(`%{[^}]*}`)
var emptyInterpolations = map[string]bool{
	``:     true,
	`::`:   true,
	`""`:   true,
	"''":   true,
	`"::"`: true,
	"'::'": true,
}

// Interpolate resolves interpolations in the given value and returns the result
func (ic *ivContext) Interpolate(value dgo.Value, allowMethods bool) dgo.Value {
	if result, changed := ic.doInterpolate(value, allowMethods); changed {
		return result
	}
	return value
}

func (ic *ivContext) doInterpolate(value dgo.Value, allowMethods bool) (dgo.Value, bool) {
	if s, ok := value.(dgo.String); ok {
		return ic.InterpolateString(s.String(), allowMethods)
	}
	if a, ok := value.(dgo.Array); ok {
		cp := a.AppendToSlice(make([]dgo.Value, 0, a.Len()))
		changed := false
		for i, e := range cp {
			v, c := ic.doInterpolate(e, allowMethods)
			if c {
				changed = true
				cp[i] = v
			}
		}
		if changed {
			a = vf.Array(cp)
		}
		return a, changed
	}
	if h, ok := value.(dgo.Map); ok {
		cp := vf.MapWithCapacity(h.Len())
		changed := false
		h.EachEntry(func(e dgo.MapEntry) {
			k, kc := ic.doInterpolate(e.Key(), allowMethods)
			v, vc := ic.doInterpolate(e.Value(), allowMethods)
			cp.Put(k, v)
			if kc || vc {
				changed = true
			}
		})
		if changed {
			cp.Freeze()
			h = cp
		}
		return h, changed
	}
	return value, false
}

type iplMethod int

const (
	scopeMethod = iplMethod(iota)
	aliasMethod
	strictAliasMethod
	lookupMethod
	literalMethod
)

func (m iplMethod) isAlias() bool {
	return m == aliasMethod || m == strictAliasMethod
}

var methodMatch = regexp.MustCompile(`^(\w+)\((?:["]([^"]+)["]|[']([^']+)['])\)$`)

func getMethodAndData(expr string, allowMethods bool) (iplMethod, string) {
	if groups := methodMatch.FindStringSubmatch(expr); groups != nil {
		if !allowMethods {
			panic(errors.New(`interpolation using method syntax is not allowed in this context`))
		}
		data := groups[2]
		if data == `` {
			data = groups[3]
		}
		switch groups[1] {
		case `alias`:
			return aliasMethod, data
		case `strict_alias`:
			return strictAliasMethod, data
		case `hiera`, `lookup`:
			return lookupMethod, data
		case `literal`:
			return literalMethod, data
		case `scope`:
			return scopeMethod, data
		default:
			panic(fmt.Errorf(`unknown interpolation method '%s'`, groups[1]))
		}
	}
	return scopeMethod, expr
}

// InterpolateString resolves a string containing interpolation expressions
func (ic *ivContext) InterpolateString(str string, allowMethods bool) (dgo.Value, bool) {
	if !strings.Contains(str, `%{`) {
		return vf.String(str), false
	}

	return ic.WithInterpolation(str, func() dgo.Value {
		var result dgo.Value
		var methodKey iplMethod
		str = iplPattern.ReplaceAllStringFunc(str, func(match string) string {
			expr := strings.TrimSpace(match[2 : len(match)-1])
			if emptyInterpolations[expr] {
				return ``
			}
			methodKey, expr = getMethodAndData(expr, allowMethods)
			if methodKey.isAlias() && match != str {
				panic(errors.New(`'alias'/'strict_alias' interpolation is only permitted if the expression is equal to the entire string`))
			}

			switch methodKey {
			case literalMethod:
				return expr
			case scopeMethod:
				if val := ic.InterpolateInScope(expr, allowMethods); val != nil {
					return val.String()
				}
				return ``
			default:
				val := ic.Lookup(api.NewKey(expr), nil)
				if methodKey.isAlias() {
					result = val
					return ``
				}
				if val == nil {
					return ``
				}
				return val.String()
			}
		})
		if result == nil && methodKey != strictAliasMethod {
			result = vf.String(str)
		}
		return result
	}), true
}

// InterpolateInScope resolves a key expression in the invocation scope
func (ic *ivContext) InterpolateInScope(expr string, allowMethods bool) dgo.Value {
	key := api.NewKey(expr)
	if val := ic.Scope().Get(key.Root()); val != nil {
		val, _ = ic.doInterpolate(val, allowMethods)
		return key.Dig(ic, val)
	}
	return nil
}
