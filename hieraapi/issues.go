package hieraapi

import (
	"fmt"
	"strings"

	"github.com/lyraproj/issue/issue"
)

const (
	DigMismatch                         = `HIERA_DIG_MISMATCH`
	EmptyKeySegment                     = `HIERA_EMPTY_KEY_SEGMENT`
	EndlessRecursion                    = `HIERA_ENDLESS_RECURSION`
	FirstKeySegmentInt                  = `HIERA_FIRST_KEY_SEGMENT_INT`
	HierarchyNameMultiplyDefined        = `HIERA_HIERARCHY_NAME_MULTIPLY_DEFINED`
	InterpolationAliasNotEntireString   = `HIERA_INTERPOLATION_ALIAS_NOT_ENTIRE_STRING`
	InterpolationMethodSyntaxNotAllowed = `HIERA_INTERPOLATION_METHOD_SYNTAX_NOT_ALLOWED`
	JsonNotHash                         = `HIERA_JSON_NOT_HASH`
	KeyNotFound                         = `HIERA_KEY_NOT_FOUND`
	MissingDataProviderFunction         = `HIERA_MISSING_DATA_PROVIDER_FUNCTION`
	MissingRequiredOption               = `HIERA_MISSING_REQUIRED_OPTION`
	MultipleDataProviderFunctions       = `HIERA_MULTIPLE_DATA_PROVIDER_FUNCTIONS`
	MultipleLocationSpecs               = `HIERA_MULTIPLE_LOCATION_SPECS`
	NameNotFound                        = `HIERA_NAME_NOT_FOUND`
	NotAnyNameFound                     = `HIERA_NOT_ANY_NAME_FOUND`
	NotInitialized                      = `HIERA_NOT_INITIALIZED`
	OptionReservedByHiera               = `HIERA_OPTION_RESERVED_BY_HIERA`
	UnterminatedQuote                   = `HIERA_UNTERMINATED_QUOTE`
	UnknownInterpolationMethod          = `HIERA_UNKNOWN_INTERPOLATION_METHOD`
	UnknownMergeStrategy                = `HIERA_UNKNOWN_MERGE_STRATEGY`
	YamlNotHash                         = `HIERA_YAML_NOT_HASH`
)

func joinNames(v interface{}) string {
	if names, ok := v.([]string); ok {
		return strings.Join(names, `, `)
	}
	return fmt.Sprintf("%v", v)
}

func init() {
	issue.Hard(DigMismatch,
		`lookup() Got %{type} when a hash-like object was expected to access value using '%{segment}' from key '%{key}'`)

	issue.Hard(EmptyKeySegment, `lookup() key '%{key}' contains an empty segment`)

	issue.Hard2(EndlessRecursion, `Recursive lookup detected in [%{name_stack}]`, issue.HF{`name_stack`: joinNames})

	issue.Hard(FirstKeySegmentInt, `lookup() key '%{key}' first segment cannot be an index`)

	issue.Hard(HierarchyNameMultiplyDefined, `Hierarchy name '%{name}' defined more than once`)

	issue.Hard(InterpolationAliasNotEntireString, `'alias' interpolation is only permitted if the expression is equal to the entire string`)

	issue.Hard(InterpolationMethodSyntaxNotAllowed, `Interpolation using method syntax is not allowed in this context`)

	issue.Hard(JsonNotHash, `File '%{path}' does not contain a JSON object`)

	issue.Hard(KeyNotFound, `key not found`)

	issue.Hard2(MissingDataProviderFunction, `One of %{keys} must be defined in hierarchy '%{name}'`,
		issue.HF{`keys`: joinNames})

	issue.Hard(MissingRequiredOption, `Missing required provider option '%{option}'`)

	issue.Hard2(MultipleDataProviderFunctions, `Only one of %{keys} can be defined in hierarchy '%{name}'`,
		issue.HF{`keys`: joinNames})

	issue.Hard2(MultipleLocationSpecs, `Only one of %{keys} can be defined in hierarchy '%{name}'`,
		issue.HF{`keys`: joinNames})

	issue.Hard(NameNotFound, `lookup() did not find a value for the name '%{name}'`)

	issue.Hard2(NotAnyNameFound, `lookup() did not find a value for any of the names [%{name_list}]`,
		issue.HF{`name_list`: joinNames})

	issue.Hard(NotInitialized, `Given px.Context is not initialized with Hiera`)

	issue.Hard(OptionReservedByHiera, `Option key '%{key}' used in hierarchy '%{name}' is reserved by Hiera`)

	issue.Hard(UnknownInterpolationMethod, `Unknown interpolation method '%{name}'`)

	issue.Hard(UnknownMergeStrategy, `Unknown merge strategy '%{name}'`)

	issue.Hard(UnterminatedQuote, `Unterminated quote in key '%{key}'`)

	issue.Hard(YamlNotHash, `File '%{path}' does not contain a YAML hash`)
}
