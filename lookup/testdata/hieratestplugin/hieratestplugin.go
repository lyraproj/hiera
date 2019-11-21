package main

import (
	"errors"

	"github.com/lyraproj/dgo/vf"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/hierasdk/hiera"
	"github.com/lyraproj/hierasdk/plugin"
	"github.com/lyraproj/hierasdk/register"
)

func main() {
	register.LookupKey(`test_lookup_key`, lookupOption)
	register.DataHash(`test_data_hash`, sampleHash)
	register.DataHash(`test_refuse_to_die`, refuseToDie)
	register.DataHash(`test_panic`, panicAttack)
	plugin.ServeAndExit()
}

// lookupOption returns the option for the given key or nil if no such option exist
func lookupOption(c hiera.ProviderContext, key string) dgo.Value {
	return c.Option(key)
}

func sampleHash(c hiera.ProviderContext) dgo.Map {
	h := c.Option(`the_hash`).(dgo.Map)
	if h.Get(`c`) != nil {
		h = h.Merge(vf.Map(`d`, `interpolate c is %{lookup("c")}`))
	}
	return h
}

// refuseToDie hangs indefinitely
func refuseToDie(c hiera.ProviderContext) dgo.Map {
	x := make(chan bool)
	<-x
	return nil
}

// panicAttack panics with an error
func panicAttack(c hiera.ProviderContext) dgo.Map {
	panic(errors.New(`dit dit dit daah daah daah dit dit dit`))
}
