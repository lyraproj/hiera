package hiera

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/lyraproj/hiera/explain"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hiera/internal"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/yaml"
)

// A CommandOptions contains the options given by to the CLI lookup command or a REST invocation.
type CommandOptions struct {
	// Type is a pcore Type string such as "String" or "Array[Integer]" used for assertion of the
	// found value.
	Type string

	// Merge is the name of a merge strategy
	Merge string

	// Default is a pointer to the string representation of a default value or nil if no default value exists
	Default *string

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
}

// NewInvocation creates a new lookup invocation using the given scope and explainer.
func NewInvocation(c px.Context, scope px.Keyed, explainer explain.Explainer) hieraapi.Invocation {
	return internal.NewInvocation(c, scope, explainer)
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
func Lookup(ic hieraapi.Invocation, name string, defaultValue px.Value, options map[string]px.Value) px.Value {
	return internal.Lookup(ic, name, defaultValue, options)
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
// block - Optional block to produce a default value
func Lookup2(
	ic hieraapi.Invocation,
	names []string,
	valueType px.Type,
	defaultValue px.Value,
	override px.OrderedMap,
	defaultValuesHash px.OrderedMap,
	options map[string]px.Value,
	block px.Lambda) px.Value {
	return internal.Lookup2(ic, names, valueType, defaultValue, override, defaultValuesHash, options, block)
}

// TryWithParent initializes a lookup context with global options and a top-level lookup key function and then calls
// the given consumer function with that context. If the given function panics, the panic will be recovered and returned
// as an error.
func TryWithParent(parent context.Context, tp hieraapi.LookupKey, options map[string]px.Value, consumer func(px.Context) error) error {
	return pcore.TryWithParent(parent, func(c px.Context) error {
		internal.InitContext(c, tp, options)
		defer internal.KillPlugins(c)
		return consumer(c)
	})
}

// DoWithParent initializes a lookup context with global options and a top-level lookup key function and then calls
// the given consumer function with that context.
func DoWithParent(parent context.Context, tp hieraapi.LookupKey, options map[string]px.Value, consumer func(px.Context)) {
	pcore.DoWithParent(parent, func(c px.Context) {
		internal.InitContext(c, tp, options)
		defer internal.KillPlugins(c)
		consumer(c)
	})
}

// varSplit splits on either ':' or '=' but not on '::', ':=', '=:' or '=='
var varSplit = regexp.MustCompile(`\A(.*?[^:=])[:=]([^:=].*)\z`)
var needParsePrefix = []string{`{`, `[`, `"`, `'`}

// LookupAndRender performs a lookup using the given command options and arguments and renders the result on the given
// io.Writer in accordance with the `RenderAs` option.
func LookupAndRender(c px.Context, opts *CommandOptions, args []string, out io.Writer) bool {
	var tp px.Type = types.DefaultAnyType()
	if opts.Type != `` {
		tp = c.ParseType(opts.Type)
	}

	options := make(map[string]px.Value)
	if !(opts.Merge == `` || opts.Merge == `first`) {
		options[`merge`] = types.WrapString(opts.Merge)
	}

	var dv px.Value
	if opts.Default != nil {
		if !tp.Equals(types.DefaultAnyType(), nil) {
			dv = types.CoerceTo(c, `default value`, tp, types.ParseFile(`<default value>`, *opts.Default))
		} else {
			dv = types.WrapString(*opts.Default)
		}
	}

	var explainer explain.Explainer
	if opts.ExplainData || opts.ExplainOptions {
		explainer = explain.NewExplainer(opts.ExplainOptions, opts.ExplainOptions && !opts.ExplainData)
	}

	found := Lookup2(internal.NewInvocation(c, createScope(c, opts), explainer), args, tp, dv, nil, nil, options, nil)
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

func parseCommandLineValue(c px.Context, key, vs string) px.Value {
	vs = strings.TrimSpace(vs)
	for _, pfx := range needParsePrefix {
		if strings.HasPrefix(vs, pfx) {
			return types.ResolveDeferred(c, types.ParseFile(`var `+key, vs), c.Scope())
		}
	}
	return types.WrapString(vs)
}

func createScope(c px.Context, opts *CommandOptions) px.OrderedMap {
	scope := px.EmptyMap
	if vl := len(opts.Variables); vl > 0 {
		ve := make([]*types.HashEntry, vl)
		for i, e := range opts.Variables {
			if m := varSplit.FindStringSubmatch(e); m != nil {
				key := strings.TrimSpace(m[1])
				ve[i] = types.WrapHashEntry2(key, parseCommandLineValue(c, key, m[2]))
			} else {
				panic(fmt.Errorf("unable to parse variable '%s'", e))
			}
		}
		scope = types.WrapHash(ve)
	}

	for _, vars := range opts.VarPaths {
		var content *types.Binary
		if vars == `-` {
			data, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				panic(err)
			}
			content = types.WrapBinary(data)
		} else {
			content = types.BinaryFromFile(vars)
		}
		bs := content.Bytes()
		if len(bs) == 0 {
			continue
		}
		yv := yaml.Unmarshal(c, bs)
		if data, ok := yv.(px.OrderedMap); ok {
			scope = scope.Merge(data)
		} else {
			panic(px.Error(hieraapi.YamlNotHash, issue.H{`path`: vars}))
		}
	}
	return scope
}
