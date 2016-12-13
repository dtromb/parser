package lexl

import (
	"fmt"
	"testing"

	"github.com/dtromb/parser"
)

var lexl0Text string = `
0:{{
	COMMENT_OPEN	/\/\//	{comment}
	{{outer}}
}}

comment:{{
	NL				/\n/	{0}
	COMMENT			/[^\n]+/	
}}

outer:{{
	DIGIT			/[0-9]/
	ALPHA			/[a-zA-Z_]/
	MOPEN			/\{\{/ {matchset}
	COLON			/:/
	_				/\s+/
}}`

func TestLexlLexer(t *testing.T) {
	lexl0Grammar := GenerateLexl0Grammar()
	lexlL0sr := GenerateLexl0SR()
	for _, block := range lexlL0sr {
		fmt.Println(MatchBlockToString(block))
	}
	ndfa, err := lexlL0sr.ConstructLexlNdfa()
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println("NDFA:")
	fmt.Println(NdfaToString(ndfa))
	dfa, err := ndfa.TransformToDfa()
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println("DFA:")
	fmt.Println(DfaToString(dfa))
	lexer, err := GenerateLexlLexerFromDfa(dfa, lexl0Grammar)
	if err != nil {
		t.Error(err)
		return
	}
	in := parser.NewStringReader(lexl0Text)
	lex, err := lexer.Open(in)
	if err != nil {
		t.Error(err)
		return
	}
	for {
		hasMore, err := lex.HasMoreTokens()
		if err != nil {
			t.Error(err)
			return
		}
		if !hasMore {
			break
		}
		tok, err := lex.NextToken()
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Println(">" + tok.Terminal().Name())
	}
	fmt.Println("finished")
}
