package lexr

import (
	"fmt"
	"bytes"
	"testing"
)

var input string = `

0:{{
    _ /[\n\f\r \t]+/
    COMMENT_OPEN /\/\// {comment}
    LABEL /[_0-9A-Za-z]+/ {0}
    MOPEN /\{\{/ {matchset}
    COLON /:/ {0}
}}

comment:{{
    NL /\n/ {0}
    COMMENT /[^\n]+/ {comment}
}}

outer:{{
    _ /[\t\n\f\r ]+/
    LABEL /[_0-9A-Za-z]+/ {outer}
    MOPEN /\{\{/ {matchset}
    COLON /:/ {outer}
}}
matchset:{{
    WS /[\t\n\f\r ]+/ {matchset}
    FS /\// {match}
    IDENT /[-_0-9A-Za-z]+/ {matchset}
    LC /\{/ {transition}
    MCLOSE /\}\}/ {0}
}}

transition:{{
    LABEL /[_0-9A-Za-z]+/ {transition}
    LC /\{/ {inclusion}
    RC /\}/ {matchset}
}}
inclusion:{{
    LABEL /[_0-9A-Za-z]+/ {inclusion}
    RC /\}/ {transition}
}}

match:{{
    DOT /\./ {match}
    CARET /\^/ {match}
    DOLLAR /\$/ {match}
    BS /\\/ {escape}
    LC /\{/ {quantifier}
    RC /\}/ {match}
    LP /\(/ {match}
    RP /\)/ {match}
    QM /\?/ {match}
    STAR /\*/ {match}
    PLUS /\+/ {match}
    PIPE /\|/ {match}
    LS /\[/ {class}
    FS /\// {matchset}
    CHARLIT /./ {match}
}}

escape:{{
    CPOINT /x[0-9]{4}/ {match}
    NL /n/ {match}
    FF /f/ {match}
    RT /r/ {match}
    TAB /t/ {match}
    ZERO /0/ {match}
    WS /s/ {match}
    CHARLIT /./ {match}
}}

quantifier:{{
    _ /[ \t\n\f\r]+/
    RC /\}/ {match}
    COMMA /,/ {quantifier}
    NUM /0|[1-9][0-9]*/ {quantifier}
}}

class:{{
    MINUS /-/ {class}
    RS /\]/ {match}
    BS /\\/ {classescape}
    DOT /\./ {class}
    CARET /\^/ {class}
    DOLLAR /\$/ {class}
    LC /\{/ {quantifier}
    RC /\}/ {class}
    LP /\(/ {class}
    RP /\)/ {class}
    QM /\?/ {class}
    STAR /\*/ {class}
    PLUS /\+/ {class}
    PIPE /\|/ {class}
    LS /\[/ {class}
    FS /\// {matchset}
    CHARLIT /./ {class}
}}

classescape:{{
    CPOINT /x[0-9]{4}/ {class}
    NL /n/ {class}
    FF /f/ {class}
    RT /r/ {class}
    TAB /t/ {class}
    ZERO /0/ {class}
    WS /s/ {class}
    CHARLIT /./ {class}
}}
`

func TestLexer(t *testing.T) {
	
	lexr0Grammar := GenerateLexr0Grammar()
	lexr0Domain := GenerateLexr0Domain(lexr0Grammar)
	lexer, err := CreateLexrLexer(lexr0Domain)
	if err != nil {
		t.Error(err)
		return
	}
	in := bytes.NewReader([]byte(input))
	lex, err := lexer.Open(in)
	if err != nil {
		t.Error()
		return
	}
	for {
		more, err := lex.HasMoreTokens()
		if err != nil {
			t.Error(err)
			return
		}
		if !more { break }
		tok, err := lex.NextToken()
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Println("<<"+tok.Terminal().Name()+" "+tok.Literal()+">>")
	}
}