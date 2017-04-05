package lexr

import (
	"math"
	"github.com/dtromb/parser"
	"fmt"
	"io"
)


type Ndfa interface {
	NumNodes() int
	Node(idx int) NdfaNode
}

type NdfaNode interface {
	Id() uint32
	Literals() []rune
	LiteralTransitions(c rune) []NdfaNode
	CharacterRanges() []CharacterRange
	CharacterRangeTransitions(cr CharacterRange) []NdfaNode
	EpsilonTransitions() []NdfaNode	
	IsTerminal() bool
	Query(c rune) []NdfaNode
}

type DomainNdfaNode interface {
	NdfaNode
	Domain() Domain
	Block() Block
	Termdef() Termdef
 	IsIgnore() bool
}

type NdfaNodeGenerator interface {
	GenerateNdfaNodes(firstId uint32) ([]NdfaNode,int)
}

type Dfa interface {
	NumStates() int
	State(idx int) DfaNode
}

type DfaNode interface{
	Dfa() Dfa
	Id() int
	TransitionRange(c rune) CharacterRange
	TransitionLookup(cr CharacterRange) (DfaNode, bool)
	TransitionQuery(c rune) (DfaNode, bool)
	IsInitial() bool
	IsAccepting() bool
	AcceptTerm() (parser.Term, bool)
	// AcceptTermNext(nxt rune) (DfaNode, bool)
	AcceptTermNext() (DfaNode, bool)
}

///

func WriteDfa(dfa Dfa, out io.Writer) {
	for i := 0; i < dfa.NumStates(); i++ {
		WriteDfaState(dfa.State(i), out)
	}
}

func WriteDfaState(dn DfaNode, out io.Writer) {
	out.Write([]byte(fmt.Sprintf("(%d)\n", dn.Id())))
	c := rune(0)
	for {
		l := c
		toNode, _ := dn.TransitionQuery(l)
		//fmt.Println(c)
		r := dn.TransitionRange(c)
		c = r.Greatest()
		tx, has := dn.TransitionLookup(r)
		if has {
			out.Write([]byte(fmt.Sprintf("  '%s' %d-%d '%s' -> (%d)\n", matchEscapeCharacterLiteral(l, false), l, c, matchEscapeCharacterLiteral(c, false), tx.Id())))
			if tx != toNode {
				panic("incorrect query result")
			}
		} else {
			out.Write([]byte(fmt.Sprintf("  '%s' %d-%d '%s' X\n", matchEscapeCharacterLiteral(l, false), l, c, matchEscapeCharacterLiteral(c, false))))
			if toNode != nil {
				panic("incorrect query result (empty)")
			}
		}
		c++
		if c >= math.MaxInt32 || c < 0{
			break
		}
	}
	if dn.IsAccepting() {
		var termName string
		if term, ok := dn.AcceptTerm(); ok {
			termName = term.Name()
		} else {
			termName = "`e"
		}
		if nxt, ok := dn.AcceptTermNext(); ok {
			out.Write([]byte(fmt.Sprintf("   `* %s %d\n", termName, nxt.Id())))
		} else {
			out.Write([]byte(fmt.Sprintf("   `* %s X\n", termName)))
		}
	}
}

func WriteNdfa(ndfa Ndfa, out io.Writer) {
	for i := 0; i < ndfa.NumNodes(); i++ {
		WriteNdfaNode(ndfa.Node(i), out)
	}
}

func WriteNdfaNode(nn NdfaNode, out io.Writer) {
	var tdef, accChar string
	if nn.IsTerminal() {
		accChar = "* "
	} else {
		accChar = " "
	}
	if dnn, ok := nn.(DomainNdfaNode); ok {
		if dnn.Block() != nil {
			if dnn.Termdef() != nil {
				tdef = fmt.Sprintf("{%s/%s}", dnn.Block().Name(), dnn.Termdef().Terminal().Name())
			} else if dnn.IsIgnore() {
				tdef = fmt.Sprintf("{%s/_}", dnn.Block().Name())
			} else {
				tdef = fmt.Sprintf("{%s}", dnn.Block().Name())
			}
		} else {
			tdef = " "
		}
	}
	out.Write([]byte(fmt.Sprintf("[[%d]] %s%s\n", nn.Id(), accChar, tdef)))
	for _, c := range nn.Literals() {
		trs := nn.LiteralTransitions(c)
		out.Write([]byte(fmt.Sprintf("     '%s' -> [", matchEscapeCharacterLiteral(c, false))))
		for i, tr := range trs {
			out.Write([]byte(fmt.Sprintf("[%d]", tr.Id())))
			if i < len(trs)-1 {
				out.Write([]byte{','})
			}
		}
		out.Write([]byte("]\n"))
	}
	for _, r := range nn.CharacterRanges() {
		trs := nn.CharacterRangeTransitions(r)
		if r.Greatest() < 0 || r.Greatest() == math.MaxInt32 {
			out.Write([]byte(fmt.Sprintf("     [%s-] -> [", matchEscapeCharacterLiteral(r.Least(), true))))
		} else {
			out.Write([]byte(fmt.Sprintf("     [%s-%s] -> [", matchEscapeCharacterLiteral(r.Least(), true),
		    	           									  matchEscapeCharacterLiteral(r.Greatest(), true))))	
		}
		for i, tr := range trs {
			out.Write([]byte(fmt.Sprintf("[%d]", tr.Id())))
			if i < len(trs)-1 {
				out.Write([]byte{','})
			}
		}
		out.Write([]byte("]\n"))
	}
	eps := nn.EpsilonTransitions()
	if len(eps) > 0 {
		out.Write([]byte("     `e -> ["))
		for i, tr := range eps {
			out.Write([]byte(fmt.Sprintf("[%d]", tr.Id())))
			if i < len(eps)-1 {
				out.Write([]byte{','})
			}
		}
		out.Write([]byte("]\n"))
	}
}
