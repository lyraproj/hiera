package hieraapi_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"

	// Ensure internal is initialized
	_ "github.com/lyraproj/hiera/internal"
)

func ExampleNewKey_simple() {
	key := hieraapi.NewKey(`simple`)
	fmt.Printf(`%s, %d`, key, len(key.Parts()))
	// Output: simple, 1
}

func ExampleNewKey_dotted() {
	key := hieraapi.NewKey(`a.b.c`)
	fmt.Printf(`%s, %d`, key, len(key.Parts()))
	// Output: a.b.c, 3
}

func ExampleNewKey_dotted_int() {
	key := hieraapi.NewKey(`a.3`)
	fmt.Printf(`%T`, key.Parts()[1])
	// Output: int
}

func ExampleNewKey_quoted() {
	key := hieraapi.NewKey(`'a.b.c'`)
	fmt.Printf(`%s, %d`, key, len(key.Parts()))
	// Output: 'a.b.c', 1
}

func ExampleNewKey_doubleQuoted() {
	key := hieraapi.NewKey(`"a.b.c"`)
	fmt.Printf(`%s, %d`, key, len(key.Parts()))
	// Output: "a.b.c", 1
}

func ExampleNewKey_quotedDot() {
	key := hieraapi.NewKey(`a.'b.c'`)
	fmt.Printf(`%s, %d, %s`, key, len(key.Parts()), key.Parts()[1])
	// Output: a.'b.c', 2, b.c
}

func ExampleNewKey_quotedDotX() {
	key := hieraapi.NewKey(`a.'b.c'.d`)
	fmt.Printf(`%s, %d, %s`, key, len(key.Parts()), key.Parts()[1])
	// Output: a.'b.c'.d, 3, b.c
}

func ExampleNewKey_quotedQuote() {
	key := hieraapi.NewKey(`a.b.'c"d"e'`)
	fmt.Printf(`%s, %d, %s`, key, len(key.Parts()), key.Parts()[2])
	// Output: a.b.'c"d"e', 3, c"d"e
}

func ExampleNewKey_doubleQuotedQuote() {
	key := hieraapi.NewKey(`a.b."c'd'e"`)
	fmt.Printf(`%s, %d, %s`, key, len(key.Parts()), key.Parts()[2])
	// Output: a.b."c'd'e", 3, c'd'e
}

func ExampleNewKey_unterminatedQuoted() {
	printErr(pcore.TryWithParent(context.Background(), func(c px.Context) error {
		hieraapi.NewKey(`a.b."c`)
		return nil
	}))
	// Output: Unterminated quote in key 'a.b."c'
}

func ExampleNewKey_empty() {
	printErr(pcore.TryWithParent(context.Background(), func(c px.Context) error {
		hieraapi.NewKey(``)
		return nil
	}))
	// Output: lookup() key '' contains an empty segment
}

func ExampleNewKey_emptySegment() {
	printErr(pcore.TryWithParent(context.Background(), func(c px.Context) error {
		hieraapi.NewKey(`a..b`)
		return nil
	}))
	// Output: lookup() key 'a..b' contains an empty segment
}

func ExampleNewKey_emptySegmentStart() {
	printErr(pcore.TryWithParent(context.Background(), func(c px.Context) error {
		hieraapi.NewKey(`.b`)
		return nil
	}))
	// Output: lookup() key '.b' contains an empty segment
}

func ExampleNewKey_emptySegmentEnd() {
	printErr(pcore.TryWithParent(context.Background(), func(c px.Context) error {
		hieraapi.NewKey(`a.`)
		return nil
	}))
	// Output: lookup() key 'a.' contains an empty segment
}

func ExampleNewKey_firstSegmentIndex() {
	printErr(pcore.TryWithParent(context.Background(), func(c px.Context) error {
		hieraapi.NewKey(`1.a`)
		return nil
	}))
	// Output: lookup() key '1.a' first segment cannot be an index
}

func printErr(e error) {
	s := e.Error()
	if ix := strings.Index(s, ` (file: `); ix > 0 {
		s = s[0:ix]
	}
	fmt.Println(s)
}

func ExampleKey_Bury_dotted() {
	v := hieraapi.NewKey(`a.b.c`).Bury(types.WrapString(`x`))
	fmt.Println(v)
	// Output: {'b' => {'c' => 'x'}}
}

func ExampleKey_Bury_dotted_int() {
	v := hieraapi.NewKey(`a.3`).Bury(types.WrapString(`x`))
	fmt.Println(v)
	// Output: {3 => 'x'}
}

func ExampleKey_Bury_untouched() {
	v := hieraapi.NewKey(`a`).Bury(types.WrapString(`x`))
	fmt.Println(v)
	// Output: x
}
