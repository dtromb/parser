package ngen

/*
import (
	"bufio"
	"io"
	"unicode"
)

var global_LexlGrammarBnf0 = "\n" +
	""

type charRangeTransition struct {
	firstChar byte
	lastChar  byte
	nxt       *stdLexlNdfaState
}

type stdLexlNdfaState struct {
	id       uint32
	schar    map[byte][]*stdLexlNdfaState
	charmaps []*charRangeTransition
	epsTrans []*stdLexlNdfaState
}

type stdLexlDfaState struct {
	id       uint32
	schar    map[byte]lexlDfaState
	lmaps    map[byte]int
	rmaps    map[byte]int
	mapTrans []lexlDfaState
}

/*
char literals:  any char except reserved  		.[{}()\*+?|^$
String literal   "asdf"
Character class [a-z]


<match> :=  CHARLIT
		|   WILDCARD
		| 	START
		|	END
		|   <subexp>
		|   <optional>
		|   <star>
		| 	<quantified>
		| 	<backref>
		|   <charset>

<subexp> := LP <match> RP

<optional> := <match> OPT

<star> := <match> STAR

<quantified> := <match> LC <num> RC
		     |  <match> LC <num> COMMA RC
			 |  <match> LC <num> COMMA <num> RC

<backref> := <esc> NZDIGIT

<digit> := NZDIGIT | ZERO

<numstr> := <digit> | <digit> <numstr>

<num> := ZERO | NZDIGIT | NZDIGIT <numstr>

<charset> := LS <nerslist> RS
          |  LS CARET <nerslist> RS

<nerslistpart> := <escape>
				| CHARLIT
				| <range>
				| <eqclass>

<nerslist> := <nerslistpart>
           |  <nerslistpart> <nerslist>

<escape> := <backslash> <special>

<range> := CHARLIT MINUS CHARLIT

<eqclass> := LS COLON <id> COLON RS

<id> := CHARLIT
     |  CHARLIT <id>


*/ /*

type lexlLexer struct {
	grammar Grammar
}

func GenerateLexlGrammar() (Grammar, error) {
	gb := NewGrammarBuilder()
	gb.Rule("match").Terminal("CHARLIT")
	panic("unimplemented")
}

func NewLexlLexer() (Lexer, error) {
	g, err := GenerateLexlGrammar()
	if err != nil {
		return nil, err
	}
	lex := &lexlLexer{
		grammar: g,
	}
	return lex, nil
}

func (ll *lexlLexer) Grammar() Grammar {
	return ll.grammar
}

func (ll *lexlLexer) Open(in io.Reader) (LexerState, error) {
	var bin *bufio.Reader
	if b, ok := in.(*bufio.Reader); !ok {
		bin = bufio.NewReaderSize(in, 2)
	} else {
		bin = b
	}
	ls := &lexlLexerState{
		lexer: ll,
		in:    bin,
		line:  1,
		col:   1,
	}
	ig := GetIndexedGrammar(ll.grammar)
	termIndexIf, err := ig.GetIndex(GrammarIndexTypeTerm)
	if err != nil {
		return nil, err
	}
	termIndex := termIndexIf.(TermGrammarIndex)
	ls.tComment, err = termIndex.GetTerminal("COMMENT_OPEN")
	if err != nil {
		return nil, err
	}
	return ls, nil
}

type lexlLexerState struct {
	lexer        *lexlLexer
	in           *bufio.Reader
	line         int
	col          int
	pos          int
	blockstate   int
	eof          bool
	bottomSent   bool
	tCommentOpen Term
	tComment     Term
	tNl          Term
	tBottom      Term
	tMOPEN       Term
	tCOLON       Term
	tDIGIT       Term
	tALPHA       Term
	tFS          Term
	tWS          Term
	tIDENT       Term
	tWILDCARD    Term
	tCARET       Term
	tDOLLAR      Term
	tLC          Term
	tRC          Term
	tLP          Term
	tRP          Term
	tQM          Term
	tSTAR        Term
	tPLUS        Term
	tLS          Term
	tESCAPE      Term
	tCHARLIT     Term
	tPIPE        Term
}

func (ls *lexlLexerState) Lexer() Lexer {
	return ls.lexer
}

func (ls *lexlLexerState) Reader() io.Reader {
	return ls.in
}

func (ls *lexlLexerState) HasMoreTokens() (bool, error) {
	if ls.eof {
		return !ls.bottomSent, nil
	}
	_, err := ls.in.Peek(1)
	if err != nil {
		if err == io.EOF {
			ls.eof = true
			return true, nil
		}
		return false, err
	}
	return true, nil
}

func (ls *lexlLexerState) makeToken(term Term, lit string) Token {
	panic("unimplemented")
}

func (ls *lexlLexerState) NextToken() (Token, error) {
	if ls.eof {
		if ls.bottomSent {
			return nil, io.EOF
		}
		ls.bottomSent = true
		return ls.makeToken(ls.lexer.grammar.Bottom(), ""), nil
	}
	for {
		p, err := ls.in.Peek(2)
		if err != nil {
			if err == io.EOF {
				ls.eof = true
				return ls.NextToken()
			}
			return nil, err
		}
		switch ls.blockstate {
		case 0: // {0}
			{
				if string(p) == "//" {
					ls.in.Read(p)
					ls.blockstate = 1 // {comment}
					return ls.makeToken(ls.tCommentOpen, "//"), nil
				}
				ls.blockstate = 2 // {outer}
				continue
			}
		case 1: // {comment}
			{
				var comment []byte
				if p[0] == '\n' {
					ls.in.ReadByte()
					ls.blockstate = 0 // {0}
					return ls.makeToken(ls.tNl, ""), nil
				}
				for p[0] != '\n' {
					c, err := ls.in.ReadByte()
					if err != nil {
						return nil, err
					}
					comment = append(comment, c)
					p, err = ls.in.Peek(1)
					if err != nil {
						if err == io.EOF {
							ls.eof = true
							return ls.makeToken(ls.tComment, string(comment)), nil
						}
						return nil, err
					}
				}
				return ls.makeToken(ls.tComment, string(comment)), nil
			}
		case 2: // {outer}
			{
				for unicode.IsSpace(rune(p[0])) {
					ls.in.ReadByte()
					p, err = ls.in.Peek(2)
					if err != nil {
						if err == io.EOF {
							ls.eof = true
							ls.bottomSent = true
							return ls.makeToken(ls.tBottom, ""), nil
						}
						return nil, err
					}
				}
				if string(p) == "{{" {
					ls.in.Read(p)
					ls.blockstate = 3 // {matchset}
					return ls.makeToken(ls.tMOPEN, "{{"), nil
				}
				if p[0] == ':' {
					ls.in.ReadByte()
					return ls.makeToken(ls.tCOLON, ":"), nil
				}
				if p[0] >= '0' && p[0] <= '9' {
					ls.in.ReadByte()
					return ls.makeToken(ls.tDIGIT, string(p[0:1])), nil
				}
				if (p[0] >= 'a' && p[0] <= 'z') ||
					(p[0] >= 'A' && p[0] >= 'Z') ||
					(p[0] == '_') {
					return ls.makeToken(ls.tALPHA, string(p[0:1])), nil
				}
			}
		case 3: // {matchset}
			{
				if p[0] == '/' {
					ls.blockstate = 4 // {match}
					ls.in.ReadByte()
					return ls.makeToken(ls.tFS, "/"), nil
				}
				if unicode.IsSpace(rune(p[0])) {
					var ws []byte
					for unicode.IsSpace(rune(p[0])) {
						ws = append(ws, p[0])
						ls.in.ReadByte()
						p, err = ls.in.Peek(2)
						if err != nil {
							if err == io.EOF {
								ls.eof = true
								break
							}
							return nil, err
						}
					}
					return ls.makeToken(ls.tWS, string(ws)), nil
				}
				var ident []byte
				for (p[0] >= 'a' && p[0] <= 'z') ||
					(p[0] >= 'A' && p[0] <= 'Z') ||
					(p[0] >= '0' && p[0] <= '9') ||
					(p[0] == '_' || p[0] == '-') {
					ls.in.ReadByte()
					ident = append(ident, p[0])
					p, err = ls.in.Peek(2)
					if err != nil {
						if err == io.EOF {
							ls.eof = true
							break
						}
						return nil, err
					}
				}
				if len(ident) > 0 {
					return ls.makeToken(ls.tIDENT, string(ident)), nil
				}
			}
		case 4: // {match}
			{
				switch p[0] {
				case '.':
					{
						ls.in.ReadByte()
						return ls.makeToken(ls.tWILDCARD, "."), nil
					}
				case '^':
					{
						ls.in.ReadByte()
						return ls.makeToken(ls.tCARET, "^"), nil
					}
				case '$':
					{
						ls.in.ReadByte()
						return ls.makeToken(ls.tDOLLAR, "$"), nil
					}
				case '{':
					{
						ls.in.ReadByte()
						ls.blockstate = 5 // {quantifier}
						return ls.makeToken(ls.tLC, "{"), nil
					}
				case '}':
					{
						ls.in.ReadByte()
						return ls.makeToken(ls.tRC, "}"), nil
					}
				case '(':
					{
						ls.in.ReadByte()
						return ls.makeToken(ls.tLP, "("), nil
					}
				case ')':
					{
						ls.in.ReadByte()
						return ls.makeToken(ls.tRP, ")"), nil
					}
				case '?':
					{
						ls.in.ReadByte()
						return ls.makeToken(ls.tQM, "?"), nil
					}
				case '*':
					{
						ls.in.ReadByte()
						return ls.makeToken(ls.tSTAR, "*"), nil
					}
				case '+':
					{
						ls.in.ReadByte()
						return ls.makeToken(ls.tPLUS, "+"), nil
					}
				case '|':
					{
						ls.in.ReadByte()
						return ls.makeToken(ls.tPIPE, "|"), nil
					}
				case '[':
					{
						ls.in.ReadByte()
						ls.blockstate = 6 // {charclass}
						return ls.makeToken(ls.tLS, "["), nil
					}
				case '\\':
					{
						if len(p) == 2 {
							ls.in.Read(p)
							return ls.makeToken(ls.tESCAPE, string(p)), nil
						}
					}
				case '/':
					{
						ls.in.ReadByte()
						ls.blockstate = 3 // {matchset}
						return ls.makeToken(ls.tFS, "/"), nil
					}
				default:
					{
						ls.in.ReadByte()
						return ls.makeToken(ls.tCHARLIT, string(p[0:1])), nil
					}
				}
			}
		case 5:
			{
				if unicode.IsSpace(rune(p[0])) {
					var ws []byte
					for unicode.IsSpace(rune(p[0])) {
						ws = append(ws, p[0])
						ls.in.ReadByte()
						p, err = ls.in.Peek(2)
						if err != nil {
							if err == io.EOF {
								ls.eof = true
								break
							}
							return nil, err
						}
					}
					return ls.makeToken(ls.tWS, string(ws)), nil
				}

			}
		}
	}
}

/*
quantifier:{{
	RC			/\}/	{match}
	COMMA		/,/
	NUM			/0|[1-9][0-9]
}}
*/ /*

func (ls *lexlLexerState) CurrentLine() int {
	return ls.line
}

func (ls *lexlLexerState) CurrentColumn() int {
	return ls.col
}

func (ls *lexlLexerState) CurrentPosition() int {
	return ls.pos
}
*/
