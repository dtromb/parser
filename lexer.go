package ngen

import (
	"io"
)

type Lexer interface {
	Grammar() Grammar
	Open(in io.Reader) (LexerState, error)
}

type LexerState interface {
	Lexer() Lexer
	Reader() io.Reader
	HasMoreTokens() (bool, error)
	NextToken() (Token, error)
	CurrentLine() int
	CurrentColumn() int
	CurrentPosition() int
	// ExpectTokens() []Term
}

type Token interface {
	LexerState() LexerState
	FirstPosition() int
	LastPosition() int
	FirstLine() int
	LastLine() int
	FirstColumn() int
	LastColumn() int
	Terminal() Term
	Literal() string
}
