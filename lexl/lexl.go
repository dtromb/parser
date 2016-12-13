package lexl

import (
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/dtromb/ngen"
)

// MatchBlock - A semantic representation of a single lexer token recognizer.
type MatchBlock interface {
	Name() string
	NumTermdefs() int
	Termdef(idx int) Termdef
	NumInclusions() int
	Inclusion(idx int) MatchBlock
	Ignore() MatchExpr
}

// Termdef - A semantic representation of a single lexer token recognizer option,
// consisting of a lexl regular expression and its next recognizer state.
type Termdef interface {
	Name() string
	Block() MatchBlock
	NextBlock() MatchBlock
	NextBlockDefault() bool
	Match() MatchExpr
}

// MatchExprType - Enumeration describing the kinds of lexl regular subexpressions.
type MatchExprType int

const (
	// LexlMatchNever never matches.
	LexlMatchNever MatchExprType = iota
	// LexlMatchAlways always matches.
	LexlMatchAlways
	// LexlMatchCharacterLiteral matches iff the next character is a specified literal character.
	LexlMatchCharacterLiteral
	// LexlMatchStart matches iff the lexer is at the start-of-stream position.
	LexlMatchStart
	// LexlMatchEnd matches iff the lexer is at the end-of-stream position
	LexlMatchEnd
	// LexlMatchSubmatch matches iff the specified submatch matches.  It generates a capturing group which
	// may be later accessed through the recognized token representation.
	LexlMatchSubmatch
	// LexlMatchOptional always matches, consuming the input of a specified submatch if that submatch matches.
	LexlMatchOptional
	// LexlMatchStar always matches, consuming the maximum sequence of repeated inputs to the specified
	// submatch, as long as it continues to match.
	LexlMatchStar
	// LexlMatchPlus matches iff the specified submatch matches at least once.  It consumes input
	// corresponding to the maximum number of repeated submatches.
	LexlMatchPlus
	// LexlMatchQuantified matches iff a specified number (or range) of matches of a specified submatch succeed.
	LexlMatchQuantified
	// LexlMatchCharset matches iff the next character is in the specified character subset.
	LexlMatchCharset
	// LexlMatchSequence matches iff all of the child submatches match the current input, in order.
	LexlMatchSequence
	// LexlMatchAlternation matches iff exactly one of the child submatches match the current input.
	LexlMatchAlternation
)

// MatchExpr - Abstract base interface representing a lexl regular expression.  Implementing structs
// may also implement at most one of the derived interfaces {CharacterLiteralExpression, SequenceExpression, etc}.
type MatchExpr interface {
	Type() MatchExprType
}

type LexlRepresentation []MatchBlock

////

type stdLexlMatchBlock struct {
	blockName  string
	termdefs   []*stdLexlTermdef
	inclusions []*stdLexlMatchBlock
	ignoreExpr MatchExpr
}

type stdLexlTermdef struct {
	terminalName string
	defBlock     *stdLexlMatchBlock
	fwdBlock     *stdLexlMatchBlock
	expr         MatchExpr
}

type lexlDfaState interface {
	Id() uint32
	Transition(c byte) (*lexlDfaState, bool)
}

func (se *stdLexlSequenceExpression) Match(idx int) MatchExpr {
	if idx < 0 || idx >= len(se.matches) {
		return nil
	}
	return se.matches[idx]
}

func (mb *stdLexlMatchBlock) Name() string {
	return mb.blockName
}

func (mb *stdLexlMatchBlock) NumTermdefs() int {
	return len(mb.termdefs)
}

func (mb *stdLexlMatchBlock) Termdef(idx int) Termdef {
	if idx < 0 || idx >= len(mb.termdefs) {
		return nil
	}
	return mb.termdefs[idx]
}

func (mb *stdLexlMatchBlock) NumInclusions() int {
	return len(mb.inclusions)
}

func (mb *stdLexlMatchBlock) Inclusion(idx int) MatchBlock {
	if idx < 0 || idx >= len(mb.inclusions) {
		return nil
	}
	return mb.inclusions[idx]
}

func (mb *stdLexlMatchBlock) Ignore() MatchExpr {
	return mb.ignoreExpr
}

func (mb *stdLexlMatchBlock) ToString() string {
	var buf []byte
	buf = append(buf, fmt.Sprintf("{{%s}}\n", mb.blockName)...)
	if mb.ignoreExpr != nil && mb.ignoreExpr.Type() != LexlMatchNever {
		buf = append(buf, fmt.Sprintf("   _: /%s/\n", MatchExprToString(mb.ignoreExpr))...)
	}
	for _, td := range mb.termdefs {
		if td.fwdBlock != nil && td.fwdBlock != mb {
			buf = append(buf, fmt.Sprintf("   %s: /%s/ {%s}\n", td.terminalName, MatchExprToString(td.expr), td.fwdBlock.blockName)...)
		} else {
			buf = append(buf, fmt.Sprintf("   %s: /%s/\n", td.terminalName, MatchExprToString(td.expr))...)
		}
	}
	for _, incl := range mb.inclusions {
		buf = append(buf, fmt.Sprintf("   {{%s}}\n", incl.blockName)...)
	}
	return string(buf)
}

func (td *stdLexlTermdef) Name() string {
	return td.terminalName
}

func (td *stdLexlTermdef) Block() MatchBlock {
	return td.defBlock
}

func (td *stdLexlTermdef) NextBlockDefault() bool {
	return td.fwdBlock == nil
}

func (td *stdLexlTermdef) NextBlock() MatchBlock {
	if td.fwdBlock == nil {
		return td.defBlock
	}
	return td.fwdBlock
}

func (td *stdLexlTermdef) Match() MatchExpr {
	return td.expr
}

// if error rv is nil, MatchExpr is guaranteed to be ndfaStateGenerator and stringable.
func cloneMatchExpr(me MatchExpr) (MatchExpr, error) {
	return nil, errors.New("cloneMatchExpr() unimplemented")
}

// if error rv is nil, MatchBlock is guaranteed to be stringable.
func cloneMatchBlock(mb MatchBlock) (MatchBlock, error) {
	return nil, errors.New("cloneMatchBlock() unimplemented")
}

func (sr LexlRepresentation) ConstructLexlNdfa() (LexlNdfa, error) {
	blockIndex := make(map[string]int)
	blockStateIndex := make([]map[string]*stdLexlNdfaState, 0, len(sr))
	blockStates := make([][][]*stdLexlNdfaState, len(sr))
	// Index the blocks by name so that we can map MatchBlock to their
	// corresponding state sets as we build the NDFA.
	for blockIdx, block := range sr {
		blockIndex[block.Name()] = blockIdx
	}
	// Iterate the terms in each block and generate the expression subgraphs
	// for those terms.
	for blockIdx, block := range sr {
		blockStateIndex = append(blockStateIndex, make(map[string]*stdLexlNdfaState))
		blockStates[blockIdx] = make([][]*stdLexlNdfaState, block.NumTermdefs())
		for termIdx := 0; termIdx < block.NumTermdefs(); termIdx++ {
			termdef := block.Termdef(termIdx)
			termdefExpr := termdef.Match()
			if _, has := blockStateIndex[blockIdx][termdef.Name()]; has {
				return nil, errors.New(fmt.Sprintf("duplicate termdef name '%s' in block '%s'", termdef.Name(), block.Name()))
			}
			stateGen, ok := termdefExpr.(ndfaStateGenerator)
			if !ok {
				stateGenIf, err := cloneMatchExpr(termdefExpr)
				if err != nil {
					return nil, err
				}
				stateGen = stateGenIf.(ndfaStateGenerator)
			}
			termdefStates, err := stateGen.GenerateNdfaStates()
			if err != nil {
				return nil, err
			}
			blockStateIndex[blockIdx][termdef.Name()] = termdefStates[0]
			// Store the terminal we can accept in the accepting states -
			// the post-accept state may not yet exist, however.
			for i := len(termdefStates) - 1; i >= 0; i-- {
				if termdefStates[i].accepting {
					if termdefStates[i].acceptTransitions == nil {
						termdefStates[i].acceptTransitions = make(map[string]*stdLexlNdfaState)
					}
					termdefStates[i].acceptTransitions[termdef.Name()] = nil
				} else {
					break
				}
			}
			// We abuse the id field of the first state in the term temporarily
			// during generation to store the next block reached after the term
			// is accepted.
			termdefStates[0].id = blockIndex[termdef.NextBlock().Name()]
			blockStates[blockIdx][termIdx] = termdefStates
		}
	}
	fmt.Println("CLOSE")
	// Compute the inclusion closure of each block.
	closureAdds := make([][][]*stdLexlNdfaState, len(sr))
	includes := make([]map[string]bool, len(sr))
	for blockIdx, block := range sr {
		fmt.Printf("BLOCKIDX %d\n", blockIdx)
		inclusions := make(map[string]bool)
		next := make(map[string]bool)
		next[block.Name()] = true
		changed := true
		for changed {
			changed = false
			for len(next) > 0 {
				fmt.Printf("len(next) == %d\n", len(next))
				var name string
				for blockName, _ := range next {
					name = blockName
					break
				}
				delete(next, name)
				if _, has := inclusions[name]; has {
					continue
				}
				changed = true
				inclusions[name] = true
				inclIdx := blockIndex[name]
				if name != block.Name() {
					fmt.Printf("Will add {%s} to {%s}\n", name, block.Name())
					for termIdx, termStates := range blockStates[inclIdx] {
						fmt.Printf("termIdx = %d\n", termIdx)

						term := sr[inclIdx].Termdef(termIdx)
						// Allow locally-defined terms to override the inclusions.
						if _, has := blockStateIndex[blockIdx][term.Name()]; has {
							fmt.Printf("TERM INCLUSION OVERRIDE: %s/%s\n", block.Name(), term.Name())
							continue
						}
						nStates, err := cloneNdfaStates(termStates)
						if err != nil {
							return nil, err
						}

						if term.NextBlockDefault() {
							nStates[0].id = blockIdx
						} else {
							nStates[0].id = blockIndex[term.NextBlock().Name()]
						}

						closureAdds[blockIdx] = append(closureAdds[blockIdx], nStates)
					}
				}
				for i := 0; i < sr[inclIdx].NumInclusions(); i++ {
					nxtIncl := sr[inclIdx].Inclusion(i).Name()
					if _, has := inclusions[nxtIncl]; !has {
						next[nxtIncl] = true
					}
				}
			}
		}
		includes[blockIdx] = inclusions
	}

	fmt.Println("FINISHED")
	// Add the computed inclusions as new terms in the blocks that include them.
	for blockIdx, _ := range sr {
		blockStates[blockIdx] = append(blockStates[blockIdx], closureAdds[blockIdx]...)
	}

	// Alternate the states in each term under a block entry point.  Add the ignore
	// expression into the graph if it exists, with an epsilon transition back to the
	// block head.
	var extraIgnoreStates []*stdLexlNdfaState
	blockHeads := make([]*stdLexlNdfaState, len(sr))
	for blockIdx, blockState := range blockStates {
		blockHeads[blockIdx] = newStdLexlNdfaState()
		for _, termStates := range blockState {
			blockHeads[blockIdx].epsilons = append(blockHeads[blockIdx].epsilons, termStates[0])
		}
		ignoreExpr := sr[blockIdx].Ignore()
		if ignoreExpr == nil {
			ignoreExpr = newLexlNeverMatchExpr()
		}
		for inclBlockName, _ := range includes[blockIdx] {
			if inclBlockName == sr[blockIdx].Name() {
				continue
			}
			fmt.Printf("importing ignore expr from {%s} to {%s}\n", inclBlockName, sr[blockIdx].Name())
			block := blockIndex[inclBlockName]
			inclIgnore := sr[block].Ignore()
			fmt.Println("imported expression is typed " + reflect.TypeOf(inclIgnore).String())
			if inclIgnore != nil && inclIgnore.Type() != LexlMatchNever {
				fmt.Println("merged")
				ignoreExpr = newLexlAlternationExpr(ignoreExpr, inclIgnore)
			}
		}
		if ignoreExpr != nil && ignoreExpr.Type() != LexlMatchNever {
			fmt.Printf("printing ignore transitions from {%s}\n", sr[blockIdx].Name())
			ignoreGen, ok := ignoreExpr.(ndfaStateGenerator)
			if !ok {
				ignoreGenIf, err := cloneMatchExpr(ignoreExpr)
				if err != nil {
					return nil, err
				}
				ignoreGen = ignoreGenIf.(ndfaStateGenerator)
			}
			ignoreStates, err := ignoreGen.GenerateNdfaStates()
			if err != nil {
				return nil, err
			}
			for i := len(ignoreStates) - 1; i >= 0; i-- {
				state := ignoreStates[i]
				if state.accepting {
					state.accepting = false
					state.epsilons = append(state.epsilons, blockHeads[blockIdx])
				} else {
					break
				}
			}
			fmt.Printf("Preprint: blockHeads[%d] has %d epsilons\n", blockIdx, len(blockHeads[blockIdx].epsilons))
			blockHeads[blockIdx].epsilons = append(blockHeads[blockIdx].epsilons, ignoreStates[0])
			fmt.Printf("Postprint: blockHeads[%d] has %d epsilons\n", blockIdx, len(blockHeads[blockIdx].epsilons))
			extraIgnoreStates = append(extraIgnoreStates, ignoreStates...)
		}
	}

	// Set the block forwarding transitions for all of the accepting states
	// in each block.  (We've stored the target block for each term in its
	// first state's id field already).
	for _, bs := range blockStates {
		for _, termStates := range bs {
			fwdBlock := sr[termStates[0].id]
			for i := len(termStates) - 1; i >= 0; i-- {
				state := termStates[i]
				if state.accepting {
					// state.epsilons = append(state.epsilons, blockHeads[blockIndex[fwdBlock.Name()]])
					for k, _ := range state.acceptTransitions {
						state.acceptTransitions[k] = blockHeads[blockIndex[fwdBlock.Name()]]
					}
				} else {
					break
				}
			}
		}
	}

	var res []*stdLexlNdfaState
	// Arrange all of the states into a linear array and set their unique ids.
	// The first block in the representation list is at index zero and is the
	// entry point for the lexer NDFA.
	idx := 0
	for i := 0; i < len(sr); i++ {
		blockHeads[i].id = idx
		fmt.Printf("HEAD %d id=%d\n", i, idx)
		idx++
		res = append(res, blockHeads[i])
		for j := 0; j < len(blockStates[i]); j++ {
			for _, st := range blockStates[i][j] {
				st.id = idx
				idx++
				res = append(res, st)
			}
		}
	}
	for _, st := range extraIgnoreStates {
		st.id = idx
		idx++
		res = append(res, st)
	}
	// Return the result.
	return &stdLexlNdfa{states: res}, nil
}

func GenerateLexlLexerFromDfa(dfa LexlDfa, grammar ngen.Grammar) (ngen.Lexer, error) {
	stdDfa, ok := dfa.(*stdLexlDfa)
	if !ok {
		stdDfaIf, err := cloneDfa(stdDfa)
		if err != nil {
			return nil, err
		}
		stdDfa = stdDfaIf.(*stdLexlDfa)
	}
	return stdDfa.GenerateLexer(grammar)
}

type stdLexlDfaLexer struct {
	grammar   ngen.Grammar
	dfa       *stdLexlDfa
	terminals []ngen.Term
}

type stdLexlDfaLexerState struct {
	lexer        *stdLexlDfaLexer
	in           LexlReader
	line         int
	markLine     int
	column       int
	markColumn   int
	pos          int
	markPos      int
	sentEof      bool
	atEof        bool
	currentState int
	llinelen     int
}

func (dfa *stdLexlDfa) GenerateLexer(grammar ngen.Grammar) (ngen.Lexer, error) {
	g := ngen.GetIndexedGrammar(grammar)
	termIndexIf, err := g.GetIndex(ngen.GrammarIndexTypeTerm)
	if err != nil {
		return nil, err
	}
	termIndex := termIndexIf.(ngen.TermGrammarIndex)
	for _, tn := range termIndex.GetTerminalNames() {
		fmt.Println(" * " + tn)
	}
	lexer := &stdLexlDfaLexer{
		grammar:   grammar,
		dfa:       dfa,
		terminals: make([]ngen.Term, len(dfa.terminals)),
	}
	for i, termName := range dfa.terminals {
		term, err := termIndex.GetTerminal(termName)
		if err != nil {
			return nil, err
		}
		lexer.terminals[i] = term
	}
	return lexer, nil
}

func (ldl *stdLexlDfaLexer) Grammar() ngen.Grammar {
	return ldl.grammar
}

func (ldl *stdLexlDfaLexer) Open(in io.Reader) (ngen.LexerState, error) {
	var reader LexlReader
	var ok bool
	if reader, ok = in.(LexlReader); !ok {
		reader = GetLexlReader(in, 256)
	}
	return &stdLexlDfaLexerState{
		lexer:        ldl,
		in:           reader,
		line:         1,
		column:       1,
		currentState: 0,
	}, nil
}

type stringable interface {
	ToString() string
}

func MatchBlockToString(block MatchBlock) string {
	if strBlock, ok := block.(stringable); !ok {
		strBlockIf, err := cloneMatchBlock(block)
		if err != nil {
			panic(err.Error())
		}
		return strBlockIf.(stringable).ToString()
	} else {
		return strBlock.ToString()
	}
}

func MatchExprToString(expr MatchExpr) string {
	if strExpr, ok := expr.(stringable); !ok {
		strExprIf, err := cloneMatchExpr(expr)
		if err != nil {
			panic(err.Error())
		}
		return strExprIf.(stringable).ToString()
	} else {
		return strExpr.ToString()
	}
}

func (dls *stdLexlDfaLexerState) Lexer() ngen.Lexer {
	return dls.lexer
}

func (dls *stdLexlDfaLexerState) Reader() io.Reader {
	return dls.in
}

func (dls *stdLexlDfaLexerState) HasMoreTokens() (bool, error) {
	return !dls.in.Eof() || !dls.sentEof, nil
}

type stdLexlToken struct {
	lexer    *stdLexlDfaLexerState
	terminal ngen.Term
	literal  string
	fpos     int
	lpos     int
	fline    int
	lline    int
	fcol     int
	lcol     int
}

func (dls *stdLexlDfaLexerState) makeToken(terminal ngen.Term, literal string) ngen.Token {
	tok := &stdLexlToken{
		lexer:    dls,
		terminal: terminal,
		literal:  literal,
		fpos:     dls.markPos,
		lpos:     dls.pos,
		fline:    dls.markLine,
		lline:    dls.line,
		fcol:     dls.markColumn,
		lcol:     dls.column,
	}
	dls.markPos = dls.pos
	dls.markLine = dls.line
	dls.markColumn = dls.column
	return tok
}

func (lt *stdLexlToken) LexerState() ngen.LexerState {
	return lt.lexer
}

func (lt *stdLexlToken) FirstPosition() int {
	return lt.fpos
}

func (lt *stdLexlToken) LastPosition() int {
	return lt.lpos - 1
}

func (lt *stdLexlToken) FirstLine() int {
	return lt.fline
}

func (lt *stdLexlToken) LastLine() int {
	if lt.lcol == 1 {
		return lt.lline - 1
	}
	return lt.lline
}

func (lt *stdLexlToken) FirstColumn() int {
	return lt.fcol
}

func (lt *stdLexlToken) LastColumn() int {
	if lt.lcol == 1 {
		return lt.lexer.llinelen
	}
	return lt.lcol - 1
}

func (lt *stdLexlToken) Terminal() ngen.Term {
	return lt.terminal
}

func (lt *stdLexlToken) Literal() string {
	return lt.literal
}

func (dls *stdLexlDfaLexerState) NextToken() (ngen.Token, error) {
	if dls.atEof {
		if !dls.sentEof {
			dls.sentEof = true
			return dls.makeToken(dls.lexer.grammar.Bottom(), ""), nil
		} else {
			return nil, errors.New("no more tokens in stream")
		}
	}
	fmt.Printf("current state: %d\n", dls.currentState)
	cs := dls.lexer.dfa.State(dls.currentState)
	literal := make([]rune, 0, 32)
	for {
		c, err := dls.in.PeekRune()
		if err != nil {
			if err == io.EOF {
				panic("EOF")
				dls.atEof = true
				return dls.NextToken()
			}
			return nil, err
		}
		ns := cs.Query(c)
		if ns == nil {
			if !cs.CanAccept() {
				// XXX - Improve this error message
				return nil, errors.New(fmt.Sprintf("illegal character during lexer read ('%c' at %d:%d(%d), state=%d)", c, dls.line, dls.column, dls.pos, dls.currentState))
			}
			fmt.Println("ACCEPT")
			termId, nxt := cs.AcceptTransition()
			dls.currentState = nxt.Id()
			return dls.makeToken(dls.lexer.terminals[termId], string(literal)), nil
		}
		dls.in.ReadRune()
		cs = ns
		literal = append(literal, c)
		dls.pos++
		if c == '\n' {
			dls.llinelen = dls.column
			dls.column = 1
			dls.line++
		} else {
			dls.column++
		}
	}
}

func (dls *stdLexlDfaLexerState) CurrentLine() int {
	return dls.line
}

func (dls *stdLexlDfaLexerState) CurrentColumn() int {
	return dls.column
}

func (dls *stdLexlDfaLexerState) CurrentPosition() int {
	return dls.pos
}
