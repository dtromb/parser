package earley

import (
	"fmt"
	"testing"
	"strings"
	"github.com/dtromb/parser"
)

func TestNNF(t *testing.T) {
	// Build a simple BNF aGrammar description aGrammar.
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
		return
	}

	var aGrammar parser.Grammar
	var rTransform parser.SyntaxTreeTransform
	
	nnf, err := IsNihilisticNormalForm(g) 
	if err != nil {
		t.Error()
		return
	}
	if !nnf {
		fmt.Println("Grammar is not NNF, transforming.")
		aGrammar, rTransform, err = GetNihilisticAugmentGrammar(g)
		if err != nil {
			t.Error(err)
			return
		}
	} else {
		t.Error("Grammar returned NNF.")
		return
	}
	
	fmt.Println("Name: "+aGrammar.Name())
	terms := make([]string, aGrammar.NumTerminals())
	for i, t := range aGrammar.Terminals() {
		terms[i] = t.String()
	}
	nterms := make([]string, aGrammar.NumNonterminals())
	for i, t := range aGrammar.Nonterminals() {
		nterms[i] = t.String()
	}
	fmt.Println("Terminals: "+strings.Join(terms, ", "))
	fmt.Println("Nonterminals: "+strings.Join(nterms, ", "))
	fmt.Println("Productions:")
	for _, p := range aGrammar.Productions() {
		fmt.Println("   "+p.String())
	}
	rTransform = rTransform
}