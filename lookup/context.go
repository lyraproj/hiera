package lookup

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"context"
	"fmt"
	"github.com/puppetlabs/go-issues/issue"
)

// DoWithParent is like eval.DoWithParent but enables lookup
func DoWithParent(parent context.Context, provider LookupKey, consumer func(eval.Context) error) error {
	return eval.Puppet.DoWithParent(parent, func(c eval.Context) error {
		c.Set(`lookupCache`, NewConcurrentMap(37))
		c.Set(`lookupProvider`, provider)
		return consumer(c)
	})
}

func Lookup(c eval.Context, names []string, dflt eval.PValue, options eval.KeyedValue) (v eval.PValue, err error) {
	cv, ok := c.Get(`lookupCache`)
	if !ok {
		return eval.UNDEF, fmt.Errorf(`lookup called without lookup.Context`)
	}
	pv, _ := c.Get(`lookupProvider`)

	cache := cv.(*ConcurrentMap)
	provider := pv.(LookupKey)

	for _, name := range names {
		v, ok, err = lookupViaCache(c, NewKey(name), options, cache, provider)
		if err != nil || ok {
			return
		}
	}
	if dflt == nil {
		// nil (as opposed to UNDEF) means that no default was provided.
		if len(names) == 1 {
			err = eval.Error(c, LOOKUP_NAME_NOT_FOUND, issue.H{`name`: names[1]})
		} else {
			err = eval.Error(c, LOOKUP_NOT_ANY_NAME_FOUND, issue.H{`name_list`: names})
		}
	} else {
		v = dflt
	}
	return
}

type notFound struct {}

var notFoundSingleton = &notFound{}

func lookupViaCache(c eval.Context, key Key, options eval.KeyedValue, cache *ConcurrentMap, provider LookupKey) (eval.PValue, bool, error) {
	rootKey := key.Root()

	var err error = nil
	val := cache.EnsureSet(rootKey, func() interface{} {
		v, f, e := provider(c, rootKey, options)
		if e != nil {
			err = e
			return notFoundSingleton
		}
		if !f {
			return notFoundSingleton
		}
		return v
	})
	if err != nil {
		return eval.UNDEF, false, err
	}
	if val == notFoundSingleton {
		return eval.UNDEF, false, nil
	}
	return val.(eval.PValue), true, nil
}