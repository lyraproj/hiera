package api

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/tf"
	"github.com/lyraproj/dgo/vf"
)

// A Key is a parsed version of the possibly dot-separated key to lookup. The
// parts of a key will be strings or integers
type (
	Key interface {
		dgo.Value

		// Return the result of using this key to dig into the given value. Nil is returned
		// unless the dig was a success
		Dig(Invocation, dgo.Value) dgo.Value

		// Bury is the opposite of Dig. It returns the value that represents what would be found
		// using the root of this key. If this key has one part, the value itself is returned, otherwise
		// a nested chain of single entry hashes is returned.
		Bury(dgo.Value) dgo.Value

		// Return the parts of this key. Each part is either a string or an int value
		Parts() []interface{}

		// Return the root key, i.e. the first part.
		Root() string

		// Source returns the string that this key was created from
		Source() string
	}

	key struct {
		source string
		parts  []interface{}
	}
)

// NewKey parses the given string into a Key
func NewKey(str string) Key {
	b := bytes.NewBufferString(``)
	return &key{str, parseUnquoted(b, str, str, []interface{}{})}
}

var keyType = tf.NewNamed(`hiera.key`,
	func(v dgo.Value) dgo.Value {
		return NewKey(v.String())
	},
	func(v dgo.Value) dgo.Value {
		return vf.String(v.(*key).source)
	},
	reflect.TypeOf(&key{}),
	reflect.TypeOf((*Key)(nil)).Elem(), nil)

func (k *key) Bury(value dgo.Value) dgo.Value {
	for i := len(k.parts) - 1; i > 0; i-- {
		p := k.parts[i]
		var kx dgo.Value
		if ix, ok := p.(int); ok {
			kx = vf.Integer(int64(ix))
		} else {
			kx = vf.String(p.(string))
		}
		value = vf.Map(kx, value)
	}
	return value
}

func (k *key) Dig(ic Invocation, v dgo.Value) dgo.Value {
	t := len(k.parts)
	if t == 1 {
		return v
	}

	return ic.WithSubLookup(k, func() dgo.Value {
		for i := 1; i < t; i++ {
			p := k.parts[i]
			v = ic.WithSegment(p, func() dgo.Value {
				switch vc := v.(type) {
				case dgo.Array:
					if ix, ok := p.(int); ok {
						if ix >= 0 && ix < vc.Len() {
							v = vc.Get(ix)
							ic.ReportFound(p, v)
							return v
						}
					}
				case dgo.Map:
					var kx dgo.Value
					if ix, ok := p.(int); ok {
						kx = vf.Integer(int64(ix))
					} else {
						kx = vf.String(p.(string))
					}
					if v := vc.Get(kx); v != nil {
						ic.ReportFound(p, v)
						return v
					}
				}
				ic.ReportNotFound(p)
				return nil
			})
			if v == nil {
				break
			}
		}
		return v
	})
}

func (k *key) Equals(value interface{}) bool {
	if ov, ok := value.(*key); ok {
		return k.source == ov.source
	}
	return false
}

func (k *key) HashCode() int {
	panic("implement me")
}

func (k *key) Parts() []interface{} {
	return k.parts
}

func (k *key) Type() dgo.Type {
	return keyType
}

func (k *key) Root() string {
	return k.parts[0].(string)
}

func (k *key) Source() string {
	return k.source
}

func (k *key) String() string {
	return k.Type().(dgo.NamedType).ValueString(k)
}

func parseUnquoted(b *bytes.Buffer, key, part string, parts []interface{}) []interface{} {
	mungedPart := func(ix int, part string) interface{} {
		if i, err := strconv.ParseInt(part, 10, 32); err == nil {
			if ix == 0 {
				panic(fmt.Errorf(`key '%s' first segment cannot be an index`, key))
			}
			return int(i)
		}
		if part == `` {
			panic(fmt.Errorf(`key '%s' contains an empty segment`, key))
		}
		return part
	}

	for i, c := range part {
		switch c {
		case '\'', '"':
			return parseQuoted(b, c, key, part[i+1:], parts)
		case '.':
			parts = append(parts, mungedPart(len(parts), b.String()))
			b.Reset()
		default:
			_, _ = b.WriteRune(c)
		}
	}
	return append(parts, mungedPart(len(parts), b.String()))
}

func parseQuoted(b *bytes.Buffer, q rune, key, part string, parts []interface{}) []interface{} {
	for i, c := range part {
		if c == q {
			if i == len(part)-1 {
				return append(parts, b.String())
			}
			return parseUnquoted(b, key, part[i+1:], parts)
		}
		_, _ = b.WriteRune(c)
	}
	panic(fmt.Errorf(`unterminated quote in key '%s'`, key))
}
