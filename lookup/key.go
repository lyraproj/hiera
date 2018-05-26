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
	Dig(eval.Context, eval.PValue) (eval.PValue, bool, error)
	Parts() []interface{}
	Root() string
}

type key struct {
	orig string
	parts []interface{}
}

func NewKey(str string) Key {
	b := bytes.NewBufferString(``)
	return &key{str, parseUnquoted(b, str, []interface{}{})}
}

func (k *key) Dig(c eval.Context, v eval.PValue) (eval.PValue, bool, error) {
	for i := 1; i < len(k.parts); i++ {
		p := k.parts[i]
		if ix, ok := p.(int); ok {
			if iv, ok := v.(*types.ArrayValue); ok {
				if ix >= 0 && ix < iv.Len() {
					v = iv.At(ix)
					continue
				}
				return eval.UNDEF, false, nil
			}
		} else {
			kx := p.(string)
			if kv, ok := v.(*types.HashValue); ok {
				if v, ok = kv.Get4(kx); ok {
					continue
				}
				return eval.UNDEF, false, nil
			}
		}
		eval.Error(c, HIERA_DIG_MISMATCH, issue.H{`type`: v.Type(), `segment`: p, `key`: k.orig})
	}
	return v, true, nil
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

func parseUnquoted(b *bytes.Buffer, key string, parts []interface{}) []interface{} {
	mungePart := func(part string) interface{} {
		if i, err := strconv.ParseInt(part, 10, 32); err == nil {
			return int(i)
		}
		return part
	}

	for i, c := range key {
		switch c {
		case '\'', '"':
			return parseQuoted(b, c, key[i+1:], parts)
		case '.':
			parts = append(parts, mungePart(b.String()))
			b.Reset()
		default:
			b.WriteRune(c)
		}
	}
	return append(parts, mungePart(b.String()))
}

func parseQuoted(b *bytes.Buffer, q rune, key string, parts []interface{}) []interface{} {
	for i, c := range key {
		if c == q {
			return parseUnquoted(b, key[i+1:], parts)
		}
		b.WriteRune(c)
	}
	return parts
}
