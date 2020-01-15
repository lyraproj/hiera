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

type mappedPaths struct {
	// Name of variable that contains an array of strings
	sourceVar string

	// Variable name to use when resolving template
	key string

	// Template containing interpolation of the key
	template string
}

var mappedPathsType = tf.NewNamed(
	`hiera.mappedPaths`,
	func(v dgo.Value) dgo.Value {
		m := v.(dgo.Map)
		return &mappedPaths{
			sourceVar: m.Get(`sourceVar`).(dgo.String).GoString(),
			key:       m.Get(`key`).(dgo.String).GoString(),
			template:  m.Get(`template`).(dgo.String).GoString()}
	},
	func(v dgo.Value) dgo.Value {
		p := v.(*mappedPaths)
		return vf.Map(
			`sourceVar`, p.sourceVar,
			`key`, p.key,
			`template`, p.template)
	},
	reflect.TypeOf(&mappedPaths{}),
	reflect.TypeOf((*api.Location)(nil)).Elem(),
	nil)

// NewMappedPaths returns a Location that initially consist of three strings:
//
// sourceVar: Name of variable that contains an array of strings
//
// key: Variable name to use when resolving template
//
// template: Template containing interpolation of the key
//
func NewMappedPaths(sourceVar, key, template string) api.Location {
	return &mappedPaths{sourceVar: sourceVar, key: key, template: template}
}

func (m *mappedPaths) Equals(other interface{}) bool {
	if om, ok := other.(*mappedPaths); ok {
		return *m == *om
	}
	return false
}

func (m *mappedPaths) Exists() bool {
	return false
}

func (m *mappedPaths) HashCode() int {
	return util.StringHash(m.sourceVar)*31 + util.StringHash(m.key)*31 + util.StringHash(m.template)
}

func (m *mappedPaths) Kind() api.LocationKind {
	return api.LcMappedPaths
}

func (m *mappedPaths) Original() string {
	return m.String()
}

func (m *mappedPaths) String() string {
	return fmt.Sprintf("mapped_paths{sourceVar:%s, key:%s, template:%s}", m.sourceVar, m.key, m.template)
}

func (m *mappedPaths) Type() dgo.Type {
	return mappedPathsType
}

func (m *mappedPaths) Resolve(ic api.Invocation, dataDir string) []api.Location {
	var mappedVars dgo.Array
	v := ic.InterpolateInScope(m.sourceVar, false)
	switch v := v.(type) {
	case dgo.String:
		mappedVars = vf.Values(v)
	case dgo.Array:
		mappedVars = v
	default:
		return []api.Location{}
	}
	paths := make([]api.Location, mappedVars.Len())

	mappedVars.EachWithIndex(func(mv dgo.Value, i int) {
		ic.DoWithScope(&scopeWithVar{ic.Scope(), vf.String(m.key), mv}, func() {
			r, _ := ic.InterpolateString(m.template, false)
			rp := filepath.Join(dataDir, r.String())
			_, err := os.Stat(rp)
			paths[i] = &path{m.template, rp, err == nil}
		})
	})
	return paths
}

func (m *mappedPaths) Resolved() string {
	// This should never happen.
	panic(fmt.Errorf(`resolved requested on mapped paths`))
}

type scopeWithVar struct {
	s dgo.Keyed
	k dgo.Value
	v dgo.Value
}

func (s *scopeWithVar) Get(key interface{}) dgo.Value {
	if s.k.Equals(key) {
		return s.v
	}
	return s.s.Get(key)
}
