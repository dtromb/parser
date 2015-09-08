package relex

import (
	"io"
	"bufio"
	"errors"
	"github.com/dtromb/parser"
)

type LexerBuilder struct {
	
}

type RuneBuffer struct {
	data []rune
	zero int
}

type TokenBuffer struct {
	data []parser.GrammarParticle
	zero int
}

type TokenAcceptorFn func(part parser.GrammarParticle, match string, 
						  la *RuneBuffer, lb *RuneBuffer, buf *TokenBuffer) (bool, interface{})

func OpenLexerBuilder() *LexerBuilder {
	return &LexerBuilder{}
}

func (lb *LexerBuilder) Lookahead(n int) *LexerBuilder {
	return lb
}

func (lb *LexerBuilder) LookBehind(n int) *LexerBuilder {
	return lb
}

func (lb *LexerBuilder) Buffer(n int) *LexerBuilder {
	return lb
}

func (lb *LexerBuilder) Token() *LexerBuilder {
	return lb
}

func (lb *LexerBuilder) Expr(re string) *LexerBuilder {
	return lb
}

func (lb *LexerBuilder) Acceptor(fn TokenAcceptorFn) *LexerBuilder {
	return lb
}

func (lb *LexerBuilder) Build() (parser.Lexer,error) {
	return nil, errors.New("unimplemented")
}

var _global_MRE_Grammar parser.Grammar
func GetMREGrammar() parser.Grammar {
	if _global_MRE_Grammar == nil {
		gb := parser.OpenGrammarBuilder()
		gb.Name("relex-lexer-generator-re-subset"). 
		 	Terminals("NCHAR","CHARESC","LC","NUM","RC","COMMA","QM","PLUS","LS","RS","CARET","LP","RP","DASH","DOT"). 
			Nonterminals("char","expr","quantified","unit","quantifier","charclass","group","charstr","atom","range"). 
			Rule().Lhs("expr").Rhs("quantified"). 
			Rule().Lhs("expr").Rhs("quantified","expr"). 
			Rule().Lhs("quantified").Rhs("unit"). 
			Rule().Lhs("quantified").Rhs("unit","quantifier"). 
			Rule().Lhs("unit").Rhs("char"). 
			Rule().Lhs("unit").Rhs("CHARESC").
			Rule().Lhs("unit").Rhs("charclass"). 
			Rule().Lhs("unit").Rhs("group").
			Rule().Lhs("quantifier").Rhs("LC","NUM","RC"). 
			Rule().Lhs("quantifier").Rhs("LC","NUM","COMMA","NUM","RC"). 
			Rule().Lhs("quantifier").Rhs("QM"). 
			Rule().Lhs("quantifier").Rhs("STAR").
			Rule().Lhs("quantifier").Rhs("PLUS"). 
			Rule().Lhs("quantifier").Rhs("STAR","QM"). 
			Rule().Lhs("qauntifier").Rhs("PLUS","QM"). 
			Rule().Lhs("charclass").Rhs("LS","charstr","RS"). 
			Rule().Lhs("charclass").Rhs("LS","CARET","charstr","RS"). 
			Rule().Lhs("group").Rhs("LP","expr","RP"). 
			Rule().Lhs("charstr").Rhs("atom"). 
			Rule().Lhs("charstr").Rhs("atom","charstr"). 
			Rule().Lhs("atom").Rhs("char"). 
			Rule().Lhs("atom").Rhs("CHARESC"). 
			Rule().Lhs("atom").Rhs("range").
			Rule().Lhs("range").Rhs("NCHAR","DASH","NCHAR"). 
			Rule().Lhs("char").Rhs("DOT"). 
			Rule().Lhs("char").Rhs("NCHAR")
		g, err := gb.Build()
		if err != nil {
			panic("could not build relex grammar: "+err.Error())
		}
		_global_MRE_Grammar = g
	}
	return _global_MRE_Grammar
}

type mreBootstrapLexer struct {
	grammar parser.Grammar
	isEof bool
	in *bufio.Reader
}

func getMREBootstrapLexer() *mreBootstrapLexer {
	return &mreBootstrapLexer{
		grammar: GetMREGrammar(),
	}
}

func (mbl *mreBootstrapLexer) Eof() bool {
	return true
}

func (mbl *mreBootstrapLexer) Next() (parser.GrammarParticle,error) {
	return nil, errors.New("unimplemented")
}

func (mbl *mreBootstrapLexer) Reset(in io.Reader) {
	mbl.isEof = false
	if br, isBuffered := in.(*bufio.Reader); isBuffered {
		mbl.in = br
	} else {
		mbl.in = bufio.NewReader(in)
	}
}