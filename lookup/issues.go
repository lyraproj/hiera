package lookup

import (
	"github.com/puppetlabs/go-issues/issue"
	"strings"
	"fmt"
)

const(
	HIERA_DIG_MISMATCH = `HIERA_DIG_MISMATCH`
	HIERA_EMPTY_KEY_SEGMENT = `HIERA_EMPTY_KEY_SEGMENT`
	HIERA_FIRST_KEY_SEGMENT_INT = `HIERA_FIRST_KEY_SEGMENT_INT`
	HIERA_HIERARCHY_NAME_MULTIPLY_DEFINED = `HIERA_HIERARCHY_NAME_MULTIPLY_DEFINED`
	HIERA_INTERPOLATION_ALIAS_NOT_ENTIRE_STRING = `HIERA_INTERPOLATION_ALIAS_NOT_ENTIRE_STRING`
	HIERA_INTERPOLATION_METHOD_SYNTAX_NOT_ALLOWED = `HIERA_INTERPOLATION_METHOD_SYNTAX_NOT_ALLOWED`
	HIERA_INTERPOLATION_UNKNOWN_INTERPOLATION_METHOD = `HIERA_INTERPOLATION_UNKNOWN_INTERPOLATION_METHOD`
	HIERA_MISSING_DATA_PROVIDER_FUNCTION = `HIERA_MISSING_DATA_PROVIDER_FUNCTION`
	HIERA_MULTIPLE_DATA_PROVIDER_FUNCTIONS = `HIERA_MULTIPLE_DATA_PROVIDER_FUNCTIONS`
	HIERA_MULTIPLE_LOCATION_SPECS = `HIERA_MULTIPLE_LOCATION_SPECS`
	HIERA_NAME_NOT_FOUND = `HIERA_NAME_NOT_FOUND`
	HIERA_NOT_ANY_NAME_FOUND = `HIERA_NOT_ANY_NAME_FOUND`
	HIERA_OPTION_RESERVED_BY_PUPPET = `HIERA_OPTION_RESERVED_BY_PUPPET`
	HIERA_UNTERMINATED_QUOTE = `HIERA_UNTERMINATED_QUOTE`
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

	issue.Hard(HIERA_EMPTY_KEY_SEGMENT, `lookup() key '%{key}' contains an empty segment`)

	issue.Hard(HIERA_FIRST_KEY_SEGMENT_INT, `lookup() key '%{key}' first segment cannot be an index`)

	issue.Hard(HIERA_HIERARCHY_NAME_MULTIPLY_DEFINED, `Hierarchy name '%{name}' defined more than once`)

	issue.Hard(HIERA_INTERPOLATION_ALIAS_NOT_ENTIRE_STRING, `'alias' interpolation is only permitted if the expression is equal to the entire string`)

	issue.Hard(HIERA_INTERPOLATION_METHOD_SYNTAX_NOT_ALLOWED, `Interpolation using method syntax is not allowed in this context`)

	issue.Hard(HIERA_INTERPOLATION_UNKNOWN_INTERPOLATION_METHOD, `Unknown interpolation method '%{name}'`)

	issue.Hard2(HIERA_MISSING_DATA_PROVIDER_FUNCTION, `One of %{keys} must be defined in hierarchy '%{name}'`,
		issue.HF{`keys`: joinNames})

	issue.Hard2(HIERA_MULTIPLE_DATA_PROVIDER_FUNCTIONS, `Only one of %{keys} can be defined in hierarchy '%{name}'`,
		issue.HF{`keys`: joinNames})

	issue.Hard2(HIERA_MULTIPLE_LOCATION_SPECS, `Only one of %{keys} can be defined in hierarchy '%{name}'`,
		issue.HF{`keys`: joinNames})

	issue.Hard(HIERA_NAME_NOT_FOUND, `lookup() did not find a value for the name '%{name}'`)

	issue.Hard2(HIERA_NOT_ANY_NAME_FOUND, `lookup() did not find a value for any of the names [%{name_list}]`,
		issue.HF{`name_list`: joinNames})

	issue.Hard(HIERA_OPTION_RESERVED_BY_PUPPET, `Option key '%{key}' used in hierarchy '%{name}' is reserved by Puppet`)

	issue.Hard(HIERA_UNTERMINATED_QUOTE, `Unterminated quote in key '%{key}'`)
}
