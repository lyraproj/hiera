package api

// FunctionKind denotes what kind of function this is.
type FunctionKind string

// A Function is a definition of a Hiera lookup function, i.e. a data_dig, data_hash, or lookup_key.
type Function interface {
	// FunctionKind returns the function kind
	Kind() FunctionKind

	// Name returns the name of the function
	Name() string

	// Resolve resolves the function on behalf of the given invocation
	Resolve(ic Invocation) (Function, bool)
}

// KindDataDig is the function kind for data_dig functions
const KindDataDig = FunctionKind(`data_dig`)

// KindDataHash is the function kind for data_dig functions
const KindDataHash = FunctionKind(`data_hash`)

// KindLookupKey is the function kind for data_dig functions
const KindLookupKey = FunctionKind(`lookup_key`)
