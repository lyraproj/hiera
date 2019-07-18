package internal

import (
	"fmt"
	"io"
	"strconv"

	"github.com/lyraproj/pcore/types"

	"github.com/lyraproj/hiera/explain"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/utils"
)

func init() {
	explain.NewExplainer = newExplainer
}

type event int

const (
	found = event(iota) + 1
	foundInDefaults
	foundInOverrides
	locationNotFound
	notFound
	result
)

type explainNode interface {
	px.Value
	utils.Indentable
	appendBranch(branch explainNode)
	appendText(text string)
	branches() []explainNode
	event() event
	found(key interface{}, v px.Value)
	foundInDefaults(key interface{}, v px.Value)
	foundInOverrides(key interface{}, v px.Value)
	key() string
	locationNotFound()
	notFound(k interface{})
	parent() explainNode
	result(result px.Value)
	texts() []string
	value() px.Value
}

type explainTreeNode struct {
	p  explainNode
	bs []explainNode
	ts []string
	e  event
	v  px.Value
	k  string
}

var explainNodeMetaType px.ObjectType
var explainDataProviderMetaType px.ObjectType
var explainInterpolateMetaType px.ObjectType
var explainInvalidKeyMetaType px.ObjectType
var explainKeySegmentMetaType px.ObjectType
var explainLocationMetaType px.ObjectType
var explainLookupMetaType px.ObjectType
var explainMergeMetaType px.ObjectType
var explainMergeSourceMetaType px.ObjectType
var explainSubLookupMetaType px.ObjectType
var explainerMetaType px.ObjectType

func init() {
	explainNodeMetaType = px.NewObjectType(`Hiera::ExplainNode`, `{
		attributes => {
      branches => Optional[Array[Hiera::ExplainNode]],
      texts => Optional[Array[String]],
      event => Optional[Integer[1,6]],
      value => { type => RichData, value => undef },
      key => Optional[String]
    }
	}`)

	explainDataProviderMetaType = px.NewObjectType(`Hiera::ExplainProvider`, `Hiera::ExplainNode{
		attributes => {
      providerName => String
    }
	}`,
		types.NoPositionalConstructor,
		func(c px.Context, args []px.Value) px.Value {
			en := &explainDataProvider{}
			en.initialize(args[0].(px.OrderedMap))
			return en
		})

	explainInterpolateMetaType = px.NewObjectType(`Hiera::ExplainInterpolate`, `Hiera::ExplainNode{
		attributes => {
      expression => String
    }
	}`,
		types.NoPositionalConstructor,
		func(c px.Context, args []px.Value) px.Value {
			en := &explainInterpolate{}
			en.initialize(args[0].(px.OrderedMap))
			return en
		})

	explainInvalidKeyMetaType = px.NewObjectType(`Hiera::ExplainInvalidKey`, `Hiera::ExplainNode{}`,
		types.NoPositionalConstructor,
		func(c px.Context, args []px.Value) px.Value {
			en := &explainInvalidKey{}
			en.initialize(args[0].(px.OrderedMap))
			return en
		})

	explainKeySegmentMetaType = px.NewObjectType(`Hiera::ExplainKeySegment`, `Hiera::ExplainNode{
		attributes => {
			segment => Variant[String,Integer]
		}
	}`,
		types.NoPositionalConstructor,
		func(c px.Context, args []px.Value) px.Value {
			en := &explainKeySegment{}
			en.initialize(args[0].(px.OrderedMap))
			return en
		})

	explainLocationMetaType = px.NewObjectType(`Hiera::ExplainLocation`, `Hiera::ExplainNode{
		attributes => {
			location => Hiera::Location
		}
	}`,
		types.NoPositionalConstructor,
		func(c px.Context, args []px.Value) px.Value {
			en := &explainLocation{}
			en.initialize(args[0].(px.OrderedMap))
			return en
		})

	explainLookupMetaType = px.NewObjectType(`Hiera::ExplainLookup`, `Hiera::ExplainNode{}`,
		types.NoPositionalConstructor,
		func(c px.Context, args []px.Value) px.Value {
			en := &explainLookup{}
			en.initialize(args[0].(px.OrderedMap))
			return en
		})

	explainMergeMetaType = px.NewObjectType(`Hiera::ExplainMerge`, `Hiera::ExplainNode{
		attributes => {
			strategy => String,
      options => { type => Hash[String,Data], value => {} }
		}
	}`,
		types.NoPositionalConstructor,
		func(c px.Context, args []px.Value) px.Value {
			en := &explainMerge{}
			en.initialize(args[0].(px.OrderedMap))
			return en
		})

	explainMergeSourceMetaType = px.NewObjectType(`Hiera::ExplainMergeSource`, `Hiera::ExplainNode{
		attributes => {
			mergeSource => String
		}
	}`,
		types.NoPositionalConstructor,
		func(c px.Context, args []px.Value) px.Value {
			en := &explainMergeSource{}
			en.initialize(args[0].(px.OrderedMap))
			return en
		})

	explainSubLookupMetaType = px.NewObjectType(`Hiera::ExplainSubLookup`, `Hiera::ExplainNode{
		attributes => {
			subKey => String
		}
	}`,
		types.NoPositionalConstructor,
		func(c px.Context, args []px.Value) px.Value {
			en := &explainSubLookup{}
			en.initialize(args[0].(px.OrderedMap))
			return en
		})

	explainerMetaType = px.NewObjectType(`Hiera::Explainer`, `Hiera::ExplainNode{
		attributes => {
			current => Optional[Hiera::ExplainNode],
			options => { type => Boolean, value => false },
      onlyOptions => { type => Boolean, value => false },
		}
	}`,
		types.NoPositionalConstructor,
		func(c px.Context, args []px.Value) px.Value {
			en := &explainer{}
			en.initialize(args[0].(px.OrderedMap))
			return en
		})
}

func (en *explainTreeNode) initialize(ih px.OrderedMap) {
	if v, ok := ih.Get4(`branches`); ok {
		ba := v.(px.List)
		bs := make([]explainNode, ba.Len())
		ba.EachWithIndex(func(b px.Value, i int) {
			bx := b.(*explainTreeNode)
			bx.p = en
			bs[i] = bx
		})
		en.bs = bs
	}
	if v, ok := ih.Get4(`texts`); ok {
		ta := v.(px.List)
		ts := make([]string, ta.Len())
		ta.EachWithIndex(func(t px.Value, i int) {
			ts[i] = t.String()
		})
		en.ts = ts
	}
	if v, ok := ih.Get4(`event`); ok {
		en.e = event(v.(px.Integer).Int())
	}
	if v, ok := ih.Get4(`value`); ok {
		en.v = v
	}
	if v, ok := ih.Get4(`key`); ok {
		en.k = v.String()
	}
}

func (en *explainTreeNode) String() string {
	return px.ToString(en)
}

func (en *explainTreeNode) Equals(value interface{}, guard px.Guard) bool {
	return en == value
}

func (en *explainTreeNode) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	types.ObjectToString(en, format, bld, g)
}

func (en *explainTreeNode) PType() px.Type {
	return explainNodeMetaType
}

func (en *explainTreeNode) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `branches`:
		if len(en.bs) > 0 {
			return px.Wrap(nil, en.bs), true
		}
		return px.Undef, true
	case `texts`:
		if len(en.ts) > 0 {
			return px.Wrap(nil, en.ts), true
		}
		return px.Undef, true
	case `event`:
		if en.e > 0 {
			return types.WrapInteger(int64(en.e)), true
		}
		return px.Undef, true
	case `value`:
		if en.v != nil {
			return en.v, true
		}
		return px.Undef, true
	case `key`:
		if en.k != `` {
			return types.WrapString(en.k), true
		}
		return px.Undef, true
	}
	return nil, false
}

func (en *explainTreeNode) InitHash() px.OrderedMap {
	return explainNodeMetaType.InstanceHash(en)
}

func keyToString(k interface{}) string {
	switch k := k.(type) {
	case string:
		return k
	case int:
		return strconv.Itoa(k)
	case fmt.Stringer:
		return k.String()
	default:
		return fmt.Sprintf(`%v`, k)
	}
}

func (en *explainTreeNode) AppendTo(w *utils.Indenter) {
	en.dumpTexts(w)
}

func (en *explainTreeNode) appendBranch(branch explainNode) {
	en.bs = append(en.bs, branch)
}

func (en *explainTreeNode) appendText(text string) {
	en.ts = append(en.ts, text)
}

func (en *explainTreeNode) branches() []explainNode {
	return en.bs
}

func (en *explainTreeNode) dumpOutcome(w *utils.Indenter) {
	switch en.e {
	case notFound:
		w.NewLine()
		w.Append(`No such key: "`)
		w.Append(en.k)
		w.AppendRune('"')
	case found, foundInDefaults, foundInOverrides:
		w.NewLine()
		w.Append(`Found key: "`)
		w.Append(en.k)
		w.Append(`" value: `)
		dumpValue(en.v, w)
		if en.e == foundInDefaults {
			w.Append(` in defaults`)
		} else if en.e == foundInOverrides {
			w.Append(` in overrides`)
		}
	}
	en.dumpTexts(w)
}

func dumpValue(v px.Value, w *utils.Indenter) {
	w.AppendIndented(px.ToPrettyString(v))
}

func (en *explainTreeNode) dumpTexts(w *utils.Indenter) {
	for _, t := range en.ts {
		w.NewLine()
		w.Append(t)
	}
}

func (en *explainTreeNode) dumpBranches(w *utils.Indenter) {
	for _, b := range en.bs {
		b.AppendTo(w)
	}
}

func (en *explainTreeNode) event() event {
	return en.e
}

func (en *explainTreeNode) found(key interface{}, v px.Value) {
	en.k = keyToString(key)
	en.v = v
	en.e = found
}

func (en *explainTreeNode) foundInDefaults(key interface{}, v px.Value) {
	en.k = keyToString(key)
	en.v = v
	en.e = foundInDefaults
}

func (en *explainTreeNode) foundInOverrides(key interface{}, v px.Value) {
	en.k = keyToString(key)
	en.v = v
	en.e = foundInOverrides
}

func (en *explainTreeNode) key() string {
	return en.k
}

func (en *explainTreeNode) locationNotFound() {
	en.e = locationNotFound
}

func (en *explainTreeNode) notFound(k interface{}) {
	en.k = keyToString(k)
	en.e = notFound
}

func (en *explainTreeNode) result(v px.Value) {
	en.v = v
	en.e = result
}

func (en *explainTreeNode) parent() explainNode {
	return en.p
}

func (en *explainTreeNode) texts() []string {
	return en.ts
}

func (en *explainTreeNode) value() px.Value {
	return en.v
}

type explainDataProvider struct {
	explainTreeNode
	providerName string
}

func (en *explainDataProvider) initialize(ih px.OrderedMap) {
	en.explainTreeNode.initialize(ih)
	en.providerName = ih.Get5(`providerName`, px.EmptyString).String()
}

func (en *explainDataProvider) AppendTo(w *utils.Indenter) {
	w.NewLine()
	w.Append(en.providerName)
	w = w.Indent()
	en.dumpBranches(w)
	en.dumpOutcome(w)
}

func (en *explainDataProvider) String() string {
	return utils.IndentedString(en)
}

func (en *explainDataProvider) Equals(value interface{}, guard px.Guard) bool {
	return en == value
}

func (en *explainDataProvider) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	types.ObjectToString(en, format, bld, g)
}

func (en *explainDataProvider) PType() px.Type {
	return explainDataProviderMetaType
}

func (en *explainDataProvider) Get(key string) (value px.Value, ok bool) {
	if key == `providerName` {
		return types.WrapString(en.providerName), true
	}
	return en.explainTreeNode.Get(key)
}

func (en *explainDataProvider) InitHash() px.OrderedMap {
	return explainDataProviderMetaType.InstanceHash(en)
}

type explainInterpolate struct {
	explainTreeNode
	expression string
}

func (en *explainInterpolate) initialize(ih px.OrderedMap) {
	en.explainTreeNode.initialize(ih)
	en.expression = ih.Get5(`expression`, px.EmptyString).String()
}

func (en *explainInterpolate) Equals(value interface{}, guard px.Guard) bool {
	return en == value
}

func (en *explainInterpolate) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	types.ObjectToString(en, format, bld, g)
}

func (en *explainInterpolate) PType() px.Type {
	return explainInterpolateMetaType
}

func (en *explainInterpolate) Get(key string) (value px.Value, ok bool) {
	if key == `expression` {
		return types.WrapString(en.expression), true
	}
	return en.explainTreeNode.Get(key)
}

func (en *explainInterpolate) InitHash() px.OrderedMap {
	return explainInterpolateMetaType.InstanceHash(en)
}

func (en *explainInterpolate) AppendTo(w *utils.Indenter) {
	w.NewLine()
	w.Append(`Interpolation on "`)
	w.Append(en.expression)
	w.AppendRune('"')
	en.dumpBranches(w.Indent())
}

func (en *explainInterpolate) String() string {
	return utils.IndentedString(en)
}

type explainInvalidKey struct {
	explainTreeNode
}

func (en *explainInvalidKey) Equals(value interface{}, guard px.Guard) bool {
	return en == value
}

func (en *explainInvalidKey) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	types.ObjectToString(en, format, bld, g)
}

func (en *explainInvalidKey) String() string {
	return utils.IndentedString(en)
}

func (en *explainInvalidKey) PType() px.Type {
	return explainInvalidKeyMetaType
}

func (en *explainInvalidKey) InitHash() px.OrderedMap {
	return explainInvalidKeyMetaType.InstanceHash(en)
}

func (en *explainInvalidKey) AppendTo(w *utils.Indenter) {
	w.NewLine()
	w.Append(`Invalid key "`)
	w.Append(en.key())
	w.AppendRune('"')
}

type explainKeySegment struct {
	explainTreeNode
	segment interface{}
}

func (en *explainKeySegment) initialize(ih px.OrderedMap) {
	en.explainTreeNode.initialize(ih)
	sg := ih.Get5(`expression`, px.EmptyString)
	if i, ok := sg.(px.Integer); ok {
		en.segment = i.Int()
	} else {
		en.segment = sg.String()
	}
}

func (en *explainKeySegment) Equals(value interface{}, guard px.Guard) bool {
	return en == value
}

func (en *explainKeySegment) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	types.ObjectToString(en, format, bld, g)
}

func (en *explainKeySegment) PType() px.Type {
	return explainKeySegmentMetaType
}

func (en *explainKeySegment) Get(key string) (value px.Value, ok bool) {
	if key == `segment` {
		return px.Wrap(nil, en.segment), true
	}
	return en.explainTreeNode.Get(key)
}

func (en *explainKeySegment) InitHash() px.OrderedMap {
	return explainKeySegmentMetaType.InstanceHash(en)
}

func (en *explainKeySegment) AppendTo(w *utils.Indenter) {
	en.dumpOutcome(w)
}

func (en *explainKeySegment) String() string {
	return utils.IndentedString(en)
}

type explainLocation struct {
	explainTreeNode
	location hieraapi.Location
}

func (en *explainLocation) initialize(ih px.OrderedMap) {
	en.explainTreeNode.initialize(ih)
	if loc, ok := ih.Get4(`location`); ok {
		en.location = loc.(hieraapi.Location)
	}
}

func (en *explainLocation) Equals(value interface{}, guard px.Guard) bool {
	return en == value
}

func (en *explainLocation) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	types.ObjectToString(en, format, bld, g)
}

func (en *explainLocation) PType() px.Type {
	return explainLocationMetaType
}

func (en *explainLocation) Get(key string) (value px.Value, ok bool) {
	if key == `location` {
		return en.location.(px.Value), true
	}
	return en.explainTreeNode.Get(key)
}

func (en *explainLocation) InitHash() px.OrderedMap {
	return explainLocationMetaType.InstanceHash(en)
}

func (en *explainLocation) AppendTo(w *utils.Indenter) {
	w.NewLine()
	if en.location.Kind() == hieraapi.LcPath {
		w.Append(`Path "`)
	} else {
		w.Append(`URI "`)
	}
	w.Append(en.location.Resolved())
	w.AppendRune('"')

	w = w.Indent()
	w.NewLine()
	w.Append(`Original `)
	w.Append(string(en.location.Kind()))
	w.Append(`: "`)
	w.Append(en.location.Original())
	w.AppendRune('"')

	en.dumpBranches(w)
	if en.e == notFound {
		w.NewLine()
		w.Append(string(en.location.Kind()))
		w.Append(` not found`)
	}
	en.dumpOutcome(w)
}

func (en *explainLocation) String() string {
	return utils.IndentedString(en)
}

type explainLookup struct {
	explainTreeNode
}

func (en *explainLookup) Equals(value interface{}, guard px.Guard) bool {
	return en == value
}

func (en *explainLookup) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	types.ObjectToString(en, format, bld, g)
}

func (en *explainLookup) PType() px.Type {
	return explainLookupMetaType
}

func (en *explainLookup) InitHash() px.OrderedMap {
	return explainLookupMetaType.InstanceHash(en)
}

func (en *explainLookup) AppendTo(w *utils.Indenter) {
	if w.Len() > 0 || w.Level() > 0 {
		w.NewLine()
	}
	w.Append(`Searching for "`)
	w.Append(en.key())
	w.AppendRune('"')
	en.dumpBranches(w.Indent())
}

func (en *explainLookup) String() string {
	return utils.IndentedString(en)
}

type explainMerge struct {
	explainTreeNode
	merge hieraapi.MergeStrategy
}

func (en *explainMerge) initialize(ih px.OrderedMap) {
	en.explainTreeNode.initialize(ih)
	name := hieraapi.First
	var opts map[string]px.Value
	if v, ok := ih.Get4(`strategy`); ok {
		name = hieraapi.MergeStrategyName(v.String())
	}
	if v, ok := ih.Get4(`options`); ok {
		opts = v.(px.OrderedMap).ToStringMap()
	}
	en.merge = getMergeStrategy(name, opts)
}

func (en *explainMerge) Equals(value interface{}, guard px.Guard) bool {
	return en == value
}

func (en *explainMerge) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	types.ObjectToString(en, format, bld, g)
}

func (en *explainMerge) PType() px.Type {
	return explainMergeMetaType
}

func (en *explainMerge) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `strategy`:
		return types.WrapString(string(en.merge.Name())), true
	case `options`:
		return en.merge.Options(), true
	}
	return en.explainTreeNode.Get(key)
}

func (en *explainMerge) InitHash() px.OrderedMap {
	return explainMergeMetaType.InstanceHash(en)
}

func (en *explainMerge) AppendTo(w *utils.Indenter) {
	switch len(en.bs) {
	case 0:
		// No action
	case 1:
		en.bs[0].AppendTo(w)
	default:
		w.NewLine()
		w.Append(`Merge strategy "`)
		w.Append(en.merge.Label())
		w.AppendRune('"')
		w = w.Indent()
		opts := en.merge.Options()
		if !opts.IsEmpty() {
			w.NewLine()
			w.Append(`Options: `)
			dumpValue(opts, w.Indent())
		}
		en.dumpBranches(w)
		if en.e == result {
			w.NewLine()
			w.Append(`Merged result: `)
			dumpValue(en.v, w.Indent())
		}
	}
}

func (en *explainMerge) String() string {
	return utils.IndentedString(en)
}

type explainMergeSource struct {
	explainTreeNode
	mergeSource string
}

func (en *explainMergeSource) initialize(ih px.OrderedMap) {
	en.explainTreeNode.initialize(ih)
	if ms, ok := ih.Get4(`mergeSource`); ok {
		en.mergeSource = ms.String()
	}
}

func (en *explainMergeSource) Equals(value interface{}, guard px.Guard) bool {
	return en == value
}

func (en *explainMergeSource) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	types.ObjectToString(en, format, bld, g)
}

func (en *explainMergeSource) PType() px.Type {
	return explainMergeSourceMetaType
}

func (en *explainMergeSource) Get(key string) (value px.Value, ok bool) {
	if key == `mergeSource` {
		return types.WrapString(en.mergeSource), true
	}
	return en.explainTreeNode.Get(key)
}

func (en *explainMergeSource) InitHash() px.OrderedMap {
	return explainMergeSourceMetaType.InstanceHash(en)
}

func (en *explainMergeSource) AppendTo(w *utils.Indenter) {
	w.NewLine()
	w.Append(`Using merge options from `)
	w.Append(en.mergeSource)
}

func (en *explainMergeSource) String() string {
	return utils.IndentedString(en)
}

type explainSubLookup struct {
	explainTreeNode
	subKey hieraapi.Key
}

func (en *explainSubLookup) initialize(ih px.OrderedMap) {
	en.explainTreeNode.initialize(ih)
	if ms, ok := ih.Get4(`subKey`); ok {
		en.subKey = newKey(ms.String())
	}
}

func (en *explainSubLookup) Equals(value interface{}, guard px.Guard) bool {
	return en == value
}

func (en *explainSubLookup) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	types.ObjectToString(en, format, bld, g)
}

func (en *explainSubLookup) PType() px.Type {
	return explainSubLookupMetaType
}

func (en *explainSubLookup) Get(key string) (value px.Value, ok bool) {
	if key == `subKey` {
		return types.WrapString(en.subKey.String()), true
	}
	return en.explainTreeNode.Get(key)
}

func (en *explainSubLookup) InitHash() px.OrderedMap {
	return explainSubLookupMetaType.InstanceHash(en)
}

func (en *explainSubLookup) AppendTo(w *utils.Indenter) {
	w.NewLine()
	w.Append(`Sub key: "`)
	for i, s := range en.subKey.Parts()[1:] {
		if i > 0 {
			w.AppendRune('.')
		}
		w.Append(keyToString(s))
	}
	w.AppendRune('"')
	w = w.Indent()
	en.dumpBranches(w)
	en.dumpOutcome(w)
}

func (en *explainSubLookup) String() string {
	return utils.IndentedString(en)
}

type explainer struct {
	explainTreeNode
	current     explainNode
	options     bool
	onlyOptions bool
}

func newExplainer(options, onlyOptions bool) explain.Explainer {
	ex := &explainer{options: options, onlyOptions: onlyOptions}
	ex.current = ex
	return ex
}

func (en *explainer) initialize(ih px.OrderedMap) {
	en.explainTreeNode.initialize(ih)
	if v, ok := ih.Get4(`current`); ok {
		if en.current != en {
			en.current = v.(explainNode)
		}
	} else {
		en.current = en
	}
	if v, ok := ih.Get4(`options`); ok {
		en.options = v.(px.Boolean).Bool()
	}
	if v, ok := ih.Get4(`onlyOptions`); ok {
		en.onlyOptions = v.(px.Boolean).Bool()
	}
}

func (en *explainer) Equals(value interface{}, guard px.Guard) bool {
	return en == value
}

func (en *explainer) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	types.ObjectToString(en, format, bld, g)
}

func (en *explainer) PType() px.Type {
	return explainerMetaType
}

func (en *explainer) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `current`:
		if en.current != en {
			return en.current, true
		}
		return px.Undef, true
	case `options`:
		return types.WrapBoolean(en.options), true
	case `onlyOptions`:
		return types.WrapBoolean(en.onlyOptions), true
	}
	return en.explainTreeNode.Get(key)
}

func (en *explainer) InitHash() px.OrderedMap {
	return explainerMetaType.InstanceHash(en)
}

func (ex *explainer) AcceptFound(key interface{}, value px.Value) {
	ex.current.found(key, value)
}

func (ex *explainer) AcceptFoundInDefaults(key string, value px.Value) {
	ex.current.foundInDefaults(key, value)
}

func (ex *explainer) AcceptFoundInOverrides(key string, value px.Value) {
	ex.current.foundInOverrides(key, value)
}

func (ex *explainer) AcceptLocationNotFound() {
	ex.current.locationNotFound()
}

func (ex *explainer) AcceptMergeSource(mergeSource string) {
	en := &explainMergeSource{mergeSource: mergeSource}
	en.p = ex.current
	ex.current.appendBranch(en)
}

func (ex *explainer) AcceptNotFound(key interface{}) {
	ex.current.notFound(key)
}

func (ex *explainer) AcceptMergeResult(value px.Value) {
	ex.current.result(value)
}

func (ex *explainer) AcceptText(text string) {
	ex.current.appendText(text)
}

func (ex *explainer) PushDataProvider(pvd hieraapi.DataProvider) {
	en := &explainDataProvider{providerName: pvd.FullName()}
	en.p = ex.current
	ex.current.appendBranch(en)
	ex.current = en
}

func (ex *explainer) PushInterpolation(expr string) {
	en := &explainInterpolate{expression: expr}
	en.p = ex.current
	ex.current.appendBranch(en)
	ex.current = en
}

func (ex *explainer) PushInvalidKey(key interface{}) {
	en := &explainInvalidKey{}
	en.k = keyToString(key)
	en.p = ex.current
	ex.current.appendBranch(en)
	ex.current = en
}

func (ex *explainer) PushLocation(loc hieraapi.Location) {
	en := &explainLocation{location: loc}
	en.p = ex.current
	ex.current.appendBranch(en)
	ex.current = en
}

func (ex *explainer) PushLookup(key hieraapi.Key) {
	en := &explainLookup{}
	en.p = ex.current
	en.k = keyToString(key)
	ex.current.appendBranch(en)
	ex.current = en
}

func (ex *explainer) PushMerge(mrg hieraapi.MergeStrategy) {
	en := &explainMerge{merge: mrg}
	en.p = ex.current
	ex.current.appendBranch(en)
	ex.current = en
}

func (ex *explainer) PushSegment(seg interface{}) {
	en := &explainKeySegment{segment: seg}
	en.p = ex.current
	ex.current.appendBranch(en)
	ex.current = en
}

func (ex *explainer) PushSubLookup(key hieraapi.Key) {
	en := &explainSubLookup{subKey: key}
	en.p = ex.current
	ex.current.appendBranch(en)
	ex.current = en
}

func (ex *explainer) Pop() {
	if ex.current != nil {
		ex.current = ex.current.parent()
	}
}

func (ex *explainer) OnlyOptions() bool {
	return ex.onlyOptions
}

func (ex *explainer) Options() bool {
	return ex.options
}

func (ex *explainer) AppendTo(w *utils.Indenter) {
	ex.dumpBranches(w)
	ex.dumpTexts(w)
}

func (ex *explainer) String() string {
	return utils.IndentedString(ex)
}
