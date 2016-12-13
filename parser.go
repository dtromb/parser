package ngen

type Parser interface {
	Grammar() Grammar
	Open(lexState LexerState) (ParserState, error)
}

type ParserState interface {
	Parser() Parser
	LexerState() LexerState
	Parse() (ParseTreeNode, error)
}

type ParseTreeNode interface {
	Parser() Parser
	Token() Token
	Production() ProductionRule
	NumChildren() int
	Child(idx int) ParseTreeNode
	Children() []ParseTreeNode
}
