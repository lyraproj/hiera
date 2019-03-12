package impl

import (
	"github.com/lyraproj/pcore/px"
	"strings"
)

type TrackingScope interface {
	px.Keyed

	// GetRead returns a map of all variables that has been read from this scope. The
	// map contains the last value read.
	GetRead() map[string]px.Value
}

type trackingScope struct {
	tracked px.Keyed
	read    map[string]px.Value
}

func NewTrackingScope(tracked px.Keyed) TrackingScope {
	return &trackingScope{tracked, make(map[string]px.Value, 13)}
}

func (t *trackingScope) Fork() px.Keyed {
	// Multi threaded use of TrackingScope is not permitted
	panic(`attempt to fork TrackingScope`)
}

func (t *trackingScope) Get(nv px.Value) (px.Value, bool) {
	name := nv.String()
	value, found := t.tracked.Get(nv)

	key := name
	if strings.HasPrefix(name, `::`) {
		key = name[2:]
	}
	if found {
		// A Global variable that has a value is immutable. No need to track it
		if vs, ok := t.tracked.(px.VariableStates); ok && vs.State(name) == px.Global {
			delete(t.read, key)
		} else {
			t.read[key] = value
		}
	} else {
		t.read[key] = nil // explicit nil denotes "not found"
	}
	return value, found
}

func (t *trackingScope) GetRead() map[string]px.Value {
	return t.read
}
