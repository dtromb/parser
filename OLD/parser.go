package parser

import "io"

type Stringable interface {
	String() string
}

type Lexer interface {
	Eof() bool
	Next() (GrammarParticle,error)
	Reset(in io.Reader)
}

type ParserError error

type Parser interface {
	Parse(lexer Lexer, svf SyntaxValueFactory) (ast SyntaxTreeNode, err ParserError)
}

type SyntaxTreeNode interface {
	Part() GrammarParticle
	First() int
	Last() int
	Value() interface{}
	Rule() Production
	NumChildren() int
	Child(idx int) SyntaxTreeNode
}

type BasicSyntaxTreeNode struct {
	Particle GrammarParticle
	FirstTokenIdx int
	LastTokenIdx int
	SyntacticValue interface{}
	Prod Production
	Expansion []SyntaxTreeNode
}

type SyntaxValue interface {
	Supports(p Production) bool
	ChildValue(p Production, idx int) SyntaxValue
}

type SyntaxValueFactory func(p Production, values []SyntaxValue) SyntaxValue

type SyntaxTreeTransform func(treeNode SyntaxTreeNode) (SyntaxTreeNode, error)

type GrammarTransform func(grammar Grammar) (Grammar, SyntaxTreeTransform, error)

func (bsn *BasicSyntaxTreeNode) Part() GrammarParticle {
	return bsn.Particle
}

func (bsn *BasicSyntaxTreeNode) First() int {
	return bsn.FirstTokenIdx
}

func (bsn *BasicSyntaxTreeNode) Last() int {
	return bsn.LastTokenIdx
}

func (bsn *BasicSyntaxTreeNode) Value() interface{} {
	return bsn.SyntacticValue
}

func (bsn *BasicSyntaxTreeNode) Rule() Production {
	return bsn.Prod
}

func (bsn *BasicSyntaxTreeNode) NumChildren() int {
	return len(bsn.Expansion)
}

func (bsn *BasicSyntaxTreeNode) Child(idx int) SyntaxTreeNode {
	return bsn.Expansion[idx]
}

type StringReader struct {
	buf []byte
	pos int
}

func NewStringReader(str string) *StringReader {
	return &StringReader{
		buf: []byte(str),
	}
}

func (sr *StringReader) Read(p []byte) (n int, err error) {
	l := len(p)
	if l > len(sr.buf) - sr.pos {
		l = len(sr.buf) - sr.pos
	} 
	if l == 0 {
		return 0, io.EOF
	}
	copy(p,sr.buf[sr.pos:sr.pos+l])
	sr.pos += l
	return l, nil
}
