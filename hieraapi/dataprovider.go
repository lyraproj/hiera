package hieraapi

import "github.com/lyraproj/pcore/px"

type DataProvider interface {
	UncheckedLookup(key Key, invocation Invocation, merge MergeStrategy) px.Value
	FullName() string
}
