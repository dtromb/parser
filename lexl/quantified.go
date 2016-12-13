package lexl

import (
	"fmt"
)

type QuantifiedExpression interface {
	MatchExpr
	LowerCount() int
	UpperCount() int
	LowerVaradic() bool
	UpperVaradic() bool
	Match() MatchExpr
}

///

type stdLexlQuantifiedExpression struct {
	lower    int
	upper    int
	submatch MatchExpr
}

func newLexlQuantifiedExpr(lower, upper int, subexpr MatchExpr) QuantifiedExpression {
	if lower > upper {
		panic("invalid lower/upper bounds for quantified expression")
	}
	if lower < 0 {
		lower = -1
	}
	if upper < 0 {
		upper = -1
	}
	return &stdLexlQuantifiedExpression{
		lower:    lower,
		upper:    upper,
		submatch: subexpr,
	}
}

func (*stdLexlQuantifiedExpression) Type() MatchExprType {
	return LexlMatchQuantified
}

func (qe *stdLexlQuantifiedExpression) LowerCount() int {
	return qe.lower
}

func (qe *stdLexlQuantifiedExpression) UpperCount() int {
	return qe.upper
}

func (qe *stdLexlQuantifiedExpression) LowerVaradic() bool {
	return qe.lower < 0
}

func (qe *stdLexlQuantifiedExpression) UpperVaradic() bool {
	return qe.upper < 0
}

func (qe *stdLexlQuantifiedExpression) Match() MatchExpr {
	return qe.submatch
}

func (ae *stdLexlQuantifiedExpression) GenerateNdfaStates() (states []*stdNdfaState, err error) {
	fmt.Println("QUANTIFIED")
	var lower int
	if ae.lower < 0 {
		lower = 0
	} else {
		lower = ae.lower
	}
	upper := ae.upper
	submatch, ok := ae.submatch.(ndfaStateGenerator)
	if !ok {
		submatchIf, err := cloneMatchExpr(ae.submatch)
		if err != nil {
			return nil, err
		}
		submatch = submatchIf.(ndfaStateGenerator)
	}
	submatchStates, err := submatch.GenerateNdfaStates()
	if err != nil {
		return nil, err
	}
	var res []*stdNdfaState
	var accept []*stdNdfaState
	var lastInit *stdNdfaState
	for i := 0; i < lower; i++ {
		newStates, err := cloneNdfaStates(submatchStates)
		if err != nil {
			return nil, err
		}
		lastInit = newStates[0]
		for _, ast := range accept {
			ast.epsilons = append(ast.epsilons, newStates[0])
		}
		accept = accept[0:0]
		if (i == lower-1) && (lower == upper) {
			res = append(res, newStates...)
			break
		}
		for _, st := range newStates {
			if st.accepting {
				st.accepting = false
				accept = append(accept, st)
			}
			res = append(res, st)
		}
	}
	if upper < 0 {
		if len(res) == 0 {
			res = []*stdNdfaState{&stdNdfaState{accepting: true}}
			accept = []*stdNdfaState{res[0]}
			lastInit = res[0]
		}
		for _, st := range accept {
			st.epsilons = append(st.epsilons, lastInit)
		}
		return res, nil
	}
	for i := lower; i < upper; i++ {
		newStates, err := cloneNdfaStates(submatchStates)
		if err != nil {
			return nil, err
		}
		lastInit = newStates[0]
		for _, ast := range accept {
			ast.epsilons = append(ast.epsilons, newStates[0])
		}
		accept = accept[0:0]
		for _, st := range newStates {
			if st.accepting {
				accept = append(accept, st)
			}
			res = append(res, st)
		}
	}
	return res, nil
}

func (ae *stdLexlQuantifiedExpression) ToString() string {
	var buf []rune
	buf = append(buf, []rune(MatchExprToString(ae.submatch))...)
	buf = append(buf, '{')
	if ae.lower <= 0 {
		buf = append(buf, '0')
	} else {
		buf = append(buf, []rune(fmt.Sprintf("%d", ae.lower))...)
	}
	if ae.lower == ae.upper || (ae.lower < 0 && ae.upper == 0) {
		buf = append(buf, '}')
		return string(buf)
	}
	buf = append(buf, ',')
	if ae.upper >= 0 {
		buf = append(buf, []rune(fmt.Sprintf("%d", ae.upper))...)
	}
	buf = append(buf, '}')
	return string(buf)
}
