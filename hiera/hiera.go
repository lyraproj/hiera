// Package hiera contains the Lookup functions to use when using Hiera as a library.
package hiera

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/streamer"
	"github.com/lyraproj/dgo/typ"
	"github.com/lyraproj/dgo/util"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/dgoyaml/yaml"
	"github.com/lyraproj/hiera/api"
	"github.com/lyraproj/hiera/explain"
	"github.com/lyraproj/hiera/session"
	"github.com/lyraproj/hierasdk/hiera"
)

// A CommandOptions contains the options given by to the CLI lookup command or a REST invocation.
type CommandOptions struct {
	// Type is a  Type string such as "string" or "[]int" used for assertion of the
	// found value.
	Type string

	// Merge is the name of a merge strategy
	Merge string

	// Default is a pointer to the string representation of a default value or nil if no default value exists
	Default *string

	// FactPaths are an optional paths to a files containing extra variables to add to the lookup scope
	// and as a copy under the lookup scope "facts" key.
	FactPaths []string

	// VarPaths are an optional paths to a files containing extra variables to add to the lookup scope
	VarPaths []string

	// Variables are an optional paths to a files containing extra variables to add to the lookup scope
	Variables []string

	// RenderAs is the name of the desired rendering
	RenderAs string

	// ExplainData should be set to true to explain the progress of a lookup
	ExplainData bool

	// ExplainOptions should be set to true to explain how lookup options were found for the lookup
	ExplainOptions bool

	LookupAll bool
}

// Lookup performs a lookup using the given parameters.
//
// ic - The lookup invocation
//
// name - The name to lookup
//
// defaultValue - Optional value to use as default when no value is found
//
// options - Optional map with merge strategy and options
func Lookup(ic api.Invocation, name string, defaultValue dgo.Value, options interface{}) dgo.Value {
	return Lookup2(ic, []string{name}, typ.Any, defaultValue, nil, nil, api.ToMap(`lookup options`, options), nil)
}

// Lookup2 performs a lookup using the given parameters.
//
// ic - The lookup invocation
//
// names[] - The name or names to lookup
//
// valueType - Optional expected type of the found value
//
// defaultValue - Optional value to use as default when no value is found
//
// override - Optional map to use as override. Values found here are returned immediately (no merge)
//
// defaultValuesHash - Optional map to use as the last resort (but before defaultValue)
//
// options - Optional map with merge strategy and options
//
// defaultFunc - Optional function to produce a default value
func Lookup2(
	ic api.Invocation,
	names []string,
	valueType dgo.Type,
	defaultValue dgo.Value,
	override dgo.Map,
	defaultValuesHash dgo.Map,
	options dgo.Map,
	defaultFunc dgo.Producer) dgo.Value {
	if v := lookupInMap(names, override); v != nil {
		return ensureType(valueType, v)
	}
	for _, name := range names {
		if v := ic.Lookup(api.NewKey(name), options); v != nil {
			return ensureType(valueType, v)
		}
	}
	if v := lookupInMap(names, defaultValuesHash); v != nil {
		return ensureType(valueType, v)
	}
	if defaultValue != nil {
		return ensureType(valueType, defaultValue)
	}
	if defaultFunc != nil {
		return ensureType(valueType, defaultFunc())
	}
	return nil
}

// LookupAll performs a lookup using the given parameters for all of the names passed in.
//
// ic - The lookup invocation
//
// names[] - The name or names to lookup
//
// valueType - Optional expected type of the found value
//
// override - Optional map to use as override. Values found here are returned immediately (no merge)
//
// defaultValuesHash - Optional map to use as the last resort
//
// options - Optional map with merge strategy and options
func LookupAll(
	ic api.Invocation,
	names []string,
	valueType dgo.StructMapType,
	override dgo.Map,
	defaultValuesHash dgo.Map,
	options dgo.Map) dgo.Value {
	response := vf.MutableMap()
	for _, name := range names {
		a := []string{name}
		if v := lookupInMap(a, override); v != nil {
			response.Put(name, ensureTypeFromMap(valueType, name, v))
			continue
		}
		if v := ic.Lookup(api.NewKey(name), options); v != nil {
			response.Put(name, ensureTypeFromMap(valueType, name, v))
			continue
		}
		if v := lookupInMap(a, defaultValuesHash); v != nil {
			response.Put(name, ensureTypeFromMap(valueType, name, v))
			continue
		}
	}
	return response
}

func lookupInMap(names []string, m dgo.Map) dgo.Value {
	if m != nil && m.Len() > 0 {
		for _, name := range names {
			if dv := m.Get(name); dv != nil {
				return dv
			}
		}
	}
	return nil
}

func ensureTypeFromMap(t dgo.StructMapType, k string, v dgo.Value) dgo.Value {
	if t == nil {
		return v
	}
	if e := t.Get(k); e != nil {
		return ensureType(e.Value().(dgo.Type), v)
	}
	panic(fmt.Errorf("key '%s' was not found in the type map", k))
}

func ensureType(t dgo.Type, v dgo.Value) dgo.Value {
	if t == nil || t.Instance(v) {
		return v
	}
	return vf.New(t, v)
}

// TryWithParent initializes a lookup context with global options and a top-level lookup key function and then calls
// the given consumer function with that context. If the given function panics, the panic will be recovered and returned
// as an error.
func TryWithParent(parent context.Context, tp hiera.LookupKey, options interface{}, consumer func(api.Session) error) error {
	return util.Catch(func() {
		s := session.New(parent, tp, options, nil)
		defer s.KillPlugins()
		err := consumer(s)
		if err != nil {
			panic(err)
		}
	})
}

// DoWithParent initializes a lookup context with global options and a top-level lookup key function and then calls
// the given consumer function with that context.
func DoWithParent(parent context.Context, tp hiera.LookupKey, options interface{}, consumer func(api.Session)) {
	s := session.New(parent, tp, options, nil)
	defer s.KillPlugins()
	consumer(s)
}

// varSplit splits on either ':' or '=' but not on '::', ':=', '=:' or '=='
var varSplit = regexp.MustCompile(`\A(.*?[^:=])[:=]([^:=].*)\z`)
var needParsePrefix = []string{`{`, `[`, `"`, `'`}

// LookupAndRender performs a lookup using the given command options and arguments and renders the result on the given
// io.Writer in accordance with the `RenderAs` option.
func LookupAndRender(c api.Session, opts *CommandOptions, args []string, out io.Writer) bool {
	tp := parseType(opts.Type, c.Dialect())

	var options dgo.Map
	if !(opts.Merge == `` || opts.Merge == `first`) {
		options = vf.Map(`merge`, opts.Merge)
	}

	var dv dgo.Value
	if opts.Default != nil {
		s := *opts.Default
		if s == `` {
			dv = vf.String(``)
		} else {
			dv = parseCommandLineValue(c, s)
		}
	}

	var explainer api.Explainer
	if opts.ExplainData || opts.ExplainOptions {
		explainer = explain.NewExplainer(opts.ExplainOptions, opts.ExplainOptions && !opts.ExplainData)
	}

	var found dgo.Value
	invocation := c.Invocation(createScope(c, opts), explainer)
	if opts.LookupAll {
		stp, ok := tp.(dgo.StructMapType)
		if !ok && opts.Type != `` {
			panic(fmt.Errorf("type must be a map"))
		}
		found = LookupAll(invocation, args, stp, nil, nil, options)
	} else {
		found = Lookup2(invocation, args, tp, dv, nil, nil, options, nil)
	}
	if explainer != nil {
		renderAs := Text
		if opts.RenderAs != `` {
			renderAs = RenderName(opts.RenderAs)
		}
		Render(c, renderAs, explainer, out)
		return found != nil
	}

	if found == nil {
		return false
	}

	renderAs := YAML
	if opts.RenderAs != `` {
		renderAs = RenderName(opts.RenderAs)
	}
	Render(c, renderAs, found, out)
	return true
}

func parseType(t string, dl streamer.Dialect) dgo.Type {
	tp := typ.Any
	if t != `` {
		tp = dl.ParseType(nil, vf.String(t))
	}
	return tp
}

func parseCommandLineValue(c api.Session, vs string) dgo.Value {
	vs = strings.TrimSpace(vs)
	for _, pfx := range needParsePrefix {
		if strings.HasPrefix(vs, pfx) {
			var v dgo.Value
			c.AliasMap().Collect(func(aa dgo.AliasAdder) {
				v = typ.ExactValue(c.Dialect().ParseType(aa, vf.String(vs)))
			})
			return v
		}
	}
	return vf.String(vs)
}

func createScope(c api.Session, opts *CommandOptions) dgo.Map {
	scope := vf.MutableMap()
	if vl := len(opts.Variables); vl > 0 {
		for _, e := range opts.Variables {
			if m := varSplit.FindStringSubmatch(e); m != nil {
				key := strings.TrimSpace(m[1])
				scope.Put(key, parseCommandLineValue(c, m[2]))
			} else {
				panic(fmt.Errorf("unable to parse variable '%s'", e))
			}
		}
	}

	addVarPaths(opts.VarPaths, scope)
	if len(opts.FactPaths) > 0 {
		facts := vf.MutableMap()
		addVarPaths(opts.FactPaths, facts)
		scope.PutAll(facts)
		scope.Put(`facts`, facts)
	}
	return scope
}

func addVarPaths(varPaths []string, m dgo.Map) {
	for _, vars := range varPaths {
		var bs []byte
		var err error
		if vars == `-` {
			bs, err = ioutil.ReadAll(os.Stdin)
		} else {
			bs, err = ioutil.ReadFile(vars)
		}
		if err == nil && len(bs) > 0 {
			var yv dgo.Value
			if yv, err = yaml.Unmarshal(bs); err == nil {
				if data, ok := yv.(dgo.Map); ok {
					m.PutAll(data)
				} else {
					err = fmt.Errorf(`file '%s' does not contain a YAML hash`, vars)
				}
			}
		}
		if err != nil {
			panic(err)
		}
	}
}
