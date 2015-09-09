package relex

import (
	"io"
	"bufio"
	"errors"
	"github.com/dtromb/parser"
	"github.com/dtromb/parser/index"
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

// Simplified interface for lexers which don't look ahead/back.
type TokenValueFn func(part parser.GrammarParticle, match string) interface{}

func OpenLexerBuilder(g parser.Grammar) *LexerBuilder {
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

func (lb *LexerBuilder) Token(name string) *LexerBuilder {
	return lb
}

func (lb *LexerBuilder) Expr(re string) *LexerBuilder {
	return lb
}

func (lb *LexerBuilder) Acceptor(fn TokenAcceptorFn) *LexerBuilder {
	return lb
}

func (lb *LexerBuilder) Value(fn TokenValueFn) *LexerBuilder {
	return lb
}

func (lb *LexerBuilder) Build() (parser.Lexer,error) {
	GetMREGrammar()
	return nil, errors.New("unimplemented")
}

var _global_MRE_Grammar parser.Grammar
func GetMREGrammar() parser.Grammar {
	if _global_MRE_Grammar == nil {
		gb := parser.OpenGrammarBuilder()
		gb.Name("relex-lexer-generator-re-subset"). 
			Nonterminals("expr","quantified","unit","quantifier","char","charesc","atom",
						 "rchar","id","charclass","group","num","nznum","digits","digit",
						 "charstr","range","uclass","alpha").
			Terminals("LC","RC","COMMA","QM","STAR","PLUS","ZERO","NZDIGIT","SLASH", 
			          "LS","RS","LP","RP","CARET","COLON","DASH","DOT","NCHAR","U","ALPHANU").
			Rule().Lhs("`*").Rhs("expr","`.").
			Rule().Lhs("expr").Rhs("quantified"). 
			Rule().Lhs("expr").Rhs("quantified","expr"). 
			Rule().Lhs("quantified").Rhs("unit"). 
			Rule().Lhs("quantified").Rhs("unit","quantifier"). 
			Rule().Lhs("unit").Rhs("char"). 
			Rule().Lhs("unit").Rhs("charesc").
			Rule().Lhs("unit").Rhs("charclass"). 
			Rule().Lhs("unit").Rhs("group").
			Rule().Lhs("quantifier").Rhs("LC","num","RC"). 
			Rule().Lhs("quantifier").Rhs("LC","num","COMMA","num","RC"). 
			Rule().Lhs("quantifier").Rhs("QM"). 
			Rule().Lhs("quantifier").Rhs("STAR").
			Rule().Lhs("quantifier").Rhs("PLUS"). 
			Rule().Lhs("quantifier").Rhs("STAR","QM"). 
			Rule().Lhs("quantifier").Rhs("PLUS","QM"). 
			Rule().Lhs("num").Rhs("ZERO"). 
			Rule().Lhs("num").Rhs("nznum"). 
			Rule().Lhs("nznum").Rhs("NZDIGIT"). 
			Rule().Lhs("nznum").Rhs("NZDIGIT","digits").
			Rule().Lhs("digit").Rhs("NZDIGIT").
			Rule().Lhs("digit").Rhs("ZERO").
			Rule().Lhs("digits").Rhs("digit"). 
			Rule().Lhs("digits").Rhs("digit","digits"). 
			Rule().Lhs("charclass").Rhs("LS","charstr","RS"). 
			Rule().Lhs("charclass").Rhs("LS","CARET","charstr","RS"). 
			Rule().Lhs("group").Rhs("LP","expr","RP"). 
			Rule().Lhs("group").Rhs("LP","QM","COLON","expr","RP").
			Rule().Lhs("charstr").Rhs("atom"). 
			Rule().Lhs("charstr").Rhs("atom","charstr"). 
			Rule().Lhs("atom").Rhs("char"). 
			Rule().Lhs("atom").Rhs("charesc"). 
			Rule().Lhs("atom").Rhs("range").
			Rule().Lhs("range").Rhs("rchar","DASH","rchar"). 
			Rule().Lhs("char").Rhs("DOT"). 
			Rule().Lhs("char").Rhs("NCHAR").
			Rule().Lhs("char").Rhs("ZERO").
			Rule().Lhs("char").Rhs("NZDIGIT"). 
			Rule().Lhs("char").Rhs("COMMA").
			Rule().Lhs("char").Rhs("ALPHANU").
			Rule().Lhs("char").Rhs("U").
			Rule().Lhs("uclass").Rhs("LS","COLON","id","COLON","RS").
			Rule().Lhs("id").Rhs("alpha").
			Rule().Lhs("id").Rhs("alpha","id").
			Rule().Lhs("rchar").Rhs("NCHAR").
			Rule().Lhs("rchar").Rhs("alpha").
			Rule().Lhs("charesc").Rhs("SLASH","ALPHANU").
			Rule().Lhs("charesc").Rhs("SLASH","U","digit","digit","digit","digit").
			Rule().Lhs("charesc").Rhs("SLASH","ZERO"). 
			Rule().Lhs("alpha").Rhs("ALPHANU"). 
			Rule().Lhs("alpha").Rhs("U")
			
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
	pos, col, line int
	t_LC, t_RC, t_COMMA, t_QM, t_PLUS, t_LS, t_ZERO,
		t_RS, t_CARET, t_LP, t_RP, t_DASH, t_DOT, 
		t_COLON, t_SLASH, t_NZDIGIT, t_U, t_ALPHA, t_NCHAR parser.GrammarParticle
}

func getMREBootstrapLexer() *mreBootstrapLexer {
	lex := &mreBootstrapLexer{
		grammar: GetMREGrammar(),
	}
	ig := parser.GetIndexedGrammar(lex.grammar)
	idx, err := ig.GetIndex(index.NAME_INDEX)
	if err != nil {
		panic("could not name-index MRE grammar: "+err.Error())
	}
	nidx := idx.(*index.NameIndex)
	lex.t_LC = nidx.Terminal("LC")
	lex.t_RC = nidx.Terminal("RC")
	lex.t_COMMA = nidx.Terminal("COMMA")
	lex.t_QM = nidx.Terminal("QM")
	lex.t_PLUS = nidx.Terminal("PLUS")
	lex.t_LS = nidx.Terminal("LS")
	lex.t_RS = nidx.Terminal("RS")
	lex.t_ZERO = nidx.Terminal("ZERO")
	lex.t_CARET = nidx.Terminal("CARET")
	lex.t_LP = nidx.Terminal("LP")
	lex.t_RP = nidx.Terminal("RP")
	lex.t_DASH = nidx.Terminal("DASH")
	lex.t_DOT = nidx.Terminal("DOT")
	lex.t_COLON = nidx.Terminal("COLON")
	lex.t_SLASH = nidx.Terminal("SLASH")
	lex.t_NZDIGIT = nidx.Terminal("NZDIGIT")
	lex.t_U = nidx.Terminal("U")
	lex.t_ALPHA = nidx.Terminal("ALPHA")
	lex.t_NCHAR = nidx.Terminal("NCHAR")
	return lex
}

func (mbl *mreBootstrapLexer) Eof() bool {
	return true
}
		
func (mbl *mreBootstrapLexer) Next() (parser.GrammarParticle,error) {
	if mbl.isEof {
		return nil, errors.New("attempt to read token after end of input")
	}
	c, s, err := mbl.in.ReadRune()
	if err == io.EOF {
		mbl.isEof = true
		return mbl.grammar.Bottom(), nil
	}
	if err != nil {
		return nil, err
	}
	mbl.pos += s
	if c == '\n' {
		mbl.line++
		mbl.col = 1
	} else {
		mbl.col++
	}		
	switch c {
		case '{': return mbl.t_LC, nil
		case '}': return mbl.t_RC, nil
		case ',': return mbl.t_COMMA, nil
		case '?': return mbl.t_QM, nil
		case '+': return mbl.t_PLUS, nil
		case '[': return mbl.t_LS, nil
		case ']': return mbl.t_RS, nil
		case '^': return mbl.t_CARET, nil
		case '(': return mbl.t_LP, nil
		case ')': return mbl.t_RP, nil
		case '-': return mbl.t_DASH, nil
		case '.': return mbl.t_DOT, nil
		case ':': return mbl.t_COLON, nil
		case '\\': return mbl.t_SLASH, nil
		case '0': return parser.NewValueTerminal(mbl.t_ZERO, int8(0)), nil
	}		
	if c > '0' && c <= '9' {
		return parser.NewValueTerminal(mbl.t_NZDIGIT, int8(c-'0')), nil
	}
	if c == 'u' {
		return parser.NewValueTerminal(mbl.t_U, c), nil
	}
	if (c >= 'a' && c <= 'z') ||
	   (c >= 'A' && c <= 'Z') {
		return parser.NewValueTerminal(mbl.t_ALPHA, c), nil
	}
	return parser.NewValueTerminal(mbl.t_NCHAR, c), nil
}

func (mbl *mreBootstrapLexer) Reset(in io.Reader) {
	mbl.isEof = false
	if br, isBuffered := in.(*bufio.Reader); isBuffered {
		mbl.in = br
	} else {
		mbl.in = bufio.NewReader(in)
	}
	mbl.pos = 1
	mbl.col = 1
	mbl.line = 1
}