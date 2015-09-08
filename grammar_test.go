package parser

import (
	"fmt"
	"testing"
	"strings"
)

/*

First:

bnf:	ntdecl, NONTERM
ntdecl:	ntdecl, NONTERM
def: ntort, IDENTIFIER, NONTERM
ntort: IDENTIFIER, NONTERM


<bnf>		`.
<ntdecl>	<bnf>. <bnf> PIPE
NONTERM		COLEQ <ntort>.
COLEQ		<def>
<def>		<ntdecl>.
PIPE		<def>
<ntort>		<def>. <def>
IDENTIFIER	<ntort>.


<bnf>		`.
<ntdecl>	<bnf>. <bnf> PIPE `.
NONTERM		COLEQ <ntort>. <def>. <def> <ntdecl>. <bnf>. <bnf> PIPE `.
COLEQ		<def> 
<def>		<ntdecl>. <bnf>. <bnf> PIPE `.
PIPE		<def>
<ntort>		<def>. <def> <ntdecl>. <bnf>. <bnf> PIPE `.
IDENTIFIER	<ntort>. <def>. <def> <ntdecl>. <bnf>. <bnf> PIPE `.

<bnf>		`.
<ntdecl>	<bnf> PIPE `.
<def>		<bnf> PIPE `.
<ntort>		<def> <bnf> PIPE `.
NONTERM		<def> <bnf> COLEQ PIPE `.
COLEQ		<def>
PIPE		<def>
IDENTIFIER	<def> <bnf> PIPE `.


0	`* := <bnf> `.

1	<bnf> := <ntdecl> 
2		   | <ntdecl> <bnf>
		
3	<ntdecl> := NONTERM COLEQ <def>
4			  | <ntdecl> PIPE <def>
			
5	<def> := <ntort>
6	       | <ntort> <def>
		
7	<ntort> := IDENTIFIER
	         | NONTERM

Reading <ntdecl> ...

    .NT
		3,ntdecl: . NT C <def>
		4,ntdecl: . <ntdecl> P <def>
		3,ntdecl: NT C . <def>
		4,ntdecl: <ntdecl> P . <def>
	.ID 
		3,ntdecl: NT C . <def>
		4,ntdecl: <ntdecl> P . <def>
	.C
		3,ntdecl: NT . C <def>
	.P
		4,ntdecl: <ntdecl> . P <def>
	

*/
func TestGrammarBuilder(t *testing.T) {
	// Build a simple BNF grammar description grammar.
	gb := OpenGrammarBuilder()

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
	
	grammar, err := gb.Build()
	if err != nil {
		t.Error(err)
	}
	fmt.Println("Name: "+grammar.Name())
	terms := make([]string, grammar.NumTerminals())
	for i, t := range grammar.Terminals() {
		terms[i] = t.String()
	}
	nterms := make([]string, grammar.NumNonterminals())
	for i, t := range grammar.Nonterminals() {
		nterms[i] = t.String()
	}
	fmt.Println("Terminals: "+strings.Join(terms, ", "))
	fmt.Println("Nonterminals: "+strings.Join(nterms, ", "))
	fmt.Println("Productions:")
	for _, p := range grammar.Productions() {
		fmt.Println("   "+p.String())
	}
}

func Foo(j int) bool {
	return j % 2 == 0
}

func TestCrud(t *testing.T) {
	for i := 0; i < 10; i++ {
	   	for j := 0; j < i; j++ {
			if j < 3 {
				fmt.Printf("%d < %d\n", j, 3)
				break
			} else if j > 3 {
				fmt.Printf("%d > %d\n", j, 3) 
				if !Foo(j) {
					break
				}
			} else {
				fmt.Printf("%d = 3\n", j)
				break
			}
			fmt.Println("WTF")
		}
	}
}