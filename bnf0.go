package ngen

import (
	"bufio"
	"errors"
	"io"
	"reflect"
)

/*

	`* := <bnf0> `.

	<bnf0> 	:= <decl>
	       	|  <decl> <bnf0>

	<decl> 	:= <nt> EQDEF <optlist>

	<optlist> 	:= <opt>
	 			|  <opt> PIPE <optlist>

	<opt> 	:= 	<term>
			|	<term> <opt>

	<term> 	:=	<nt> | <t>

	<nt> 	:= 	LT ID RT
			|	AST

	<t>		:= ID
			|  EPS
			|  BOT

*/

func GenerateBnf0Grammar() Grammar {
	bnf0 := &stdGrammar{}
	bnf0.asterisk = &stdTerm{
		grammar: bnf0,
		nonterm: true,
		special: true,
		name:    "`*",
		id:      1,
	}
	bnf0.epsilon = &stdTerm{
		grammar: bnf0,
		special: true,
		name:    "`e",
		id:      2,
	}
	bnf0.bottom = &stdTerm{
		grammar: bnf0,
		name:    "`.",
		id:      3,
	}
	nextid := uint32(100)
	tr := int(nextid)
	terminals := []string{"AST", "BOT", "EPS", "EQDEF", "ID", "LT", "PIPE", "RT"}
	bnf0.terminals = make([]*stdTerm, len(terminals))
	for i, n := range terminals {
		bnf0.terminals[i] = &stdTerm{
			grammar: bnf0,
			name:    n,
			id:      nextid,
		}
		nextid++
	}
	nt := int(nextid)
	nonterminals := []string{"bnf0", "decl", "nt", "opt", "optlist", "t", "term"}
	bnf0.nonterminals = make([]*stdTerm, len(nonterminals))
	for i, n := range nonterminals {
		bnf0.nonterminals[i] = &stdTerm{
			grammar: bnf0,
			name:    n,
			id:      nextid,
			nonterm: true,
		}
		nextid++
	}
	productions := [][]int{
		[]int{-1, nt + 0, -3},
		[]int{nt + 0, nt + 1},
		[]int{nt + 0, nt + 1, nt + 0},
		[]int{nt + 1, nt + 2, tr + 3, nt + 4},
		[]int{nt + 4, nt + 3},
		[]int{nt + 4, nt + 3, tr + 6, nt + 4},
		[]int{nt + 3, nt + 6},
		[]int{nt + 3, nt + 6, nt + 3},
		[]int{nt + 6, nt + 2},
		[]int{nt + 6, nt + 5},
		[]int{nt + 2, tr + 5, tr + 4, tr + 7},
		[]int{nt + 2, tr + 0},
		[]int{nt + 5, tr + 4},
		[]int{nt + 5, tr + 2},
		[]int{nt + 5, tr + 1},
	}
	bnf0.productions = make([]*stdProduction, len(productions))
	for i, k := range productions {
		p := &stdProduction{
			grammar: bnf0,
			id:      nextid,
		}
		nextid++
		if k[0] < 0 {
			p.lhs = bnf0.asterisk
		} else {
			p.lhs = bnf0.nonterminals[k[0]-nt]
		}
		p.rhs = make([]Term, len(k)-1)
		for j, v := range k[1:] {
			//fmt.Printf("j: %d\n", j)
			if v < 0 {
				//fmt.Println("bot")
				p.rhs[j] = bnf0.bottom
			} else if v >= nt {
				//fmt.Println("nt")
				p.rhs[j] = bnf0.nonterminals[v-nt]
			} else {
				//fmt.Println("t")
				p.rhs[j] = bnf0.terminals[v-tr]
			}
			if reflect.ValueOf(p.rhs[j]).IsNil() {
				//panic("wtF")
			}
		}
		bnf0.productions[i] = p
		//fmt.Printf("bnf0: %s\n", ProductionRuleToString(p))
	}
	return bnf0
}

type bnf0Lexer struct {
	grammar Grammar
	ast     Term
	bot     Term
	eps     Term
	pipe    Term
	coleq   Term
	lt      Term
	rt      Term
	id      Term
}

type bnf0LexerState struct {
	lexer      *bnf0Lexer
	in         *bufio.Reader
	eof        bool
	sentBottom bool
	line       int
	col        int
	pos        int
}

func NewBnf0Lexer() (Lexer, error) {
	l := &bnf0Lexer{
		grammar: GenerateBnf0Grammar(),
	}
	ig := GetIndexedGrammar(l.grammar)
	idxIf, err := ig.GetIndex(GrammarIndexTypeTerm)
	if err != nil {
		return nil, err
	}
	termIndex := idxIf.(TermGrammarIndex)
	l.ast, _ = termIndex.GetTerm("AST")
	l.bot, _ = termIndex.GetTerm("BOT")
	l.eps, _ = termIndex.GetTerm("EPS")
	l.coleq, _ = termIndex.GetTerm("EQDEF")
	l.lt, _ = termIndex.GetTerm("LT")
	l.rt, _ = termIndex.GetTerm("RT")
	l.pipe, _ = termIndex.GetTerm("PIPE")
	l.id, _ = termIndex.GetTerm("ID")
	return l, nil
}

func (l *bnf0Lexer) Grammar() Grammar {
	return l.grammar
}

func (l *bnf0Lexer) Open(in io.Reader) (LexerState, error) {
	var bin *bufio.Reader
	if b, ok := in.(*bufio.Reader); !ok {
		bin = bufio.NewReaderSize(in, 2)
	} else {
		bin = b
	}
	state := &bnf0LexerState{
		lexer: l,
		in:    bin,
		line:  1,
		col:   1,
	}
	return state, nil
}

func (ls *bnf0LexerState) Lexer() Lexer {
	return ls.lexer
}

func (ls *bnf0LexerState) Reader() io.Reader {
	return ls.in
}

func (ls *bnf0LexerState) skipWhitespace() {
	k, err := ls.in.Peek(1)
	for err == nil {
		switch k[0] {
		case ' ':
			fallthrough
		case '\t':
			fallthrough
		case '\r':
			{
				ls.col++
				ls.pos++
				ls.in.ReadByte()
				k, err = ls.in.Peek(1)
				break
			}
		case '\n':
			{
				ls.col = 1
				ls.line++
				ls.pos++
				ls.in.ReadByte()
				k, err = ls.in.Peek(1)
				break
			}
		default:
			err = io.EOF
			break
		}
	}
}

func (ls *bnf0LexerState) HasMoreTokens() (bool, error) {
	if ls.eof {
		return !ls.sentBottom, nil
	}
	ls.skipWhitespace()
	if _, err := ls.in.Peek(1); err != nil {
		ls.eof = true
		if err == io.EOF {
			ls.eof = true
			return !ls.sentBottom, nil
		}
		return false, err
	}
	return true, nil
}

func (ls *bnf0LexerState) NextToken() (Token, error) {
	if ls.eof {
		if !ls.sentBottom {
			ls.sentBottom = true
			return ls.makeToken(ls.lexer.grammar.Bottom(), ""), nil
		}
		return nil, io.EOF
	}
	ls.skipWhitespace()
	pre, err := ls.in.Peek(2)
	if err != nil {
		return nil, err
	}
	k := string(pre)
	var tok Token
	switch k {
	case "`.":
		tok = ls.makeToken(ls.lexer.bot, k)
	case "`e":
		tok = ls.makeToken(ls.lexer.eps, k)
	case "`*":
		tok = ls.makeToken(ls.lexer.ast, k)
	case ":=":
		tok = ls.makeToken(ls.lexer.coleq, k)
	default:
		{
			switch k[0] {
			case byte('<'):
				tok = ls.makeToken(ls.lexer.lt, "<")
			case byte('>'):
				tok = ls.makeToken(ls.lexer.rt, ">")
			case byte('|'):
				tok = ls.makeToken(ls.lexer.pipe, "|")
			default:
				{
					var id []byte
					ca, err := ls.in.Peek(1)
					for err == nil && ls.isIdentifierPart(ca[0]) {
						id = append(id, ca[0])
						ls.in.ReadByte()
						ca, err = ls.in.Peek(1)
					}
					if err != nil {
						if err != io.EOF {
							return nil, err
						}
						ls.eof = true
					}
					if len(id) == 0 {
						return nil, errors.New("expected token")
					}
					tok = ls.makeToken(ls.lexer.id, string(id))
					return tok, nil
				}
			}
		}
	}
	ls.in.Discard(len(tok.Literal()))
	return tok, nil
}

func (ls *bnf0LexerState) CurrentLine() int {
	return ls.line
}

func (ls *bnf0LexerState) CurrentColumn() int {
	return ls.col
}

func (ls *bnf0LexerState) CurrentPosition() int {
	return ls.pos
}

func (ls *bnf0LexerState) makeToken(term Term, literal string) Token {
	tok := &bnf0Token{
		state: ls,
		term:  term,
	}
	ipos, iline, icol := ls.pos, ls.line, ls.col
	epos, eline, ecol := ipos, iline, icol
	if literal != "" {
		for c := range []byte(literal[1:]) {
			if c == '\n' {
				eline++
				ecol = 1
			} else {
				ecol++
			}
			epos++
		}
	}
	tok.pos = [6]int{ipos, iline, icol, epos, eline, ecol}
	tok.lit = literal
	return tok
}

func (ls *bnf0LexerState) isIdentifierPart(c byte) bool {
	return ((c >= 'a') && (c <= 'z')) ||
		((c >= 'A') && (c <= 'Z')) ||
		((c >= '0') && (c <= '9')) ||
		(c == '_') || (c == '-')
}

type bnf0Token struct {
	state LexerState
	pos   [6]int
	term  Term
	lit   string
}

func (t *bnf0Token) LexerState() LexerState {
	return t.state
}

func (t *bnf0Token) FirstPosition() int {
	return t.pos[0]
}

func (t *bnf0Token) LastPosition() int {
	return t.pos[3]
}

func (t *bnf0Token) FirstLine() int {
	return t.pos[1]
}

func (t *bnf0Token) LastLine() int {
	return t.pos[4]
}

func (t *bnf0Token) FirstColumn() int {
	return t.pos[2]
}

func (t *bnf0Token) LastColumn() int {
	return t.pos[5]
}

func (t *bnf0Token) Terminal() Term {
	return t.term
}

func (t *bnf0Token) Literal() string {
	return t.lit
}

type bnf0Decl struct {
	nt   string
	opts [][]Term
}

func GetGrammarFromBnf0Ast(bnf0 ParseTreeNode) (Grammar, error) {
	gb := NewGrammarBuilder()
	ntBnf := bnf0.Child(0)
	for {
		var lhsName string
		decl := ntBnf.Child(0)
		nt := decl.Child(0)
		optList := decl.Child(2)
		if nt.NumChildren() == 3 {
			lhsName = nt.Child(1).Token().Literal()
		} else {
			lhsName = nt.Child(0).Token().Literal()
		}
		for {
			opt := optList.Child(0)
			gb.Rule(lhsName)
			for {
				term := opt.Child(0).Child(0)
				if term.Production().Lhs().Name() == "nt" {
					if term.NumChildren() == 3 {
						gb.Nonterminal(term.Child(1).Token().Literal())
					} else {
						gb.Nonterminal(term.Child(0).Token().Literal())
					}
				} else {
					gb.Terminal(term.Child(0).Token().Literal())
				}
				if opt.NumChildren() == 1 {
					break
				}
				opt = opt.Child(1)
			}
			if optList.NumChildren() == 1 {
				break
			}
			optList = optList.Child(2)
		}
		if ntBnf.NumChildren() == 1 {
			break
		}
		ntBnf = ntBnf.Child(1)
	}
	return gb.Build()
}

/*
	<bnf0> 	:= <decl>
	       	|  <decl> <bnf0>

	<decl> 	:= <nt> EQDEF <optlist>

	<optlist> 	:= <opt>
	 			|  <opt> PIPE <optlist>

	<opt> 	:= 	<term>
			|	<term> <opt>

	<term> 	:=	<nt> | <t>

	<nt> 	:= 	LT ID RT
			|	AST
			|	BOT

	<t>		:= ID
			|  EPS */
