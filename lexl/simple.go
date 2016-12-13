package lexl

import (
	"errors"
	"fmt"
)

type simpleExpr struct {
	exprType MatchExprType
}

func newLexlAlwaysMatchExpr() MatchExpr {
	return &simpleExpr{exprType: LexlMatchAlways}
}

func newLexlNeverMatchExpr() MatchExpr {
	return &simpleExpr{exprType: LexlMatchNever}
}

func newLexlMatchStartExpr() MatchExpr {
	return &simpleExpr{exprType: LexlMatchStart}
}

func newLexlMatchEndExpr() MatchExpr {
	return &simpleExpr{exprType: LexlMatchEnd}
}

func (se *simpleExpr) Type() MatchExprType {
	return se.exprType
}

func (se *simpleExpr) GenerateNdfaStates() ([]*stdNdfaState, error) {
	fmt.Println("SIMPLE")
	switch se.exprType {
	case LexlMatchAlways:
		{ /*
					   [0]: * -> [1]
				       [1]: `.
			*/
			s0 := newStdNdfaState()
			s1 := newStdNdfaState()
			s1.accepting = true
			//s0.stars = []*stdNdfaState{newStdNdfaState()}
			//s0.stars[0].accepting = true
			starRange := &characterRange{0, -1}
			s0.ranges[starRange] = []*stdNdfaState{s1}
			//return []*stdNdfaState{s0, s0.stars[0]}, nil
			return []*stdNdfaState{s0, s1}, nil
		}
	case LexlMatchNever:
		{
			/*
				[0]:
			*/
			return []*stdNdfaState{newStdNdfaState()}, nil
		}
	case LexlMatchStart:
		{
			s0 := newStdNdfaState()
			s1 := newStdNdfaState()
			s1.accepting = true
			s0.literals[rune(0xFEFF)] = []*stdNdfaState{s1}
			return []*stdNdfaState{s0, s1}, nil
		}
	case LexlMatchEnd:
		{
			s0 := newStdNdfaState()
			s1 := newStdNdfaState()
			s1.accepting = true
			s0.literals[rune(0x0004)] = []*stdNdfaState{s1}
			return []*stdNdfaState{s0, s1}, nil
		}
	default:
		{
			return nil, errors.New("invalid simpleExpr expression type")
		}
	}
}

func (se *simpleExpr) ToString() string {
	switch se.exprType {
	case LexlMatchAlways:
		{
			return "."
		}
	case LexlMatchStart:
		{
			return "^"
		}
	case LexlMatchEnd:
		{
			return "$"
		}
	default:
		{
			return "\\xFFFD"
		}
	}
}
