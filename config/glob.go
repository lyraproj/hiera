package config

import (
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/bmatcuk/doublestar"
	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/tf"
	"github.com/lyraproj/dgo/util"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/hieraapi"
)

type glob string

var globType = tf.NewNamed(
	`hiera.glob`,
	func(v dgo.Value) dgo.Value {
		return glob(v.(dgo.String).GoString())
	},
	func(v dgo.Value) dgo.Value {
		return vf.String(string(v.(glob)))
	},
	reflect.TypeOf(glob(``)),
	reflect.TypeOf((*hieraapi.Location)(nil)).Elem(),
	nil)

func NewGlob(pattern string) hieraapi.Location {
	return glob(pattern)
}

func (g glob) Type() dgo.Type {
	return globType
}

func (g glob) Equals(other interface{}) bool {
	return g == other
}

func (g glob) HashCode() int {
	return util.StringHash(string(g))
}

func (g glob) Exists() bool {
	return false
}

func (g glob) Kind() hieraapi.LocationKind {
	return hieraapi.LcGlob
}

func (g glob) String() string {
	return fmt.Sprintf("glob{pattern:%s}", g.Original())
}

func (g glob) Original() string {
	return string(g)
}

func (g glob) Resolve(ic hieraapi.Invocation, dataDir string) []hieraapi.Location {
	r, _ := ic.InterpolateString(g.Original(), false)
	rp := filepath.Join(dataDir, r.String())
	matches, _ := doublestar.Glob(rp)
	ls := make([]hieraapi.Location, len(matches))
	for i, m := range matches {
		ls[i] = &path{g.Original(), m, true}
	}
	return ls
}

func (g glob) Resolved() string {
	// This should never happen.
	panic(fmt.Errorf(`resolved requested on a glob`))
}
