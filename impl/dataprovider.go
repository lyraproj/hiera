package impl

import (
	"fmt"
	"github.com/lyraproj/hiera/config"
	"github.com/lyraproj/hiera/lookup"

	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

func CheckedLookup(dp lookup.DataProvider, key lookup.Key, invocation lookup.Invocation, merge lookup.MergeStrategy) (px.Value, bool) {
	return invocation.Check(key, func() (px.Value, bool) { return dp.UncheckedLookup(key, invocation, merge) })
}

type basicProvider struct {
	function config.Function

	// Set if the designated function has a return type that is equal to or more
	// strict than RichData.
	valueIsValidated bool
}

type dataHashProvider struct {
	basicProvider
	locations []lookup.Location
}

func (dh *dataHashProvider) UncheckedLookup(key lookup.Key, invocation lookup.Invocation, merge lookup.MergeStrategy) (px.Value, bool) {
	return invocation.WithDataProvider(dh, func() (px.Value, bool) {
		return merge.Lookup(dh.locations, invocation, func(location lookup.Location) (px.Value, bool) {
			return dh.invokeWithLocation(invocation, location, key.Root())
		})
	})
}

func (dh *dataHashProvider) invokeWithLocation(invocation lookup.Invocation, location lookup.Location, root string) (px.Value, bool) {
	if location == nil {
		return dh.lookupKey(invocation, nil, root)
	}
	return invocation.WithLocation(location, func() (px.Value, bool) {
		if location.Exist() {
			return dh.lookupKey(invocation, location, root)
		}
		invocation.ReportLocationNotFound()
		return nil, false
	})
}

func (dh *dataHashProvider) lookupKey(invocation lookup.Invocation, location lookup.Location, root string) (px.Value, bool) {
	if value, ok := dh.dataValue(invocation, location, root); ok {
		invocation.ReportFound(root, value)
		return value, true
	}
	return nil, false
}

func (dh *dataHashProvider) dataValue(invocation lookup.Invocation, location lookup.Location, root string) (px.Value, bool) {
	hash := dh.dataHash(invocation, location)
	value, found := hash.Get4(root)
	if !found {
		return nil, false
	}
	value = dh.validateDataValue(invocation, value, func() string {
		msg := fmt.Sprintf(`Value for key '%s' in hash returned from %s`, root, dh.FullName())
		if location != nil {
			msg = fmt.Sprintf(`%s, when using location '%s'`, msg, location)
		}
		return msg
	})
	return Interpolate(invocation, value, true), true
}

func (dh *dataHashProvider) dataHash(invocation lookup.Invocation, location lookup.Location) px.OrderedMap {
	// TODO
	return nil
}

func (dh *basicProvider) validateDataHash(c px.Context, value px.Value, pfx func() string) px.OrderedMap {
	return px.AssertInstance(pfx, types.DefaultHashType(), value).(px.OrderedMap)
}

func (dh *basicProvider) validateDataValue(c px.Context, value px.Value, pfx func() string) px.Value {
	if !dh.valueIsValidated {
		px.AssertInstance(pfx, types.DefaultRichDataType(), value)
	}
	return value
}

func (dh *dataHashProvider) FullName() string {
	return fmt.Sprintf(`data_hash function '%s'`, dh.function.Name())
}

func newDataHashProvider(ic lookup.Invocation, he config.HierarchyEntry) lookup.DataProvider {
	// TODO
	return nil
}

func newDataDigProvider(ic lookup.Invocation, he config.HierarchyEntry) lookup.DataProvider {
	// TODO
	return nil
}

func newLookupKeyProvider(ic lookup.Invocation, he config.HierarchyEntry) lookup.DataProvider {
	// TODO
	return nil
}
