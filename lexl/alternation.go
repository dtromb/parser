package lexl

import (
	"fmt"
)

// AlternationExpression - Derived interface of MatchExpr - describles a regular subexpression
// with type LexlSequenceAlternation
type AlternationExpression interface {
	MatchExpr
	Isa_AlternationExpression() bool
	NumMatch() int
	Match(idx int) MatchExpr
}

type stdLexlAlternationExpression struct {
	matches []MatchExpr
}

func newLexlAlternationExpr(matches ...MatchExpr) AlternationExpression {
	m := make([]MatchExpr, len(matches))
	copy(m, matches)
	return &stdLexlAlternationExpression{
		matches: m,
	}
}

func (*stdLexlAlternationExpression) Type() MatchExprType {
	return LexlMatchAlternation
}

func (*stdLexlAlternationExpression) Isa_AlternationExpression() bool {
	return true
}

func (ae *stdLexlAlternationExpression) NumMatch() int {
	return len(ae.matches)
}

func (ae *stdLexlAlternationExpression) Match(idx int) MatchExpr {
	if idx < 0 || idx >= len(ae.matches) {
		return nil
	}
	return ae.matches[idx]
}

func (ae *stdLexlAlternationExpression) GenerateNdfaStates() (states []*stdNdfaState, err error) {
	fmt.Println("ALTERNATION")
	var initial *stdNdfaState
	var res, aRes []*stdNdfaState
	for i, sub := range ae.matches {
		submatch, ok := sub.(ndfaStateGenerator)
		if !ok {
			submatchIf, err := cloneMatchExpr(sub)
			if err != nil {
				return nil, err
			}
			submatch = submatchIf.(ndfaStateGenerator)
		}
		submatchStates, err := submatch.GenerateNdfaStates()
		if err != nil {
			return nil, err
		}
		if i == 0 {
			initial = submatchStates[0]
			for j := 0; j < len(submatchStates); j++ {
				tState := submatchStates[j]
				if tState.accepting {
					aRes = append(aRes, tState)
				} else {
					res = append(res, tState)
				}
			}
		} else {
			for j := 1; j < len(submatchStates); j++ {
				tState := submatchStates[j]
				if tState.accepting {
					aRes = append(aRes, tState)
				} else {
					res = append(res, tState)
				}
			}
			mInit := submatchStates[0]
			if mInit.accepting {
				initial.accepting = true
			}
			initial.epsilons = append(initial.epsilons, mInit.epsilons...)
			// mInit.stars = append(initial.stars, mInit.stars...)
			for k, v := range mInit.literals {
				if _, has := initial.literals[k]; !has {
					initial.literals[k] = v
				} else {
					initial.literals[k] = append(initial.literals[k], v...)
				}
			}
			for k, v := range mInit.ranges {
				if _, has := initial.ranges[k]; !has {
					initial.ranges[k] = v
				} else {
					initial.ranges[k] = append(initial.ranges[k], v...)
				}
			}
		}
	}
	res = append(res, aRes...)
	return res, nil
}

func (ae *stdLexlAlternationExpression) ToString() string {
	var buf []byte
	for i, m := range ae.matches {
		buf = append(buf, MatchExprToString(m)...)
		if i < len(ae.matches)-1 {
			buf = append(buf, '|')
		}
	}
	return string(buf)
}
