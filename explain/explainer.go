// Package explain contains the Hiera explainer logic
package explain

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/tf"
	"github.com/lyraproj/dgo/util"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/api"
	"github.com/lyraproj/hiera/merge"
)

type event string

const (
	found            = `found`
	foundInDefaults  = `found_in_defaults`
	foundInOverrides = `found_in_overrides`
	locationNotFound = `location_not_found`
	moduleNotFound   = `module_not_found`
	notFound         = `not_found`
	result           = `result`
)

func (e event) String() (s string) {
	return string(e)
}

type explainNode interface {
	dgo.Value
	dgo.Indentable
	appendBranch(branch explainNode)
	appendText(text string)
	branches() []explainNode
	event() event
	found(key interface{}, v dgo.Value)
	foundInDefaults(key interface{}, v dgo.Value)
	foundInOverrides(key interface{}, v dgo.Value)
	initMap() dgo.Map
	initialize(dgo.Map)
	key() string
	locationNotFound()
	moduleNotFound()
	notFound(k interface{})
	parent() explainNode
	result(result dgo.Value)
	setBranches([]explainNode)
	setEvent(event)
	setKey(string)
	setParent(explainNode)
	setTexts([]string)
	setValue(dgo.Value)
	texts() []string
	value() dgo.Value
}

type explainTreeNode struct {
	p  explainNode
	bs []explainNode
	ts []string
	e  event
	v  dgo.Value
	k  string
}

var explainNodeRType = reflect.TypeOf((*explainNode)(nil)).Elem()

var extractFunc = func(v dgo.Value) dgo.Value { return v.(explainNode).initMap() }
var createFunc = func(value dgo.Value, en explainNode) dgo.Value {
	en.initialize(value.(dgo.Map))
	return en
}

func (en *explainTreeNode) Equals(other interface{}) bool {
	return en == other
}

func (en *explainTreeNode) HashCode() int {
	return int(reflect.ValueOf(en).Pointer())
}

func initialize(en explainNode, ih dgo.Map) {
	if ba, ok := ih.Get(`branches`).(dgo.Array); ok {
		bs := make([]explainNode, ba.Len())
		ba.EachWithIndex(func(b dgo.Value, i int) {
			bx := b.(explainNode)
			bx.setParent(en)
			bs[i] = bx
		})
		en.setBranches(bs)
	}
	if ta, ok := ih.Get(`texts`).(dgo.Array); ok {
		ts := make([]string, ta.Len())
		ta.EachWithIndex(func(t dgo.Value, i int) {
			ts[i] = t.String()
		})
		en.setTexts(ts)
	}
	if v, ok := ih.Get(`event`).(dgo.String); ok {
		en.setEvent(event(v.GoString()))
	}
	en.setValue(ih.Get(`value`))
	if v, ok := ih.Get(`key`).(dgo.String); ok {
		en.setKey(v.GoString())
	}
}

func initMap(en explainNode) dgo.Map {
	m := vf.MapWithCapacity(7)
	if bs := en.branches(); len(bs) > 0 {
		m.Put(`branches`, vf.Array(bs))
	}
	if e := en.event(); e != event(``) {
		m.Put(`event`, e.String())
	}
	if k := en.key(); k != `` {
		m.Put(`key`, k)
	}
	if v := en.value(); v != nil {
		m.Put(`value`, v)
	}
	if ts := en.texts(); len(ts) > 0 {
		m.Put(`texts`, vf.Array(ts))
	}
	return m
}

func keyToString(k interface{}) string {
	switch k := k.(type) {
	case string:
		return k
	case int:
		return strconv.Itoa(k)
	case api.Key:
		return k.Source()
	case fmt.Stringer:
		return k.String()
	default:
		return fmt.Sprintf(`%v`, k)
	}
}

func (en *explainTreeNode) AppendTo(w dgo.Indenter) {
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

func (en *explainTreeNode) dumpOutcome(w dgo.Indenter) {
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

func dumpValue(v dgo.Value, w dgo.Indenter) {
	w.AppendValue(v)
}

func (en *explainTreeNode) dumpTexts(w dgo.Indenter) {
	for _, t := range en.ts {
		w.NewLine()
		w.Append(t)
	}
}

func (en *explainTreeNode) dumpBranches(w dgo.Indenter) {
	for _, b := range en.bs {
		b.AppendTo(w)
	}
}

func (en *explainTreeNode) event() event {
	return en.e
}

func (en *explainTreeNode) found(key interface{}, v dgo.Value) {
	en.k = keyToString(key)
	en.v = v
	en.e = found
}

func (en *explainTreeNode) foundInDefaults(key interface{}, v dgo.Value) {
	en.k = keyToString(key)
	en.v = v
	en.e = foundInDefaults
}

func (en *explainTreeNode) foundInOverrides(key interface{}, v dgo.Value) {
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

func (en *explainTreeNode) moduleNotFound() {
	en.e = moduleNotFound
}

func (en *explainTreeNode) notFound(k interface{}) {
	en.k = keyToString(k)
	en.e = notFound
}

func (en *explainTreeNode) result(v dgo.Value) {
	en.v = v
	en.e = result
}

func (en *explainTreeNode) parent() explainNode {
	return en.p
}

func (en *explainTreeNode) setBranches(bs []explainNode) {
	en.bs = bs
}

func (en *explainTreeNode) setEvent(e event) {
	en.e = e
}

func (en *explainTreeNode) setKey(k string) {
	en.k = k
}

func (en *explainTreeNode) setParent(p explainNode) {
	en.p = p
}

func (en *explainTreeNode) setTexts(ts []string) {
	en.ts = ts
}

func (en *explainTreeNode) setValue(v dgo.Value) {
	en.v = v
}

func (en *explainTreeNode) texts() []string {
	return en.ts
}

func (en *explainTreeNode) value() dgo.Value {
	return en.v
}

type explainDataProvider struct {
	explainTreeNode
	providerName string
}

var explainDataProviderType = tf.NewNamed(
	`hiera.explainDataProvider`,
	func(value dgo.Value) dgo.Value { return createFunc(value, &explainDataProvider{}) },
	extractFunc,
	reflect.TypeOf(&explainDataProvider{}),
	explainNodeRType,
	nil)

func (en *explainDataProvider) Type() dgo.Type {
	return tf.ExactNamed(explainDataProviderType, en)
}

func (en *explainDataProvider) initialize(ih dgo.Map) {
	initialize(en, ih)
	if pn, ok := ih.Get(`providerName`).(dgo.String); ok {
		en.providerName = pn.GoString()
	}
}

func (en *explainDataProvider) initMap() dgo.Map {
	m := initMap(en)
	m.Put(`providerName`, en.providerName)
	return m
}

func (en *explainDataProvider) AppendTo(w dgo.Indenter) {
	w.NewLine()
	w.Append(en.providerName)
	w = w.Indent()
	en.dumpBranches(w)
	en.dumpOutcome(w)
}

func (en *explainDataProvider) String() string {
	return util.ToIndentedString(en)
}

func (en *explainDataProvider) Equals(value interface{}) bool {
	return en == value
}

type explainInterpolate struct {
	explainTreeNode
	expression string
}

var explainInterpolateType = tf.NewNamed(
	`hiera.explainInterpolate`,
	func(value dgo.Value) dgo.Value { return createFunc(value, &explainInterpolate{}) },
	extractFunc,
	reflect.TypeOf(&explainInterpolate{}),
	explainNodeRType,
	nil)

func (en *explainInterpolate) Type() dgo.Type {
	return tf.ExactNamed(explainInterpolateType, en)
}

func (en *explainInterpolate) initialize(ih dgo.Map) {
	initialize(en, ih)
	if es, ok := ih.Get(`expression`).(dgo.String); ok {
		en.expression = es.GoString()
	}
}

func (en *explainInterpolate) initMap() dgo.Map {
	m := initMap(en)
	m.Put(`expression`, en.expression)
	return m
}

func (en *explainInterpolate) Equals(value interface{}) bool {
	return en == value
}

func (en *explainInterpolate) AppendTo(w dgo.Indenter) {
	w.NewLine()
	w.Append(`Interpolation on "`)
	w.Append(en.expression)
	w.AppendRune('"')
	en.dumpBranches(w.Indent())
}

func (en *explainInterpolate) String() string {
	return util.ToIndentedString(en)
}

type explainInvalidKey struct {
	explainTreeNode
}

var explainInvalidKeyType = tf.NewNamed(
	`hiera.explainInvalidKey`,
	func(value dgo.Value) dgo.Value { return createFunc(value, &explainInvalidKey{}) },
	extractFunc,
	reflect.TypeOf(&explainInvalidKey{}),
	explainNodeRType,
	nil)

func (en *explainInvalidKey) initialize(ih dgo.Map) {
	initialize(en, ih)
}

func (en *explainInvalidKey) initMap() dgo.Map {
	return initMap(en)
}

func (en *explainInvalidKey) Type() dgo.Type {
	return tf.ExactNamed(explainInvalidKeyType, en)
}

func (en *explainInvalidKey) Equals(value interface{}) bool {
	return en == value
}

func (en *explainInvalidKey) String() string {
	return util.ToIndentedString(en)
}

func (en *explainInvalidKey) AppendTo(w dgo.Indenter) {
	w.NewLine()
	w.Append(`Invalid key "`)
	w.Append(en.key())
	w.AppendRune('"')
}

type explainKeySegment struct {
	explainTreeNode
	segment interface{}
}

var explainKeySegmentType = tf.NewNamed(
	`hiera.explainKeySegment`,
	func(value dgo.Value) dgo.Value { return createFunc(value, &explainKeySegment{}) },
	extractFunc,
	reflect.TypeOf(&explainKeySegment{}),
	explainNodeRType,
	nil)

func (en *explainKeySegment) Type() dgo.Type {
	return tf.ExactNamed(explainKeySegmentType, en)
}

func (en *explainKeySegment) initialize(ih dgo.Map) {
	initialize(en, ih)
	sg := ih.Get(`segment`)
	if i, ok := sg.(dgo.Integer); ok {
		en.segment = i.GoInt()
	} else {
		en.segment = sg.String()
	}
}

func (en *explainKeySegment) initMap() dgo.Map {
	m := initMap(en)
	m.Put(`segment`, en.segment)
	return m
}

func (en *explainKeySegment) Equals(value interface{}) bool {
	return en == value
}

func (en *explainKeySegment) AppendTo(w dgo.Indenter) {
	en.dumpOutcome(w)
}

func (en *explainKeySegment) String() string {
	return util.ToIndentedString(en)
}

type explainLocation struct {
	explainTreeNode
	location api.Location
}

var explainLocationType = tf.NewNamed(
	`hiera.explainLocation`,
	func(value dgo.Value) dgo.Value { return createFunc(value, &explainLocation{}) },
	extractFunc,
	reflect.TypeOf(&explainLocation{}),
	explainNodeRType,
	nil)

func (en *explainLocation) Type() dgo.Type {
	return tf.ExactNamed(explainLocationType, en)
}

func (en *explainLocation) initialize(ih dgo.Map) {
	initialize(en, ih)
	if loc, ok := ih.Get(`location`).(api.Location); ok {
		en.location = loc
	}
}

func (en *explainLocation) initMap() dgo.Map {
	m := initMap(en)
	m.Put(`location`, en.location)
	return m
}

func (en *explainLocation) Equals(value interface{}) bool {
	return en == value
}

func (en *explainLocation) AppendTo(w dgo.Indenter) {
	w.NewLine()
	if en.location.Kind() == api.LcPath {
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
	if en.e == locationNotFound {
		w.NewLine()
		w.Append(string(en.location.Kind()))
		w.Append(` not found`)
	}
	en.dumpOutcome(w)
}

func (en *explainLocation) String() string {
	return util.ToIndentedString(en)
}

type explainLookup struct {
	explainTreeNode
}

var explainLookupType = tf.NewNamed(
	`hiera.explainLookup`,
	func(value dgo.Value) dgo.Value { return createFunc(value, &explainLookup{}) },
	extractFunc,
	reflect.TypeOf(&explainLookup{}),
	explainNodeRType,
	nil)

func (en *explainLookup) initialize(ih dgo.Map) {
	initialize(en, ih)
}

func (en *explainLookup) initMap() dgo.Map {
	return initMap(en)
}

func (en *explainLookup) Type() dgo.Type {
	return tf.ExactNamed(explainLookupType, en)
}

func (en *explainLookup) Equals(value interface{}) bool {
	return en == value
}

func (en *explainLookup) AppendTo(w dgo.Indenter) {
	if w.Len() > 0 || w.Level() > 0 {
		w.NewLine()
	}
	w.Append(`Searching for "`)
	w.Append(en.key())
	w.AppendRune('"')
	en.dumpBranches(w.Indent())
}

func (en *explainLookup) String() string {
	return util.ToIndentedString(en)
}

type explainMerge struct {
	explainTreeNode
	merge api.MergeStrategy
}

var explainMergeType = tf.NewNamed(
	`hiera.explainMerge`,
	func(value dgo.Value) dgo.Value { return createFunc(value, &explainMerge{}) },
	extractFunc,
	reflect.TypeOf(&explainMerge{}),
	explainNodeRType,
	nil)

func (en *explainMerge) Type() dgo.Type {
	return tf.ExactNamed(explainMergeType, en)
}

func (en *explainMerge) initialize(ih dgo.Map) {
	initialize(en, ih)
	name := `first`
	var opts dgo.Map
	if v, ok := ih.Get(`strategy`).(dgo.String); ok {
		name = v.GoString()
	}
	if v, ok := ih.Get(`options`).(dgo.Map); ok {
		opts = v
	}
	en.merge = merge.GetStrategy(name, opts)
}

func (en *explainMerge) initMap() dgo.Map {
	m := initMap(en)
	if en.merge.Name() != `first` {
		m.Put(`strategy`, en.merge.Name())
		if opts := en.merge.Options(); opts != nil && opts.Len() > 0 {
			m.Put(`options`, opts)
		}
	}
	return m
}

func (en *explainMerge) Equals(value interface{}) bool {
	return en == value
}

func (en *explainMerge) AppendTo(w dgo.Indenter) {
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
		if opts.Len() > 0 {
			w.NewLine()
			w.Append(`Options: `)
			dumpValue(opts, w)
		}
		en.dumpBranches(w)
		if en.e == result {
			w.NewLine()
			w.Append(`Merged result: `)
			dumpValue(en.v, w)
		}
	}
}

func (en *explainMerge) String() string {
	return util.ToIndentedString(en)
}

type explainMergeSource struct {
	explainTreeNode
	mergeSource string
}

var explainMergeSourceType = tf.NewNamed(
	`hiera.explainMergeSource`,
	func(value dgo.Value) dgo.Value { return createFunc(value, &explainMergeSource{}) },
	extractFunc,
	reflect.TypeOf(&explainMergeSource{}),
	explainNodeRType,
	nil)

func (en *explainMergeSource) Type() dgo.Type {
	return tf.ExactNamed(explainMergeSourceType, en)
}

func (en *explainMergeSource) initialize(ih dgo.Map) {
	initialize(en, ih)
	if ms, ok := ih.Get(`mergeSource`).(dgo.String); ok {
		en.mergeSource = ms.GoString()
	}
}

func (en *explainMergeSource) initMap() dgo.Map {
	m := initMap(en)
	m.Put(`mergeSource`, en.mergeSource)
	return m
}

func (en *explainMergeSource) Equals(value interface{}) bool {
	return en == value
}

func (en *explainMergeSource) AppendTo(w dgo.Indenter) {
	w.NewLine()
	w.Append(`Using merge options from `)
	w.Append(en.mergeSource)
}

func (en *explainMergeSource) String() string {
	return util.ToIndentedString(en)
}

type explainModule struct {
	explainTreeNode
	moduleName string
}

var explainModuleType = tf.NewNamed(
	`hiera.explainModule`,
	func(value dgo.Value) dgo.Value { return createFunc(value, &explainModule{}) },
	extractFunc,
	reflect.TypeOf(&explainModule{}),
	explainNodeRType,
	nil)

func (en *explainModule) Type() dgo.Type {
	return tf.ExactNamed(explainModuleType, en)
}

func (en *explainModule) initialize(ih dgo.Map) {
	initialize(en, ih)
	if ms, ok := ih.Get(`moduleName`).(dgo.String); ok {
		en.moduleName = ms.GoString()
	}
}

func (en *explainModule) initMap() dgo.Map {
	m := initMap(en)
	m.Put(`moduleName`, en.moduleName)
	return m
}

func (en *explainModule) Equals(value interface{}) bool {
	return en == value
}

func (en *explainModule) AppendTo(w dgo.Indenter) {
	switch en.e {
	case moduleNotFound:
		w.NewLine()
		w.Append(`Module "`)
		w.Append(en.moduleName)
		w.Append(`" not found`)
	default:
		en.dumpBranches(w)
		en.dumpOutcome(w)
	}
}

func (en *explainModule) String() string {
	return util.ToIndentedString(en)
}

type explainSubLookup struct {
	explainTreeNode
	subKey api.Key
}

var explainSubLookupType = tf.NewNamed(
	`hiera.explainSubLookup`,
	func(value dgo.Value) dgo.Value { return createFunc(value, &explainSubLookup{}) },
	extractFunc,
	reflect.TypeOf(&explainSubLookup{}),
	explainNodeRType,
	nil)

func (en *explainSubLookup) Type() dgo.Type {
	return tf.ExactNamed(explainSubLookupType, en)
}

func (en *explainSubLookup) initialize(ih dgo.Map) {
	initialize(en, ih)
	if ms, ok := ih.Get(`subKey`).(dgo.String); ok {
		en.subKey = api.NewKey(ms.GoString())
	}
}

func (en *explainSubLookup) initMap() dgo.Map {
	m := initMap(en)
	m.Put(`subKey`, en.subKey.Source())
	return m
}

func (en *explainSubLookup) Equals(value interface{}) bool {
	return en == value
}

func (en *explainSubLookup) AppendTo(w dgo.Indenter) {
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
	return util.ToIndentedString(en)
}

type explainer struct {
	explainTreeNode
	current     explainNode
	options     bool
	onlyOptions bool
}

var explainerType = tf.NewNamed(
	`hiera.explainer`,
	func(value dgo.Value) dgo.Value { return createFunc(value, &explainer{}) },
	extractFunc,
	reflect.TypeOf(&explainer{}),
	reflect.TypeOf((*api.Explainer)(nil)).Elem(),
	nil)

func (ex *explainer) Type() dgo.Type {
	return tf.ExactNamed(explainerType, ex)
}

// NewExplainer creates a new configured Explainer instance.
func NewExplainer(options, onlyOptions bool) api.Explainer {
	ex := &explainer{options: options, onlyOptions: onlyOptions}
	ex.current = ex
	return ex
}

func (ex *explainer) initialize(ih dgo.Map) {
	initialize(ex, ih)
	if v, ok := ih.Get(`current`).(explainNode); ok {
		if ex.current != ex {
			ex.current = v
		}
	} else {
		ex.current = ex
	}
	if v, ok := ih.Get(`options`).(dgo.Boolean); ok {
		ex.options = v.GoBool()
	}
	if v, ok := ih.Get(`onlyOptions`).(dgo.Boolean); ok {
		ex.onlyOptions = v.GoBool()
	}
}

func (ex *explainer) initMap() dgo.Map {
	m := initMap(ex)
	if ex.current != ex {
		m.Put(`current`, ex.current)
	}
	if ex.options {
		m.Put(`options`, true)
	}
	if ex.onlyOptions {
		m.Put(`onlyOptions`, true)
	}
	return m
}

func (ex *explainer) Equals(value interface{}) bool {
	return ex == value
}

func (ex *explainer) AcceptFound(key interface{}, value dgo.Value) {
	ex.current.found(key, value)
}

func (ex *explainer) AcceptFoundInDefaults(key string, value dgo.Value) {
	ex.current.foundInDefaults(key, value)
}

func (ex *explainer) AcceptFoundInOverrides(key string, value dgo.Value) {
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

func (ex *explainer) AcceptModuleNotFound() {
	ex.current.moduleNotFound()
}

func (ex *explainer) AcceptNotFound(key interface{}) {
	ex.current.notFound(key)
}

func (ex *explainer) AcceptMergeResult(value dgo.Value) {
	ex.current.result(value)
}

func (ex *explainer) AcceptText(text string) {
	ex.current.appendText(text)
}

func (ex *explainer) push(en explainNode) {
	en.setParent(ex.current)
	ex.current.appendBranch(en)
	ex.current = en
}

func (ex *explainer) PushDataProvider(pvd api.DataProvider) {
	ex.push(&explainDataProvider{providerName: pvd.FullName()})
}

func (ex *explainer) PushInterpolation(expr string) {
	ex.push(&explainInterpolate{expression: expr})
}

func (ex *explainer) PushInvalidKey(key interface{}) {
	ex.push(&explainInvalidKey{explainTreeNode{k: keyToString(key)}})
}

func (ex *explainer) PushLocation(loc api.Location) {
	ex.push(&explainLocation{location: loc})
}

func (ex *explainer) PushLookup(key api.Key) {
	ex.push(&explainLookup{explainTreeNode{k: keyToString(key)}})
}

func (ex *explainer) PushMerge(mrg api.MergeStrategy) {
	ex.push(&explainMerge{merge: mrg})
}

func (ex *explainer) PushModule(mn string) {
	ex.push(&explainModule{moduleName: mn})
}

func (ex *explainer) PushSegment(seg interface{}) {
	ex.push(&explainKeySegment{segment: seg})
}

func (ex *explainer) PushSubLookup(key api.Key) {
	ex.push(&explainSubLookup{subKey: key})
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

func (ex *explainer) AppendTo(w dgo.Indenter) {
	ex.dumpBranches(w)
	ex.dumpTexts(w)
}

func (ex *explainer) String() string {
	return util.ToIndentedString(ex)
}
