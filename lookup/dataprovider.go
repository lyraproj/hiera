package lookup

import (
	"fmt"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
)

func CheckedLookup(dp DataProvider, key Key, invocation Invocation, merge MergeStrategy) eval.Value {
	return invocation.Check(key, func() eval.Value { return dp.UncheckedLookup(key, invocation, merge) })
}

type DataProvider interface {
	UncheckedLookup(key Key, invocation Invocation, merge MergeStrategy) eval.Value
	FullName() string
}

type provider struct {
	function Function

	// Set if the designated function has a return type that is equal to or more
	// strict than RichData.
	valueIsValidated bool
}

type dataHashProvider struct {
	provider
	locations []Location
}

func (dh *dataHashProvider) UncheckedLookup(key Key, invocation Invocation, merge MergeStrategy) eval.Value {
	return invocation.WithDataProvider(dh, func() eval.Value {
		return merge.Lookup(dh.locations, invocation, func(location Location) eval.Value {
			return dh.invokeWithLocation(invocation, location, key.Root())
		})
	})
}

func (dh *dataHashProvider) invokeWithLocation(invocation Invocation, location Location, root string) eval.Value {
	if location == nil {
		return dh.lookupKey(invocation, nil, root)
	}
	return invocation.WithLocation(location, func() eval.Value {
		if location.Exist() {
			return dh.lookupKey(invocation, location, root)
		}
		invocation.ReportLocationNotFound()
		panic(notFoundSingleton)
	})
}

func (dh *dataHashProvider) lookupKey(invocation Invocation, location Location, root string) eval.Value {
	value := dh.dataValue(invocation, location, root)
	invocation.ReportFound(root, value)
	return value
}

func (dh *dataHashProvider) dataValue(invocation Invocation, location Location, root string) eval.Value {
	hash := dh.dataHash(invocation, location)
	value, found := hash.Get4(root)
	if !found {
		invocation.ReportNotFound(root)
		panic(notFoundSingleton)
	}
	value = dh.validateDataValue(invocation, value, func() string {
		msg := fmt.Sprintf(`Value for key '%s' in hash returned from %s`, root, dh.FullName())
		if location != nil {
			msg = fmt.Sprintf(`%s, when using location '%s'`, msg, location)
		}
		return msg
	})
	return Interpolate(invocation, value, true)
}

func (dh *dataHashProvider) dataHash(invocation Invocation, location Location) eval.OrderedMap {
	// TODO
	return nil
}

func (dh *provider) validateDataHash(c Context, value eval.Value, pfx func() string) eval.OrderedMap {
	return eval.AssertInstance(pfx, types.DefaultHashType(), value).(eval.OrderedMap)
}

func (dh *provider) validateDataValue(c Context, value eval.Value, pfx func() string) eval.Value {
	if !dh.valueIsValidated {
		eval.AssertInstance(pfx, types.DefaultRichDataType(), value)
	}
	return value
}

func (dh *dataHashProvider) FullName() string {
	return fmt.Sprintf(`data_hash function '%s'`, dh.function.Name())
}

func newDataHashProvider(ic Invocation, he HierarchyEntry) DataProvider {
	// TODO
	return nil
}

func newDataDigProvider(ic Invocation, he HierarchyEntry) DataProvider {
	// TODO
	return nil
}

func newLookupKeyProvider(ic Invocation, he HierarchyEntry) DataProvider {
	// TODO
	return nil
}
