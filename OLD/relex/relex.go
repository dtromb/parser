package relex

import (
	"io"
	"bufio"
	"errors"
	"strconv"
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

type BuildableLexerElement interface {
	Build() parser.NdfaState
}

type CharacterClass interface {
	BuildableLexerElement
	TestChar(c rune) bool
	Union(cc CharacterClass) CharacterClass
	Complement() CharacterClass
}

type BuiltinCharacterClassType int
const (
	ALNUM	BuiltinCharacterClassType = iota
	ALPHA
	ASCII
	BLANK
	CNTRL
	DIGIT
	GRAPH
	LOWER
	UPPER
	PRINT
	PUNCT
	SPACE
	WORD
	XDIGIT
)
func CharacterClassTypeByName(name string) BuiltinCharacterClassType {
	switch(name) {
		case "alnum": return ALNUM
		case "alpha": return ALPHA
		case "ascii": return ASCII
		case "blank": return BLANK
		case "cntrl": return CNTRL
		case "digit": return DIGIT
		case "graph": return GRAPH
		case "lower": return LOWER
		case "upper": return UPPER
		case "print": return PRINT
		case "punct": return PUNCT
		case "space": return SPACE
		case "word":  return WORD
		case "xdigit": return XDIGIT
	}
	panic("unknown character class '"+name+"'")
}
type BuiltinCharacterClass struct {
	classType BuiltinCharacterClassType
}
type RangedCharacterClass struct {
	first rune
	last rune
}

type NoncapturingExpression struct {
	expr BuildableLexerElement
}

type Quantifier struct {
	min, max uint32
	infinite, reluctant bool
}

type QuantifiedUnit struct {
	unit BuildableLexerElement
	quantifier *Quantifier
}

type SingletonCharacterClass struct {
	c rune
}

func GetEscapeCharacterClass(code rune) CharacterClass {
	panic("unimplemented")
}

func hexval(digit rune) int {
	if digit >= '0' && digit <= '9' {
		return int(digit - '0')
	}
	if digit >= 'a' && digit <= 'f' {
		return int(digit - 'a') + 10
	}
	if digit >= 'A' && digit <= 'F' {
		return int(digit - 'A') + 10
	}
	panic("invalid hex digit '"+string(digit)+"'")
}

var _global_MRE_Grammar parser.Grammar
func GetMREGrammar() parser.Grammar {
	if _global_MRE_Grammar == nil {
		gb := parser.OpenGrammarBuilder()
		gb.Name("relex-lexer-generator-re-subset"). 
			Nonterminals("expr","quantified","unit","quantifier","char","charesc","atom",
						 "rchar","id","charclass","group","num","nznum","digits","digit","alphanu",
						 "charstr","range","uclass","alpha").
			Terminals("LC","RC","COMMA","QM","STAR","PLUS","ZERO","NZDIGIT","SLASH", 
			          "LS","RS","LP","RP","CARET","COLON","DASH","DOT","NCHAR","U","ATOF","ALPHANUAF").
			
			// Initial rule.
			Rule().Lhs("`*").Rhs("expr","`.")
			
			// An expr is the toplevel expression that matches tokens. It consists of 
			// a left-to-right list of quantified matching units.
			// <expr> := <quantified> | <quantified> <expr> {[]BuildableLexerElement}
			gb.Rule().Lhs("expr").Rhs("quantified").
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return []BuildableLexerElement{values[0].(BuildableLexerElement)},nil
				}).
			Rule().Lhs("expr").Rhs("quantified","expr"). 
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return append(values[0].([]BuildableLexerElement), values[1].(BuildableLexerElement)),nil
				})
			 
			// A quantified is the top level re element.  All expressions are lists of 
			// (possibly nested) quantified matching units.
			// <quantified> := <unit> | <unit> <quantifier>
			gb.Rule().Lhs("quantified").Rhs("unit"). 
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return &QuantifiedUnit{unit: values[0].(BuildableLexerElement), quantifier: &Quantifier{min: values[1].(uint32), max: values[1].(uint32)}},nil
				}).
			Rule().Lhs("quantified").Rhs("unit","quantifier").
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return &QuantifiedUnit{unit: values[0].(BuildableLexerElement), quantifier: values[1].(*Quantifier)},nil
				})
			 
			// A unit is a matching subunit that can be quantified in the RE.
			// <unit> := <char> | <charesc> | <charclass> | <uclass> | <group> {BuildableLexerElement}
			gb.Rule().Lhs("unit").Rhs("char").  
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return &SingletonCharacterClass{c: ([]rune)(values[0].(string))[0]},nil
				}).
			Rule().Lhs("unit").Rhs("charesc").
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return GetEscapeCharacterClass(([]rune)(values[0].(string))[0]),nil
				}).
			Rule().Lhs("unit").Rhs("charclass"). 
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return values[0].(CharacterClass),nil
				}).
			Rule().Lhs("unit").Rhs("uclass").
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return values[0].(CharacterClass),nil
				}).
			Rule().Lhs("unit").Rhs("group")
			
			// A quantifier determines the number of times its precedent unit must match.
			// Note here that reluctant quantifiers (eg. .*?) may only disambiguate between
			// states chosen in the final DFA - they are just a sort of shorthand for an 
			// acceptor func.
			// <quantifier> := LC <num> RC
			//              |  LC <num> COMMA <num> RC
			//              |  QM
			//              |  STAR
			//				|  PLUS
			// 				|  STAR QM
			//              |  PLUS QM {Quantifier}
			gb.Rule().Lhs("quantifier").Rhs("LC","num","RC"). 
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return &Quantifier{min: values[1].(uint32), max: values[1].(uint32)},nil
				}).
			Rule().Lhs("quantifier").Rhs("LC","num","COMMA","num","RC"). 
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return &Quantifier{min: values[1].(uint32), max: values[3].(uint32)},nil
				}).
			Rule().Lhs("quantifier").Rhs("QM"). 
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return &Quantifier{max: 1},nil
				}).			
			Rule().Lhs("quantifier").Rhs("STAR").
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return &Quantifier{infinite: true},nil
				}).			
			Rule().Lhs("quantifier").Rhs("PLUS").
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return &Quantifier{min: 1, infinite: true},nil
				}).			 
			Rule().Lhs("quantifier").Rhs("STAR","QM").
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return &Quantifier{infinite: true, reluctant: true},nil
				}).			 
			Rule().Lhs("quantifier").Rhs("PLUS","QM").
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return &Quantifier{min: 1, infinite: true, reluctant: true},nil
				})	 
			
			// A num is any zero or positive integer.  At most a single leading zero,
			// and only in the zero case.
			// <num> := ZERO | <nznum>  {uint32}
			gb.Rule().Lhs("num").Rhs("ZERO"). 
			Rule().Lhs("num").Rhs("nznum") 
			
			// An nznum is a string of digits interpreted as a positive integer which
			// does not start with a zero.
			// <nznum> := NZDIGIT
			//         |  NZDIGIT <digits> {uint32}
			gb.Rule().Lhs("nznum").Rhs("NZDIGIT").
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return uint32(values[0].(string)[0] - '0'),nil
				}).
			Rule().Lhs("nznum").Rhs("NZDIGIT","digits"). 
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					// XXX - Meeeeeh.   Better/faster way to handle this, please.
					str := values[0].(string) + strconv.Itoa(int(values[1].(uint32)))
					v, _ := strconv.Atoi(str)
					return uint32(v),nil
				})
			
			// A digit is a member of [0-9].
			// <digit> := NZDIGIT | ZERO {string}
			gb.Rule().Lhs("digit").Rhs("NZDIGIT").
			Rule().Lhs("digit").Rhs("ZERO")
			
			// A digits is a string of digits interpreted as a positive integer.
			// <digits> := <digit>
			//          |  <digits> <digit> {uint32}
			gb.Rule().Lhs("digits").Rhs("digit"). 
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return uint32(values[0].(string)[0] - '0'),nil
				}).
			Rule().Lhs("digits").Rhs("digits","digit"). 
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					d := uint32(values[1].(string)[0] - '0')
					return values[0].(uint32)*10 + d,nil
				})
			
			// A charclass is an entire composite character class, delimited by [].
			// <charclass> := LS <charstr> RS
			//             |  LS CARET <charstr> RS {CharacterClass}
			gb.Rule().Lhs("charclass").Rhs("LS","charstr","RS"). 
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return values[1],nil
				}).
			Rule().Lhs("charclass").Rhs("LS","CARET","charstr","RS"). 
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return values[2].(CharacterClass).Complement(),nil
				})
			
			// A group is a parenthesis-delimited expression which can be independently
			// quantified. It may be non-capturing (?:), which here means that it will
			// be overlooked when the lexer looks for a match of the first capturing group.
			// <group> := LP <expr> RP
			//         |  LP QM COLON <expr> RP {BuildableLexerElement}
			gb.Rule().Lhs("group").Rhs("LP","expr","RP"). 
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return values[1],nil
				}).
			Rule().Lhs("group").Rhs("LP","QM","COLON","expr","RP").
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return &NoncapturingExpression{expr: values[3].(BuildableLexerElement)},nil
				})
			
			// A charstr is a list of atoms that appear in a character class expression.
			// <charstr> := <atom>
			//           |  <atom> <charstr> {CharacterClass}
			gb.Rule().Lhs("charstr").Rhs("atom"). 
			Rule().Lhs("charstr").Rhs("atom","charstr"). 
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return values[0].(CharacterClass).Union(values[1].(CharacterClass)),nil
				})
			
			// An atom is an element of a [] composite character class expression.
			// <atom> := <char> | <charesc> | <range>  {CharacterClass}
			gb.Rule().Lhs("atom").Rhs("char").
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return &SingletonCharacterClass{c:([]rune)(values[0].(string))[0]},nil
				}).
			Rule().Lhs("atom").Rhs("charesc"). 
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return GetEscapeCharacterClass(([]rune)(values[0].(string))[0]),nil
				}).
			Rule().Lhs("atom").Rhs("range")
			
			// A range is a character range in a [] expression such as the A-Z in [a-zA-Z0_\s]
			// <range> := <rchar> DASH <rchar> {RangedCharacterClass}
			gb.Rule().Lhs("range").Rhs("rchar","DASH","rchar").
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return &RangedCharacterClass{
						first: ([]rune)(values[0].(string))[0],
						last: ([]rune)(values[2].(string))[0],
					},nil
				}) 
			
			// The char NT covers the entire class of characters that can appear in a 
			// normal expression without being escaped. 
			// <char> := DOT | NCHAR | ZERO | NZDIGIT | COMMA | ALPHANUAF | ATOF | U {string}
			gb.Rule().Lhs("char").Rhs("DOT"). 
			Rule().Lhs("char").Rhs("NCHAR").
			Rule().Lhs("char").Rhs("ZERO").
			Rule().Lhs("char").Rhs("NZDIGIT"). 
			Rule().Lhs("char").Rhs("COMMA").
			Rule().Lhs("char").Rhs("ALPHANUAF").
			Rule().Lhs("char").Rhs("ATOF").
			Rule().Lhs("char").Rhs("U")
			
			// Uclass is one of the builtin character classes such as [:digit:]. 
			// <uclass> := LS COLON <id> COLON RS {BuiltinCharacterClass}
			gb.Rule().Lhs("uclass").Rhs("LS","COLON","id","COLON","RS").
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return &BuiltinCharacterClass{classType: CharacterClassTypeByName(values[2].(string))},nil
				})
				
			// Id is an alpha-only identifier.
			// <id> := <alpha> 
			//       | <alpha> <id> {string}
			gb.Rule().Lhs("id").Rhs("alpha").
			Rule().Lhs("id").Rhs("alpha","id").
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return values[0].(string) + values[1].(string),nil
				})
			
			// Rchar is any character that may delimit a range expression.
			// <rchar> := <alpha> | NCHAR {string}
			gb.Rule().Lhs("rchar").Rhs("alpha").
			Rule().Lhs("rchar").Rhs("NCHAR")
			
			// <hexdigit> := <digit> | ATOF {string}
			gb.Rule().Lhs("hexdigit").Rhs("digit").
			Rule().Lhs("hexdigit").Rhs("ATOF")
			
			
			// Charesc is a character escape sequence.  \u takes four hex digits and 
			// represents a UTF-16 code, as a special case.  \0 is the NUL character.
			// <charesc> := SLASH <alphanu>
			//           |  SLASH U <hexdigit> <hexdigit> <hexdigit> <hexdigit>
			//           |  SLASH ZERO {string}
			gb.Rule().Lhs("charesc").Rhs("SLASH","alphanu").
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return values[1],nil
				}).
			Rule().Lhs("charesc").Rhs("SLASH","U","hexdigit","hexdigit","hexdigit","hexdigit").
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					var k uint16
					k = (uint16(hexval(([]rune)(values[2].(string))[0])) << 12) |
						(uint16(hexval(([]rune)(values[3].(string))[0])) << 8) |
						(uint16(hexval(([]rune)(values[4].(string))[0])) << 4) |
						uint16(hexval(([]rune)(values[5].(string))[0]))
					return string(rune(k)),nil
				}).			
			Rule().Lhs("charesc").Rhs("SLASH","ZERO"). 
				Value(func(p parser.Production, values []interface{}) (interface{},error) {
					return string(rune(0)),nil
				})
			
			// <alphanu> := ALPHANUAF | ATOF {string}
			gb.Rule().Lhs("alphanu").Rhs("ALPHANUAF"). 
			Rule().Lhs("alphanu").Rhs("ATOF")
			
			// <alpha> := ALPHANUAF | ATOF | U {string}
			gb.Rule().Lhs("alpha").Rhs("ALPHANUAF"). 
			Rule().Lhs("alpha").Rhs("ATOF").
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
		case '0': return parser.NewValueTerminal(mbl.t_ZERO, c), nil
	}		
	if c > '0' && c <= '9' {
		return parser.NewValueTerminal(mbl.t_NZDIGIT, c), nil
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