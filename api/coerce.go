package api

import (
	"fmt"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
)

// ToMap coerces the given interface{} argument to a dgo.Map and returns it. A panic
// is raised if the argument cannot be coerced into a map.
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
