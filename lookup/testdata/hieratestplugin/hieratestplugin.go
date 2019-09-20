package main

import (
	"errors"

	"github.com/lyraproj/hierasdk/hiera"
	"github.com/lyraproj/hierasdk/plugin"
	"github.com/lyraproj/hierasdk/register"
	"github.com/lyraproj/hierasdk/vf"
)

func main() {
	register.LookupKey(`test_lookup_key`, lookupOption)
	register.DataHash(`test_refuse_to_die`, refuseToDie)
	register.DataHash(`test_panic`, panicAttack)
	plugin.ServeAndExit()
}

// lookupOption returns the option for the given key or nil if no such option exist
func lookupOption(c hiera.ProviderContext, key string) vf.Data {
	return c.Option(key)
}

// refuseToDie hangs indefinitely
func refuseToDie(c hiera.ProviderContext) vf.Data {
	x := make(chan bool)
	<-x
	return nil
}

// panicAttack panics with an error
func panicAttack(c hiera.ProviderContext) vf.Data {
	panic(errors.New(`dit dit dit daah daah daah dit dit dit`))
}
