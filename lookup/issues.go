package lookup

import (
	"github.com/puppetlabs/go-issues/issue"
	"strings"
	"fmt"
)

const(
	HIERA_DIG_MISMATCH = `LOOKUP_DIG_MISMATCH`
	HIERA_HIERARCHY_NAME_MULTIPLY_DEFINED = `HIERA_HIERARCHY_NAME_MULTIPLY_DEFINED`
	HIERA_NAME_NOT_FOUND = `LOOKUP_NAME_NOT_FOUND`
	HIERA_NOT_ANY_NAME_FOUND = `LOOKUP_NOT_ANY_NAME_FOUND`
	HIERA_MISSING_DATA_PROVIDER_FUNCTION = `HIERA_MISSING_DATA_PROVIDER_FUNCTION`
	HIERA_MULTIPLE_DATA_PROVIDER_FUNCTIONS = `HIERA_MULTIPLE_DATA_PROVIDER_FUNCTIONS`
	HIERA_MULTIPLE_LOCATION_SPECS = `HIERA_MULTIPLE_LOCATION_SPECS`
	HIERA_OPTION_RESERVED_BY_PUPPET = `HIERA_OPTION_RESERVED_BY_PUPPET`
)

func joinNames(v interface{}) string {
	if names, ok := v.([]string); ok {
		return strings.Join(names, `, `)
	}
	return fmt.Sprintf("%v", v)
}

func init() {
	issue.Hard(HIERA_DIG_MISMATCH,
		`lookup() Got %{type} when a hash-like object was expected to access value using '%{segment}' from key '%{key}'`)

	issue.Hard(HIERA_HIERARCHY_NAME_MULTIPLY_DEFINED, `Hierarchy name '%{name}' defined more than once`)

	issue.Hard(HIERA_NAME_NOT_FOUND, `lookup() did not find a value for the name '{name}'`)

	issue.Hard2(HIERA_NOT_ANY_NAME_FOUND, `lookup() did not find a value for any of the names [%{name_list}]'`,
		issue.HF{`name_list`: joinNames})

	issue.Hard2(HIERA_MISSING_DATA_PROVIDER_FUNCTION, `One of %{keys} must be defined in hierarchy '%{name}'`,
		issue.HF{`keys`: joinNames})

	issue.Hard2(HIERA_MULTIPLE_DATA_PROVIDER_FUNCTIONS, `Only one of %{keys} can be defined in hierarchy '%{name}'`,
		issue.HF{`keys`: joinNames})

	issue.Hard2(HIERA_MULTIPLE_LOCATION_SPECS, `Only one of %{keys} can be defined in hierarchy '%{name}'`,
		issue.HF{`keys`: joinNames})

	issue.Hard(HIERA_OPTION_RESERVED_BY_PUPPET, `Option key '%{key}' used in hierarchy '%{name}' is reserved by Puppet`)
}
