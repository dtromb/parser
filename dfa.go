package parser

import c "github.com/dtromb/collections"

type DfaState interface {
	Label() string
	Id() int
	HasTransition(k c.Comparable) bool
	Transition(k c.Comparable) DfaState
	Terminal() bool
	NumTransitions() int
	TransitionKey(idx int) c.Comparable
}

type Dfa interface {
	InitialState() DfaState
	NumStates() int
	State(idx int) DfaState
	NumTerminals() int
	Terminal(idx int) DfaState
}

type NdfaState interface {
	Label() string
	Id() int
	HasTransition(k c.Comparable) bool
	NumTransitions(k c.Comparable) int
	Transition(k c.Comparable, idx int) NdfaState
	Terminal() bool
	NumTransitionKeys() int
	TransitionKey(idx int) c.Comparable
}

type Ndfa interface {
	InitialState() NdfaState
	NumStates() int
	State(idx int) NdfaState
	NumTerminals() int
	Terminal(idx int) NdfaState
}