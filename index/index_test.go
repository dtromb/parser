package index

import (
	"fmt"
	"testing"
	"strings"
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
	
	idx, err := grammar.GetIndex("basic")
	if err != nil {
		t.Error()
		return
	}
	basicIndex := idx.(*BasicGrammarIndex)
	fmt.Println("Basic index type: '"+basicIndex.IndexName()+"'")
	fmt.Println("Production RHS starts: ")
	for idx := 0; idx < grammar.NumTerminals(); idx++ {
		term := grammar.Terminal(idx)
		starts := basicIndex.RhsStarts(term)
		if len(starts) == 0 {
			continue
		}
		fmt.Println("  "+term.String()+":")
		for _, p := range starts {
			fmt.Println("   "+p.String())
		}
	}	
	fmt.Println("\nProduction RHS ends: ")
	for idx := 0; idx < grammar.NumTerminals(); idx++ {
		term := grammar.Terminal(idx)
		starts := basicIndex.RhsEnds(term)
		if len(starts) == 0 {
			continue
		}
		fmt.Println("  "+term.String()+":")
		for _, p := range starts {
			fmt.Println("   "+p.String())
		}
	}	
	fmt.Println("\nProduction RHS contains: ")
	for idx := 0; idx < grammar.NumTerminals(); idx++ {
		term := grammar.Terminal(idx)
		starts := basicIndex.RhsContains(term)
		if len(starts) == 0 {
			continue
		}
		fmt.Println("  "+term.String()+":")
		for _, p := range starts {
			fmt.Println("   "+p.String())
		}
	}
	fmt.Println("Grammar class:") 
	idx, err = grammar.GetIndex(GRAMMAR_CLASS_INDEX)
	if err != nil {
		t.Error(err)
		return
	}
	gcidx := idx.(*GrammarClassIndex)
	fmt.Println("       type: "+gcidx.Class().String())
	fmt.Println(" regularity: "+gcidx.Regularity().String())
	idx, err = grammar.GetIndex(FIRST_FOLLOW_INDEX)
	if err != nil {
		t.Error(err)
		return
	}
	ffidx := idx.(*FFIndex)
	fmt.Println("FIRST(x): ")
	for _, p := range g.Nonterminals() {
		fmt.Println("  "+p.String())
		for _, k := range ffidx.Firsts(p) {
			fmt.Println("    "+k.String())
		}
	}
	fmt.Println("FOLLOW(x): ")
	for _, p := range g.Nonterminals() {
		fmt.Println("  "+p.String())
		for _, k := range ffidx.Follows(p) {
			fmt.Println("    "+k.String())
		}
	}
	for _, p := range g.Terminals() {
		fmt.Println("  "+p.String())
		for _, k := range ffidx.Firsts(p) {
			fmt.Println("    "+k.String())
		}
	}
}