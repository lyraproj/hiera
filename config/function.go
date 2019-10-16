package config

import (
	"github.com/lyraproj/hiera/hieraapi"
)

type (
	function struct {
		kind hieraapi.FunctionKind
		name string
	}
)

func (f *function) Kind() hieraapi.FunctionKind {
	return f.kind
}

func (f *function) Name() string {
	return f.name
}

func (f *function) Resolve(ic hieraapi.Invocation) (hieraapi.Function, bool) {
	if n, changed := ic.InterpolateString(f.name, false); changed {
		return &function{f.kind, n.String()}, true
	}
	return f, false
}
