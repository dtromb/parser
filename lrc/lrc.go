package lrc

import "github.com/dtromb/parser"

type LrcItemType int
const (
	TERMINAL					LrcItemType = iota
	UNEXPANDED_NONTERMINAL
	EXPANDED_NONTERMINAL
)

type LrcItem interface {
	Type() LrcItemType
	Head() parser.GrammarParticle
	Len() int
	Part(idx int) LrcItem
	HasCaret() bool
	Caret() int
	String() string
}


