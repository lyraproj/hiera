package lookup

import (
	"regexp"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"strings"
	"github.com/puppetlabs/go-issues/issue"
)

var iplPattern = regexp.MustCompile(`%\{[^\}]*\}`)
var emptyInterpolations = map[string]bool {
	``: true,
	`::`: true,
	`""`: true,
	"''": true,
	`"::"`: true,
	"'::'": true,
}

func Interpolate(c eval.Context, value eval.PValue, allowMethods bool) (result eval.PValue, err error) {
	result, _, err = doInterpolate(c, value, allowMethods)
	return
}

func doInterpolate(ctx eval.Context, value eval.PValue, allowMethods bool) (eval.PValue, bool, error) {
	if s, ok := value.(*types.StringValue); ok {
		return interpolateString(ctx, s.String(), allowMethods)
	}
	if a, ok := value.(*types.ArrayValue); ok {
		cp := a.AppendTo(make([]eval.PValue, 0, a.Len()))
		changed := false
		for i, e := range cp {
			v, c, err := doInterpolate(ctx, e, allowMethods)
			if err != nil {
				return nil, false, err
			}
			if c {
				changed = true
				cp[i] = v
			}
		}
		if changed {
			a = types.WrapArray(cp)
		}
		return a, changed, nil
	}
	if h, ok := value.(*types.HashValue); ok {
		cp := h.AppendEntriesTo(make([]*types.HashEntry, 0, h.Len()))
		changed := false
		for i, e := range cp {
			ec := false
			k, c, err := doInterpolate(ctx, e.Key(), allowMethods)
			if err != nil {
				return nil, false, err
			}
			if c {
				ec = true
			}
			v, c, err := doInterpolate(ctx, e.Value(), allowMethods)
			if err != nil {
				return nil, false, err
			}
			if c {
				ec = true
			}
			if ec {
				changed = true
				cp[i] = types.WrapHashEntry(k, v)
			}
		}
		if changed {
			h = types.WrapHash(cp)
		}
		return h, changed, nil
	}
	return value, false, nil
}

const empty = 0
const scopeMethod = 1
const aliasMethod = 2
const lookupMethod = 3
const literalMethod = 4

var methodMatch = regexp.MustCompile(`^(\w+)\((?:["]([^"]+)["]|[']([^']+)['])\)$`)

func getMethodAndData(c eval.Context, expr string, allowMethods bool) (int, string, error) {
	if len(expr) == 0 {
		return empty, ``, nil
	}
	if groups := methodMatch.FindStringSubmatch(expr); groups != nil {
		if !allowMethods {
			return empty, ``, eval.Error(c, LOOKUP_INTERPOLATION_METHOD_SYNTAX_NOT_ALLOWED, issue.NO_ARGS)
		}
		data := groups[2]
		if data == `` {
			data = groups[3]
		}
		switch groups[1] {
		case `alias`:
			return aliasMethod, data, nil
		case `hiera`, `lookup`:
			return lookupMethod, data, nil
		case `scope`:
			return scopeMethod, data, nil
		default:
			return empty, ``, eval.Error(c, LOOKUP_INTERPOLATION_UNKNOWN_INTERPOLATION_METHOD, issue.H{`name`: groups[1]})
		}
	} else {
		return scopeMethod, expr, nil
	}
}

func interpolateString(c eval.Context, str string, allowMethods bool) (result eval.PValue, changed bool, err error) {
	if strings.Index(str, `%{`) < 0 {
		result = types.WrapString(str)
		return
	}
	str = iplPattern.ReplaceAllStringFunc(str, func (match string) string {
		expr := strings.TrimSpace(match[2:len(match)-1])
		if emptyInterpolations[expr] {
			return ``
		}
		var methodKey int
		methodKey, expr, err = getMethodAndData(c, expr, allowMethods)
		if err != nil {
			return ``
		}
		if methodKey == aliasMethod && match != str {
			err = eval.Error(c, LOOKUP_INTERPOLATION_ALIAS_NOT_ENTIRE_STRING, issue.NO_ARGS)
			return ``
		}

		switch methodKey {
		case literalMethod:
			return expr
		case scopeMethod:
			if val, ok := c.Scope().Get(expr); ok {
				val, _, err = doInterpolate(c, val, allowMethods)
				if err != nil {
					return ``
				}
				return val.String()
			}
			return ``
		default:
			val, err := Lookup(c, []string{expr}, eval.UNDEF, eval.EMPTY_MAP)
			if err != nil {
				return ``
			}
			if methodKey == aliasMethod {
				result = val
				return ``
			}
			return val.String()
		}
	})
	if err != nil {
		return nil, false, err
	}
	changed = true
	if result == nil {
		result = types.WrapString(str)
	}
	return

}
/*
require 'hiera/scope'
require_relative 'sub_lookup'
module Puppet::Pops
module Lookup
# Adds support for interpolation expressions. The expressions may contain keys that uses dot-notation
# to further navigate into hashes and arrays
#
# @api public
module Interpolation
  include SubLookup

  # @param value [Object] The value to interpolate
  # @param context [Context] The current lookup context
  # @param allow_methods [Boolean] `true` if interpolation expression that contains lookup methods are allowed
  # @return [Object] the result of resolving all interpolations in the given value
  # @api public
  def interpolate(value, context, allow_methods)
    case value
    when String
      value.index('%{').nil? ? value : interpolate_string(value, context, allow_methods)
    when Array
      value.map { |element| interpolate(element, context, allow_methods) }
    when Hash
      result = {}
      value.each_pair { |k, v| result[interpolate(k, context, allow_methods)] = interpolate(v, context, allow_methods) }
      result
    else
      value
    end
  end

  private

  EMPTY_INTERPOLATIONS = {
    '' => true,
    '::' => true,
    '""' => true,
    "''" => true,
    '"::"' => true,
    "'::'" => true
  }.freeze

  # Matches a key that is quoted using a matching pair of either single or double quotes.
  QUOTED_KEY = /^(?:"([^"]+)"|'([^']+)')$/

  def interpolate_string(subject, context, allow_methods)
    lookup_invocation = context.is_a?(Invocation) ? context : context.invocation
    lookup_invocation.with(:interpolate, subject) do
      subject.gsub(/%\{([^\}]*)\}/) do |match|
        expr = $1
        # Leading and trailing spaces inside an interpolation expression are insignificant
        expr.strip!
        value = nil
        unless EMPTY_INTERPOLATIONS[expr]
          method_key, key = get_method_and_data(expr, allow_methods)
          is_alias = method_key == :alias

          # Alias is only permitted if the entire string is equal to the interpolate expression
          fail(Issues::HIERA_INTERPOLATION_ALIAS_NOT_ENTIRE_STRING) if is_alias && subject != match
          value = interpolate_method(method_key).call(key, lookup_invocation, subject)

          # break gsub and return value immediately if this was an alias substitution. The value might be something other than a String
          return value if is_alias

          value = lookup_invocation.check(method_key == :scope ? "scope:#{key}" : key) { interpolate(value, lookup_invocation, allow_methods) }
        end
        value.nil? ? '' : value
      end
    end
  end

  def interpolate_method(method_key)
    @@interpolate_methods ||= begin
      global_lookup = lambda do |key, lookup_invocation, _|
        scope = lookup_invocation.scope
        if scope.is_a?(Hiera::Scope) && !lookup_invocation.global_only?
          # "unwrap" the Hiera::Scope
          scope = scope.real
        end
        lookup_invocation.with_scope(scope) do |sub_invocation|
          sub_invocation.lookup(key) {  Lookup.lookup(key, nil, '', true, nil, sub_invocation) }
        end
      end
      scope_lookup = lambda do |key, lookup_invocation, subject|
        segments = split_key(key) { |problem| Puppet::DataBinding::LookupError.new("#{problem} in string: #{subject}") }
        root_key = segments.shift
        value = lookup_invocation.with(:scope, 'Global Scope') do
          ovr = lookup_invocation.override_values
          if ovr.include?(root_key)
            lookup_invocation.report_found_in_overrides(root_key, ovr[root_key])
          else
            scope = lookup_invocation.scope
            val = scope[root_key]
            if val.nil? && !nil_in_scope?(scope, root_key)
              defaults = lookup_invocation.default_values
              if defaults.include?(root_key)
                lookup_invocation.report_found_in_defaults(root_key, defaults[root_key])
              else
                nil
              end
            else
              lookup_invocation.report_found(root_key, val)
            end
          end
        end
        unless value.nil? || segments.empty?
          found = nil;
          catch(:no_such_key) { found = sub_lookup(key, lookup_invocation, segments, value) }
          value = found;
        end
        lookup_invocation.remember_scope_lookup(key, root_key, segments, value)
        value
      end

      {
        :lookup => global_lookup,
        :hiera => global_lookup, # this is just an alias for 'lookup'
        :alias => global_lookup, # same as 'lookup' but expression must be entire string and result is not subject to string substitution
        :scope => scope_lookup,
        :literal => lambda { |key, _, _| key }
      }.freeze
    end
    interpolate_method = @@interpolate_methods[method_key]
    fail(Issues::HIERA_INTERPOLATION_UNKNOWN_INTERPOLATION_METHOD, :name => method_key) unless interpolate_method
    interpolate_method
  end

  # Because the semantics of Puppet::Parser::Scope#include? differs from Hash#include?
  def nil_in_scope?(scope, key)
    if scope.is_a?(Hash)
      scope.include?(key)
    else
      scope.exist?(key)
    end
  end

  def get_method_and_data(data, allow_methods)
    if match = data.match(/^(\w+)\((?:["]([^"]+)["]|[']([^']+)['])\)$/)
      fail(Issues::HIERA_INTERPOLATION_METHOD_SYNTAX_NOT_ALLOWED) unless allow_methods
      key = match[1].to_sym
      data = match[2] || match[3] # double or single qouted
    else
      key = :scope
    end
    [key, data]
  end

  def fail(issue, args = EMPTY_HASH)
    raise Puppet::DataBinding::LookupError.new(
      issue.format(args), nil, nil, nil, nil, issue.issue_code)
  end
end
end
end

 */
