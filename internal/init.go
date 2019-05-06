package internal

import (
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/pcore/types"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
)

var NoOptions = map[string]px.Value{}

func init() {
	px.RegisterResolvableType(px.NewNamedType(`Hiera`, `TypeSet[{
		pcore_version => '1.0.0',
		version => '5.0.0',
		types => {
			Options => Hash[Pattern[/\A[A-Za-z](:?[0-9A-Za-z_-]*[0-9A-Za-z])?\z/], Data],
			Defaults => Struct[{
				Optional[options] => Options,
				Optional[data_dig] => String[1],
				Optional[data_hash] => String[1],
				Optional[lookup_key] => String[1],
				Optional[data_dir] => String[1],
			}],
			Entry => Struct[{
				name => String[1],
				Optional[options] => Options,
				Optional[data_dig] => String[1],
				Optional[data_hash] => String[1],
				Optional[lookup_key] => String[1],
				Optional[data_dir] => String[1],
				Optional[path] => String[1],
				Optional[paths] => Array[String[1], 1],
				Optional[glob] => String[1],
				Optional[globs] => Array[String[1], 1],
				Optional[uri] => String[1],
				Optional[uris] => Array[String[1], 1],
				Optional[mapped_paths] => Array[String[1], 3, 3],
			}],
			Config => Struct[{
				version => Integer[5, 5],
				Optional[defaults] => Defaults,
				Optional[hierarchy] => Array[Entry],
				Optional[default_hierarchy] => Array[Entry]
			}]
		}
  }]`).(px.ResolvableType))

	pcore.DefineSetting(`hiera_config`, types.DefaultStringType(), nil)

	hieraapi.NotFound = px.Error(hieraapi.KeyNotFound, issue.NoArgs)

	hieraapi.NewKey = newKey
}

func Lookup(ic hieraapi.Invocation, name string, dflt px.Value, options map[string]px.Value) px.Value {
	return Lookup2(ic, []string{name}, types.DefaultAnyType(), dflt, px.EmptyMap, px.EmptyMap, options, nil)
}

func Lookup2(
	ic hieraapi.Invocation,
	names []string,
	valueType px.Type,
	defaultValue px.Value,
	override px.OrderedMap,
	defaultValuesHash px.OrderedMap,
	options map[string]px.Value,
	block px.Lambda) px.Value {
	if override == nil {
		override = px.EmptyMap
	}
	if defaultValuesHash == nil {
		defaultValuesHash = px.EmptyMap
	}

	if options == nil {
		options = NoOptions
	}

	for _, name := range names {
		if ov, ok := override.Get4(name); ok {
			return ov
		}
		key := newKey(name)
		v := ic.WithKey(key, func() px.Value {
			return ic.(*invocation).lookupViaCache(key, options)
		})
		if v != nil {
			return v
		}
	}

	if defaultValuesHash.Len() > 0 {
		for _, name := range names {
			if dv, ok := defaultValuesHash.Get4(name); ok {
				return dv
			}
		}
	}

	if defaultValue == nil {
		// nil (as opposed to UNDEF) means that no default was provided.
		if len(names) == 1 {
			panic(px.Error(hieraapi.NameNotFound, issue.H{`name`: names[0]}))
		}
		panic(px.Error(hieraapi.NotAnyNameFound, issue.H{`name_list`: names}))
	}
	return defaultValue
}
