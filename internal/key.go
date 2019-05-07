package internal

import (
	"bytes"
	"strconv"

	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

type key struct {
	orig  string
	parts []interface{}
}

func newKey(str string) hieraapi.Key {
	b := bytes.NewBufferString(``)
	return &key{str, parseUnquoted(b, str, str, []interface{}{})}
}

func (k *key) Dig(v px.Value) px.Value {
	var ok bool
	var ix int
	for i := 1; i < len(k.parts); i++ {
		p := k.parts[i]
		switch vc := v.(type) {
		case *types.Array:
			if ix, ok = p.(int); ok {
				if ix >= 0 && ix < vc.Len() {
					v = vc.At(ix)
					continue
				}
				return nil
			}
		case *types.Hash:
			var kx px.Value
			if ix, ok = p.(int); ok {
				kx = types.WrapInteger(int64(ix))
			} else {
				kx = types.WrapString(p.(string))
			}
			if v, ok = vc.Get(kx); ok {
				continue
			}
			return nil
		}
		panic(px.Error(hieraapi.DigMismatch, issue.H{`type`: px.GenericValueType(v), `segment`: p, `key`: k.orig}))
	}
	return v
}

func (k *key) Bury(value px.Value) px.Value {
	for i := len(k.parts) - 1; i > 0; i-- {
		p := k.parts[i]
		var kx px.Value
		if ix, ok := p.(int); ok {
			kx = types.WrapInteger(int64(ix))
		} else {
			kx = types.WrapString(p.(string))
		}
		value = types.WrapHash([]*types.HashEntry{types.WrapHashEntry(kx, value)})
	}
	return value
}

func (k *key) Parts() []interface{} {
	return k.parts
}

func (k *key) String() string {
	return k.orig
}

func (k *key) Root() string {
	return k.parts[0].(string)
}

func parseUnquoted(b *bytes.Buffer, key, part string, parts []interface{}) []interface{} {
	mungedPart := func(ix int, part string) interface{} {
		if i, err := strconv.ParseInt(part, 10, 32); err == nil {
			if ix == 0 {
				panic(px.Error(hieraapi.FirstKeySegmentInt, issue.H{`key`: key}))
			}
			return int(i)
		}
		if part == `` {
			panic(px.Error(hieraapi.EmptyKeySegment, issue.H{`key`: key}))
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
			b.WriteRune(c)
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
		b.WriteRune(c)
	}
	panic(px.Error(hieraapi.UnterminatedQuote, issue.H{`key`: key}))
}
