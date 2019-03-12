package impl

import (
	"bytes"
	"github.com/lyraproj/hiera/lookup"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"strconv"
)

type key struct {
	orig  string
	parts []interface{}
}

func NewKey(str string) lookup.Key {
	b := bytes.NewBufferString(``)
	return &key{str, parseUnquoted(b, str, str, []interface{}{})}
}

func (k *key) Dig(v px.Value) (px.Value, bool) {
	for i := 1; i < len(k.parts); i++ {
		p := k.parts[i]
		if ix, ok := p.(int); ok {
			if iv, ok := v.(*types.Array); ok {
				if ix >= 0 && ix < iv.Len() {
					v = iv.At(ix)
					continue
				}
				return nil, false
			}
		} else {
			kx := p.(string)
			if kv, ok := v.(*types.Hash); ok {
				if v, ok = kv.Get4(kx); ok {
					continue
				}
				return nil, false
			}
		}
		panic(px.Error(HIERA_DIG_MISMATCH, issue.H{`type`: px.GenericValueType(v), `segment`: p, `key`: k.orig}))
	}
	return v, true
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
	mungePart := func(ix int, part string) interface{} {
		if i, err := strconv.ParseInt(part, 10, 32); err == nil {
			if ix == 0 {
				panic(px.Error(HIERA_FIRST_KEY_SEGMENT_INT, issue.H{`key`: key}))
			}
			return int(i)
		}
		if part == `` {
			panic(px.Error(HIERA_EMPTY_KEY_SEGMENT, issue.H{`key`: key}))
		}
		return part
	}

	for i, c := range part {
		switch c {
		case '\'', '"':
			return parseQuoted(b, c, key, part[i+1:], parts)
		case '.':
			parts = append(parts, mungePart(len(parts), b.String()))
			b.Reset()
		default:
			b.WriteRune(c)
		}
	}
	return append(parts, mungePart(len(parts), b.String()))
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
	panic(px.Error(HIERA_UNTERMINATED_QUOTE, issue.H{`key`: key}))
}
