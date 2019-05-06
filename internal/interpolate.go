package internal

import (
	"regexp"
	"strings"

	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
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
func Interpolate(ic hieraapi.Invocation, value px.Value, allowMethods bool) px.Value {
	result, _ := doInterpolate(ic, value, allowMethods)
	return result
}

func doInterpolate(ic hieraapi.Invocation, value px.Value, allowMethods bool) (px.Value, bool) {
	if s, ok := value.(px.StringValue); ok {
		return interpolateString(ic, s.String(), allowMethods)
	}
	if a, ok := value.(*types.Array); ok {
		cp := a.AppendTo(make([]px.Value, 0, a.Len()))
		changed := false
		for i, e := range cp {
			v, c := doInterpolate(ic, e, allowMethods)
			if c {
				changed = true
				cp[i] = v
			}
		}
		if changed {
			a = types.WrapValues(cp)
		}
		return a, changed
	}
	if h, ok := value.(*types.Hash); ok {
		cp := h.AppendEntriesTo(make([]*types.HashEntry, 0, h.Len()))
		changed := false
		for i, e := range cp {
			k, kc := doInterpolate(ic, e.Key(), allowMethods)
			v, vc := doInterpolate(ic, e.Value(), allowMethods)
			if kc || vc {
				changed = true
				cp[i] = types.WrapHashEntry(k, v)
			}
		}
		if changed {
			h = types.WrapHash(cp)
		}
		return h, changed
	}
	return value, false
}

const scopeMethod = 1
const aliasMethod = 2
const lookupMethod = 3
const literalMethod = 4

var methodMatch = regexp.MustCompile(`^(\w+)\((?:["]([^"]+)["]|[']([^']+)['])\)$`)

func getMethodAndData(expr string, allowMethods bool) (int, string) {
	if groups := methodMatch.FindStringSubmatch(expr); groups != nil {
		if !allowMethods {
			panic(px.Error(hieraapi.InterpolationMethodSyntaxNotAllowed, issue.NoArgs))
		}
		data := groups[2]
		if data == `` {
			data = groups[3]
		}
		switch groups[1] {
		case `alias`:
			return aliasMethod, data
		case `hiera`, `lookup`:
			return lookupMethod, data
		case `literal`:
			return literalMethod, data
		case `scope`:
			return scopeMethod, data
		default:
			panic(px.Error(hieraapi.UnknownInterpolationMethod, issue.H{`name`: groups[1]}))
		}
	}
	return scopeMethod, expr
}

func interpolateString(ic hieraapi.Invocation, str string, allowMethods bool) (result px.Value, changed bool) {
	changed = false
	if !strings.Contains(str, `%{`) {
		result = types.WrapString(str)
		return
	}
	str = iplPattern.ReplaceAllStringFunc(str, func(match string) string {
		expr := strings.TrimSpace(match[2 : len(match)-1])
		if emptyInterpolations[expr] {
			return ``
		}
		var methodKey int
		methodKey, expr = getMethodAndData(expr, allowMethods)
		if methodKey == aliasMethod && match != str {
			panic(px.Error(hieraapi.InterpolationAliasNotEntireString, issue.NoArgs))
		}

		switch methodKey {
		case literalMethod:
			return expr
		case scopeMethod:
			key := newKey(expr)
			if val, ok := ic.Scope().Get(types.WrapString(key.Root())); ok {
				val, _ = doInterpolate(ic, val, allowMethods)
				val = key.Dig(val)
				if val == nil {
					return ``
				}
				return val.String()
			}
			return ``
		default:
			val := Lookup(ic, expr, px.Undef, nil)
			if methodKey == aliasMethod {
				result = val
				return ``
			}
			return val.String()
		}
	})
	changed = true
	if result == nil {
		result = types.WrapString(str)
	}
	return

}

func resolveInScope(ic hieraapi.Invocation, expr string, allowMethods bool) px.Value {
	key := newKey(expr)
	if val, ok := ic.Scope().Get(types.WrapString(key.Root())); ok {
		val, _ = doInterpolate(ic, val, allowMethods)
		return key.Dig(val)
	}
	return nil
}
