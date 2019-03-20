package impl

import (
	"fmt"
	"strings"

	"github.com/lyraproj/issue/issue"
)

const (
	HieraDigMismatch                             = `HIERA_DIG_MISMATCH`
	HieraEmptyKeySegment                         = `HIERA_EMPTY_KEY_SEGMENT`
	HieraEndlessRecursion                        = `HIERA_ENDLESS_RECURSION`
	HieraFirstKeySegmentInt                      = `HIERA_FIRST_KEY_SEGMENT_INT`
	HieraHierarchyNameMultiplyDefined            = `HIERA_HIERARCHY_NAME_MULTIPLY_DEFINED`
	HieraInterpolationAliasNotEntireString       = `HIERA_INTERPOLATION_ALIAS_NOT_ENTIRE_STRING`
	HieraInterpolationMethodSyntaxNotAllowed     = `HIERA_INTERPOLATION_METHOD_SYNTAX_NOT_ALLOWED`
	HieraInterpolationUnknownInterpolationMethod = `HIERA_INTERPOLATION_UNKNOWN_INTERPOLATION_METHOD`
	HieraMissingDataProviderFunction             = `HIERA_MISSING_DATA_PROVIDER_FUNCTION`
	HieraMissingRequiredOption                   = `HIERA_MISSING_REQUIRED_OPTION`
	HieraMultipleDataProviderFunctions           = `HIERA_MULTIPLE_DATA_PROVIDER_FUNCTIONS`
	HieraMultipleLocationSpecs                   = `HIERA_MULTIPLE_LOCATION_SPECS`
	HieraNameNotFound                            = `HIERA_NAME_NOT_FOUND`
	HieraNotAnyNameFound                         = `HIERA_NOT_ANY_NAME_FOUND`
	HieraNotInitialized                          = `HIERA_NOT_INITIALIZED`
	HieraOptionReservedByPuppet                  = `HIERA_OPTION_RESERVED_BY_PUPPET`
	HieraUnterminatedQuote                       = `HIERA_UNTERMINATED_QUOTE`
	HieraYamlNotHash                             = `HIERA_YAML_NOT_HASH`
)

func joinNames(v interface{}) string {
	if names, ok := v.([]string); ok {
		return strings.Join(names, `, `)
	}
	return fmt.Sprintf("%v", v)
}

func init() {
	issue.Hard(HieraDigMismatch,
		`lookup() Got %{type} when a hash-like object was expected to access value using '%{segment}' from key '%{key}'`)

	issue.Hard(HieraEmptyKeySegment, `lookup() key '%{key}' contains an empty segment`)

	issue.Hard2(HieraEndlessRecursion, `Recursive lookup detected in [%{name_stack}]`, issue.HF{`name_stack`: joinNames})

	issue.Hard(HieraFirstKeySegmentInt, `lookup() key '%{key}' first segment cannot be an index`)

	issue.Hard(HieraHierarchyNameMultiplyDefined, `Hierarchy name '%{name}' defined more than once`)

	issue.Hard(HieraInterpolationAliasNotEntireString, `'alias' interpolation is only permitted if the expression is equal to the entire string`)

	issue.Hard(HieraInterpolationMethodSyntaxNotAllowed, `Interpolation using method syntax is not allowed in this context`)

	issue.Hard(HieraInterpolationUnknownInterpolationMethod, `Unknown interpolation method '%{name}'`)

	issue.Hard2(HieraMissingDataProviderFunction, `One of %{keys} must be defined in hierarchy '%{name}'`,
		issue.HF{`keys`: joinNames})

	issue.Hard(HieraMissingRequiredOption, `Missing required provider option '%{option}'`)

	issue.Hard2(HieraMultipleDataProviderFunctions, `Only one of %{keys} can be defined in hierarchy '%{name}'`,
		issue.HF{`keys`: joinNames})

	issue.Hard2(HieraMultipleLocationSpecs, `Only one of %{keys} can be defined in hierarchy '%{name}'`,
		issue.HF{`keys`: joinNames})

	issue.Hard(HieraNameNotFound, `lookup() did not find a value for the name '%{name}'`)

	issue.Hard2(HieraNotAnyNameFound, `lookup() did not find a value for any of the names [%{name_list}]`,
		issue.HF{`name_list`: joinNames})

	issue.Hard(HieraNotInitialized, `Given px.Context is not initialized with Hiera`)

	issue.Hard(HieraOptionReservedByPuppet, `Option key '%{key}' used in hierarchy '%{name}' is reserved by Puppet`)

	issue.Hard(HieraUnterminatedQuote, `Unterminated quote in key '%{key}'`)

	issue.Hard(HieraYamlNotHash, `File '%{path}' does not contain a YAML hash`)
}
