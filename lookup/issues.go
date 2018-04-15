package lookup

import (
	"github.com/puppetlabs/go-issues/issue"
	"strings"
	"fmt"
)

const(
	LOOKUP_DIG_MISMATCH = `LOOKUP_DIG_MISMATCH`
	LOOKUP_NAME_NOT_FOUND = `LOOKUP_NAME_NOT_FOUND`
	LOOKUP_NOT_ANY_NAME_FOUND = `LOOKUP_NOT_ANY_NAME_FOUND`
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

	issue.Hard(LOOKUP_NAME_NOT_FOUND, `lookup() did not find a value for the name '{name}'`)

	issue.Hard2(LOOKUP_NOT_ANY_NAME_FOUND, `lookup() did not find a value for any of the names [%{name_list}]'`,
		issue.HF{`name_list`: joinNames})
}
