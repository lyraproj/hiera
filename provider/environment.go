package provider

import (
	"os"
	"strings"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hierasdk/hiera"
)

// Environment is a LookupKey function that performs a lookup in the current environment. The key can either be just
// "env" in which case all current environment variables will be returned as an OrderedMap, or
// prefixed with "env::" in which case the rest of the key is interpreted as the environment variable
// to look for.
func Environment(_ hiera.ProviderContext, key string) dgo.Value {
	if key == `env` {
		env := os.Environ()
		em := vf.MapWithCapacity(len(env))
		for _, ev := range env {
			if ei := strings.IndexRune(ev, '='); ei > 0 {
				em.Put(ev[:ei], ev[ei+1:])
			}
		}
		return em
	}
	if strings.HasPrefix(key, `env::`) {
		// Rest of key is name of environment
		if v, ok := os.LookupEnv(key[5:]); ok {
			return vf.String(v)
		}
	}
	return nil
}
