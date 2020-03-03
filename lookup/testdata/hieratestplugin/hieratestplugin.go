package main

import (
	"encoding/json"
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
	register.DataHash(`test_tf_simulation`, tfSimulation)
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

func tfSimulation(c hiera.ProviderContext) dgo.Map {
	var state map[string]interface{}
	err := json.Unmarshal([]byte(`{
    "dns_resource_group_name": "cbuk-shared-sharedproduction-dns-uksouth",
    "dns_zones": {
			"cbinnovation.uk": {
        "id": "/subscriptions/xxx/resourceGroups/cbuk-shared-sharedproduction-dns-uksouth/providers/Microsoft.Network/dnszones/cbinnovation.uk",
        "max_number_of_record_sets": 10000,
        "name": "cbinnovation.uk",
        "name_servers": [
          "ns1-04.azure-dns.com.",
          "ns2-04.azure-dns.net.",
          "ns3-04.azure-dns.org.",
          "ns4-04.azure-dns.info."
        ],
        "number_of_record_sets": 2,
        "registration_virtual_network_ids": null,
        "resolution_virtual_network_ids": null,
        "resource_group_name": "cbuk-shared-sharedproduction-dns-uksouth",
        "tags": {
          "environment": "sharedproduction",
          "organisation": "cbuk",
          "system": "shared",
          "technology": "dns"
        },
        "zone_type": "Public"
      }
    }
  }`), &state)
	if err != nil {
		panic(err)
	}
	v := vf.MutableMap()
	for k, os := range state {
		v.Put(k, os)
	}
	return vf.Map(`root`, v)
}
