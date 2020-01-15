package config

import (
	"github.com/lyraproj/hiera/api"
)

type (
	function struct {
		kind api.FunctionKind
		name string
	}
)

func (f *function) Kind() api.FunctionKind {
	return f.kind
}

func (f *function) Name() string {
	return f.name
}

func (f *function) Resolve(ic api.Invocation) (api.Function, bool) {
	if n, changed := ic.InterpolateString(f.name, false); changed {
		return &function{f.kind, n.String()}, true
	}
	return f, false
}
