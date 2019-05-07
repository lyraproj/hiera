package hieraapi

import (
	"fmt"

	"github.com/lyraproj/pcore/px"
)

// A Key is a parsed version of the possibly dot-separated key to lookup. The
// parts of a key will be strings or integers
type Key interface {
	fmt.Stringer

	// Return the result of using this key to dig into the given value. Nil is returned
	// unless the dig was a success
	Dig(px.Value) px.Value

	// Bury is the opposite of Dig. It returns the value that represents what would be found
	// using the root of this key. If this key has one part, the value itself is returned, otherwise
	// a nested chain of single entry hashes is returned.
	Bury(px.Value) px.Value

	// Return the parts of this key. Each part is either a string or an int value
	Parts() []interface{}

	// Return the root key, i.e. the first part.
	Root() string
}

// NewKey parses the given string into a Key
var NewKey func(str string) Key
