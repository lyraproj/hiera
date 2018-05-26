package lookup

import (
	"fmt"
	"bytes"
	"github.com/puppetlabs/go-evaluator/eval"
	"strconv"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
)

type Key interface {
	fmt.Stringer
	Dig(Context, eval.PValue) (eval.PValue, bool)
	Parts() []interface{}
	Root() string
}

type key struct {
	orig string
	parts []interface{}
}

func NewKey(c eval.Context, str string) Key {
	b := bytes.NewBufferString(``)
	return &key{str, parseUnquoted(c, b, str, str, []interface{}{})}
}

func (k *key) Dig(c Context, v eval.PValue) (eval.PValue, bool) {
	for i := 1; i < len(k.parts); i++ {
		p := k.parts[i]
		if ix, ok := p.(int); ok {
			if iv, ok := v.(*types.ArrayValue); ok {
				if ix >= 0 && ix < iv.Len() {
					v = iv.At(ix)
					continue
				}
				return nil, false
			}
		} else {
			kx := p.(string)
			if kv, ok := v.(*types.HashValue); ok {
				if v, ok = kv.Get4(kx); ok {
					continue
				}
				return nil, false
			}
		}
		panic(eval.Error(c, HIERA_DIG_MISMATCH, issue.H{`type`: eval.GenericValueType(v), `segment`: p, `key`: k.orig}))
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

func parseUnquoted(ctx eval.Context, b *bytes.Buffer, key, part string, parts []interface{}) []interface{} {
	mungePart := func(ix int, part string) interface{} {
		if i, err := strconv.ParseInt(part, 10, 32); err == nil {
			if ix == 0 {
				panic(eval.Error(ctx, HIERA_FIRST_KEY_SEGMENT_INT, issue.H{`key`: key}))
			}
			return int(i)
		}
		if part == `` {
			panic(eval.Error(ctx, HIERA_EMPTY_KEY_SEGMENT, issue.H{`key`: key}))
		}
		return part
	}

	for i, c := range part {
		switch c {
		case '\'', '"':
			return parseQuoted(ctx, b, c, key, part[i+1:], parts)
		case '.':
			parts = append(parts, mungePart(len(parts), b.String()))
			b.Reset()
		default:
			b.WriteRune(c)
		}
	}
	return append(parts, mungePart(len(parts), b.String()))
}

func parseQuoted(ctx eval.Context, b *bytes.Buffer, q rune, key, part string, parts []interface{}) []interface{} {
	for i, c := range part {
		if c == q {
			if i == len(part) - 1 {
				return append(parts, b.String())
			}
			return parseUnquoted(ctx, b, key, part[i+1:], parts)
		}
		b.WriteRune(c)
	}
	panic(eval.Error(ctx, HIERA_UNTERMINATED_QUOTE, issue.H{`key`: key}))
}
