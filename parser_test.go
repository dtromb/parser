package parser

import (
	"fmt"
	"testing"
)

var bnf1InText string = "\n" +
	"`* := <bnf1> `.						\n" +
	"										\n" +
	"<bnf1> 	:= <decl>					\n" +
	"       	|  <decl> <bnf1>			\n" +
	"										\n" +
	"<decl> 	:= <nt> EQDEF <optlist>		\n" +
	"										\n" +
	"<optlist> 	:= <opt>					\n" +
	" 			|  <opt> PIPE <optlist>		\n" +
	"										\n" +
	"<opt> 	:= 	<term>						\n" +
	"		|	<term> <opt>				\n" +
	"										\n" +
	"<term> 	:=	<nt> | <t>				\n" +
	"										\n" +
	"<nt> 	:= 	LT ID RT					\n" +
	"		|	AST							\n" +
	"										\n" +
	"<t>		:= ID						\n" +
	"		|  EPS							\n" +
	"		|  BOT							\n"

func TestParser(t *testing.T) {
	lexer, err := NewBnf0Lexer()
	if err != nil {
		t.Error(err)
		return
	}
	parser, err := GenerateEarleyParser(lexer.Grammar())
	if err != nil {
		t.Error(err)
		return
	}
	in := NewStringReader(bnf1InText)
	lex, _ := lexer.Open(in)
	ok, err := lex.HasMoreTokens()
	for ok && err == nil {
		tok, err := lex.NextToken()
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Printf("%s (%s)\n", tok.Terminal().Name(), tok.Literal())
		ok, err = lex.HasMoreTokens()
	}
	if err != nil {
		t.Error(err)
		return
	}
	in = NewStringReader(bnf1InText)
	lex, _ = lexer.Open(in)
	ps, err := parser.Open(lex)
	if err != nil {
		t.Error(err)
		return
	}
	ast, err := ps.Parse()
	if err != nil {
		t.Error(err)
		return
	}
	printAst(ast, 0)
	g2, err := GetGrammarFromBnf0Ast(ast)
	if err != nil {
		t.Error(err)
		return
	}
	g2 = g2
	fmt.Println("OK")
}

func printAst(n ParseTreeNode, indent int) {
	for i := 0; i < indent; i++ {
		fmt.Print(" ")
	}
	if n.Production() != nil {
		fmt.Printf("%s\n", TermToString(n.Production().Lhs()))
		for i := 0; i < n.Production().RhsLen(); i++ {
			printAst(n.Child(i), indent+2)
		}
	} else {
		fmt.Printf("%s '%s'\n", TermToString(n.Token().Terminal()), n.Token().Literal())
	}
}
