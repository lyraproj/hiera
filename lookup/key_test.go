package lookup_test

import (
	"github.com/puppetlabs/go-hiera/lookup"
	"fmt"
	"context"
	"github.com/puppetlabs/go-evaluator/eval"
)

func ExampleNewKey_simple() {
	key := lookup.NewKey(`simple`)
	fmt.Printf(`%s, %d`, key, len(key.Parts()))
	// Output: simple, 1
}

func ExampleNewKey_dotted() {
	key := lookup.NewKey(`a.b.c`)
	fmt.Printf(`%s, %d`, key, len(key.Parts()))
	// Output: a.b.c, 3
}

func ExampleNewKey_dotted_int() {
	key := lookup.NewKey(`a.3`)
	fmt.Printf(`%T`, key.Parts()[1])
	// Output: int
}

func ExampleNewKey_quoted() {
	key := lookup.NewKey(`'a.b.c'`)
	fmt.Printf(`%s, %d`, key, len(key.Parts()))
	// Output: 'a.b.c', 1
}

func ExampleNewKey_doubleQuoted() {
	key := lookup.NewKey(`"a.b.c"`)
	fmt.Printf(`%s, %d`, key, len(key.Parts()))
	// Output: "a.b.c", 1
}

func ExampleNewKey_quotedDot() {
	key := lookup.NewKey(`a.'b.c'`)
	fmt.Printf(`%s, %d, %s`, key, len(key.Parts()), key.Parts()[1])
	// Output: a.'b.c', 2, b.c
}

func ExampleNewKey_quotedDotX() {
	key := lookup.NewKey(`a.'b.c'.d`)
	fmt.Printf(`%s, %d, %s`, key, len(key.Parts()), key.Parts()[1])
	// Output: a.'b.c'.d, 3, b.c
}

func ExampleNewKey_quotedQuote() {
	key := lookup.NewKey(`a.b.'c"d"e'`)
	fmt.Printf(`%s, %d, %s`, key, len(key.Parts()), key.Parts()[2])
	// Output: a.b.'c"d"e', 3, c"d"e
}

func ExampleNewKey_doubleQuotedQuote() {
	key := lookup.NewKey(`a.b."c'd'e"`)
	fmt.Printf(`%s, %d, %s`, key, len(key.Parts()), key.Parts()[2])
	// Output: a.b."c'd'e", 3, c'd'e
}

func ExampleNewKey_unterminatedQuoted() {
	fmt.Println(eval.Puppet.TryWithParent(context.Background(), func(c eval.Context) error {
		lookup.NewKey(`a.b."c`)
		return nil
	}))
	// Output: Unterminated quote in key 'a.b."c'
}

func ExampleNewKey_empty() {
	fmt.Println(eval.Puppet.TryWithParent(context.Background(), func(c eval.Context) error {
		lookup.NewKey(``)
		return nil
	}))
	// Output: lookup() key '' contains an empty segment
}

func ExampleNewKey_emptySegment() {
	fmt.Println(eval.Puppet.TryWithParent(context.Background(), func(c eval.Context) error {
		lookup.NewKey(`a..b`)
		return nil
	}))
	// Output: lookup() key 'a..b' contains an empty segment
}

func ExampleNewKey_emptySegmentStart() {
	fmt.Println(eval.Puppet.TryWithParent(context.Background(), func(c eval.Context) error {
		lookup.NewKey(`.b`)
		return nil
	}))
	// Output: lookup() key '.b' contains an empty segment
}

func ExampleNewKey_emptySegmentEnd() {
	fmt.Println(eval.Puppet.TryWithParent(context.Background(), func(c eval.Context) error {
		lookup.NewKey(`a.`)
		return nil
	}))
	// Output: lookup() key 'a.' contains an empty segment
}

func ExampleNewKey_firstSegmentIndex() {
	fmt.Println(eval.Puppet.TryWithParent(context.Background(), func(c eval.Context) error {
		lookup.NewKey(`1.a`)
		return nil
	}))
	// Output: lookup() key '1.a' first segment cannot be an index
}
