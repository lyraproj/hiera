package lookup_test

import (
	"github.com/puppetlabs/go-hiera/lookup"
	"context"
	"github.com/puppetlabs/go-evaluator/eval"
	"fmt"

	// Ensure initialization
	_ "github.com/puppetlabs/go-evaluator/pcore"
	_ "github.com/puppetlabs/go-hiera/functions"
	"github.com/puppetlabs/go-evaluator/impl"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-evaluator/types"
)

var sampleData = map[string]eval.PValue {
  `first`: eval.Wrap(`value of first`),
  `array`: eval.Wrap([]string{`one`, `two`, `three`}),
	`hash`: eval.Wrap(map[string]interface{}{`int`: 1, `string`: `one`, `array`: []string{`two`, `%{hiera('first')}`}}),
  `second`: eval.Wrap(`includes '%{lookup('first')}'`),
	`ipAlias`: eval.Wrap(`%{alias('array')}`),
	`ipBadAlias`: eval.Wrap(`x %{alias('array')}`),
	`ipScope`: eval.Wrap(`hello %{world}`),
	`ipScope2`: eval.Wrap(`hello %{scope('world')}`),
	`ipLiteral`: eval.Wrap(`some %{literal('literal')} text`),
	`ipBad`: eval.Wrap(`hello %{bad('world')}`),
	`empty1`: eval.Wrap(`start%{}end`),
	`empty2`: eval.Wrap(`start%{''}end`),
	`empty3`: eval.Wrap(`start%{""}end`),
	`empty4`: eval.Wrap(`start%{::}end`),
	`empty5`: eval.Wrap(`start%{'::'}end`),
	`empty6`: eval.Wrap(`start%{"::"}end`),
}

func provider(c lookup.Context, key string, options eval.KeyedValue) eval.PValue {
	if v, ok := sampleData[key]; ok {
		return v
	}
	c.NotFound()
	return nil
}

func ExampleLookup_first() {
	lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		fmt.Println(lookup.Lookup(c, `first`, nil, eval.EMPTY_MAP))
		return nil
	})
	// Output: value of first
}

func ExampleLookup_dottedInt() {
	lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		fmt.Println(lookup.Lookup(c, `array.1`, nil, eval.EMPTY_MAP))
		return nil
	})
	// Output: two
}

func ExampleLookup_dottedMix() {
	lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		fmt.Println(lookup.Lookup(c, `hash.array.1`, nil, eval.EMPTY_MAP))
		return nil
	})
	// Output: value of first
}

func ExampleLookup_interpolate() {
	lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		fmt.Println(lookup.Lookup(c, `second`, nil, eval.EMPTY_MAP))
		return nil
	})
	// Output: includes 'value of first'
}

func ExampleLookup_interpolateScope() {
	eval.Puppet.DoWithParent(context.Background(), func(c eval.Context) error {
		c = c.WithScope(impl.NewScope2(types.WrapHash4(c, issue.H{
			`world`: `cruel world`,
		})))
		lookup.DoWithParent(c, provider, func(c lookup.Context) error {
			fmt.Println(lookup.Lookup(c, `ipScope`, nil, eval.EMPTY_MAP))
			fmt.Println(lookup.Lookup(c, `ipScope2`, nil, eval.EMPTY_MAP))
			return nil
		})
		return nil
	})
	// Output:
	// hello cruel world
	// hello cruel world
}

func ExampleLookup_interpolateEmpty() {
	lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		fmt.Println(lookup.Lookup(c, `empty1`, nil, eval.EMPTY_MAP))
		fmt.Println(lookup.Lookup(c, `empty2`, nil, eval.EMPTY_MAP))
		fmt.Println(lookup.Lookup(c, `empty3`, nil, eval.EMPTY_MAP))
		fmt.Println(lookup.Lookup(c, `empty4`, nil, eval.EMPTY_MAP))
		fmt.Println(lookup.Lookup(c, `empty5`, nil, eval.EMPTY_MAP))
		fmt.Println(lookup.Lookup(c, `empty6`, nil, eval.EMPTY_MAP))
		return nil
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
	lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		fmt.Println(lookup.Lookup(c, `ipLiteral`, nil, eval.EMPTY_MAP))
		return nil
	})
	// Output: some literal text
}

func ExampleLookup_interpolateAlias() {
	lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		v := lookup.Lookup(c, `ipAlias`, nil, eval.EMPTY_MAP)
		fmt.Printf(`%s %s`, eval.GenericValueType(v), v)
		return nil
	})
	// Output: Array[Enum] ['one', 'two', 'three']
}

func ExampleLookup_interpolateBadAlias() {
	fmt.Println(lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		lookup.Lookup(c, `ipBadAlias`, nil, eval.EMPTY_MAP)
		return nil
	}))
	// Output: 'alias' interpolation is only permitted if the expression is equal to the entire string
}

func ExampleLookup_interpolateBadFunction() {
	fmt.Println(lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		lookup.Lookup(c, `ipBad`, nil, eval.EMPTY_MAP)
		return nil
	}))
	// Output: Unknown interpolation method 'bad'
}

func ExampleLookup_notFoundWithoutDefault() {
	fmt.Println(lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		lookup.Lookup(c, `nonexistent`, nil, eval.EMPTY_MAP)
		return nil
	}))
	// Output: lookup() did not find a value for the name '{name}'
}

func ExampleLookup_notFoundDflt() {
	lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		fmt.Println(lookup.Lookup(c, `nonexistent`, eval.Wrap(`default value`), eval.EMPTY_MAP))
		return nil
	})
	// Output: default value
}

func ExampleLookup_notFoundDottedIdx() {
	lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		fmt.Println(lookup.Lookup(c, `array.3`, eval.Wrap(`default value`), eval.EMPTY_MAP))
		return nil
	})
	// Output: default value
}

func ExampleLookup_notFoundDottedMix() {
	lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		fmt.Println(lookup.Lookup(c, `hash.float`, eval.Wrap(`default value`), eval.EMPTY_MAP))
		return nil
	})
	// Output: default value
}

func ExampleLookup_badStringDig() {
	fmt.Println(lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		lookup.Lookup(c, `hash.int.v`, nil, eval.EMPTY_MAP)
		return nil
	}))
	// Output: lookup() Got Integer when a hash-like object was expected to access value using 'v' from key 'hash.int.v'
}

func ExampleLookup_badIntDig() {
	fmt.Println(lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		lookup.Lookup(c, `hash.3`, nil, eval.EMPTY_MAP)
		return nil
	}))
	// Output: lookup() Got Hash[Enum, Data] when a hash-like object was expected to access value using '3' from key 'hash.3'
}

func ExampleLookup2_findFirst() {
	lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		fmt.Println(lookup.Lookup2(c, []string{`first`, `second`}, nil, eval.EMPTY_MAP))
		return nil
	})
	// Output: value of first
}

func ExampleLookup2_findSecond() {
	lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		fmt.Println(lookup.Lookup2(c, []string{`nonexisting`, `second`}, nil, eval.EMPTY_MAP))
		return nil
	})
	// Output: includes 'value of first'
}

func ExampleLookup2_notFoundWithoutDflt() {
	fmt.Println(lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		lookup.Lookup2(c, []string{`nonexisting`, `notthere`}, nil, eval.EMPTY_MAP)
		return nil
	}))
	// Output: lookup() did not find a value for any of the names [nonexisting, notthere]
}

func ExampleLookup2_notFoundDflt() {
	lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		fmt.Println(lookup.Lookup2(c, []string{`nonexisting`, `notthere`}, eval.Wrap(`default value`), eval.EMPTY_MAP))
		return nil
	})
	// Output: default value
}

func ExampleContextCachedValue() {

	cachingProvider := func(c lookup.Context, key string, options eval.KeyedValue) eval.PValue{
		if v, ok := c.CachedValue(key); ok {
			fmt.Printf("Returning cached value for %s\n", key)
			return v
		}
		fmt.Printf("Creating and caching value for %s\n", key)
		v := c.Interpolate(types.WrapString(fmt.Sprintf("generated value for %%{%s}", key)))
		c.Cache(key, v)
		return v
	}

	lookup.DoWithParent(context.Background(), cachingProvider, func(c lookup.Context) error {
		c = c.WithScope(impl.NewScope2(types.WrapHash4(c, map[string]interface{}{
			`a`: `scope 'a'`,
			`b`: `scope 'b'`,
		}))).(lookup.Context)
		fmt.Println(lookup.Lookup(c, `a`, nil, eval.EMPTY_MAP))
		fmt.Println(lookup.Lookup(c, `b`, nil, eval.EMPTY_MAP))
		fmt.Println(lookup.Lookup(c, `a`, nil, eval.EMPTY_MAP))
		fmt.Println(lookup.Lookup(c, `b`, nil, eval.EMPTY_MAP))
		return nil
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
	lookup.DoWithParent(context.Background(), provider, func(c lookup.Context) error {
		v := lookup.Lookup(c, `hash.array.0`, nil, nil)
		fmt.Println(v)
		return nil
	})
	// Output: two
}