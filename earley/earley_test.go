package earley

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"io"
	"bufio"
	"unicode"
	"github.com/dtromb/parser"
	"github.com/dtromb/parser/index"
)

func TestEarleyDFA(t *testing.T) {
	gb := parser.OpenGrammarBuilder()

	gb.Terminals("NONTERM","COLEQ","PIPE","IDENTIFIER").
	   Nonterminals("bnf","ntdecl","def","ntort"). 
	   Rule().Lhs("bnf").Rhs("ntdecl"). 
	   Rule().Lhs("bnf").Rhs("ntdecl","bnf").
       Rule().Lhs("ntdecl").Rhs("NONTERM","COLEQ","def"). 
	   Rule().Lhs("ntdecl").Rhs("ntdecl","PIPE","def"). 
	   Rule().Lhs("def").Rhs("ntort"). 
	   Rule().Lhs("def").Rhs("ntort","def"). 
	   Rule().Lhs("ntort").Rhs("IDENTIFIER"). 
	   Rule().Lhs("ntort").Rhs("NONTERM"). 
	   Rule().Lhs("`*").Rhs("bnf","`.").
	   Name("simple-bnf")
	
	g, err := gb.Build()
	if err != nil {
		t.Error(err)
	}
	
	grammar := parser.GetIndexedGrammar(g)
		dfa, err := BuildEpsilonLR0Dfa(grammar)
	if err != nil {
		t.Error()
		return
	}
	
	fmt.Printf("DFA has %d states\n", len(dfa.states))
	for i := 0; i < len(dfa.states); i++ {
		fmt.Println(dfa.states[i].String())
	}
}


func TestEarleyDFAEpsilons(t *testing.T) {
	// Build a simple BNF grammar description grammar.
	gb := parser.OpenGrammarBuilder()

	gb.Name("a4"). 
	   Terminals("a").
	   Nonterminals("S","A","E").
	   Rule().Lhs("`*").Rhs("S","`.").
	   Rule().Lhs("S").Rhs("A","A","A","A").
	   Rule().Lhs("A").Rhs("a").
	   Rule().Lhs("A").Rhs("E").
	   Rule().Lhs("E").Rhs("`e")
	
	g, err := gb.Build()
	if err != nil {
		t.Error(err)
	}
	grammar := parser.GetIndexedGrammar(g)

	dfa, err := BuildEpsilonLR0Dfa(grammar)
	if err != nil {
		t.Error()
		return
	}
	
	fmt.Printf("DFA has %d states\n", len(dfa.states))
	for i := 0; i < len(dfa.states); i++ {
		fmt.Println(dfa.states[i].String())
	}
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

type SimpleBnfLexer struct {
	in *bufio.Reader
	end bool
	eof bool
	line, col, pos int
	grammar parser.Grammar
	identifier, nonterm, coleq, pipe parser.GrammarTerminal 
}

func NewSimpleBnfLexer(g parser.Grammar) (*SimpleBnfLexer, error) {
	ig := parser.GetIndexedGrammar(g)
	idx, err := ig.GetIndex(index.NAME_INDEX)
	if err != nil {
		return nil, err
	}
	nidx := idx.(*index.NameIndex)
	identifier := nidx.Terminal("IDENTIFIER")
	if identifier == nil {
		return nil, errors.New("bnf grammar missing IDENTIFIER terminal")
	}
	nonterm := nidx.Terminal("NONTERM")
	if nonterm == nil {
		return nil, errors.New("bnf grammar missing NONTERM terminal")
	}
	coleq := nidx.Terminal("COLEQ")
	if coleq == nil {
		return nil, errors.New("bnf grammar missing COLEQ terminal")
	}
	pipe := nidx.Terminal("PIPE")
	if pipe == nil {
		return nil, errors.New("bnf grammar missing PIPE terminal")
	}
	return &SimpleBnfLexer{
		grammar: g,
		identifier: identifier.(parser.GrammarTerminal),
		nonterm: nonterm.(parser.GrammarTerminal),
		coleq: coleq.(parser.GrammarTerminal),
		pipe: pipe.(parser.GrammarTerminal),
	}, nil
}


func (sbl *SimpleBnfLexer) Eof() bool {
	return sbl.end
}

func (sbl *SimpleBnfLexer) Next() (parser.GrammarParticle,error) {
	var buf []rune
	var c rune
	var err error
	
	if sbl.end {
		if sbl.eof {
			return nil, errors.New("attempt to read past end of input")
		}
		sbl.eof = true
		return sbl.grammar.Bottom(), nil
	}
	
	// Ignore whitespace
	c, _, err = sbl.in.ReadRune()
	if err == io.EOF {
		sbl.end = true
		sbl.eof = true
		return sbl.grammar.Bottom(), nil
	}
	for unicode.IsSpace(c) {
		c, _, err = sbl.in.ReadRune()
		if err == io.EOF {
			sbl.end = true
			sbl.eof = true
			return sbl.grammar.Bottom(), nil
		}
		if err != nil {
			return nil, err
		}
	}
		
	if c == '<' { // Start of a NONTERM
		c, _, err  = sbl.in.ReadRune()
		if err != nil {
			return nil, errors.New("unexpected end of input after '<'")
		}
		for c != '>' {
			if c < 'a' || c > 'z' {
				return nil, errors.New("illegal character '"+string(c)+"' in nonterminal")
			}
			buf = append(buf, c)
			c, _, err = sbl.in.ReadRune()
			if err != nil {
				return nil, errors.New("unexpected end of input after '<'")
			}
		}
		return sbl.nonterm.Value(string(buf)), nil
	}
	
	if c == '|' {
		return sbl.pipe, nil
	}
	
	if c == ':' {
		c, _, err  = sbl.in.ReadRune()
		if err != nil {
			return nil, errors.New("unexpected end of input after ':'")
		}
		if c != '=' {
			return nil, errors.New("invalid character '"+string(c)+"' after ':'; expected ':='")
		}
		return sbl.coleq, nil
	}
	
	// start of identifier
	for c >= 'A' && c <= 'Z' {
		buf = append(buf, c)
		c, _, err  = sbl.in.ReadRune()
		if err == io.EOF {
			sbl.end = true
		}
	}
	
	if len(buf) == 0 {
		return nil, errors.New("invalid character '"+string(c)+"'")
	}
	
	return sbl.identifier.Value(string(buf)), nil
}

func (sbl *SimpleBnfLexer) Reset(in io.Reader) {
	sbl.in = bufio.NewReader(in)
	sbl.end = false
	sbl.line = 1
	sbl.col = 1
	sbl.pos = 1
}

func TestParser(t *testing.T) {
	gb := parser.OpenGrammarBuilder()

	gb.Terminals("NONTERM","COLEQ","PIPE","IDENTIFIER").
	   Nonterminals("bnf","ntdecl","def","ntort"). 
	   Rule().Lhs("bnf").Rhs("ntdecl"). 
	   Rule().Lhs("bnf").Rhs("ntdecl","bnf").
       Rule().Lhs("ntdecl").Rhs("NONTERM","COLEQ","def"). 
	   Rule().Lhs("ntdecl").Rhs("ntdecl","PIPE","def"). 
	   Rule().Lhs("def").Rhs("ntort"). 
	   Rule().Lhs("def").Rhs("ntort","def"). 
	   Rule().Lhs("ntort").Rhs("IDENTIFIER"). 
	   Rule().Lhs("ntort").Rhs("NONTERM"). 
	   Rule().Lhs("`*").Rhs("bnf","`.").
	   Name("simple-bnf")
	
	g, err := gb.Build()
	if err != nil {
		t.Error(err)
	}
	
	metaBnf := `
	
		<bnf> := <ntdecl>
		      |  <ntdecl> <bnf>
		
		<ntdecl> := NONTERM COLEQ <def>
		         |  <ntdecl> PIPE <def>
				
		<def> := <ntort>
			  |  <ntort> <def>
		
		<ntort> := IDENTIFIER
		        |  NONTERM
				
	`
	lexer, err := NewSimpleBnfLexer(g)
	if err != nil {
		t.Error(err)
		return
	}

	lexer.Reset(NewStringReader(metaBnf))
	p, err := GenerateParser(g)
	if err != nil {
		t.Error(err)
		return
	}
	
	var ast parser.SyntaxTreeNode
	ast, err = p.Parse(lexer, nil) 
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println("Got AST.")
	fmt.Printf("Top is: %s\n", ast.Part())
	
	cmap := make(map[parser.SyntaxTreeNode]int)
	outmap := make(map[int]string)
	var cnid, chid int
	
	DumpTree(ast.(*astNode))
	
	stack := []parser.SyntaxTreeNode{ast}
	for len(stack) > 0 {
		var buf []byte 
		cn := stack[len(stack)-1]
		stack = stack[0:len(stack)-1]
		if id, has := cmap[cn]; !has {
			cnid = len(cmap)
			cmap[cn] = cnid
		} else {
			cnid = id
		}
		var desc string
		if cn.Part() == nil {
			desc = "-"
		} else {
			desc = cn.Part().String()
		}
		var valstr string
		if cn.NumChildren() == 0 {
			if vt, isVal := cn.Part().(*parser.ValueTerminal); isVal {
				val := vt.Value()
				if s, isStr := val.(string); isStr {
					valstr = fmt.Sprintf("\"%s\"", s)
				} else if s, isStr := val.(Stringable); isStr {
					valstr = fmt.Sprintf("\"%s\"",s.String())
				} else {
					valstr = fmt.Sprintf("0x%8.8X", reflect.ValueOf(val).Pointer())
				}
			} else {
				valstr = "-"
			}
			var teststr string
			if cn.(*astNode).left == nil {
				teststr = "."
			} else {
				teststr = fmt.Sprintf("%d", cn.(*astNode).left.first)
			}
			buf = append(buf, fmt.Sprintf("[%d: leaf %s %s %s (%d-%d)]", cnid, teststr, desc, valstr, cn.First(), cn.Last())...)
		} else {
			buf = append(buf, fmt.Sprintf("[%d: %s {", cnid, cn.Rule().String())...)
			for i := 0; i < cn.NumChildren(); i++ {
				child := cn.Child(i)
				stack = append(stack,child)
				if id, has := cmap[child]; !has {
					chid = len(cmap)
					cmap[child] = chid
				} else {
					chid = id
				}
				buf = append(buf, fmt.Sprintf("%d",chid)...)
				if i < cn.NumChildren()-1 {
					buf = append(buf, ","...)
				}
			}
			buf = append(buf, fmt.Sprintf("} (%d-%d)]", cn.First(), cn.Last())...)
		}
		outmap[cnid] = string(buf)
	}
	for i := 0; i < len(outmap); i++ {
		fmt.Println(outmap[i])
	}
}


func TestPValues(t *testing.T) {
	gb := parser.OpenGrammarBuilder()

	gb.Name("simple-calculator").
	   Terminals().
	   Nonterminals().
	   
	