package lookup

import (
	"github.com/puppetlabs/go-issues/issue"
	"strings"
	"fmt"
)

const(
	LOOKUP_DIG_MISMATCH = `LOOKUP_DIG_MISMATCH`
	LOOKUP_EMPTY_KEY_SEGMENT = `LOOKUP_EMPTY_KEY_SEGMENT`
	LOOKUP_FIRST_KEY_SEGMENT_INT = `LOOKUP_FIRST_KEY_SEGMENT_INT`
	LOOKUP_NAME_NOT_FOUND = `LOOKUP_NAME_NOT_FOUND`
	LOOKUP_NOT_ANY_NAME_FOUND = `LOOKUP_NOT_ANY_NAME_FOUND`
	LOOKUP_INTERPOLATION_ALIAS_NOT_ENTIRE_STRING = `LOOKUP_INTERPOLATION_ALIAS_NOT_ENTIRE_STRING`
	LOOKUP_INTERPOLATION_METHOD_SYNTAX_NOT_ALLOWED = `LOOKUP_INTERPOLATION_METHOD_SYNTAX_NOT_ALLOWED`
	LOOKUP_INTERPOLATION_UNKNOWN_INTERPOLATION_METHOD = `LOOKUP_INTERPOLATION_UNKNOWN_INTERPOLATION_METHOD`
	LOOKUP_UNTERMINATED_QUOTE = `LOOKUP_UNTERMINATED_QUOTE`
)

func joinNames(v interface{}) string {
	if names, ok := v.([]string); ok {
		return strings.Join(names, `, `)
	}
	return fmt.Sprintf("%v", v)
}

func init() {
	issue.Hard(LOOKUP_DIG_MISMATCH,
		`lookup() Got %{type} when a hash-like object was expected to access value using '%{segment}' from key '%{key}'`)

	issue.Hard(LOOKUP_EMPTY_KEY_SEGMENT, `lookup() key '%{key}' contains an empty segment`)

	issue.Hard(LOOKUP_FIRST_KEY_SEGMENT_INT, `lookup() key '%{key}' first segment cannot be an index`)

	issue.Hard(LOOKUP_NAME_NOT_FOUND, `lookup() did not find a value for the name '{name}'`)

	issue.Hard2(LOOKUP_NOT_ANY_NAME_FOUND, `lookup() did not find a value for any of the names [%{name_list}]`,
		issue.HF{`name_list`: joinNames})

	issue.Hard(LOOKUP_INTERPOLATION_ALIAS_NOT_ENTIRE_STRING, `'alias' interpolation is only permitted if the expression is equal to the entire string`)

	issue.Hard(LOOKUP_INTERPOLATION_METHOD_SYNTAX_NOT_ALLOWED, `Interpolation using method syntax is not allowed in this context`)

	issue.Hard(LOOKUP_INTERPOLATION_UNKNOWN_INTERPOLATION_METHOD, `Unknown interpolation method '%{name}'`)

	issue.Hard(LOOKUP_UNTERMINATED_QUOTE, `Unterminated quote in key '%{key}'`)
}
