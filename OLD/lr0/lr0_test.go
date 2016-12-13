package lr0

import (
	"fmt"
	"testing"
	"github.com/dtromb/parser"
)

func TestGrammarIndex(t *testing.T) {
	// Build a simple BNF grammar description grammar.
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
	fmt.Printf("`*: 0x%8.8X\n", grammar.Asterisk())
	dfa, err := BuildDfa(grammar, false)
	
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("DFA has %d states\n", len(dfa.states))
	
	for i := 0; i < len(dfa.states); i++ {
		fmt.Println(dfa.states[i])
	}
	fmt.Println("Done.")
}
	