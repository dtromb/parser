package lexr

import "github.com/dtromb/parser"

func GenerateLexr0Grammar() parser.Grammar {
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

func GenerateLexr0Domain(lexr0Grammar parser.Grammar) Domain {
	
	ccWhitespace := OpenCharacterClassBuilder(). 
						AddCharacter(' ').
						AddCharacter('\t').
						AddCharacter('\n').
						AddCharacter('\r').
						AddCharacter('\f').MustBuild()
						
	ccLabel := OpenCharacterClassBuilder(). 
						AddRange('0','9').
						AddRange('a','z').
						AddRange('A','Z').
						AddCharacter('_').MustBuild()
						
	builder, err := OpenDomainBuilder(lexr0Grammar)
	if err != nil {
		panic(err.Error())
	}
	
	builder.Block("0").
				Termdef("COMMENT_OPEN",	SequenceExpression(
						CharacterLiteralExpression('/'),
						CharacterLiteralExpression('/'))).ToBlock("comment"). 
				Include("outer"). 
				
			Block("comment"). 
				Termdef("NL", CharacterLiteralExpression('\n')).ToBlock("0"). 
				Termdef("COMMENT", PlusExpression(CharacterClassExpression(OpenCharacterClassBuilder(). 
																				AddCharacter('\n').
																				Negate(). 
																				MustBuild()))). 
																				
			Block("outer"). 
				Termdef("LABEL", PlusExpression(CharacterClassExpression(ccLabel))). 
				Termdef("MOPEN", SequenceExpression(
									CharacterLiteralExpression('{'),
									CharacterLiteralExpression('{'))).ToBlock("matchset"). 
				Termdef("COLON", CharacterLiteralExpression(':')). 
				Ignore(PlusExpression(CharacterClassExpression(ccWhitespace))). 
				
			Block("matchset"). 
				Termdef("WS", PlusExpression(CharacterClassExpression(ccWhitespace))). 
				Termdef("FS", CharacterLiteralExpression('/')).ToBlock("match"). 
				Termdef("IDENT", PlusExpression(CharacterClassExpression(OpenCharacterClassBuilder(). 
																			AddRange('a','z').
																			AddRange('A','Z').
																			AddRange('0','9').
																			AddCharacter('-').
																			AddCharacter('_').MustBuild()))). 
				Termdef("LC", CharacterLiteralExpression('{')).ToBlock("transition"). 
				Termdef("MCLOSE", SequenceExpression( 
									CharacterLiteralExpression('}'),
									CharacterLiteralExpression('}'))).ToBlock("0"). 
									
			Block("transition"). 
				Termdef("LABEL", PlusExpression(CharacterClassExpression(ccLabel))). 
				Termdef("LC", CharacterLiteralExpression('{')).ToBlock("inclusion"). 
				Termdef("RC", CharacterLiteralExpression('}')).ToBlock("matchset"). 
				
			Block("inclusion"). 
				Termdef("LABEL", PlusExpression(CharacterClassExpression(ccLabel))). 
				Termdef("RC", CharacterLiteralExpression('}')).ToBlock("transition"). 
				
			
			Block("match"). 
				Termdef("DOT", CharacterLiteralExpression('.')). 
				Termdef("CARET", CharacterLiteralExpression('^')). 
				Termdef("DOLLAR", CharacterLiteralExpression('$')). 
				Termdef("BS", CharacterLiteralExpression('\\')).ToBlock("escape"). 
				Termdef("LC", CharacterLiteralExpression('{')).ToBlock("quantifier"). 
				Termdef("RC", CharacterLiteralExpression('}')). 
				Termdef("LP", CharacterLiteralExpression('(')). 
				Termdef("RP", CharacterLiteralExpression(')')). 
				Termdef("QM", CharacterLiteralExpression('?')). 
				Termdef("STAR", CharacterLiteralExpression('*')). 
				Termdef("PLUS", CharacterLiteralExpression('+')). 
				Termdef("PIPE", CharacterLiteralExpression('|')). 
				Termdef("LS", CharacterLiteralExpression('[')).ToBlock("class"). 
				Termdef("FS", CharacterLiteralExpression('/')).ToBlock("matchset").
				Termdef("CHARLIT", AlwaysMatchExpression()). 
				
			Block("escape"). 
				DefaultToBlock("match"). 
				Termdef("CPOINT", SequenceExpression(
									CharacterLiteralExpression('x'),
									QuantifiedExpression(
										CharacterClassExpression(OpenCharacterClassBuilder(). 
																	AddRange('0','9').MustBuild()),
										4, 4))). 
				Termdef("NL", CharacterLiteralExpression('n')). 
				Termdef("FF", CharacterLiteralExpression('f')). 
				Termdef("RT", CharacterLiteralExpression('r')). 
				Termdef("TAB", CharacterLiteralExpression('t')). 
				Termdef("ZERO", CharacterLiteralExpression('0')).
				Termdef("WS", CharacterLiteralExpression('s')). 
				Termdef("CHARLIT", AlwaysMatchExpression()). 
				
			Block("quantifier"). 
				Termdef("RC", CharacterLiteralExpression('}')).ToBlock("match"). 
				Termdef("COMMA", CharacterLiteralExpression(',')). 
				Termdef("NUM", AlternationExpression(
								CharacterLiteralExpression('0'),
								SequenceExpression(
									CharacterClassExpression(OpenCharacterClassBuilder(). 
																AddRange('1','9').MustBuild()),
									StarExpression(CharacterClassExpression(OpenCharacterClassBuilder(). 
																				AddRange('0','9').MustBuild()))))). 
			Ignore(PlusExpression(CharacterClassExpression(ccWhitespace))). 
			
			
			Block("class"). 
				Termdef("MINUS", CharacterLiteralExpression('-')).
				Termdef("RS", CharacterLiteralExpression(']')).ToBlock("match"). 
				Termdef("BS", CharacterLiteralExpression('\\')).ToBlock("classescape"). 
				Include("match").
		
			Block("classescape").
				DefaultToBlock("class"). 
				Termdef("CPOINT", SequenceExpression(
									CharacterLiteralExpression('x'),
									QuantifiedExpression(
										CharacterClassExpression(OpenCharacterClassBuilder(). 
																	AddRange('0','9').MustBuild()),
										4, 4))).
				Termdef("NL", CharacterLiteralExpression('n')).
				Termdef("FF", CharacterLiteralExpression('f')).
				Termdef("RT", CharacterLiteralExpression('r')).
				Termdef("TAB", CharacterLiteralExpression('t')).
				Termdef("ZERO", CharacterLiteralExpression('0')).
				Termdef("WS", CharacterLiteralExpression('s')). 
				Termdef("CHARLIT", AlwaysMatchExpression())
				
	return builder.MustBuild()
}
				