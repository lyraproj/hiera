package lookup

import (
	"fmt"
	"bytes"
)

type Key interface {
	fmt.Stringer
	Parts() []string
	Root() string
}

type key struct {
	orig string
	parts []string
}

func NewKey(str string) Key {
	b := bytes.NewBufferString(``)
	return &key{str, parseUnquoted(b, str, []string{})}
}

func (k *key) Parts() []string {
	return k.parts
}

func (k *key) String() string {
	return k.orig
}

func (k *key) Root() string {
	return k.parts[0]
}

func parseUnquoted(b *bytes.Buffer, key string, parts []string) []string {
	for i, c := range key {
		switch c {
		case '\'', '"':
			return parseQuoted(b, c, key[i+1:], parts)
		case '.':
			parts = append(parts, b.String())
			b.Reset()
		default:
			b.WriteRune(c)
		}
	}
	return append(parts, b.String())
}

func parseQuoted(b *bytes.Buffer, q rune, key string, parts []string) []string {
	for i, c := range key {
		if c == q {
			return parseUnquoted(b, key[i+1:], parts)
		}
		b.WriteRune(c)
	}
	return parts
}
