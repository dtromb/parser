package parser

import (
	"fmt"
	"testing"
)

func TestGrammar(t *testing.T) {
	bnf0 := GenerateBnf0Grammar()
	g := GetIndexedGrammar(bnf0)
	idxIf, err := g.GetIndex(GrammarIndexTypeTerm)
	if err != nil {
		t.Error(err)
		return
	}
	termIndex := idxIf.(TermGrammarIndex)
	for _, ntn := range termIndex.GetNonterminalNames() {
		nt, _ := termIndex.GetNonterminal(ntn)
		fmt.Printf("%d: <%s>\n", nt.Id(), nt.Name())
	}
	for _, tn := range termIndex.GetTerminalNames() {
		t, _ := termIndex.GetTerminal(tn)
		fmt.Printf("%d: %s\n", t.Id(), t.Name())
	}
}
