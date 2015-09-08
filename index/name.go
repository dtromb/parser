package index

import (
	"github.com/dtromb/parser"
	"reflect"
	"strings"
)

// Grammar particles by name

type NameIndex struct {
	grammar parser.Grammar
	terminalsByName map[string]parser.GrammarParticle
	nonterminalsByName map[string]parser.GrammarParticle
	lhsNames map[string][]parser.Production
	rhsNames map[string][]parser.Production
}

const(
	NAME_INDEX	string = "name"
)

func init() {
	parser.RegisterGrammarIndexType(reflect.TypeOf([]NameIndex{}).Elem())
}

func (ni *NameIndex) IndexName() string {
	return NAME_INDEX
}

func (ni *NameIndex) Grammar() parser.Grammar {
	return ni.grammar
}
	
func (ni *NameIndex) Initialize(g parser.Grammar) error {
	ni.nonterminalsByName = make(map[string]parser.GrammarParticle)
	ni.terminalsByName = make(map[string]parser.GrammarParticle)
	for i := 0; i < g.NumNonterminals(); i++ {
		nt := g.Nonterminal(i)
		ni.nonterminalsByName[nt.Name()] = nt
	}
	for i := 0; i < g.NumTerminals(); i++ {
		t := g.Terminal(i)
		ni.terminalsByName[t.Name()] = t
	}
	ni.lhsNames = make(map[string][]parser.Production)
	ni.rhsNames = make(map[string][]parser.Production)
	for _, p := range g.Productions() {
		var rhs, lhs []byte
		for i := 0; i < p.LhsLen(); i++ {
			lhs = append(lhs,p.Lhs(i).Name()...)
			if i < p.LhsLen()-1 {
				lhs = append(lhs,"|"...)
			}
		}
		if _, has := ni.lhsNames[string(lhs)]; !has {
			ni.lhsNames[string(lhs)] = []parser.Production{}
		}
		ni.lhsNames[string(lhs)] = append(ni.lhsNames[string(lhs)],p)
		for i := 0; i < p.RhsLen(); i++ {
			rhs = append(rhs,p.Rhs(i).Name()...)
			if i < p.RhsLen()-1 {
				rhs = append(rhs,"|"...)
			}
		}
		if _, has := ni.rhsNames[string(rhs)]; !has {
			ni.rhsNames[string(rhs)] = []parser.Production{}
		}
		ni.rhsNames[string(rhs)] = append(ni.rhsNames[string(rhs)],p)
	}
	return nil
}

func (ni *NameIndex) LhsNames(name []string) []parser.Production {
	return ni.lhsNames[strings.Join(name,"|")]
}

func (ni *NameIndex) RhsNames(name []string) []parser.Production {
	return ni.rhsNames[strings.Join(name,"|")]	
}

func (ni *NameIndex) Terminal(name string) parser.GrammarParticle {
	return ni.terminalsByName[name]
}

func (ni *NameIndex) Nonterminal(name string) parser.GrammarParticle {
	return ni.nonterminalsByName[name]
}