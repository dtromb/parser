package lexl

import (
	"fmt"
)

// StarExpression - Derived interface of MatchExpr - describes a regular subexpression
// with type LexlMatchStar.
type StarExpression interface {
	MatchExpr
	Isa_StarExpression() bool // returns true, dismbiguates derived interface
	Submatch() MatchExpr
}

type stdLexlStarExpression struct {
	submatch MatchExpr
}

func newLexlStarExpr(subexpr MatchExpr) StarExpression {
	return &stdLexlStarExpression{
		submatch: subexpr,
	}
}

func (*stdLexlStarExpression) Type() MatchExprType {
	return LexlMatchStar
}

func (*stdLexlStarExpression) Isa_StarExpression() bool {
	return true
}

func (se *stdLexlStarExpression) Submatch() MatchExpr {
	return se.submatch
}

func (se *stdLexlStarExpression) GenerateNdfaStates() (states []*stdLexlNdfaState, err error) {
	fmt.Println("STAR")
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
	subStates[0].accepting = true
	return subStates, nil
}

func (se *stdLexlStarExpression) ToString() string {
	var buf []rune
	buf = append(buf, []rune(MatchExprToString(se.submatch))...)
	buf = append(buf, '*')
	return string(buf)
}
