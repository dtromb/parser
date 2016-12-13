package lexl

import (
	"fmt"
	"unicode"
)

// CharacterLiteralExpression - Derived interface of MatchExpr - describes a regular subexpression
// with type LexlMatchCharacterLiteral
type CharacterLiteralExpression interface {
	MatchExpr
	Character() rune
}

type stdLexlCharacterLiteralExpression struct {
	c rune
}

func newLexlCharacterLiteralExpr(c rune) CharacterLiteralExpression {
	return &stdLexlCharacterLiteralExpression{c: c}
}

func (*stdLexlCharacterLiteralExpression) Type() MatchExprType {
	return LexlMatchCharacterLiteral
}

func (cle *stdLexlCharacterLiteralExpression) Character() rune {
	return cle.c
}

func (cle *stdLexlCharacterLiteralExpression) GenerateNdfaStates() (states []*stdLexlNdfaState, err error) {
	fmt.Println("CHARLIT")
	s0 := newStdLexlNdfaState()
	s1 := newStdLexlNdfaState()
	s1.accepting = true
	s0.literals[cle.c] = []*stdLexlNdfaState{s1}
	return []*stdLexlNdfaState{s0, s1}, nil
}

func (cle *stdLexlCharacterLiteralExpression) isPrintableClassChar(c rune) bool {
	return unicode.IsPrint(c) && c <= 256
}

func (cle *stdLexlCharacterLiteralExpression) isSpecialClassChar(c rune) bool {
	if c > 255 {
		return false
	}
	switch byte(c) {
	case '.':
		fallthrough
	case '^':
		fallthrough
	case '$':
		fallthrough
	case '\\':
		fallthrough
	case '{':
		fallthrough
	case '}':
		fallthrough
	case '(':
		fallthrough
	case ')':
		fallthrough
	case '?':
		fallthrough
	case '*':
		fallthrough
	case '+':
		fallthrough
	case '|':
		fallthrough
	case '[':
		fallthrough
	case '/':
		{
			return true
		}
	}
	return false
}

func (cle *stdLexlCharacterLiteralExpression) appendClassChar(buf []rune, c rune) []rune {
	if cle.isPrintableClassChar(c) {
		if cle.isSpecialClassChar(c) {
			buf = append(buf, '\\')
		}
		buf = append(buf, c)
	} else {
		switch c {
		case '\n':
			buf = append(buf, []rune("\\n")...)
		case '\t':
			buf = append(buf, []rune("\\t")...)
		case '\r':
			buf = append(buf, []rune("\\r")...)
		case '\f':
			buf = append(buf, []rune("\\f")...)
		default:
			buf = append(buf, ([]rune)(fmt.Sprintf("\\x%4.4x", c))...)
		}
	}
	return buf
}

func (cle *stdLexlCharacterLiteralExpression) ToString() string {
	var buf []rune
	buf = cle.appendClassChar(buf, cle.c)
	return string(buf)
}
