package internal

import (
	"github.com/lyraproj/hiera/explain"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

func luNames(nameOrNames px.Value) (names []string) {
	if ar, ok := nameOrNames.(*types.Array); ok {
		names = make([]string, ar.Len())
		ar.EachWithIndex(func(v px.Value, i int) {
			names[i] = v.String()
		})
	} else {
		names = []string{nameOrNames.String()}
	}
	return
}

func mergeType(nameOrHash px.Value) (merge map[string]px.Value) {
	hs, ok := nameOrHash.(*types.Hash)
	switch {
	case ok:
		merge = make(map[string]px.Value, hs.Len())
		hs.EachPair(func(k, v px.Value) { merge[k.String()] = v })
	case nameOrHash == px.Undef:
		merge = NoOptions
	default:
		merge = map[string]px.Value{`merge`: nameOrHash}
	}
	return
}

func init() {
	px.NewGoFunction2(`lookup`,
		func(l px.LocalTypes) {
			l.Type(`NameType`, `Variant[String, Array[String]]`)
			l.Type(`ValueType`, `Type`)
			l.Type(`DefaultValueType`, `Any`)
			l.Type(`MergeType`, `Variant[String[1], Hash[String, Scalar]]`)
			l.Type(`BlockType`, `Callable[NameType]`)
			l.Type(`OptionsWithName`, `Struct[{
	      name                => NameType,
  	    value_type          => Optional[ValueType],
    	  default_value       => Optional[DefaultValueType],
      	override            => Optional[Hash[String,Any]],
	      default_values_hash => Optional[Hash[String,Any]],
  	    merge               => Optional[MergeType]
    	}]`)
			l.Type(`OptionsWithoutName`, `Struct[{
	      value_type          => Optional[ValueType],
  	    default_value       => Optional[DefaultValueType],
    	  override            => Optional[Hash[String,Any]],
      	default_values_hash => Optional[Hash[String,Any]],
	      merge               => Optional[MergeType]
  	  }]`)
		},

		func(d px.Dispatch) {
			d.Param(`NameType`)
			d.OptionalParam(`ValueType`)
			d.OptionalParam(`MergeType`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				vt := px.Type(types.DefaultAnyType())
				options := px.Undef
				argc := len(args)
				if argc > 1 {
					vt = args[1].(px.Type)
					if argc > 2 {
						options = args[2]
					}
				}
				return invokeLookup(c, luNames(args[0]), vt, nil, nil, nil, options, nil)
			})
		},

		func(d px.Dispatch) {
			d.Param(`NameType`)
			d.Param(`Optional[ValueType]`)
			d.Param(`Optional[MergeType]`)
			d.Param(`DefaultValueType`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				vt := px.Type(types.DefaultAnyType())
				if arg := args[1]; arg != px.Undef {
					vt = arg.(px.Type)
				}
				return invokeLookup(c, luNames(args[0]), vt, args[3], nil, nil, args[2], nil)
			})
		},

		func(d px.Dispatch) {
			d.Param(`NameType`)
			d.OptionalParam(`ValueType`)
			d.OptionalParam(`MergeType`)
			d.Block(`BlockType`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				vt := px.Type(types.DefaultAnyType())
				if arg := args[1]; arg != px.Undef {
					vt = arg.(px.Type)
				}
				return invokeLookup(c, luNames(args[0]), vt, nil, nil, nil, args[2], block)
			})
		},

		func(d px.Dispatch) {
			d.Param(`OptionsWithName`)
			d.OptionalBlock(`BlockType`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				hash := args[0].(*types.Hash)
				names := luNames(hash.Get5(`name`, px.Undef))
				dflt := hash.Get5(`default_value`, nil)
				vt := hash.Get5(`value_type`, types.DefaultAnyType()).(px.Type)
				override := hash.Get5(`override`, px.EmptyMap).(px.OrderedMap)
				dfltHash := hash.Get5(`default_values_hash`, px.EmptyMap).(px.OrderedMap)
				return invokeLookup(c, names, vt, dflt, override, dfltHash, hash.Get5(`merge`, px.Undef), block)
			})
		},

		func(d px.Dispatch) {
			d.Param(`NameType`)
			d.Param(`OptionsWithoutName`)
			d.OptionalBlock(`BlockType`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				names := luNames(args[0])
				hash := args[1].(*types.Hash)
				dflt := hash.Get5(`default_value`, nil)
				vt := hash.Get5(`value_type`, types.DefaultAnyType()).(px.Type)
				override := hash.Get5(`override`, px.EmptyMap).(px.OrderedMap)
				dfltHash := hash.Get5(`default_values_hash`, px.EmptyMap).(px.OrderedMap)
				return invokeLookup(c, names, vt, dflt, override, dfltHash, hash.Get5(`merge`, px.Undef), block)
			})
		},
	)
}

func invokeLookup(c px.Context, names []string, vt px.Type, dflt px.Value, override, dfltHash px.OrderedMap, options px.Value, block px.Lambda) px.Value {
	ll := px.ERR
	if li, ok := c.Get(`logLevel`); ok {
		ll = li.(px.LogLevel)
	}
	var ex explain.Explainer
	if ll == px.DEBUG {
		ex = explain.NewExplainer(false, false)
	}

	v := Lookup2(NewInvocation(c, c.Scope(), ex), names, vt, dflt, override, dfltHash, mergeType(options), block)
	if ex != nil {
		c.Logger().Log(px.DEBUG, ex)
	}
	return v
}
