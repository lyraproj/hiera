package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/tf"
	"github.com/lyraproj/dgo/util"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/api"
)

type path struct {
	original string
	resolved string
	exists   bool
}

var pathType = tf.NewNamed(
	`hiera.path`,
	func(v dgo.Value) dgo.Value {
		m := v.(dgo.Map)
		return &path{
			original: m.Get(`original`).(dgo.String).GoString(),
			resolved: m.Get(`resolved`).(dgo.String).GoString(),
			exists:   m.Get(`exists`).(dgo.Boolean).GoBool()}
	},
	func(v dgo.Value) dgo.Value {
		p := v.(*path)
		return vf.Map(
			`original`, p.original,
			`resolved`, p.resolved,
			`exists`, p.exists)
	},
	reflect.TypeOf(&path{}),
	reflect.TypeOf((*api.Location)(nil)).Elem(),
	nil)

// NewPath returns a path Location
func NewPath(original string) api.Location {
	return &path{original: original}
}

func (p *path) Type() dgo.Type {
	return pathType
}

func (p *path) HashCode() int {
	return util.StringHash(p.original)
}

func (p *path) Equals(value interface{}) bool {
	op, ok := value.(*path)
	if ok {
		ok = *p == *op
	}
	return ok
}

func (p *path) Exists() bool {
	return p.exists
}

func (p *path) Kind() api.LocationKind {
	return api.LcPath
}

func (p *path) String() string {
	return fmt.Sprintf("path{ original:%s, resolved:%s, exist:%v}", p.original, p.resolved, p.exists)
}

func (p *path) Resolve(ic api.Invocation, dataDir string) []api.Location {
	r, _ := ic.InterpolateString(p.original, false)
	rp := filepath.Join(dataDir, r.String())
	_, err := os.Stat(rp)
	return []api.Location{&path{p.original, rp, err == nil}}
}

func (p *path) Original() string {
	return p.original
}

func (p *path) Resolved() string {
	return p.resolved
}
