package lookup_test

import (
	"github.com/puppetlabs/go-hiera/lookup"
	"fmt"
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

func ExampleNewKey_quotedDot() {
	key := lookup.NewKey(`a.'b.c'`)
	fmt.Printf(`%s, %d, %s`, key, len(key.Parts()), key.Parts()[1])
	// Output: a.'b.c', 2, b.c
}

func ExampleNewKey_quotedQuote() {
	key := lookup.NewKey(`a.b.'c"d"e'`)
	fmt.Printf(`%s, %d, %s`, key, len(key.Parts()), key.Parts()[2])
	// Output: a.b.'c"d"e', 3, c"d"e
}

func ExampleNewKey_empty() {
	key := lookup.NewKey(``)
	fmt.Printf(`%s, %d`, key, len(key.Parts()))
	// Output: , 1
}
