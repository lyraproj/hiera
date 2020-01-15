package config

import (
	"fmt"
	"reflect"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/tf"
	"github.com/lyraproj/dgo/util"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/api"
)

type uri struct {
	original string
	resolved string
}

var uriType = tf.NewNamed(
	`hiera.uri`,
	func(v dgo.Value) dgo.Value {
		m := v.(dgo.Map)
		return &uri{
			original: m.Get(`original`).(dgo.String).GoString(),
			resolved: m.Get(`resolved`).(dgo.String).GoString()}
	},
	func(v dgo.Value) dgo.Value {
		p := v.(*uri)
		return vf.Map(
			`original`, p.original,
			`resolved`, p.resolved)
	},
	reflect.TypeOf(&uri{}),
	reflect.TypeOf((*api.Location)(nil)).Elem(),
	nil)

// NewURI returns a URI Location
func NewURI(original string) api.Location {
	return &uri{original: original}
}

func (u *uri) Type() dgo.Type {
	return uriType
}

func (u *uri) HashCode() int {
	return util.StringHash(u.original)
}

func (u *uri) Equals(value interface{}) bool {
	ou, ok := value.(*uri)
	if ok {
		ok = *u == *ou
	}
	return ok
}

func (u *uri) Exists() bool {
	return true
}

func (u *uri) Kind() api.LocationKind {
	return api.LcURI
}

func (u *uri) String() string {
	return fmt.Sprintf("uri{original:%s, resolved:%s", u.original, u.resolved)
}

func (u *uri) Original() string {
	return u.original
}

func (u *uri) Resolve(ic api.Invocation, dataDir string) []api.Location {
	r, _ := ic.InterpolateString(u.original, false)
	return []api.Location{&uri{u.original, r.String()}}
}

func (u *uri) Resolved() string {
	return u.resolved
}
