package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

type path struct {
	original string
	resolved string
	exist    bool
}

func (p *path) Exist() bool {
	return p.exist
}

func (p *path) Kind() hieraapi.LocationKind {
	return hieraapi.LcPath
}

func (p *path) String() string {
	return fmt.Sprintf("path{ original:%s, resolved:%s, exist:%v}", p.original, p.resolved, p.exist)
}

func (p *path) Resolve(ic hieraapi.Invocation, dataDir string) []hieraapi.Location {
	r, _ := interpolateString(ic, p.original, false)
	rp := filepath.Join(dataDir, r.String())
	_, err := os.Stat(rp)
	return []hieraapi.Location{&path{p.original, rp, err == nil}}
}

func (p *path) Original() string {
	return p.original
}

func (p *path) Resolved() string {
	return p.resolved
}

type glob struct {
	pattern string
}

func (g *glob) Exist() bool {
	return false
}

func (g *glob) Kind() hieraapi.LocationKind {
	return hieraapi.LcGlob
}

func (g *glob) String() string {
	return fmt.Sprintf("glob{pattern:%s}", g.pattern)
}

func (g *glob) Original() string {
	return g.pattern
}

func (g *glob) Resolve(ic hieraapi.Invocation, dataDir string) []hieraapi.Location {
	r, _ := interpolateString(ic, g.pattern, false)
	rp := filepath.Join(dataDir, r.String())
	matches, _ := doublestar.Glob(rp)
	ls := make([]hieraapi.Location, len(matches))
	for i, m := range matches {
		ls[i] = &path{g.pattern, m, true}
	}
	return ls
}

func (g *glob) Resolved() string {
	// This should never happen.
	panic(fmt.Errorf(`resolved requested on a glob`))
}

type uri struct {
	original string
	resolved string
}

func (u *uri) Exist() bool {
	return true
}

func (u *uri) Kind() hieraapi.LocationKind {
	return hieraapi.LcUri
}

func (u *uri) String() string {
	return fmt.Sprintf("uri{original:%s, resolved:%s", u.original, u.resolved)
}

func (u *uri) Original() string {
	return u.original
}

func (u *uri) Resolve(ic hieraapi.Invocation, dataDir string) []hieraapi.Location {
	r, _ := interpolateString(ic, u.original, false)
	return []hieraapi.Location{&uri{u.original, r.String()}}
}

func (u *uri) Resolved() string {
	return u.resolved
}

type mappedPaths struct {
	// Name of variable that contains an array of strings
	sourceVar string

	// Variable name to use when resolving template
	key string

	// Template containing interpolation of the key
	template string
}

func (m *mappedPaths) Exist() bool {
	return false
}

func (m *mappedPaths) Kind() hieraapi.LocationKind {
	return hieraapi.LcMappedPaths
}

func (m *mappedPaths) Original() string {
	return m.String()
}

func (m *mappedPaths) String() string {
	return fmt.Sprintf("mapped_paths{sourceVar:%s, key:%s, template:%s}", m.sourceVar, m.key, m.template)
}

type scopeWithVar struct {
	s px.Keyed
	k px.Value
	v px.Value
}

func (s *scopeWithVar) Get(key px.Value) (px.Value, bool) {
	if s.k.Equals(key, nil) {
		return s.v, true
	}
	return s.s.Get(key)
}

func (m *mappedPaths) Resolve(ic hieraapi.Invocation, dataDir string) []hieraapi.Location {
	var mappedVars px.List
	v := resolveInScope(ic, m.sourceVar, false)
	switch v := v.(type) {
	case px.StringValue:
		mappedVars = types.SingletonArray(v)
	case px.List:
		mappedVars = v
	default:
		return []hieraapi.Location{}
	}
	paths := make([]hieraapi.Location, mappedVars.Len())

	mappedVars.EachWithIndex(func(mv px.Value, i int) {
		ic.DoWithScope(&scopeWithVar{ic.Scope(), types.WrapString(m.key), mv}, func() {
			r, _ := interpolateString(ic, m.template, false)
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
