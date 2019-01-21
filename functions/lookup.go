package functions

import (
	"github.com/lyraproj/hiera/impl"
	"github.com/lyraproj/hiera/lookup"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
)

func luNames(nameOrNames eval.Value) (names []string) {
	if ar, ok := nameOrNames.(*types.ArrayValue); ok {
		names = make([]string, ar.Len())
		ar.EachWithIndex(func(v eval.Value, i int) {
			names[i] = v.String()
		})
	} else {
		names = []string{nameOrNames.String()}
	}
	return
}

func mergeType(nameOrHash eval.Value) (merge map[string]eval.Value) {
	if hs, ok := nameOrHash.(*types.HashValue); ok {
		merge = make(map[string]eval.Value, hs.Len())
		hs.EachPair(func(k, v eval.Value) { merge[k.String()] = v })
	} else if nameOrHash == eval.UNDEF {
		merge = impl.NoOptions
	} else {
		merge = map[string]eval.Value{`merge`: nameOrHash}
	}
	return
}

func init() {
	eval.NewGoFunction2(`lookup`,
		func(l eval.LocalTypes) {
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

		func(d eval.Dispatch) {
			d.Param(`NameType`)
			d.OptionalParam(`ValueType`)
			d.OptionalParam(`MergeType`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				vtype := eval.Type(types.DefaultAnyType())
				var options map[string]eval.Value
				nargs := len(args)
				if nargs > 1 {
					vtype = args[1].(eval.Type)
					if nargs > 2 {
						options = mergeType(args[2])
					}
				}
				return lookup.Lookup2(impl.NewInvocation(c), luNames(args[0]), vtype, nil, nil, nil, options, nil)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`NameType`)
			d.Param(`Optional[ValueType]`)
			d.Param(`Optional[MergeType]`)
			d.Param(`DefaultValueType`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				vtype := eval.Type(types.DefaultAnyType())
				if arg := args[1]; arg != eval.UNDEF {
					vtype = arg.(eval.Type)
				}
				options := mergeType(args[2])
				return lookup.Lookup2(impl.NewInvocation(c), luNames(args[0]), vtype, args[3], nil, nil, options, nil)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`NameType`)
			d.OptionalParam(`ValueType`)
			d.OptionalParam(`MergeType`)
			d.Block(`BlockType`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				vtype := eval.Type(types.DefaultAnyType())
				if arg := args[1]; arg != eval.UNDEF {
					vtype = arg.(eval.Type)
				}
				options := mergeType(args[2])
				return lookup.Lookup2(impl.NewInvocation(c), luNames(args[0]), vtype, nil, nil, nil, options, block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`OptionsWithName`)
			d.OptionalBlock(`BlockType`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				hash := args[0].(*types.HashValue)
				names := luNames(hash.Get5(`name`, eval.UNDEF))
				dflt := hash.Get5(`default_value`, nil)
				vtype := hash.Get5(`value_type`, types.DefaultAnyType()).(eval.Type)
				override := hash.Get5(`override`, eval.EMPTY_MAP).(eval.OrderedMap)
				dfltHash := hash.Get5(`default_values_hash`, eval.EMPTY_MAP).(eval.OrderedMap)
				options := mergeType(hash.Get5(`merge`, eval.UNDEF))
				return lookup.Lookup2(impl.NewInvocation(c), names, vtype, dflt, override, dfltHash, options, block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`NameType`)
			d.Param(`OptionsWithoutName`)
			d.OptionalBlock(`BlockType`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				names := luNames(args[0])
				hash := args[1].(*types.HashValue)
				dflt := hash.Get5(`default_value`, nil)
				vtype := hash.Get5(`value_type`, types.DefaultAnyType()).(eval.Type)
				override := hash.Get5(`override`, eval.EMPTY_MAP).(eval.OrderedMap)
				dfltHash := hash.Get5(`default_values_hash`, eval.EMPTY_MAP).(eval.OrderedMap)
				options := mergeType(hash.Get5(`merge`, eval.UNDEF))
				return lookup.Lookup2(impl.NewInvocation(c), names, vtype, dflt, override, dfltHash, options, block)
			})
		},
	)
}
