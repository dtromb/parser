package lexl

import "github.com/dtromb/parser"

// Metagrammar that describes the LEXL lexer generator description language.

var lex0 string = `

// This is the lexer block's grammar definition.


0:{{
	COMMENT_OPEN	/\/\//	{comment}
	{{outer}}
}}

comment:{{
	NL				/\n/	{0}
	COMMENT			/[^\n]+/	
}}

outer:{{
	LABEL			/[0-9a-zA-Z_]+/
	MOPEN			/\{\{/ {matchset}
	COLON			/:/
	_				/\s+/
}}

matchset:{{
	WS				/\s+/
	FS				/\// {match}
	IDENT			/[a-zA-Z0-9_-]+/
	LC				/\{/ {transition}
	MCLOSE			/\}\}/ {0}
}}

transition:{{
	LABEL			/[0-9a-zA-Z_]+/
	LC				/\{/ {inclusion}
	RC				/\}/ {matchset}
}}

inclusion:{{
	LABEL			/[0-9a-zA-Z_]+/
	RC				/}/ {transition}
}}

match:{{
	DOT 		/\./
	CARET		/\^/
	END			/\$/
	BS			/\\/	{escape}
	LC			/\{/	{quantifier}
	RC			/\}/
	LP			/\(/
	RP			/\)/
	QM			/\?/
	STAR		/\*/
	PLUS		/\+/
	PIPE		/\|/
	LS			/\[/	{class}
	FS			/\//	{matchset}
	CHARLIT		/./
}}

escape:{{
	{match}
	CPOINT		/x[0-9]{4}/		
	NL			/n/				
	FF			/f/				
	RT			/r/				
	TAB			/t/				
	ZERO		/0/
	CHARLIT		/./
}}


quantifier:{{
	RC			/\}/	{match}
	_			/\s+/
	COMMA		/,/
	NUM			/0|[1-9][0-9]*/
}}

class:{{
	MINUS 		/-/
	RS			/\]/	{match}
	BS			/\\/	{classescape}
	{{match}}
}}

classescape:{{
	{class}
	CPOINT		/x[0-9]{4}/		
	NL			/n/				
	FF			/f/				
	RT			/r/				
	TAB			/t/				
	ZERO		/0/
	CHARLIT		/./
}}

<lexl> 			:= 	<lexerBlock>
				|	<comment>
               	| 	<lexerBlock> <lexl>
				|	<comment> <lexl>
				
<comment>		:=	COMMENT_OPEN COMMENT NL

<lexerBlock>	:=	LABEL COLON MOPEN <terminalDefList> MCLOSE

<terminalDefList> 	:=	<terminalDef>
					|	<terminalDef> <terminalDefList>

<terminalDef>		:=	<optWs> IDENT <optWs> FS <match> FS
					|	<optWs> LC LABEL RC
					|	<optWs> LC LC LABEL RC RC

<optWs>			:=	WS
				|	{e}
				
<match> :=	<nonalt>
		|	<alternation>

<nonalt>	:= 	CHARLIT
			|  	WILDCARD
			|	CARET
			|	END
			|	<subexp>
			|	<optional>
			|	<star>
			|   <plus>
			|	<quantified>
			|	<charset>
			|	<escape>
		
<alternation>	:=	<nonalt> PIPE <match>

<subexp> 	:= LP <match> RP

<optional> 	:= <match> QM

<star> 		:= <match> STAR

<plus> 		:= <match> PLUS

<quantified> 	:= 	<match> LC NUM RC
				|	<match> LC NUM COMMA RC	
				|  	<match> LC NUM COMMA NUM RC
			
<charset>		:= 	LS <charlist> RS
				|	LS CARET <charlist> RS
				
<charlist>		:=	<charitem>
				|	<charitem> <charlist>

<charitem>		:= 	<escChar>
				|	<range>
				
<escChar> 		:= 	CHARLIT
				|  	<escape>
				
<range>			:= <escChar> MINUS <escChar>

<escape>		:=	BS <special>
				|	BS CHARLIT
				|	BS CPOINT
				
<special>		:= 	NL | FF | RT | TAB | ZERO


`

func GenerateLexl0Grammar() parser.Grammar {
	g := parser.NewGrammarBuilder()
	g.Rule("lexl0").Nonterminal("lexerBlock")
	g.Rule("lexl0").Nonterminal("comment")
	g.Rule("lexl0").Nonterminal("lexerBlock").Nonterminal("lexl0")
	g.Rule("lexl0").Nonterminal("comment").Nonterminal("lexl0")
	g.Rule("comment").Terminal("COMMENT_OPEN").Terminal("COMMENT").Terminal("NL")
	g.Rule("lexerBlock").Terminal("LABEL").Terminal("COLON").Terminal("MOPEN").Nonterminal("terminalDefList").Terminal("MCLOSE")
	g.Rule("terminalDefList").Nonterminal("terminalDef")
	g.Rule("terminalDefList").Nonterminal("terminalDef").Nonterminal("terminalDefList")
	g.Rule("terminalDef").Nonterminal("optws").Terminal("IDENT").Nonterminal("optws").Terminal("FS").Nonterminal("MATCH").Terminal("FS")
	g.Rule("terminalDef").Nonterminal("optws").Terminal("LC").Nonterminal("optws").Terminal("LABEL").Nonterminal("optws").Terminal("RC")
	g.Rule("terminalDef").Nonterminal("optWs").Terminal("LC").Terminal("LC").Terminal("LABEL").Nonterminal("optws").Terminal("RC").Terminal("RC")
	g.Rule("optws").Terminal("WS")
	g.Rule("optWs").Terminal("`e")
	g.Rule("match").Nonterminal("nonalt")
	g.Rule("match").Nonterminal("alteration")
	g.Rule("nonalt").Terminal("CHARLIT")
	g.Rule("nonalt").Terminal("DOT")
	g.Rule("nonalt").Terminal("CARET")
	g.Rule("nonalt").Terminal("DOLLAR")
	g.Rule("nonalt").Nonterminal("subexp")
	g.Rule("nonalt").Nonterminal("optional")
	g.Rule("nonalt").Nonterminal("star")
	g.Rule("nonalt").Nonterminal("plus")
	g.Rule("nonalt").Nonterminal("quantified")
	g.Rule("nonalt").Nonterminal("charset")
	g.Rule("nonalt").Nonterminal("escape")
	g.Rule("alternation").Nonterminal("nonalt").Terminal("PIPE").Nonterminal("match")
	g.Rule("subexp").Terminal("LP").Nonterminal("match").Terminal("RP")
	g.Rule("optional").Nonterminal("match").Terminal("QM")
	g.Rule("star").Nonterminal("match").Terminal("STAR")
	g.Rule("plus").Nonterminal("match").Terminal("PLUS")
	g.Rule("quantified").Nonterminal("match").Terminal("LC").Terminal("NUM").Terminal("RC")
	g.Rule("quantified").Nonterminal("match").Terminal("LC").Terminal("NUM").Terminal("COMMA").Terminal("RC")
	g.Rule("quantified").Nonterminal("match").Terminal("LC").Terminal("NUM").Terminal("COMMA").Terminal("NUM").Terminal("RC")
	g.Rule("charset").Terminal("LS").Nonterminal("charlist").Terminal("RS")
	g.Rule("charset").Terminal("LS").Terminal("CARET").Nonterminal("charlist").Terminal("RS")
	g.Rule("charlist").Nonterminal("charitem")
	g.Rule("charlist").Nonterminal("charitem").Nonterminal("charlist")
	g.Rule("charitem").Nonterminal("escChar")
	g.Rule("charitem").Nonterminal("range")
	g.Rule("escChar").Terminal("CHARLIT")
	g.Rule("escChar").Nonterminal("escape")
	g.Rule("range").Nonterminal("escChar").Terminal("MINUS").Nonterminal("escChar")
	g.Rule("escape").Terminal("BS").Nonterminal("special")
	g.Rule("escape").Terminal("BS").Terminal("CPOINT")
	g.Rule("escape").Terminal("BS").Terminal("CHARLIT")
	g.Rule("special").Terminal("NL")
	g.Rule("special").Terminal("FF")
	g.Rule("special").Terminal("RT")
	g.Rule("special").Terminal("TAB")
	g.Rule("special").Terminal("ZERO")
	grammar, err := g.Build()
	if err != nil {
		panic(err.Error())
	}
	return grammar
}

func GenerateLexl0SR() LexlRepresentation {
	blocks := make([]MatchBlock, 0, 1)

	///// {{0}}
	blocks = append(blocks, &stdLexlMatchBlock{
		/*  0:{{
			   COMMENT_OPEN	/\/\//	{comment}
		      {{outer}}
			}}  */
		blockName: "0",
		termdefs: []*stdLexlTermdef{
			&stdLexlTermdef{
				terminalName: "COMMENT_OPEN",
				expr: newLexlSequenceExpr(
					newLexlCharacterLiteralExpr('/'),
					newLexlCharacterLiteralExpr('/'),
				),
			},
		},
	})
	block0 := blocks[0].(*stdLexlMatchBlock)
	tCommentOpen := block0.termdefs[0]
	tCommentOpen.defBlock = block0
	tCommentOpen.fwdBlock = block0

	///// {{comment}}
	nlClass := newCharacterClass()
	nlClass.Negate()
	nlClass.AddCharacter('\n')
	blocks = append(blocks, &stdLexlMatchBlock{
		/*
			comment:{{
				NL				/\n/	{0}
				COMMENT			/[^\n]+/
			}}
		*/
		blockName: "comment",
		termdefs: []*stdLexlTermdef{
			&stdLexlTermdef{
				terminalName: "NL",
				expr:         newLexlCharacterLiteralExpr('\n'),
			},
			&stdLexlTermdef{
				terminalName: "COMMENT",
				expr:         newLexlPlusExpr(newLexlCharacterClassExpr(nlClass)),
			},
		},
	})

	///// {{match}}
	/*
		match:{{
			DOT 		/\./
			CARET		/\^/
			END			/\$/
			BS			/\\/	{escape}
			LC			/\{/	{quantifier}
			RC			/\}/
			LP			/\(/
			RP			/\)/
			QM			/\?/
			STAR		/ \ * /
			PLUS		/\+/
			PIPE		/\|/
			LS			/\[/	{class}
			FS			/\//	{matchset}
			CHARLIT		/./ /*
		}}
	*/
	blocks = append(blocks, &stdLexlMatchBlock{
		blockName: "match",
		termdefs: []*stdLexlTermdef{
			&stdLexlTermdef{
				terminalName: "DOT",
				expr:         newLexlCharacterLiteralExpr('.'),
			},
			&stdLexlTermdef{
				terminalName: "CARET",
				expr:         newLexlCharacterLiteralExpr('^'),
			},
			&stdLexlTermdef{
				terminalName: "DOLLAR",
				expr:         newLexlCharacterLiteralExpr('$'),
			},
			&stdLexlTermdef{
				terminalName: "BS",
				expr:         newLexlCharacterLiteralExpr('\\'),
			},
			&stdLexlTermdef{
				terminalName: "LC",
				expr:         newLexlCharacterLiteralExpr('{'),
			},
			&stdLexlTermdef{
				terminalName: "RC",
				expr:         newLexlCharacterLiteralExpr('}'),
			},
			&stdLexlTermdef{
				terminalName: "LP",
				expr:         newLexlCharacterLiteralExpr('('),
			},
			&stdLexlTermdef{
				terminalName: "RP",
				expr:         newLexlCharacterLiteralExpr(')'),
			},
			&stdLexlTermdef{
				terminalName: "QM",
				expr:         newLexlCharacterLiteralExpr('?'),
			},
			&stdLexlTermdef{
				terminalName: "STAR",
				expr:         newLexlCharacterLiteralExpr('*'),
			},
			&stdLexlTermdef{
				terminalName: "PLUS",
				expr:         newLexlCharacterLiteralExpr('+'),
			},
			&stdLexlTermdef{
				terminalName: "PIPE",
				expr:         newLexlCharacterLiteralExpr('|'),
			},
			&stdLexlTermdef{
				terminalName: "LS",
				expr:         newLexlCharacterLiteralExpr('['),
			},
			&stdLexlTermdef{
				terminalName: "FS",
				expr:         newLexlCharacterLiteralExpr('/'),
			},
			&stdLexlTermdef{
				terminalName: "CHARLIT",
				expr:         newLexlAlwaysMatchExpr(),
			},
		},
	})

	digitClass := newCharacterClass()
	digitClass.AddRange('0', '9')

	blocks = append(blocks, &stdLexlMatchBlock{
		/*
				escape:{{
					{match}
					CPOINT		/x[0-9]{4}/
					NL			/n/
					FF			/f/
					RT			/r/
					TAB			/t/
					ZERO		/0/
					CHARLIT		/./
			}}
		*/
		blockName: "escape",
		termdefs: []*stdLexlTermdef{
			&stdLexlTermdef{
				terminalName: "CPOINT",
				expr: newLexlSequenceExpr(
					newLexlCharacterLiteralExpr('x'),
					newLexlQuantifiedExpr(4, 4,
						newLexlCharacterClassExpr(digitClass)),
				),
			},
			&stdLexlTermdef{
				terminalName: "NL",
				expr:         newLexlCharacterLiteralExpr('n'),
			},
			&stdLexlTermdef{
				terminalName: "FF",
				expr:         newLexlCharacterLiteralExpr('f'),
			},
			&stdLexlTermdef{
				terminalName: "RT",
				expr:         newLexlCharacterLiteralExpr('r'),
			},
			&stdLexlTermdef{
				terminalName: "TAB",
				expr:         newLexlCharacterLiteralExpr('t'),
			},
			&stdLexlTermdef{
				terminalName: "ZERO",
				expr:         newLexlCharacterLiteralExpr('0'),
			},
			&stdLexlTermdef{
				terminalName: "CHARLIT",
				expr:         newLexlAlwaysMatchExpr(),
			},
		},
	})

	nzDigitClass := newCharacterClass()
	nzDigitClass.AddRange('1', '9')

	blocks = append(blocks, &stdLexlMatchBlock{
		blockName:  "quantifier",
		ignoreExpr: newLexlPlusExpr(newLexlWhitespaceClassExpr()),
		termdefs: []*stdLexlTermdef{
			&stdLexlTermdef{
				terminalName: "RC",
				expr:         newLexlCharacterLiteralExpr('}'),
			},
			&stdLexlTermdef{
				terminalName: "COMMA",
				expr:         newLexlCharacterLiteralExpr(','),
			},
			&stdLexlTermdef{
				terminalName: "NUM",
				expr: newLexlAlternationExpr(
					newLexlCharacterLiteralExpr('0'),
					newLexlSequenceExpr(
						newLexlCharacterClassExpr(nzDigitClass),
						newLexlStarExpr(newLexlCharacterClassExpr(digitClass)),
					),
				),
			},
		},
	})

	///// {{outer}}
	wsClass := newCharacterClass()
	wsClass.AddCharacter(' ')
	wsClass.AddCharacter('\t')
	wsClass.AddCharacter('\r')
	wsClass.AddCharacter('\f')
	wsClass.AddCharacter('\n')
	labelClass := newCharacterClass()
	labelClass.AddRange('0', '9')
	labelClass.AddRange('a', 'z')
	labelClass.AddRange('A', 'Z')
	labelClass.AddCharacter('_')
	blocks = append(blocks, &stdLexlMatchBlock{
		/*
			outer:{{
				LABEL			/[0-9a-zA-Z_]+/
				MOPEN			/\{\{/ {matchset}
				COLON			/:/
				_				/\s+/
			}}
		*/
		blockName:  "outer",
		ignoreExpr: newLexlPlusExpr(newLexlCharacterClassExpr(wsClass)),
		termdefs: []*stdLexlTermdef{
			&stdLexlTermdef{
				terminalName: "LABEL",
				expr:         newLexlPlusExpr(newLexlCharacterClassExpr(labelClass)),
			},
			&stdLexlTermdef{
				terminalName: "MOPEN",
				expr: newLexlSequenceExpr(
					newLexlCharacterLiteralExpr('{'),
					newLexlCharacterLiteralExpr('{'),
				),
			},
			&stdLexlTermdef{
				terminalName: "COLON",
				expr:         newLexlCharacterLiteralExpr(':'),
			},
		},
	})

	identClass := newCharacterClass()
	identClass.AddRange('a', 'z')
	identClass.AddRange('A', 'Z')
	identClass.AddRange('0', '9')
	identClass.AddCharacter('-')
	identClass.AddCharacter('_')
	blocks = append(blocks, &stdLexlMatchBlock{
		/*
			matchset:{{
				WS				/\s+/
				FS				/\// {match}
				IDENT			/[a-zA-Z0-9_-]+/
				LC				/\{/ {transition}
				MCLOSE			/\}\}/ {0}
			}}
		*/
		blockName: "matchset",
		termdefs: []*stdLexlTermdef{
			&stdLexlTermdef{
				terminalName: "WS",
				expr:         newLexlPlusExpr(newLexlWhitespaceClassExpr()),
			},
			&stdLexlTermdef{
				terminalName: "FS",
				expr:         newLexlCharacterLiteralExpr('/'),
			},
			&stdLexlTermdef{
				terminalName: "IDENT",
				expr:         newLexlPlusExpr(newLexlCharacterClassExpr(identClass)),
			},
			&stdLexlTermdef{
				terminalName: "LC",
				expr:         newLexlCharacterLiteralExpr('{'),
			},
			&stdLexlTermdef{
				terminalName: "MCLOSE",
				expr: newLexlSequenceExpr(
					newLexlCharacterLiteralExpr('}'),
					newLexlCharacterLiteralExpr('}'),
				),
			},
		},
	})
	blocks = append(blocks, &stdLexlMatchBlock{
		/*
			transition:{{
				LABEL			/[0-9a-zA-Z_]+/
				LC				/\{/ {inclusion}
				RC				/\}/ {matchset}
			}}
		*/
		blockName: "transition",
		termdefs: []*stdLexlTermdef{
			&stdLexlTermdef{
				terminalName: "LABEL",
				expr:         newLexlPlusExpr(newLexlCharacterClassExpr(labelClass)),
			},
			&stdLexlTermdef{
				terminalName: "LC",
				expr:         newLexlCharacterLiteralExpr('{'),
			},
			&stdLexlTermdef{
				terminalName: "RC",
				expr:         newLexlCharacterLiteralExpr('}'),
			},
		},
	})

	blocks = append(blocks, &stdLexlMatchBlock{
		/*
			inclusion:{{
				LABEL			/[0-9a-zA-Z_]+/
				RC				/}/ {transition}
			}}
		*/
		blockName: "inclusion",
		termdefs: []*stdLexlTermdef{
			&stdLexlTermdef{
				terminalName: "LABEL",
				expr:         newLexlPlusExpr(newLexlCharacterClassExpr(labelClass)),
			},
			&stdLexlTermdef{
				terminalName: "RC",
				expr:         newLexlCharacterLiteralExpr('}'),
			},
		},
	})

	blocks = append(blocks, &stdLexlMatchBlock{
		/*
			class:{{
				MINUS 		/-/
				RS			/\]/	{match}
				BS			/\\/	{class}
				{{match}}
			}}
		*/
		blockName: "class",
		termdefs: []*stdLexlTermdef{
			&stdLexlTermdef{
				terminalName: "MINUS",
				expr:         newLexlCharacterLiteralExpr('-'),
			},
			&stdLexlTermdef{
				terminalName: "RS",
				expr:         newLexlCharacterLiteralExpr(']'),
			},
			&stdLexlTermdef{
				terminalName: "BS",
				expr:         newLexlCharacterLiteralExpr('\\'),
			},
		},
	})

	/*
		classescape:{{
			{class}
			CPOINT		/x[0-9]{4}/
			NL			/n/
			FF			/f/
			RT			/r/
			TAB			/t/
			ZERO		/0/
			CHARLIT		/./
		}}
	*/
	blocks = append(blocks, &stdLexlMatchBlock{
		blockName: "classescape",
		termdefs: []*stdLexlTermdef{
			&stdLexlTermdef{
				terminalName: "CPOINT",
				expr: newLexlSequenceExpr(
					newLexlCharacterLiteralExpr('x'),
					newLexlQuantifiedExpr(4, 4,
						newLexlCharacterClassExpr(digitClass)),
				),
			},
			&stdLexlTermdef{
				terminalName: "NL",
				expr:         newLexlCharacterLiteralExpr('n'),
			},
			&stdLexlTermdef{
				terminalName: "FF",
				expr:         newLexlCharacterLiteralExpr('f'),
			},
			&stdLexlTermdef{
				terminalName: "RT",
				expr:         newLexlCharacterLiteralExpr('r'),
			},
			&stdLexlTermdef{
				terminalName: "TAB",
				expr:         newLexlCharacterLiteralExpr('t'),
			},
			&stdLexlTermdef{
				terminalName: "ZERO",
				expr:         newLexlCharacterLiteralExpr('0'),
			},
			&stdLexlTermdef{
				terminalName: "CHARLIT",
				expr:         newLexlAlwaysMatchExpr(),
			},
		},
	})

	blocksByName := make(map[string]*stdLexlMatchBlock)
	termdefIndex := make(map[string]*stdLexlTermdef)
	for _, b := range blocks {
		block := b.(*stdLexlMatchBlock)
		blocksByName[b.Name()] = block
		for i, td := range block.termdefs {
			td.defBlock = block
			td.index = i
			// td.fwdBlock = block
			termdefIndex[b.Name()+"/"+td.terminalName] = td
		}
		if block.ignoreExpr == nil {
			block.ignoreExpr = newLexlNeverMatchExpr()
		}
	}

	blocksByName["0"].inclusions = []*stdLexlMatchBlock{blocksByName["outer"]}
	termdefIndex["0/COMMENT_OPEN"].fwdBlock = blocksByName["comment"]
	termdefIndex["comment/NL"].fwdBlock = blocksByName["0"]
	termdefIndex["outer/MOPEN"].fwdBlock = blocksByName["matchset"]
	termdefIndex["matchset/FS"].fwdBlock = blocksByName["match"]
	termdefIndex["matchset/LC"].fwdBlock = blocksByName["transition"]
	termdefIndex["matchset/MCLOSE"].fwdBlock = blocksByName["0"]
	termdefIndex["match/BS"].fwdBlock = blocksByName["escape"]
	termdefIndex["match/LC"].fwdBlock = blocksByName["quantifier"]
	termdefIndex["match/FS"].fwdBlock = blocksByName["matchset"]
	termdefIndex["match/LS"].fwdBlock = blocksByName["class"]
	termdefIndex["quantifier/RC"].fwdBlock = blocksByName["match"]
	termdefIndex["transition/LC"].fwdBlock = blocksByName["inclusion"]
	termdefIndex["transition/RC"].fwdBlock = blocksByName["matchset"]
	termdefIndex["inclusion/RC"].fwdBlock = blocksByName["transition"]
	blocksByName["class"].inclusions = []*stdLexlMatchBlock{blocksByName["match"]}
	termdefIndex["class/RS"].fwdBlock = blocksByName["match"]
	termdefIndex["class/BS"].fwdBlock = blocksByName["classescape"]
	for _, td := range blocksByName["escape"].termdefs {
		td.fwdBlock = blocksByName["match"]
	}
	for _, td := range blocksByName["classescape"].termdefs {
		td.fwdBlock = blocksByName["class"]
	}
	return blocks
}
