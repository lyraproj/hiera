package impl_test

import (
	"context"
	"fmt"
	"github.com/puppetlabs/go-evaluator/eval"
	evalimpl "github.com/puppetlabs/go-evaluator/impl"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-hiera/impl"
	"github.com/puppetlabs/go-hiera/lookup"
	"github.com/puppetlabs/go-hiera/provider"
	"github.com/puppetlabs/go-issues/issue"

	// Ensure initialization
	_ "github.com/puppetlabs/go-evaluator/pcore"
	_ "github.com/puppetlabs/go-hiera/functions"
)

var options map[string]eval.Value

func init() {
	options = map[string]eval.Value{`path`: types.WrapString(`./testdata/sample_data.yaml`)}
}

func ExampleLookup_first() {
	lookup.DoWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) {
		fmt.Println(lookup.Lookup(impl.NewInvocation(c), `first`, nil, nil))
	})
	// Output: value of first
}

func ExampleLookup_dottedInt() {
	lookup.DoWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) {
		fmt.Println(lookup.Lookup(impl.NewInvocation(c), `array.1`, nil, nil))
	})
	// Output: two
}

func ExampleLookup_dottedMix() {
	lookup.DoWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) {
		fmt.Println(lookup.Lookup(impl.NewInvocation(c), `hash.array.1`, nil, nil))
	})
	// Output: value of first
}

func ExampleLookup_interpolate() {
	lookup.DoWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) {
		fmt.Println(lookup.Lookup(impl.NewInvocation(c), `second`, nil, nil))
	})
	// Output: includes 'value of first'
}

func ExampleLookup_interpolateScope() {
	eval.Puppet.DoWithParent(context.Background(), func(c eval.Context) {
		c.DoWithScope(evalimpl.NewScope2(types.WrapStringToInterfaceMap(c, issue.H{
			`world`: `cruel world`,
		}), false), func() {
			lookup.DoWithParent(c, provider.Yaml, options, func(c eval.Context) {
				fmt.Println(lookup.Lookup(impl.NewInvocation(c), `ipScope`, nil, nil))
				fmt.Println(lookup.Lookup(impl.NewInvocation(c), `ipScope2`, nil, nil))
			})
		})
	})
	// Output:
	// hello cruel world
	// hello cruel world
}

func ExampleLookup_interpolateEmpty() {
	lookup.DoWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) {
		fmt.Println(lookup.Lookup(impl.NewInvocation(c), `empty1`, nil, nil))
		fmt.Println(lookup.Lookup(impl.NewInvocation(c), `empty2`, nil, nil))
		fmt.Println(lookup.Lookup(impl.NewInvocation(c), `empty3`, nil, nil))
		fmt.Println(lookup.Lookup(impl.NewInvocation(c), `empty4`, nil, nil))
		fmt.Println(lookup.Lookup(impl.NewInvocation(c), `empty5`, nil, nil))
		fmt.Println(lookup.Lookup(impl.NewInvocation(c), `empty6`, nil, nil))
	})
	// Output:
	// startend
	// startend
	// startend
	// startend
	// startend
	// startend
}

func ExampleLookup_interpolateLiteral() {
	lookup.DoWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) {
		fmt.Println(lookup.Lookup(impl.NewInvocation(c), `ipLiteral`, nil, options))
	})
	// Output: some literal text
}

func ExampleLookup_interpolateAlias() {
	lookup.DoWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) {
		v := lookup.Lookup(impl.NewInvocation(c), `ipAlias`, nil, options)
		fmt.Printf(`%s %s`, eval.GenericValueType(v), v)
	})
	// Output: Array[Enum] ['one', 'two', 'three']
}

func ExampleLookup_interpolateBadAlias() {
	fmt.Println(lookup.TryWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) error {
		lookup.Lookup(impl.NewInvocation(c), `ipBadAlias`, nil, options)
		return nil
	}))
	// Output: 'alias' interpolation is only permitted if the expression is equal to the entire string
}

func ExampleLookup_interpolateBadFunction() {
	fmt.Println(lookup.TryWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) error {
		lookup.Lookup(impl.NewInvocation(c), `ipBad`, nil, options)
		return nil
	}))
	// Output: Unknown interpolation method 'bad'
}

func ExampleLookup_notFoundWithoutDefault() {
	fmt.Println(lookup.TryWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) error {
		lookup.Lookup(impl.NewInvocation(c), `nonexistent`, nil, options)
		return nil
	}))
	// Output: lookup() did not find a value for the name 'nonexistent'
}

func ExampleLookup_notFoundDflt() {
	lookup.DoWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) {
		fmt.Println(lookup.Lookup(impl.NewInvocation(c), `nonexistent`, types.WrapString(`default value`), options))
	})
	// Output: default value
}

func ExampleLookup_notFoundDottedIdx() {
	lookup.DoWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) {
		fmt.Println(lookup.Lookup(impl.NewInvocation(c), `array.3`, types.WrapString(`default value`), options))
	})
	// Output: default value
}

func ExampleLookup_notFoundDottedMix() {
	lookup.DoWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) {
		fmt.Println(lookup.Lookup(impl.NewInvocation(c), `hash.float`, types.WrapString(`default value`), options))
	})
	// Output: default value
}

func ExampleLookup_badStringDig() {
	fmt.Println(lookup.TryWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) error {
		lookup.Lookup(impl.NewInvocation(c), `hash.int.v`, nil, options)
		return nil
	}))
	// Output: lookup() Got Integer when a hash-like object was expected to access value using 'v' from key 'hash.int.v'
}

func ExampleLookup_badIntDig() {
	fmt.Println(lookup.TryWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) error {
		lookup.Lookup(impl.NewInvocation(c), `hash.3`, nil, options)
		return nil
	}))
	// Output: lookup() Got Hash[Enum, Data] when a hash-like object was expected to access value using '3' from key 'hash.3'
}

func ExampleLookup2_findFirst() {
	lookup.DoWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) {
		fmt.Println(lookup.Lookup2(impl.NewInvocation(c), []string{`first`, `second`}, types.DefaultAnyType(), nil, nil, nil, options, nil))
	})
	// Output: value of first
}

func ExampleLookup2_findSecond() {
	lookup.DoWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) {
		fmt.Println(lookup.Lookup2(impl.NewInvocation(c), []string{`nonexisting`, `second`}, types.DefaultAnyType(), nil, nil, nil, options, nil))
	})
	// Output: includes 'value of first'
}

func ExampleLookup2_notFoundWithoutDflt() {
	fmt.Println(lookup.TryWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) error {
		lookup.Lookup2(impl.NewInvocation(c), []string{`nonexisting`, `notthere`}, types.DefaultAnyType(), nil, nil, nil, options, nil)
		return nil
	}))
	// Output: lookup() did not find a value for any of the names [nonexisting, notthere]
}

func ExampleLookup2_notFoundDflt() {
	lookup.DoWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) {
		fmt.Println(lookup.Lookup2(impl.NewInvocation(c), []string{`nonexisting`, `notthere`}, types.DefaultAnyType(), types.WrapString(`default value`), nil, nil, options, nil))
	})
	// Output: default value
}

func ExampleContextCachedValue() {

	cachingProvider := func(ic lookup.ProviderContext, key string, options map[string]eval.Value) (eval.Value, bool) {
		if v, ok := ic.CachedValue(key); ok {
			fmt.Printf("Returning cached value for %s\n", key)
			return v, true
		}
		fmt.Printf("Creating and caching value for %s\n", key)
		v := ic.Interpolate(types.WrapString(fmt.Sprintf("generated value for %%{%s}", key)))
		ic.Cache(key, v)
		return v, true
	}

	lookup.DoWithParent(context.Background(), cachingProvider, nil, func(c eval.Context) {
		c.DoWithScope(evalimpl.NewScope2(types.WrapStringToInterfaceMap(c, map[string]interface{}{
			`a`: `scope 'a'`,
			`b`: `scope 'b'`,
		}), false), func() {
			ic := impl.NewInvocation(c)
			fmt.Println(lookup.Lookup(ic, `a`, nil, nil))
			fmt.Println(lookup.Lookup(ic, `b`, nil, nil))
			fmt.Println(lookup.Lookup(ic, `a`, nil, nil))
			fmt.Println(lookup.Lookup(ic, `b`, nil, nil))
		})
	})
	// Output:
	// Creating and caching value for a
	// generated value for scope 'a'
	// Creating and caching value for b
	// generated value for scope 'b'
	// generated value for scope 'a'
	// generated value for scope 'b'
}

func ExampleLookup_dottedStringInt() {
	lookup.DoWithParent(context.Background(), provider.Yaml, options, func(c eval.Context) {
		v := lookup.Lookup(impl.NewInvocation(c), `hash.array.0`, nil, options)
		fmt.Println(v)
	})
	// Output: two
}