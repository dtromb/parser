package lexl

import (
	"fmt"
)

// SequenceExpression - Derived interface of MatchExpr - describles a regular subexpression
// with type LexlSequenceExpression
type SequenceExpression interface {
	MatchExpr
	Isa_SequenceExpression() bool
	NumMatch() int
	Match(idx int) MatchExpr
}

type stdLexlSequenceExpression struct {
	matches []MatchExpr
}

func newLexlSequenceExpr(matches ...MatchExpr) SequenceExpression {
	m := make([]MatchExpr, len(matches))
	copy(m, matches)
	return &stdLexlSequenceExpression{
		matches: m,
	}
}

func (*stdLexlSequenceExpression) Isa_SequenceExpression() bool {
	return true
}

func (*stdLexlSequenceExpression) Type() MatchExprType {
	return LexlMatchSequence
}

func (se *stdLexlSequenceExpression) NumMatch() int {
	return len(se.matches)
}

func (se *stdLexlSequenceExpression) GenerateNdfaStates() ([]*stdLexlNdfaState, error) {
	fmt.Println("SEQUENCE")
	var res []*stdLexlNdfaState
	acceptPoints := make([]*stdLexlNdfaState, 0, 16)
	for i, subStateIf := range se.matches {
		subState, ok := subStateIf.(ndfaStateGenerator)
		if !ok {
			subStateIf, err := cloneMatchExpr(subStateIf)
			if err != nil {
				return nil, err
			}
			subState = subStateIf.(ndfaStateGenerator)
		}
		submatchStates, err := subState.GenerateNdfaStates()
		if err != nil {
			return nil, err
		}
		fmt.Printf("submatch %d has %d states\n", i, len(submatchStates))
		res = append(res, submatchStates...)
		for _, ap := range acceptPoints {
			ap.accepting = false
			ap.epsilons = append(ap.epsilons, submatchStates[0])
		}
		acceptPoints = acceptPoints[0:0]
		for i := len(submatchStates) - 1; i >= 0; i-- {
			tState := submatchStates[i]
			if !tState.accepting {
				break
			}
			acceptPoints = append(acceptPoints, tState)
		}
	}
	return res, nil
}

func (se *stdLexlSequenceExpression) ToString() string {
	var buf []rune
	for _, expr := range se.matches {
		buf = append(buf, []rune(MatchExprToString(expr))...)
	}
	return string(buf)
}
