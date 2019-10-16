package hieraapi

import (
	"fmt"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
)

func ToMap(argName string, vi interface{}) dgo.Map {
	value := vf.Value(vi)
	if vf.Nil != value {
		if m, ok := value.(dgo.Map); ok {
			return m
		}
		panic(fmt.Errorf(`%s does not represent a map`, argName))
	}
	return vf.Map()
}
