package internal

import (
	"fmt"
	"strconv"

	"github.com/lyraproj/hiera/explain"

	"github.com/lyraproj/pcore/utils"

	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/pcore/px"
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
	provider hieraapi.DataProvider
}

func (en *explainDataProvider) AppendTo(w *utils.Indenter) {
	w.NewLine()
	w.Append(en.provider.FullName())
	w = w.Indent()
	en.dumpBranches(w)
	en.dumpOutcome(w)
}

func (en *explainDataProvider) String() string {
	return utils.IndentedString(en)
}

type explainInterpolate struct {
	explainTreeNode
	expression string
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

func (en *explainInvalidKey) AppendTo(w *utils.Indenter) {
	w.NewLine()
	w.Append(`Invalid key "`)
	w.Append(en.key())
	w.AppendRune('"')
}

func (en *explainInvalidKey) String() string {
	return utils.IndentedString(en)
}

type explainKeySegment struct {
	explainTreeNode
	segment interface{}
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
	en := &explainDataProvider{provider: pvd}
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
