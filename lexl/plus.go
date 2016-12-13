package lexl

import (
	"fmt"
)

// PlusExpression - Derived interface of MatchExpr - describes a regular subexpression
// with type LexlMatchPlus.
type PlusExpression interface {
	MatchExpr
	Isa_PlusExpression() bool // returns true, dismbiguates derived interface
	Submatch() MatchExpr
}

///

type stdLexlPlusExpression struct {
	submatch MatchExpr
}

func newLexlPlusExpr(subexpr MatchExpr) PlusExpression {
	return &stdLexlPlusExpression{
		submatch: subexpr,
	}
}

func (pe *stdLexlPlusExpression) Type() MatchExprType {
	return LexlMatchPlus
}

func (*stdLexlPlusExpression) Isa_PlusExpression() bool {
	return true
}

func (pe *stdLexlPlusExpression) Submatch() MatchExpr {
	return pe.submatch
}

func (se *stdLexlPlusExpression) GenerateNdfaStates() (states []*stdNdfaState, err error) {
	fmt.Println("PLUS")
	sub, ok := se.submatch.(ndfaStateGenerator)
	if !ok {
		subIf, err := cloneMatchExpr(se.submatch)
		if err != nil {
			return nil, err
		}
		sub = subIf.(ndfaStateGenerator)
	}
	subStates, err := sub.GenerateNdfaStates()
	if err != nil {
		return nil, err
	}
	for i := len(subStates) - 1; i >= 0; i-- {
		tState := subStates[i]
		if !tState.accepting {
			break
		}
		tState.epsilons = append(tState.epsilons, subStates[0])
	}
	return subStates, nil
}

func (se *stdLexlPlusExpression) ToString() string {
	var buf []rune
	buf = append(buf, []rune(MatchExprToString(se.submatch))...)
	buf = append(buf, '+')
	return string(buf)
}
