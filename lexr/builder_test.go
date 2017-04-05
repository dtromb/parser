package lexr


import (
	"fmt"
	"testing"
	"os"
)

func TestDomainBuilder(t *testing.T) {
	lexr0Grammar := GenerateLexr0Grammar()
	lexr0Domain := GenerateLexr0Domain(lexr0Grammar)
	WriteDomainLexr0(os.Stdout, lexr0Domain)
	ndfas, err := lexr0Domain.GenerateNdfas()
	if err != nil {
		t.Error(err)
		return
	}
	for i, ndfa := range ndfas {
		fmt.Printf("NDFA FOR: {%s}\n", lexr0Domain.Block(i).Name())
		WriteNdfa(ndfa, os.Stdout)
	}
	offset := 0
	for i, ndfa := range ndfas {
		fmt.Printf("DFA FOR: {%s}\n", lexr0Domain.Block(i).Name())
		dfa, info, err := GenerateDomainDfaFromNdfa(ndfa, offset, lexr0Domain.Grammar())
		if err != nil {
			t.Error(err)
			return
		}
		offset += len(info)
		WriteDfa(dfa, os.Stdout)
	}
}